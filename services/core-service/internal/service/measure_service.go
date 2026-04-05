package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"

	"github.com/augno/api/services/auth-service/pkg/types"
)

var measureSvcTracer = tracing.GetTracer("core-service.measure_service")

type measureSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type MeasureSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *MeasureSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("measure service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("measure service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("measure service: tx manager is required")
	}
	return nil
}

func NewMeasureSvc(config *MeasureSvcConfig) domain.MeasureSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &measureSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *measureSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *measureSvcImpl) withTx(ctx context.Context, fn func(context.Context, *measureSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &measureSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *measureSvcImpl) checkObjectPermission(identity *types.Identity, objectType constants.ObjectType) *apierror.APIError {
	switch objectType {
	case constants.ObjectTypeItem:
		return identity.CheckHasPermission(types.PermissionDomainItems, types.ActionUpdate)
	case constants.ObjectTypeProductionStep:
		return identity.CheckHasPermission(types.PermissionDomainProductionSteps, types.ActionUpdate)
	default:
		return apierror.NewValidationErrorWithParam("Invalid object type.", "object_type")
	}
}

func (s *measureSvcImpl) verifyObjectExists(ctx context.Context, accountID string, objectID string, objectType constants.ObjectType) *apierror.APIError {
	switch objectType {
	case constants.ObjectTypeItem:
		_, apiErr := s.repos.NewItemRepo().Get(ctx, domain.GetItemParams{
			AccountID: accountID,
			ItemID:    objectID,
		})
		return apiErr
	case constants.ObjectTypeProductionStep:
		_, apiErr := s.repos.NewProductionStepRepo().Get(ctx, accountID, objectID)
		return apiErr
	default:
		return apierror.NewValidationErrorWithParam("Invalid object type.", "object_type")
	}
}

func (s *measureSvcImpl) UpdateQuantity(ctx context.Context, params domain.UpdateQuantityParams) (*domain.Quantity, *apierror.APIError) {
	ctx, span := measureSvcTracer.Start(ctx, "service.measure.update_quantity")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if params.ObjectType != nil {
		if apiErr := s.checkObjectPermission(identity, *params.ObjectType); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	accountID := identity.Target.AccountID

	if params.ObjectID != nil && params.ObjectType != nil {
		if apiErr := s.verifyObjectExists(ctx, accountID, *params.ObjectID, *params.ObjectType); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Quantity](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Quantity
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *measureSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewQuantityRepo()

			old, apiErr := txRepo.Get(txCtx, params.QuantityID)
			if apiErr != nil {
				return apiErr
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeQuantity,
				ResourceID:   updated.ID,
				Changes:      changes,
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

func (s *measureSvcImpl) UpdateRate(ctx context.Context, params domain.UpdateRateParams) (*domain.Rate, *apierror.APIError) {
	ctx, span := measureSvcTracer.Start(ctx, "service.measure.update_rate")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if params.ObjectType != nil {
		if apiErr := s.checkObjectPermission(identity, *params.ObjectType); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	accountID := identity.Target.AccountID

	if params.ObjectID != nil && params.ObjectType != nil {
		if apiErr := s.verifyObjectExists(ctx, accountID, *params.ObjectID, *params.ObjectType); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Rate](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Rate
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *measureSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewRateRepo()

			old, apiErr := txRepo.Get(txCtx, params.RateID)
			if apiErr != nil {
				return apiErr
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeRate,
				ResourceID:   updated.ID,
				Changes:      changes,
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
