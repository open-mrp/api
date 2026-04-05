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

var roleSvcTracer = tracing.GetTracer("core-service.role_service")

type roleSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type RoleSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *RoleSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("role service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("role service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("role service: tx manager is required")
	}
	return nil
}

func NewRoleSvc(config *RoleSvcConfig) domain.RoleSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &roleSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *roleSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *roleSvcImpl) withTx(ctx context.Context, fn func(context.Context, *roleSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &roleSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *roleSvcImpl) ListRoles(ctx context.Context, params domain.ListRolesParams) (*domain.ListRolesResult, *apierror.APIError) {
	ctx, span := roleSvcTracer.Start(ctx, "service.role.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainRoles, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	page, apiErr := s.repos.NewRoleRepo().List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Batch-fetch permissions for all roles in the page.
	roleIDs := make([]string, len(page.Roles))
	for i, r := range page.Roles {
		roleIDs[i] = r.ID
	}

	permsByRole, apiErr := s.repos.NewRolePermissionRepo().ListByRoleIDs(ctx, roleIDs)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rolesWithPerms := make([]*domain.RoleWithPermissions, len(page.Roles))
	for i, r := range page.Roles {
		rolesWithPerms[i] = &domain.RoleWithPermissions{
			Role:        *r,
			Permissions: permsByRole[r.ID],
		}
	}

	return &domain.ListRolesResult{Roles: rolesWithPerms, PageInfo: page.PageInfo}, nil
}

func (s *roleSvcImpl) GetRole(ctx context.Context, roleID string) (*domain.RoleWithPermissions, *apierror.APIError) {
	ctx, span := roleSvcTracer.Start(ctx, "service.role.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainRoles, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	role, apiErr := s.repos.NewRoleRepo().Get(ctx, roleID, identity.Target.AccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	permissions, apiErr := s.repos.NewRolePermissionRepo().ListByRoleID(ctx, roleID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.RoleWithPermissions{
		Role:        *role,
		Permissions: permissions,
	}, nil
}

func (s *roleSvcImpl) CreateRole(ctx context.Context, params domain.CreateRoleParams) (*domain.RoleWithPermissions, *apierror.APIError) {
	ctx, span := roleSvcTracer.Start(ctx, "service.role.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainRoles, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	roleID, apiErr := id.GenID(id.RoleIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.RoleWithPermissions](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.RoleWithPermissions
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *roleSvcImpl) *apierror.APIError {
			txRoleRepo := txSvc.repos.NewRoleRepo()
			txRolePermRepo := txSvc.repos.NewRolePermissionRepo()

			exists, apiErr := txRoleRepo.ExistsByName(txCtx, params.AccountID, params.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam(fmt.Sprintf("A role with the name '%s' already exists.", params.Name), "name")
			}

			if apiErr := txRoleRepo.Create(txCtx, roleID, params); apiErr != nil {
				return apiErr
			}

			for _, input := range params.Permissions {
				permID, apiErr := id.GenID(id.RolePermissionIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}

				if apiErr := txRolePermRepo.Create(txCtx, permID, roleID, input); apiErr != nil {
					return apiErr
				}
			}

			role, apiErr := txRoleRepo.Get(txCtx, roleID, params.AccountID)
			if apiErr != nil {
				return apiErr
			}

			permissions, apiErr := txRolePermRepo.ListByRoleID(txCtx, roleID)
			if apiErr != nil {
				return apiErr
			}

			result = &domain.RoleWithPermissions{
				Role:        *role,
				Permissions: permissions,
			}

			changes := audit.ComputeChanges(nil, &result.Role)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeRole,
				ResourceID:   result.Role.ID,
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

func (s *roleSvcImpl) UpdateRole(ctx context.Context, params domain.UpdateRoleParams) (*domain.RoleWithPermissions, *apierror.APIError) {
	ctx, span := roleSvcTracer.Start(ctx, "service.role.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainRoles, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.RoleWithPermissions](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.RoleWithPermissions
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *roleSvcImpl) *apierror.APIError {
			txRoleRepo := txSvc.repos.NewRoleRepo()
			txRolePermRepo := txSvc.repos.NewRolePermissionRepo()

			// Verify the role exists and is accessible
			old, apiErr := txRoleRepo.Get(txCtx, params.RoleID, params.AccountID)
			if apiErr != nil {
				return apiErr
			}
			if old.AccountID == nil {
				return apierror.NewValidationError("Global roles cannot be modified.")
			}

			if params.Name != nil {
				exists, apiErr := txRoleRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.RoleID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A role with this name already exists.", "name")
				}

				if apiErr := txRoleRepo.UpdateName(txCtx, params.RoleID, params.AccountID, *params.Name); apiErr != nil {
					return apiErr
				}
			}

			if params.Permissions != nil {
				if apiErr := txRolePermRepo.DeleteByRoleID(txCtx, params.RoleID); apiErr != nil {
					return apiErr
				}

				for _, input := range *params.Permissions {
					permID, apiErr := id.GenID(id.RolePermissionIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}

					if apiErr := txRolePermRepo.Create(txCtx, permID, params.RoleID, input); apiErr != nil {
						return apiErr
					}
				}
			}

			// Re-fetch the role and permissions after updates
			updatedRole, apiErr := txRoleRepo.Get(txCtx, params.RoleID, params.AccountID)
			if apiErr != nil {
				return apiErr
			}

			permissions, apiErr := txRolePermRepo.ListByRoleID(txCtx, params.RoleID)
			if apiErr != nil {
				return apiErr
			}

			result = &domain.RoleWithPermissions{
				Role:        *updatedRole,
				Permissions: permissions,
			}

			changes := audit.ComputeChanges(old, updatedRole)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeRole,
				ResourceID:   updatedRole.ID,
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

func (s *roleSvcImpl) DeleteRole(ctx context.Context, roleID string) *apierror.APIError {
	ctx, span := roleSvcTracer.Start(ctx, "service.role.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainRoles, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	role, apiErr := s.repos.NewRoleRepo().Get(ctx, roleID, accountID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeRole, roleID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This role has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}
	if role.AccountID == nil {
		return tracing.Trace(span, apierror.NewValidationError("Global roles cannot be deleted."))
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *roleSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeRole, role.ID, role); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewRolePermissionRepo().DeleteByRoleID(txCtx, roleID); apiErr != nil {
			return apiErr
		}
		if apiErr := txSvc.repos.NewRoleRepo().Delete(txCtx, roleID, accountID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(role, (*domain.Role)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeRole,
			ResourceID:   role.ID,
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
