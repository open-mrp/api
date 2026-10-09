package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// A role's holders act with every permission it grants, so a non-admin may only hand out, or widen a role by, permissions they already hold. Admins are unbounded.

// rolePermissionCodes flattens permission inputs into domain:action codes.
func rolePermissionCodes(inputs []domain.CreateRolePermissionInput) map[string]bool {
	codes := make(map[string]bool, len(inputs)*4)
	for _, in := range inputs {
		for action, granted := range map[types.Action]bool{
			types.ActionCreate: in.Create,
			types.ActionRead:   in.Read,
			types.ActionUpdate: in.Update,
			types.ActionDelete: in.Delete,
		} {
			if granted {
				codes[in.PermissionCode+":"+string(action)] = true
			}
		}
	}
	return codes
}

// addedPermissionCodes is what an update would grant that the role does not already; removing permissions is always allowed.
func addedPermissionCodes(current []*domain.RolePermission, requested []domain.CreateRolePermissionInput) map[string]bool {
	held := make([]domain.CreateRolePermissionInput, 0, len(current))
	for _, p := range current {
		held = append(held, domain.CreateRolePermissionInput{PermissionCode: p.PermissionCode, Create: p.Create, Read: p.Read, Update: p.Update, Delete: p.Delete})
	}
	heldCodes := rolePermissionCodes(held)
	added := rolePermissionCodes(requested)
	for code := range heldCodes {
		delete(added, code)
	}
	return added
}

// checkPermissionsGrantable rejects a non-admin granting a role permissions they lack.
func checkPermissionsGrantable(identity *types.Identity, granted map[string]bool) *apierror.APIError {
	if identity.IsAdmin() {
		return nil
	}
	missing := types.PermissionsNotHeld(identity.Actor.Permissions, granted)
	if len(missing) == 0 {
		return nil
	}
	return apierror.NewAuthorizationError(fmt.Sprintf("The requested permissions include ones you do not hold: %s. A role can only be given permissions you already have.", strings.Join(missing, ", "))).WithParam("permissions")
}

// checkRoleAssignable rejects a non-admin assigning a member a role wider than their own grant.
// existing marks the member's current role, which the caller must also cover to change it, so a non-admin cannot demote a wider member.
func checkRoleAssignable(ctx context.Context, identity *types.Identity, permRepo domain.RolePermissionRepo, roleID, roleType string, existing bool) *apierror.APIError {
	if identity.IsAdmin() {
		return nil
	}
	subject := "The requested role"
	if existing {
		subject = "This member's current role"
	}
	if roleType == string(constants.RoleTypeAdmin) {
		return apierror.NewAuthorizationError(subject + " is an admin role; only an admin can assign or change it.").WithParam("role_id")
	}
	granted, apiErr := permRepo.FindByRoleID(ctx, roleID)
	if apiErr != nil {
		return apiErr
	}
	missing := types.PermissionsNotHeld(identity.Actor.Permissions, granted)
	if len(missing) == 0 {
		return nil
	}
	return apierror.NewAuthorizationError(fmt.Sprintf("%s grants permissions you do not hold: %s. You can only assign or change a role whose permissions you already have.", subject, strings.Join(missing, ", "))).WithParam("role_id")
}

// checkCurrentRoleChangeable applies checkRoleAssignable to a member's current role. A role that no longer resolves grants nothing to protect.
func (s *accountUserSvcImpl) checkCurrentRoleChangeable(ctx context.Context, identity *types.Identity, roleID, accountID string) *apierror.APIError {
	if identity.IsAdmin() {
		return nil
	}
	role, apiErr := s.repos.NewRoleRepo().Get(ctx, roleID, accountID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil
		}
		return apiErr
	}
	return checkRoleAssignable(ctx, identity, s.repos.NewRolePermissionRepo(), role.ID, role.RoleType, true)
}

func roleIDOrEmpty(roleID *string) string {
	if roleID == nil {
		return ""
	}
	return *roleID
}
