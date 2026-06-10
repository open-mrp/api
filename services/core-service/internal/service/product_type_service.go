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
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var productTypeSvcTracer = tracing.GetTracer("core-service.product_type_service")

type productTypeSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ProductTypeSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *ProductTypeSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("product type service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("product type service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("product type service: tx manager is required")
	}
	return nil
}

func NewProductTypeSvc(config *ProductTypeSvcConfig) domain.ProductTypeSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &productTypeSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *productTypeSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *productTypeSvcImpl) withTx(ctx context.Context, fn func(context.Context, *productTypeSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &productTypeSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *productTypeSvcImpl) ListProductTypes(ctx context.Context, params domain.ListProductTypesParams) (*domain.ListProductTypesResult, *apierror.APIError) {
	ctx, span := productTypeSvcTracer.Start(ctx, "service.product_type.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductTypes, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewProductTypeRepo().List(ctx, params)
}

func (s *productTypeSvcImpl) GetProductType(ctx context.Context, identifier string) (*domain.ProductType, *apierror.APIError) {
	ctx, span := productTypeSvcTracer.Start(ctx, "service.product_type.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductTypes, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewProductTypeRepo().Get(ctx, identifier)
}

// BatchGetProductTypesByIDs returns product types matching the input IDs.
// ProductType is a system-only lookup so no account scoping is required.
func (s *productTypeSvcImpl) BatchGetProductTypesByIDs(ctx context.Context, ids []string) ([]*domain.ProductType, *apierror.APIError) {
	ctx, span := productTypeSvcTracer.Start(ctx, "service.product_type.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductTypes, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewProductTypeRepo().GetByIDs(ctx, ids)
}

func (s *productTypeSvcImpl) CreateProductType(ctx context.Context, params domain.CreateProductTypeParams) (*domain.ProductType, *apierror.APIError) {
	ctx, span := productTypeSvcTracer.Start(ctx, "service.product_type.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductTypes, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	productTypeID, apiErr := id.GenID(id.ProductTypeIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductType](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ProductType
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productTypeSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewProductTypeRepo()

			exists, apiErr := txRepo.ExistsByName(txCtx, params.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A product type with this name already exists.", "name")
			}

			exists, apiErr = txRepo.ExistsByCode(txCtx, params.Code, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A product type with this code already exists.", "code")
			}

			created, apiErr := txRepo.Create(txCtx, productTypeID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeProductType,
				ResourceID:   created.ID,
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

func (s *productTypeSvcImpl) UpdateProductType(ctx context.Context, params domain.UpdateProductTypeParams) (*domain.ProductType, *apierror.APIError) {
	ctx, span := productTypeSvcTracer.Start(ctx, "service.product_type.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductTypes, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductType](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ProductType
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productTypeSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewProductTypeRepo()

			old, apiErr := txRepo.Get(txCtx, params.ProductTypeID)
			if apiErr != nil {
				return apiErr
			}

			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, *params.Name, &params.ProductTypeID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A product type with this name already exists.", "name")
				}
			}

			if params.Code != nil {
				exists, apiErr := txRepo.ExistsByCode(txCtx, *params.Code, &params.ProductTypeID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A product type with this code already exists.", "code")
				}
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
				ResourceType: constants.ObjectTypeProductType,
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

func (s *productTypeSvcImpl) DeleteProductType(ctx context.Context, productTypeID string) *apierror.APIError {
	ctx, span := productTypeSvcTracer.Start(ctx, "service.product_type.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductTypes, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	productType, apiErr := s.repos.NewProductTypeRepo().Get(ctx, productTypeID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeProductType, productTypeID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This product type has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productTypeSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeProductType, productType.ID, productType); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewProductTypeRepo().Delete(txCtx, productTypeID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(productType, (*domain.ProductType)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeProductType,
			ResourceID:   productType.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
