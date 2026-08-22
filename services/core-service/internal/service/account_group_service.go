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
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

var accountGroupSvcTracer = tracing.GetTracer("core-service.account_group_service")

type accountGroupSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type AccountGroupSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *AccountGroupSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("account group service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("account group service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("account group service: tx manager is required")
	}
	return nil
}

func NewAccountGroupSvc(config *AccountGroupSvcConfig) domain.AccountGroupSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &accountGroupSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *accountGroupSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *accountGroupSvcImpl) withTx(ctx context.Context, fn func(context.Context, *accountGroupSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &accountGroupSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *accountGroupSvcImpl) ListAccountGroups(ctx context.Context, params domain.ListAccountGroupsParams) (*domain.ListAccountGroupsResult, *apierror.APIError) {
	ctx, span := accountGroupSvcTracer.Start(ctx, "service.account_group.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomerGroups, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewAccountGroupRepo().List(ctx, params)
}

func (s *accountGroupSvcImpl) GetAccountGroup(ctx context.Context, accountGroupID string) (*domain.AccountGroup, *apierror.APIError) {
	ctx, span := accountGroupSvcTracer.Start(ctx, "service.account_group.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomerGroups, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewAccountGroupRepo().Get(ctx, identity.Target.AccountID, accountGroupID)
}

// BatchGetAccountGroupsByIDs returns account groups matching the input IDs that the caller's account is authorized to read. Account groups are always account-scoped (no system rows).
func (s *accountGroupSvcImpl) BatchGetAccountGroupsByIDs(ctx context.Context, ids []string) ([]*domain.AccountGroup, *apierror.APIError) {
	ctx, span := accountGroupSvcTracer.Start(ctx, "service.account_group.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomerGroups, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewAccountGroupRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}

func (s *accountGroupSvcImpl) CreateAccountGroup(ctx context.Context, params domain.CreateAccountGroupParams) (*domain.AccountGroup, *apierror.APIError) {
	ctx, span := accountGroupSvcTracer.Start(ctx, "service.account_group.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomerGroups, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountGroupID, apiErr := id.GenID(id.AccountGroupIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.AccountGroup](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.AccountGroup
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountGroupSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewAccountGroupRepo()

			exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, params.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("An account group with this name already exists.", "name")
			}

			created, apiErr := txRepo.Create(txCtx, accountGroupID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeAccountGroup,
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

func (s *accountGroupSvcImpl) UpdateAccountGroup(ctx context.Context, params domain.UpdateAccountGroupParams) (*domain.AccountGroup, *apierror.APIError) {
	ctx, span := accountGroupSvcTracer.Start(ctx, "service.account_group.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomerGroups, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.AccountGroup](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.AccountGroup
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountGroupSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewAccountGroupRepo()

			old, apiErr := txRepo.Get(txCtx, params.AccountID, params.AccountGroupID)
			if apiErr != nil {
				return apiErr
			}

			params.Description = params.Description.BackfillUnsetPtr(old.Description)
			params.DefaultLeadTimeDays = params.DefaultLeadTimeDays.BackfillUnsetPtr(old.DefaultLeadTimeDays)

			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.AccountGroupID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("An account group with this name already exists.", "name")
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
				ResourceType: constants.ObjectTypeAccountGroup,
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

func (s *accountGroupSvcImpl) DeleteAccountGroup(ctx context.Context, accountGroupID string) *apierror.APIError {
	ctx, span := accountGroupSvcTracer.Start(ctx, "service.account_group.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomerGroups, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	accountGroup, apiErr := s.repos.NewAccountGroupRepo().Get(ctx, accountID, accountGroupID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeAccountGroup, accountGroupID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(
					span,
					apierror.NewAlreadyDeletedError("This account group has already been deleted and can no longer be modified."),
				)
			}
		}
		return tracing.Trace(span, apiErr)
	}

	if apiErr := s.repos.NewAccountGroupRepo().CheckAccountGroupNotInUse(ctx, accountGroup); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountGroupSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewAccountGroupRepo()

		if accountGroup.AccountGroupTypeCode == string(constants.AccountGroupTypePricingGroup) {
			if apiErr := txRepo.DeleteAccountRelationPriceGroupsByAccountGroupID(txCtx, accountGroupID); apiErr != nil {
				return apiErr
			}
		}

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeAccountGroup, accountGroup.ID, accountGroup); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.Delete(txCtx, domain.DeleteAccountGroupParams{
			AccountID:      accountID,
			AccountGroupID: accountGroupID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(accountGroup, (*domain.AccountGroup)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeAccountGroup,
			ResourceID:   accountGroup.ID,
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
