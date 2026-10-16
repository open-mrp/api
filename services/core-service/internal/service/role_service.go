package service

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idemresponse"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/tracing"
)

var roleSvcTracer = tracing.GetTracer("core-service.role_service")

type roleSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type RoleSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
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

	identity, ok := appctx.Identity[*types.Identity](ctx)
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

	rolesWithPerms := make([]*domain.RoleWithPermissions, len(page.Roles))
	for i, r := range page.Roles {
		rolesWithPerms[i] = &domain.RoleWithPermissions{Role: *r}
	}

	{
		roleIDs := make([]string, len(page.Roles))
		for i, r := range page.Roles {
			roleIDs[i] = r.ID
		}
		permsByRole, apiErr := s.repos.NewRolePermissionRepo().ListByRoleIDs(ctx, roleIDs)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, rwp := range rolesWithPerms {
			rwp.Permissions = permsByRole[rwp.Role.ID]
		}
	}

	return &domain.ListRolesResult{Roles: rolesWithPerms, PageInfo: page.PageInfo}, nil
}

func (s *roleSvcImpl) GetRole(ctx context.Context, roleID string, incs []string) (*domain.RoleWithPermissions, *apierror.APIError) {
	ctx, span := roleSvcTracer.Start(ctx, "service.role.get")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
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

	rwp := &domain.RoleWithPermissions{Role: *role}

	{
		permissions, apiErr := s.repos.NewRolePermissionRepo().ListByRoleID(ctx, roleID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		rwp.Permissions = permissions
	}

	return rwp, nil
}

func (s *roleSvcImpl) CreateRole(ctx context.Context, params domain.CreateRoleParams) (*domain.RoleWithPermissions, *apierror.APIError) {
	ctx, span := roleSvcTracer.Start(ctx, "service.role.create")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
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
		cached, err := idemresponse.UnmarshalCachedResponse[domain.RoleWithPermissions](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		access, effective, apiErr := createDefinition(params)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		params.Access = access
		if apiErr := checkPermissionsGrantable(identity, codeSet(effective)); apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		var result *domain.RoleWithPermissions
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *roleSvcImpl) *apierror.APIError {
			txRoleRepo := txSvc.repos.NewRoleRepo()
			txRolePermRepo := txSvc.repos.NewRolePermissionRepo()

			exists, apiErr := txRoleRepo.ExistsByName(txCtx, params.AccountID, params.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("name", fmt.Sprintf("A role with the name '%s' already exists.", params.Name))
			}

			if apiErr := txRoleRepo.Create(txCtx, roleID, params); apiErr != nil {
				return apiErr
			}

			if apiErr := writePermissionRows(txCtx, txRolePermRepo, roleID, effective); apiErr != nil {
				return apiErr
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

	identity, ok := appctx.Identity[*types.Identity](ctx)
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
		cached, err := idemresponse.UnmarshalCachedResponse[domain.RoleWithPermissions](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
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

			oldPermissions, apiErr := txRolePermRepo.ListByRoleID(txCtx, params.RoleID)
			if apiErr != nil {
				return apiErr
			}
			if old.RoleType == string(constants.RoleTypeAdmin) && !identity.IsAdmin() {
				return apierror.NewAuthorizationError("This is an admin role; only an admin can change it.")
			}
			changesDefinition := params.Access != nil || params.Grants != nil || params.Revokes != nil
			var access types.RoleAccess
			var effective []string
			if changesDefinition {
				var apiErr *apierror.APIError
				access, effective, apiErr = updateDefinition(&domain.RoleWithPermissions{Role: *old, Permissions: oldPermissions}, params)
				if apiErr != nil {
					return apiErr
				}
				added := codeSet(effective)
				for code := range domain.HeldCodes(oldPermissions) {
					delete(added, code)
				}
				if apiErr := checkPermissionsGrantable(identity, added); apiErr != nil {
					return apiErr
				}
			}

			if params.Name != nil {
				exists, apiErr := txRoleRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.RoleID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("name", "A role with this name already exists.")
				}

				if apiErr := txRoleRepo.UpdateName(txCtx, params.RoleID, params.AccountID, *params.Name); apiErr != nil {
					return apiErr
				}
			}

			if changesDefinition {
				if apiErr := txRoleRepo.UpdateAccess(txCtx, params.RoleID, params.AccountID, access); apiErr != nil {
					return apiErr
				}
				if apiErr := txRolePermRepo.DeleteByRoleID(txCtx, params.RoleID); apiErr != nil {
					return apiErr
				}
				if apiErr := writePermissionRows(txCtx, txRolePermRepo, params.RoleID, effective); apiErr != nil {
					return apiErr
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

			if permChange := computePermissionsChange(oldPermissions, permissions); permChange != nil {
				changes = append(changes, *permChange)
			}

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

	identity, ok := appctx.Identity[*types.Identity](ctx)
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
		if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().ExistsInAccount(ctx, constants.DeletedRecordResourceTypeRole, roleID, accountID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewGoneError("This role has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}
	if role.AccountID == nil {
		return tracing.Trace(span, apierror.NewValidationError("Global roles cannot be deleted."))
	}

	if apiErr := domain.RoleReferences.RequireUnused(ctx, s.repos.NewReferenceRepo(), roleID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *roleSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeRole, role.ID, accountID, role); apiErr != nil {
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

func (s *roleSvcImpl) BatchGetRolesByIDs(ctx context.Context, ids []string) ([]*domain.RoleWithPermissions, *apierror.APIError) {
	ctx, span := roleSvcTracer.Start(ctx, "service.role.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActorForRead(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainRoles, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	roles, apiErr := s.repos.NewRoleRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	roleIDs := make([]string, len(roles))
	for i, r := range roles {
		roleIDs[i] = r.ID
	}

	permsByRole, apiErr := s.repos.NewRolePermissionRepo().ListByRoleIDs(ctx, roleIDs)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := make([]*domain.RoleWithPermissions, len(roles))
	for i, r := range roles {
		result[i] = &domain.RoleWithPermissions{
			Role:        *r,
			Permissions: permsByRole[r.ID],
		}
	}

	return result, nil
}

// permissionSummary is a lightweight, stable representation of a role permission used for audit diff comparison. It excludes metadata fields (ID, RoleID, timestamps) that change on every delete+recreate cycle.
type permissionSummary struct {
	PermissionCode string `json:"permission_code"`
	Create         bool   `json:"create"`
	Read           bool   `json:"read"`
	Update         bool   `json:"update"`
	Delete         bool   `json:"delete"`
}

func toPermissionSummaries(perms []*domain.RolePermission) []permissionSummary {
	summaries := make([]permissionSummary, len(perms))
	for i, p := range perms {
		summaries[i] = permissionSummary{
			PermissionCode: p.PermissionCode,
			Create:         p.Create,
			Read:           p.Read,
			Update:         p.Update,
			Delete:         p.Delete,
		}
	}
	slices.SortFunc(summaries, func(a, b permissionSummary) int {
		if a.PermissionCode < b.PermissionCode {
			return -1
		}
		if a.PermissionCode > b.PermissionCode {
			return 1
		}
		return 0
	})
	return summaries
}

func computePermissionsChange(oldPerms, newPerms []*domain.RolePermission) *audit.FieldChange {
	oldSummaries := toPermissionSummaries(oldPerms)
	newSummaries := toPermissionSummaries(newPerms)

	if reflect.DeepEqual(oldSummaries, newSummaries) {
		return nil
	}

	change := audit.NewFieldChange("permissions", oldSummaries, newSummaries)
	return &change
}

// createDefinition settles a new role's access and what it holds: the template's access and grants, the request's
// levels over them, its grants instead of the template's when given, and its revokes.
func createDefinition(params domain.CreateRoleParams) (types.RoleAccess, []string, *apierror.APIError) {
	access := types.RoleAccess{}.Complete()
	var grants []string
	if params.Template != nil {
		spec, ok := types.RoleTemplateByCode(*params.Template)
		if !ok {
			return nil, nil, apierror.NewValidationError(fmt.Sprintf("Unknown role template %q.", *params.Template)).WithParam("template")
		}
		for area, level := range spec.Access {
			access[area] = level
		}
		grants = slices.Clone(spec.Grants)
	}
	if apiErr := mergeAccess(access, params.Access); apiErr != nil {
		return nil, nil, apiErr
	}
	if params.Grants != nil {
		grants = *params.Grants
	}
	if apiErr := checkCatalogCodes("grants", grants); apiErr != nil {
		return nil, nil, apiErr
	}
	if apiErr := checkCatalogCodes("revokes", params.Revokes); apiErr != nil {
		return nil, nil, apiErr
	}
	return access, types.EffectivePermissions(access, grants, params.Revokes), nil
}

// updateDefinition applies a role update to its current definition: the areas named change level, and grants and
// revokes are replaced when given.
func updateDefinition(current *domain.RoleWithPermissions, params domain.UpdateRoleParams) (types.RoleAccess, []string, *apierror.APIError) {
	def := current.Definition()
	access := def.Access.Complete()
	if apiErr := mergeAccess(access, params.Access); apiErr != nil {
		return nil, nil, apiErr
	}
	grants, revokes := def.Grants, def.Revokes
	if params.Grants != nil {
		grants = *params.Grants
		if apiErr := checkCatalogCodes("grants", grants); apiErr != nil {
			return nil, nil, apiErr
		}
	}
	if params.Revokes != nil {
		revokes = *params.Revokes
		if apiErr := checkCatalogCodes("revokes", revokes); apiErr != nil {
			return nil, nil, apiErr
		}
	}
	return access, types.EffectivePermissions(access, grants, revokes), nil
}

func mergeAccess(into, from types.RoleAccess) *apierror.APIError {
	for area, level := range from {
		if !area.IsValid() {
			return apierror.NewValidationError(fmt.Sprintf("Unknown area %q. Areas are %s.", area, strings.Join(constants.PermissionArea("").EnumValues(), ", "))).WithParam("access")
		}
		if !level.IsValid() {
			return apierror.NewValidationError(fmt.Sprintf("Unknown access level %q for %s. Levels are none, read, edit and full.", level, area)).WithParam("access." + string(area))
		}
		into[area] = level
	}
	return nil
}

func checkCatalogCodes(param string, codes []string) *apierror.APIError {
	if unknown := types.UnknownPermissionCodes(codes); len(unknown) > 0 {
		return apierror.NewValidationError(fmt.Sprintf("Unknown permission codes: %s. GET /v1/identity/permissions lists them.", strings.Join(unknown, ", "))).WithParam(param)
	}
	return nil
}

func codeSet(codes []string) map[string]bool {
	out := make(map[string]bool, len(codes))
	for _, c := range codes {
		out[c] = true
	}
	return out
}

// writePermissionRows stores what a role holds as its permission rows, which every permission check reads.
func writePermissionRows(ctx context.Context, repo domain.RolePermissionRepo, roleID string, effective []string) *apierror.APIError {
	for _, input := range domain.PermissionRowsFor(effective) {
		permID, apiErr := id.GenID(id.RolePermissionIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := repo.Create(ctx, permID, roleID, input); apiErr != nil {
			return apiErr
		}
	}
	return nil
}
