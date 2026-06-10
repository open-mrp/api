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

var orderDiscountSvcTracer = tracing.GetTracer("core-service.order_discount_service")

type orderDiscountSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type OrderDiscountSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *OrderDiscountSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("order discount service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("order discount service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("order discount service: tx manager is required")
	}
	return nil
}

func NewOrderDiscountSvc(config *OrderDiscountSvcConfig) domain.OrderDiscountSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &orderDiscountSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *orderDiscountSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *orderDiscountSvcImpl) withTx(ctx context.Context, fn func(context.Context, *orderDiscountSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &orderDiscountSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *orderDiscountSvcImpl) ListOrderDiscounts(ctx context.Context, params domain.ListOrderDiscountsParams) (*domain.ListOrderDiscountsResult, *apierror.APIError) {
	ctx, span := orderDiscountSvcTracer.Start(ctx, "service.order_discount.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkOrderDiscountReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewOrderDiscountRepo().List(ctx, params)
}

func (s *orderDiscountSvcImpl) GetOrderDiscount(ctx context.Context, orderDiscountID string) (*domain.OrderDiscount, *apierror.APIError) {
	ctx, span := orderDiscountSvcTracer.Start(ctx, "service.order_discount.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewOrderDiscountRepo().Get(ctx, domain.GetOrderDiscountParams{
		AccountID:       identity.Target.AccountID,
		OrderDiscountID: orderDiscountID,
	})
}

// BatchGetOrderDiscountsByIDs returns order discounts matching the input IDs
// that the caller's account is authorized to read. Account-scoped.
func (s *orderDiscountSvcImpl) BatchGetOrderDiscountsByIDs(ctx context.Context, ids []string) ([]*domain.OrderDiscount, *apierror.APIError) {
	ctx, span := orderDiscountSvcTracer.Start(ctx, "service.order_discount.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewOrderDiscountRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}

func (s *orderDiscountSvcImpl) CreateOrderDiscount(ctx context.Context, params domain.CreateOrderDiscountParams) (*domain.OrderDiscount, *apierror.APIError) {
	ctx, span := orderDiscountSvcTracer.Start(ctx, "service.order_discount.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	discountID, apiErr := id.GenID(id.OrderDiscountIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.OrderDiscount](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.OrderDiscount
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *orderDiscountSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewOrderDiscountRepo()

			exists, apiErr := txRepo.ExistsByCode(txCtx, params.AccountID, params.Code, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A discount with this code already exists.", "code")
			}

			created, apiErr := txRepo.Create(txCtx, discountID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeOrderDiscount,
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

func (s *orderDiscountSvcImpl) UpdateOrderDiscount(ctx context.Context, params domain.UpdateOrderDiscountParams) (*domain.OrderDiscount, *apierror.APIError) {
	ctx, span := orderDiscountSvcTracer.Start(ctx, "service.order_discount.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.OrderDiscount](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.OrderDiscount
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *orderDiscountSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewOrderDiscountRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetOrderDiscountParams{
				AccountID:       params.AccountID,
				OrderDiscountID: params.OrderDiscountID,
			})
			if apiErr != nil {
				return apiErr
			}

			if params.Code != nil {
				exists, apiErr := txRepo.ExistsByCode(txCtx, params.AccountID, *params.Code, &params.OrderDiscountID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A discount with this code already exists.", "code")
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
				ResourceType: constants.ObjectTypeOrderDiscount,
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

func (s *orderDiscountSvcImpl) DeleteOrderDiscount(ctx context.Context, orderDiscountID string) (*domain.OrderDiscount, *apierror.APIError) {
	ctx, span := orderDiscountSvcTracer.Start(ctx, "service.order_discount.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionDelete); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	orderDiscount, apiErr := s.repos.NewOrderDiscountRepo().Get(ctx, domain.GetOrderDiscountParams{
		AccountID:       identity.Target.AccountID,
		OrderDiscountID: orderDiscountID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeOrderDiscount, orderDiscountID)
			if deletedCheckErr != nil {
				return nil, tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return nil, tracing.Trace(span, apierror.NewAlreadyDeletedError("This order discount has already been deleted and can no longer be modified."))
			}
		}
		return nil, tracing.Trace(span, apiErr)
	}

	var result *domain.OrderDiscount
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *orderDiscountSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeOrderDiscount, orderDiscount.ID, orderDiscount); apiErr != nil {
			return apiErr
		}

		deleted, apiErr := txSvc.repos.NewOrderDiscountRepo().Delete(txCtx, domain.DeleteOrderDiscountParams{
			AccountID:       identity.Target.AccountID,
			OrderDiscountID: orderDiscountID,
		})
		if apiErr != nil {
			return apiErr
		}
		result = deleted

		changes := audit.ComputeChanges(deleted, (*domain.OrderDiscount)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeOrderDiscount,
			ResourceID:   deleted.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

// checkOrderDiscountReadPermission checks the appropriate read permission based on the target context.
// Internal actors need discounts:read for their own account, or customers:read / suppliers:read for external accounts.
func checkOrderDiscountReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionRead)
}

func (s *orderDiscountSvcImpl) FindOrderDiscountByCode(ctx context.Context, params domain.FindOrderDiscountByCodeParams) (*domain.OrderDiscount, *apierror.APIError) {
	ctx, span := orderDiscountSvcTracer.Start(ctx, "service.order_discount.find_by_code")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	// Support both internal and customer actors
	if identity.IsCustomerUser() {
		// Customer can only look up for themselves
		params.AccountID = identity.Target.AccountID
		params.BuyerAccountID = identity.ActorAccountID()
	} else {
		if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		params.AccountID = identity.Target.AccountID
	}

	repo := s.repos.NewOrderDiscountRepo()

	discount, apiErr := repo.FindByCode(ctx, params.AccountID, params.Code)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Check duplicate usage if buyer account ID is provided
	if params.BuyerAccountID != nil {
		isDuplicate, apiErr := repo.CheckDuplicateUsage(ctx, params.AccountID, *params.BuyerAccountID, discount.ID, params.SalesOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if isDuplicate {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Order discount not found."))
		}
	}

	return discount, nil
}
