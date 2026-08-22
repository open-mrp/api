package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

var productionSvcTracer = tracing.GetTracer("core-service.production_service")

type productionSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ProductionSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *ProductionSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("production service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("production service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("production service: tx manager is required")
	}
	return nil
}

func NewProductionSvc(config *ProductionSvcConfig) domain.ProductionSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &productionSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *productionSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *productionSvcImpl) withTx(ctx context.Context, fn func(context.Context, *productionSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &productionSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *productionSvcImpl) GetProduction(ctx context.Context, productionStepID, productionID string) (*domain.Production, *apierror.APIError) {
	ctx, span := productionSvcTracer.Start(ctx, "service.production.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSteps, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	// Verify the step belongs to the account.
	isInAccount, apiErr := s.repos.NewProductionStepQueryRepo().IsInAccount(ctx, accountID, productionStepID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !isInAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Production step not found."))
	}

	return s.repos.NewProductionRepo().Get(ctx, accountID, productionStepID, productionID)
}

func (s *productionSvcImpl) UpdateProduction(ctx context.Context, params domain.UpdateProductionParams) (*domain.Production, *apierror.APIError) {
	ctx, span := productionSvcTracer.Start(ctx, "service.production.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSteps, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// Verify the step belongs to the account.
	isInAccount, apiErr := s.repos.NewProductionStepQueryRepo().IsInAccount(ctx, params.AccountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !isInAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Production step not found."))
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Production](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Production
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionSvcImpl) *apierror.APIError {
			productionRepo := txSvc.repos.NewProductionRepo()

			old, apiErr := productionRepo.Get(txCtx, params.AccountID, params.ProductionStepID, params.ProductionID)
			if apiErr != nil {
				return apiErr
			}

			if params.ItemID != nil {
				if apiErr := productionRepo.UpdateItem(txCtx, params.ProductionID, *params.ItemID); apiErr != nil {
					return apiErr
				}
			}

			if params.QuantityValue != nil && params.QuantityUnitID != nil {
				if apiErr := productionRepo.UpdateQuantity(txCtx, params.ProductionID, *params.QuantityValue, *params.QuantityUnitID); apiErr != nil {
					return apiErr
				}
			}

			fetched, apiErr := productionRepo.Get(txCtx, params.AccountID, params.ProductionStepID, params.ProductionID)
			if apiErr != nil {
				return apiErr
			}
			result = fetched

			changes := audit.ComputeChanges(old, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeProduction,
				ResourceID:   result.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			// Re-link the production flow after item change (must be inside transaction for atomicity).
			if params.ItemID != nil {
				if apiErr := txSvc.mediators().ProductionFlow.LinkFlow(txCtx, params.ProductionStepID, params.AccountID); apiErr != nil {
					return apiErr
				}
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
