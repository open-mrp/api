package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
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

// GetSandboxAccountByOwner returns the first sandbox account ID owned by the given production account.
//
// 1. Query the sandbox account repository for the first sandbox matching the owner account ID.
func (s *sandboxSvcImpl) GetSandboxAccountByOwner(ctx context.Context, ownerAccountID string) (string, *apierror.APIError) {
	ctx, span := sandboxSvcTracer.Start(ctx, "service.sandbox.get_sandbox_account_by_owner")
	defer span.End()

	return s.repos.NewSandboxAccountRepo().FindFirstByOwnerAccountID(ctx, ownerAccountID)
}

// ListSandboxAccounts returns a paginated list of sandbox accounts for the caller's production account.
//
// 1. Extract and validate the caller's identity, actor type, sandbox:read permission, and non-sandbox mode.
// 2. Require the Augno-Account header.
// 3. Query the sandbox account repository with pagination and optional includes.
func (s *sandboxSvcImpl) ListSandboxAccounts(ctx context.Context, cursor *string, limit int32, query *string, includes []string) (*domain.ListSandboxAccountsResult, *apierror.APIError) {
	ctx, span := sandboxSvcTracer.Start(ctx, "service.sandbox.list_sandbox_accounts")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckNotSandboxMode(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSandbox, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewSandboxAccountRepo().List(ctx, identity.Target.AccountID, cursor, limit, query, includes)
}

// CreateSandbox provisions a new sandbox account for the caller's production account, with idempotency support.
//
// 1. Extract and validate the caller's identity, actor type, sandbox:create permission, and non-sandbox mode.
// 2. Upsert an idempotency key; if already finished, return the cached response.
// 3. Within a transaction, delegate to the sandbox mediator to create the sandbox account.
// 4. If the mode is "seeded", enqueue a seed-data message for async population.
// 5. Cache the success response for idempotent replay.
func (s *sandboxSvcImpl) CreateSandbox(ctx context.Context, name string, mode constants.SandboxMode) (*domain.SandboxAccount, *apierror.APIError) {
	ctx, span := sandboxSvcTracer.Start(ctx, "service.sandbox.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckNotSandboxMode(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSandbox, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if strings.TrimSpace(name) == "" {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Name is required.", "name"))
	}

	ownerAccountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
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

			changes := audit.ComputeChanges(nil, sandbox)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeSandbox,
				ResourceID:   sandbox.TypeID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

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

// GetSandbox retrieves a single sandbox by type ID, verifying ownership against the caller's account.
//
// 1. Extract and validate the caller's identity, actor type, sandbox:read permission, and non-sandbox mode.
// 2. Fetch the sandbox by type ID from the repository.
// 3. Verify the sandbox belongs to the caller's account; return not-found if ownership mismatches.
func (s *sandboxSvcImpl) GetSandbox(ctx context.Context, sandboxTypeID string, includes []string) (*domain.SandboxAccount, *apierror.APIError) {
	ctx, span := sandboxSvcTracer.Start(ctx, "service.sandbox.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckNotSandboxMode(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSandbox, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	sandbox, apiErr := s.repos.NewSandboxAccountRepo().FindByTypeID(ctx, sandboxTypeID, includes)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if sandbox.OwnerAccountID != identity.Target.AccountID {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Sandbox not found."))
	}

	return sandbox, nil
}

// DeleteSandbox removes a sandbox account and enqueues a purge message for async data cleanup.
//
// 1. Extract and validate the caller's identity, actor type, sandbox:delete permission, and non-sandbox mode.
// 2. Within a transaction, delegate to the sandbox mediator to delete the sandbox and its account record.
// 3. Enqueue a purge-account-data message via the outbox for downstream cleanup.
func (s *sandboxSvcImpl) DeleteSandbox(ctx context.Context, sandboxTypeID string) *apierror.APIError {
	ctx, span := sandboxSvcTracer.Start(ctx, "service.sandbox.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckNotSandboxMode(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSandbox, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	ownerAccountID := identity.Target.AccountID

	sandbox, apiErr := s.repos.NewSandboxAccountRepo().FindByTypeID(ctx, sandboxTypeID, nil)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeSandbox, sandboxTypeID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This sandbox has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}
	if sandbox.OwnerAccountID != ownerAccountID {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Sandbox not found."))
	}

	var accountID string
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *sandboxSvcImpl) *apierror.APIError {
		txMeds := txSvc.mediators()

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeSandbox, sandbox.TypeID, sandbox); apiErr != nil {
			return apiErr
		}

		deletedAccountID, deleteErr := txMeds.Sandbox.Delete(txCtx, ownerAccountID, sandboxTypeID)
		if deleteErr != nil {
			return deleteErr
		}
		accountID = deletedAccountID

		changes := audit.ComputeChanges(sandbox, (*domain.SandboxAccount)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeSandbox,
			ResourceID:   sandbox.TypeID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

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
