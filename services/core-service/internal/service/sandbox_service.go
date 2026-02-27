package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var sandboxSvcTracer = tracing.GetTracer("core-service.sandbox_service")

type sandboxSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type SandboxSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *SandboxSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("sandbox service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("sandbox service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("sandbox service: tx manager is required")
	}
	return nil
}

func NewSandboxSvc(config *SandboxSvcConfig) domain.SandboxSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &sandboxSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *sandboxSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *sandboxSvcImpl) withTx(ctx context.Context, fn func(context.Context, *sandboxSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &sandboxSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *sandboxSvcImpl) GetSandboxAccountByOwner(ctx context.Context, ownerAccountID string) (string, *apierror.APIError) {
	ctx, span := sandboxSvcTracer.Start(ctx, "service.sandbox.get_sandbox_account_by_owner")
	defer span.End()

	return s.repos.NewSandboxAccountRepo().FindFirstByOwnerAccountID(ctx, ownerAccountID)
}

func (s *sandboxSvcImpl) ListSandboxAccounts(ctx context.Context, cursor *string, limit int32) (*domain.ListSandboxAccountsResult, *apierror.APIError) {
	ctx, span := sandboxSvcTracer.Start(ctx, "service.sandbox.list_sandbox_accounts")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckNotSandboxMode(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckHasPermission(identity, types.PermissionDomainSandbox, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if identity.TargetAccountID == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Target account ID is required."))
	}

	return s.repos.NewSandboxAccountRepo().List(ctx, *identity.TargetAccountID, cursor, limit)
}

func (s *sandboxSvcImpl) CreateSandbox(ctx context.Context, name string, mode constants.SandboxMode) (*domain.SandboxAccount, *apierror.APIError) {
	ctx, span := sandboxSvcTracer.Start(ctx, "service.sandbox.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckNotSandboxMode(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckHasPermission(identity, types.PermissionDomainSandbox, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if identity.TargetAccountID == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Target account ID is required."))
	}

	ownerAccountID := *identity.TargetAccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, &domain.RequestIdentity{
		ActorID:      identity.Actor.ID,
		IdentityType: identity.Type,
	})
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.SandboxAccount](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.SandboxAccount
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *sandboxSvcImpl) *apierror.APIError {
			txMeds := txSvc.mediators()

			sandbox, createErr := txMeds.Sandbox.Create(txCtx, ownerAccountID, identity.Actor.ID, name)
			if createErr != nil {
				return createErr
			}

			result = sandbox

			if mode == constants.SandboxModeSeeded {
				payloadJSON, err := json.Marshal(map[string]string{"account_id": sandbox.AccountID})
				if err != nil {
					return apierror.NewInternalError(err, "Failed to marshal seed payload.")
				}

				msg := contracts.AmqpMessage{
					Data:     payloadJSON,
					Identity: identity,
				}
				if requestID, ok := appctx.GetRequestID(txCtx); ok {
					msg.RequestID = requestID
				}

				if _, err := txSvc.repos.NewOutboxRepo().Create(txCtx, messaging.OutboxMessageInput{
					ServiceName: domain.ServiceName,
					MessageType: string(contracts.CoreCmdSeedSandbox),
					Destination: messaging.ApplicationExchange,
					RoutingKey:  string(contracts.CoreCmdSeedSandbox),
					Payload:     msg,
				}); err != nil {
					return apierror.NewInternalError(err, "Failed to create seed outbox message.")
				}
			}

			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *sandboxSvcImpl) GetSandbox(ctx context.Context, sandboxTypeID string) (*domain.SandboxAccount, *apierror.APIError) {
	ctx, span := sandboxSvcTracer.Start(ctx, "service.sandbox.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckNotSandboxMode(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckHasPermission(identity, types.PermissionDomainSandbox, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if identity.TargetAccountID == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Target account ID is required."))
	}

	sandbox, apiErr := s.repos.NewSandboxAccountRepo().FindByTypeID(ctx, sandboxTypeID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if sandbox.OwnerAccountID != *identity.TargetAccountID {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Sandbox not found."))
	}

	return sandbox, nil
}

func (s *sandboxSvcImpl) DeleteSandbox(ctx context.Context, sandboxTypeID string) *apierror.APIError {
	ctx, span := sandboxSvcTracer.Start(ctx, "service.sandbox.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckNotSandboxMode(identity); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckHasPermission(identity, types.PermissionDomainSandbox, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if identity.TargetAccountID == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Target account ID is required."))
	}

	ownerAccountID := *identity.TargetAccountID

	var accountID string
	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *sandboxSvcImpl) *apierror.APIError {
		txMeds := txSvc.mediators()

		deletedAccountID, deleteErr := txMeds.Sandbox.Delete(txCtx, ownerAccountID, sandboxTypeID)
		if deleteErr != nil {
			return deleteErr
		}
		accountID = deletedAccountID

		payloadData := map[string]string{"account_id": accountID}
		payloadJSON, err := json.Marshal(payloadData)
		if err != nil {
			return apierror.NewInternalError(err, "Failed to marshal purge payload.")
		}

		msg := contracts.AmqpMessage{
			Data:     payloadJSON,
			Identity: identity,
		}
		if requestID, ok := appctx.GetRequestID(txCtx); ok {
			msg.RequestID = requestID
		}

		outboxInput := messaging.OutboxMessageInput{
			ServiceName: domain.ServiceName,
			MessageType: string(contracts.CoreCmdPurgeAccountData),
			Destination: messaging.ApplicationExchange,
			RoutingKey:  string(contracts.CoreCmdPurgeAccountData),
			Payload:     msg,
		}

		if _, err := txSvc.repos.NewOutboxRepo().Create(txCtx, outboxInput); err != nil {
			return apierror.NewInternalError(err, "Failed to create outbox message.")
		}

		return nil
	})

	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
