package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

// A role's holders act with every permission it grants, so a non-admin may only hand out, or widen a role by, permissions they already hold. Admins are unbounded.

// checkPermissionsGrantable rejects a non-admin granting a role permissions they lack.
func checkPermissionsGrantable(identity *types.Identity, granted map[string]bool) *apierror.APIError {
	if identity.IsAdmin() {
		return nil
	}
	missing := types.PermissionsNotHeld(identity.Actor.Permissions, withoutBaseline(granted))
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
	missing := types.PermissionsNotHeld(identity.Actor.Permissions, withoutBaseline(granted))
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
		if apiErr.Code == apierror.CodeResourceNotFound {
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

// withoutBaseline drops the permissions every member holds (their profile, messaging, notes): handing them out widens
// nothing.
func withoutBaseline(granted map[string]bool) map[string]bool {
	out := make(map[string]bool, len(granted))
	for code, ok := range granted {
		if p, known := types.CatalogPermissionByCode(code); known && p.Level == constants.AccessLevelNone {
			continue
		}
		out[code] = ok
	}
	return out
}
