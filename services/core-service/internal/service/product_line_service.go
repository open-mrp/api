package service

import (
	"context"
	"fmt"
	"slices"

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

var productLineSvcTracer = tracing.GetTracer("core-service.product_line_service")

type productLineSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ProductLineSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *ProductLineSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("product line service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("product line service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("product line service: tx manager is required")
	}
	return nil
}

func NewProductLineSvc(config *ProductLineSvcConfig) domain.ProductLineSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &productLineSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *productLineSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *productLineSvcImpl) withTx(ctx context.Context, fn func(context.Context, *productLineSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &productLineSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *productLineSvcImpl) ListProductLines(ctx context.Context, params domain.ListProductLinesParams) (*domain.ListProductLinesResult, *apierror.APIError) {
	ctx, span := productLineSvcTracer.Start(ctx, "service.product_line.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsInternalActor() {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainProductLines, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	} else if !identity.IsCustomerUser() && !identity.IsSupplierUser() {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You do not have access to this resource."))
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewProductLineRepo()

	result, apiErr := repo.List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if slices.Contains(params.Includes, "unit_group") {
		for _, pl := range result.ProductLines {
			unitGroup, apiErr := repo.GetUnitGroup(ctx, pl.UnitGroupID, params.Includes)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			pl.UnitGroup = unitGroup
		}
	}

	return result, nil
}

func (s *productLineSvcImpl) GetProductLine(ctx context.Context, params domain.GetProductLineParams) (*domain.ProductLineFull, *apierror.APIError) {
	ctx, span := productLineSvcTracer.Start(ctx, "service.product_line.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsInternalActor() {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainProductLines, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	} else if !identity.IsCustomerUser() && !identity.IsSupplierUser() {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You do not have access to this resource."))
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewProductLineRepo()

	productLine, apiErr := repo.Get(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if slices.Contains(params.Includes, "unit_group") {
		unitGroup, apiErr := repo.GetUnitGroup(ctx, productLine.UnitGroupID, params.Includes)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		productLine.UnitGroup = unitGroup
	}

	return productLine, nil
}

func (s *productLineSvcImpl) CreateProductLine(ctx context.Context, params domain.CreateProductLineParams) (*domain.ProductLineFull, *apierror.APIError) {
	ctx, span := productLineSvcTracer.Start(ctx, "service.product_line.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductLines, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	productLineID, apiErr := id.GenID(id.ProductLineIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductLineFull](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ProductLineFull
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productLineSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewProductLineRepo()

			exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, params.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A product line with this name already exists.", "name")
			}

			created, apiErr := txRepo.Create(txCtx, productLineID, params)
			if apiErr != nil {
				return apiErr
			}

			if slices.Contains(params.Includes, "unit_group") {
				unitGroup, apiErr := txRepo.GetUnitGroup(txCtx, created.UnitGroupID, params.Includes)
				if apiErr != nil {
					return apiErr
				}
				created.UnitGroup = unitGroup
			}

			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeProductLine,
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

func (s *productLineSvcImpl) UpdateProductLine(ctx context.Context, params domain.UpdateProductLineParams) (*domain.ProductLineFull, *apierror.APIError) {
	ctx, span := productLineSvcTracer.Start(ctx, "service.product_line.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductLines, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if domain.IsDefaultProductLine(params.ProductLineID) {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("Default product lines cannot be updated."))
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductLineFull](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ProductLineFull
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productLineSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewProductLineRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetProductLineParams{
				AccountID:     params.AccountID,
				ProductLineID: params.ProductLineID,
			})
			if apiErr != nil {
				return apiErr
			}

			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.ProductLineID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A product line with this name already exists.", "name")
				}
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}

			if slices.Contains(params.Includes, "unit_group") {
				unitGroup, apiErr := txRepo.GetUnitGroup(txCtx, updated.UnitGroupID, params.Includes)
				if apiErr != nil {
					return apiErr
				}
				updated.UnitGroup = unitGroup
			}

			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeProductLine,
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

func (s *productLineSvcImpl) DeleteProductLine(ctx context.Context, productLineID string) *apierror.APIError {
	ctx, span := productLineSvcTracer.Start(ctx, "service.product_line.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductLines, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if domain.IsDefaultProductLine(productLineID) {
		return tracing.Trace(span, apierror.NewAuthorizationError("Default product lines cannot be deleted."))
	}

	accountID := identity.Target.AccountID

	productLine, apiErr := s.repos.NewProductLineRepo().Get(ctx, domain.GetProductLineParams{
		AccountID:     accountID,
		ProductLineID: productLineID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeProductLine, productLineID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This product line has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productLineSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeProductLine, productLine.ID, productLine); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewProductLineRepo().Delete(txCtx, domain.DeleteProductLineParams{
			AccountID:     accountID,
			ProductLineID: productLineID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(productLine, (*domain.ProductLineFull)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeProductLine,
			ResourceID:   productLine.ID,
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

func (s *productLineSvcImpl) BatchGetProductLinesByIDs(ctx context.Context, ids []string) ([]*domain.ProductLineFull, *apierror.APIError) {
	ctx, span := productLineSvcTracer.Start(ctx, "service.product_line.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}
	if identity.IsInternalActor() {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainProductLines, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	} else if !identity.IsCustomerUser() && !identity.IsSupplierUser() {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You do not have access to this resource."))
	}
	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewProductLineRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}
