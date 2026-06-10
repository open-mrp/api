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

var customerProductLineAccessSvcTracer = tracing.GetTracer("core-service.customer_product_line_access_service")

type customerProductLineAccessSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type CustomerProductLineAccessSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *CustomerProductLineAccessSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("customer product line access service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("customer product line access service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("customer product line access service: tx manager is required")
	}
	return nil
}

func NewCustomerProductLineAccessSvc(config *CustomerProductLineAccessSvcConfig) domain.CustomerProductLineAccessSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &customerProductLineAccessSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *customerProductLineAccessSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *customerProductLineAccessSvcImpl) withTx(ctx context.Context, fn func(context.Context, *customerProductLineAccessSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &customerProductLineAccessSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *customerProductLineAccessSvcImpl) ListCustomerProductLineAccess(ctx context.Context, params domain.ListCustomerProductLineAccessParams) (*domain.ListCustomerProductLineAccessResult, *apierror.APIError) {
	ctx, span := customerProductLineAccessSvcTracer.Start(ctx, "service.customer_product_line_access.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductLineAccess, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewCustomerProductLineAccessRepo().List(ctx, params)
}

func (s *customerProductLineAccessSvcImpl) GetCustomerProductLineAccess(ctx context.Context, customerID string) (*domain.CustomerProductLineAccess, *apierror.APIError) {
	ctx, span := customerProductLineAccessSvcTracer.Start(ctx, "service.customer_product_line_access.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductLineAccess, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	return s.repos.NewCustomerProductLineAccessRepo().Get(ctx, identity.Target.AccountID, customerID)
}

// BatchGetCustomerProductLineAccessByIDs returns access records for each given
// customer_id. Loop over Get (same approach as AccountGroupProductLineAccess
// — the underlying SQL shape is awkward to batch and include-resolution
// batch sizes are small).
func (s *customerProductLineAccessSvcImpl) BatchGetCustomerProductLineAccessByIDs(ctx context.Context, customerIDs []string) ([]*domain.CustomerProductLineAccess, *apierror.APIError) {
	ctx, span := customerProductLineAccessSvcTracer.Start(ctx, "service.customer_product_line_access.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductLineAccess, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}
	if len(customerIDs) == 0 {
		return nil, nil
	}
	repo := s.repos.NewCustomerProductLineAccessRepo()
	out := make([]*domain.CustomerProductLineAccess, 0, len(customerIDs))
	for _, id := range customerIDs {
		access, apiErr := repo.Get(ctx, identity.Target.AccountID, id)
		if apiErr != nil {
			if apierror.IsNotFound(apiErr) {
				continue
			}
			return nil, tracing.Trace(span, apiErr)
		}
		out = append(out, access)
	}
	return out, nil
}

func (s *customerProductLineAccessSvcImpl) CreateCustomerProductLineAccess(ctx context.Context, params domain.CreateCustomerProductLineAccessParams) (*domain.CustomerProductLineAccess, *apierror.APIError) {
	ctx, span := customerProductLineAccessSvcTracer.Start(ctx, "service.customer_product_line_access.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductLineAccess, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.CustomerProductLineAccess](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.CustomerProductLineAccess
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *customerProductLineAccessSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewCustomerProductLineAccessRepo()

			exists, apiErr := txRepo.ExistsByCustomerID(txCtx, params.AccountID, params.CustomerID)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("Product line access for this customer already exists.", "customer_id")
			}

			created, apiErr := txRepo.Create(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeCustomerProductLineAccess,
				ResourceID:   created.CustomerID,
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

func (s *customerProductLineAccessSvcImpl) UpdateCustomerProductLineAccess(ctx context.Context, params domain.UpdateCustomerProductLineAccessParams) (*domain.CustomerProductLineAccess, *apierror.APIError) {
	ctx, span := customerProductLineAccessSvcTracer.Start(ctx, "service.customer_product_line_access.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductLineAccess, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.CustomerProductLineAccess](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.CustomerProductLineAccess
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *customerProductLineAccessSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewCustomerProductLineAccessRepo()

			old, apiErr := txRepo.Get(txCtx, params.AccountID, params.CustomerID)
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
				ResourceType: constants.ObjectTypeCustomerProductLineAccess,
				ResourceID:   updated.CustomerID,
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

func (s *customerProductLineAccessSvcImpl) DeleteCustomerProductLineAccess(ctx context.Context, customerID string) *apierror.APIError {
	ctx, span := customerProductLineAccessSvcTracer.Start(ctx, "service.customer_product_line_access.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductLineAccess, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	accountID := identity.Target.AccountID

	existing, apiErr := s.repos.NewCustomerProductLineAccessRepo().Get(ctx, accountID, customerID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeCustomerProductLineAccess, customerID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(
					span,
					apierror.NewAlreadyDeletedError("This customer product line access has already been deleted and can no longer be modified."),
				)
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *customerProductLineAccessSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeCustomerProductLineAccess, existing.CustomerID, existing); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewCustomerProductLineAccessRepo().Delete(txCtx, accountID, customerID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(existing, (*domain.CustomerProductLineAccess)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeCustomerProductLineAccess,
			ResourceID:   existing.CustomerID,
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
