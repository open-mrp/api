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

var accountGroupProductLineAccessSvcTracer = tracing.GetTracer("core-service.account_group_product_line_access_service")

type accountGroupProductLineAccessSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type AccountGroupProductLineAccessSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *AccountGroupProductLineAccessSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("account group product line access service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("account group product line access service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("account group product line access service: tx manager is required")
	}
	return nil
}

func NewAccountGroupProductLineAccessSvc(config *AccountGroupProductLineAccessSvcConfig) domain.AccountGroupProductLineAccessSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &accountGroupProductLineAccessSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *accountGroupProductLineAccessSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *accountGroupProductLineAccessSvcImpl) withTx(ctx context.Context, fn func(context.Context, *accountGroupProductLineAccessSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &accountGroupProductLineAccessSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *accountGroupProductLineAccessSvcImpl) ListAccountGroupProductLineAccess(ctx context.Context, params domain.ListAccountGroupProductLineAccessParams) (*domain.ListAccountGroupProductLineAccessResult, *apierror.APIError) {
	ctx, span := accountGroupProductLineAccessSvcTracer.Start(ctx, "service.account_group_product_line_access.list")
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

	params.AccountID = identity.Target.AccountID

	return s.repos.NewAccountGroupProductLineAccessRepo().List(ctx, params)
}

func (s *accountGroupProductLineAccessSvcImpl) GetAccountGroupProductLineAccess(ctx context.Context, accountGroupID string) (*domain.AccountGroupProductLineAccess, *apierror.APIError) {
	ctx, span := accountGroupProductLineAccessSvcTracer.Start(ctx, "service.account_group_product_line_access.get")
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

	return s.repos.NewAccountGroupProductLineAccessRepo().Get(ctx, identity.Target.AccountID, accountGroupID)
}

// BatchGetAccountGroupProductLineAccessByIDs returns access records for each given account_group_id. Implemented as a loop over Get since the underlying shape (one record per group, with N product lines) is awkward to express in a single SQL query; batch sizes for include resolution are small.
func (s *accountGroupProductLineAccessSvcImpl) BatchGetAccountGroupProductLineAccessByIDs(ctx context.Context, accountGroupIDs []string) ([]*domain.AccountGroupProductLineAccess, *apierror.APIError) {
	ctx, span := accountGroupProductLineAccessSvcTracer.Start(ctx, "service.account_group_product_line_access.batch_get_by_ids")
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
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}
	if len(accountGroupIDs) == 0 {
		return nil, nil
	}
	repo := s.repos.NewAccountGroupProductLineAccessRepo()
	out := make([]*domain.AccountGroupProductLineAccess, 0, len(accountGroupIDs))
	for _, id := range accountGroupIDs {
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

func (s *accountGroupProductLineAccessSvcImpl) CreateAccountGroupProductLineAccess(ctx context.Context, params domain.CreateAccountGroupProductLineAccessParams) (*domain.AccountGroupProductLineAccess, *apierror.APIError) {
	ctx, span := accountGroupProductLineAccessSvcTracer.Start(ctx, "service.account_group_product_line_access.create")
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

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.AccountGroupProductLineAccess](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.AccountGroupProductLineAccess
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountGroupProductLineAccessSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewAccountGroupProductLineAccessRepo()

			exists, apiErr := txRepo.ExistsByAccountGroupID(txCtx, params.AccountGroupID)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("Product line access for this account group already exists.", "account_group_id")
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
				ResourceType: constants.ObjectTypeAccountGroupProductLineAccess,
				ResourceID:   created.AccountGroupID,
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

func (s *accountGroupProductLineAccessSvcImpl) UpdateAccountGroupProductLineAccess(ctx context.Context, params domain.UpdateAccountGroupProductLineAccessParams) (*domain.AccountGroupProductLineAccess, *apierror.APIError) {
	ctx, span := accountGroupProductLineAccessSvcTracer.Start(ctx, "service.account_group_product_line_access.update")
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

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.AccountGroupProductLineAccess](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.AccountGroupProductLineAccess
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountGroupProductLineAccessSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewAccountGroupProductLineAccessRepo()

			old, apiErr := txRepo.Get(txCtx, params.AccountID, params.AccountGroupID)
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
				ResourceType: constants.ObjectTypeAccountGroupProductLineAccess,
				ResourceID:   updated.AccountGroupID,
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

func (s *accountGroupProductLineAccessSvcImpl) DeleteAccountGroupProductLineAccess(ctx context.Context, accountGroupID string) *apierror.APIError {
	ctx, span := accountGroupProductLineAccessSvcTracer.Start(ctx, "service.account_group_product_line_access.delete")
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

	accountID := identity.Target.AccountID

	existing, apiErr := s.repos.NewAccountGroupProductLineAccessRepo().Get(ctx, accountID, accountGroupID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeAccountGroupProductLineAccess, accountGroupID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This account group product line access has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountGroupProductLineAccessSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeAccountGroupProductLineAccess, existing.AccountGroupID, existing); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewAccountGroupProductLineAccessRepo().Delete(txCtx, accountID, accountGroupID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(existing, (*domain.AccountGroupProductLineAccess)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeAccountGroupProductLineAccess,
			ResourceID:   existing.AccountGroupID,
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
