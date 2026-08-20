package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var pickLineSvcTracer = tracing.GetTracer("core-service.pick_line_service")

type pickLineSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type PickLineSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *PickLineSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("pick line service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("pick line service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("pick line service: tx manager is required")
	}
	return nil
}

func NewPickLineSvc(config *PickLineSvcConfig) domain.PickLineSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &pickLineSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *pickLineSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *pickLineSvcImpl) withTx(ctx context.Context, fn func(context.Context, *pickLineSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &pickLineSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *pickLineSvcImpl) UpdatePickLine(ctx context.Context, params domain.UpdatePickLineParams) (*domain.PickLine, *apierror.APIError) {
	ctx, span := pickLineSvcTracer.Start(ctx, "service.pick_line.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPicks, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	pickRepo := s.repos.NewPickRepo()
	pickLineRepo := s.repos.NewPickLineRepo()

	// Verify pick is in account
	inAccount, apiErr := pickRepo.IsInAccount(ctx, params.AccountID, params.PickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Pick not found."))
	}

	// Verify line is in pick
	inPick, apiErr := pickLineRepo.IsInPick(ctx, params.PickLineID, params.PickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inPick {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Pick line not found."))
	}

	// Resolve the parent sales order so the audit event can be scoped to the order's history tree.
	rootSalesOrder, apiErr := pickRepo.GetSalesOrderForPick(ctx, params.AccountID, params.PickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.PickLine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.PickLine
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *pickLineSvcImpl) *apierror.APIError {
			txPickLineRepo := txSvc.repos.NewPickLineRepo()

			old, apiErr := txPickLineRepo.Get(txCtx, params.PickLineID)
			if apiErr != nil {
				return apiErr
			}

			if params.QuantityValue != nil {
				if apiErr := txPickLineRepo.UpdateQuantity(txCtx, params.PickLineID, params.QuantityValue, nil); apiErr != nil {
					return apiErr
				}
			}

			pickLine, apiErr := txPickLineRepo.Get(txCtx, params.PickLineID)
			if apiErr != nil {
				return apiErr
			}
			result = pickLine

			changes := audit.ComputeChanges(old, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionUpdate,
				ResourceType:     constants.ObjectTypePickLine,
				ResourceID:       result.ID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   rootSalesOrder.ID,
				Changes:          changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *pickLineSvcImpl) PickPickLine(ctx context.Context, pickID, pickLineID string) (*domain.PickLine, *apierror.APIError) {
	ctx, span := pickLineSvcTracer.Start(ctx, "service.pick_line.pick")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPicks, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	pickRepo := s.repos.NewPickRepo()
	pickLineRepo := s.repos.NewPickLineRepo()

	// Verify pick is in account
	inAccount, apiErr := pickRepo.IsInAccount(ctx, accountID, pickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Pick not found."))
	}

	// Verify line is in pick
	inPick, apiErr := pickLineRepo.IsInPick(ctx, pickLineID, pickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inPick {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Pick line not found."))
	}

	old, apiErr := pickLineRepo.Get(ctx, pickLineID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Resolve the parent sales order so the audit event can be scoped to the order's history tree.
	rootSalesOrder, apiErr := pickRepo.GetSalesOrderForPick(ctx, accountID, pickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.PickLine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.PickLine
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *pickLineSvcImpl) *apierror.APIError {
			txPickLineRepo := txSvc.repos.NewPickLineRepo()

			if apiErr := txPickLineRepo.PickRemainingQuantity(txCtx, pickLineID); apiErr != nil {
				return apiErr
			}

			pickLine, apiErr := txPickLineRepo.Get(txCtx, pickLineID)
			if apiErr != nil {
				return apiErr
			}
			result = pickLine

			changes := audit.ComputeChanges(old, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionUpdate,
				ResourceType:     constants.ObjectTypePickLine,
				ResourceID:       result.ID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   rootSalesOrder.ID,
				Changes:          changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *pickLineSvcImpl) VoidPickLine(ctx context.Context, pickID, pickLineID string) (*domain.PickLine, *apierror.APIError) {
	ctx, span := pickLineSvcTracer.Start(ctx, "service.pick_line.void")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPicks, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	pickRepo := s.repos.NewPickRepo()
	pickLineRepo := s.repos.NewPickLineRepo()

	// Verify pick is in account
	inAccount, apiErr := pickRepo.IsInAccount(ctx, accountID, pickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Pick not found."))
	}

	// Verify line is in pick
	inPick, apiErr := pickLineRepo.IsInPick(ctx, pickLineID, pickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inPick {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Pick line not found."))
	}

	// Get the pick line to check if it's already packed
	old, apiErr := pickLineRepo.Get(ctx, pickLineID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if old.PackedAt != nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("Cannot void a pick line that has already been packed."))
	}

	// Resolve the parent sales order so the audit event can be scoped to the order's history tree.
	rootSalesOrder, apiErr := pickRepo.GetSalesOrderForPick(ctx, accountID, pickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.PickLine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.PickLine
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *pickLineSvcImpl) *apierror.APIError {
			txPickLineRepo := txSvc.repos.NewPickLineRepo()

			if apiErr := txPickLineRepo.VoidLine(txCtx, pickLineID); apiErr != nil {
				return apiErr
			}

			pickLine, apiErr := txPickLineRepo.Get(txCtx, pickLineID)
			if apiErr != nil {
				return apiErr
			}
			result = pickLine

			changes := audit.ComputeChanges(old, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionUpdate,
				ResourceType:     constants.ObjectTypePickLine,
				ResourceID:       result.ID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   rootSalesOrder.ID,
				Changes:          changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}
