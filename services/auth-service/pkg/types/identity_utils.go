package types

import (
	"fmt"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// CheckIsUser checks that the identity is a user
func (i *Identity) CheckIsUser() *apierror.APIError {
	if apiErr := i.CheckIsAuthenticated(); apiErr != nil {
		return apiErr
	}

	if !i.IsUser() {
		return apierror.NewAuthorizationError("You must be a user to access this resource.")
	}

	return nil
}

// CheckHasUserActor checks that the identity is an authenticated user, without
// requiring an assigned account. Use this for account-agnostic user endpoints
// (e.g. tenancy discovery) where the user may not yet have selected an account.
func (i *Identity) CheckHasUserActor() *apierror.APIError {
	if apiErr := i.CheckIsAuthenticated(); apiErr != nil {
		return apiErr
	}

	if !i.HasUserActor() {
		return apierror.NewAuthorizationError("You must be a user to access this resource.")
	}

	return nil
}

// CheckIsAPIKey checks that the identity is an API key
func (i *Identity) CheckIsAPIKey() *apierror.APIError {
	if apiErr := i.CheckIsAuthenticated(); apiErr != nil {
		return apiErr
	}

	if !i.IsAPIKey() {
		return apierror.NewAuthorizationError("You must be an API key to access this resource.")
	}

	return nil
}

// CheckIsAuthenticated checks that the identity is authenticated
func (i *Identity) CheckIsAuthenticated() *apierror.APIError {
	if !i.IsAuthenticated() {
		return apierror.NewAuthenticationError("You must be authenticated to access this resource.")
	}

	return nil
}

func (i *Identity) CheckIsTargetAccountSet() *apierror.APIError {
	if !i.IsTargetAccountSet() {
		return apierror.NewAuthenticationError("The Augno-Account header is required.")
	}

	return nil
}

// CheckIsInternalActor checks that the identity is an internal user
func (i *Identity) CheckIsInternalActor() *apierror.APIError {
	if apiErr := i.CheckIsTargetAccountSet(); apiErr != nil {
		return apiErr
	}

	if apiErr := i.CheckIsAuthenticated(); apiErr != nil {
		return apiErr
	}

	if !i.IsInternalUser() {
		return apierror.NewAuthorizationError("You must be an internal user for this account to access this resource.")
	}

	return nil
}

// CheckIsAdmin checks that the identity is an admin
func (i *Identity) CheckIsAdmin() *apierror.APIError {
	if apiErr := i.CheckIsTargetAccountSet(); apiErr != nil {
		return apiErr
	}

	if apiErr := i.CheckIsAuthenticated(); apiErr != nil {
		return apiErr
	}

	if !i.IsAdmin() {
		return apierror.NewAuthorizationError("You must be an administrator to access this resource.")
	}

	return nil
}

// CheckIsSandboxMode checks that the identity is in sandbox mode
func (i *Identity) CheckIsSandboxMode() *apierror.APIError {
	if apiErr := i.CheckIsAuthenticated(); apiErr != nil {
		return apiErr
	}

	if !i.IsSandbox() {
		return apierror.NewAuthorizationError("This action is only available in sandbox mode.")
	}

	return nil
}

// CheckNotSandboxMode checks that the identity is not in sandbox mode
func (i *Identity) CheckNotSandboxMode() *apierror.APIError {
	if apiErr := i.CheckIsAuthenticated(); apiErr != nil {
		return apiErr
	}

	if !i.IsNotSandbox() {
		return apierror.NewAuthorizationError("Sandbox management is not available in sandbox mode.")
	}

	return nil
}

// CheckHasPermission checks that the identity has a specific permission. It is the single-permission case of CheckHasAnyPermission, so the gateway gate and the internal-service checks run the exact same logic.
func (i *Identity) CheckHasPermission(domain PermissionDomain, action Action) *apierror.APIError {
	return i.CheckHasAnyPermission(Permission{Domain: domain, Action: action})
}

// CheckHasAnyPermission passes if the identity holds AT LEAST ONE of the given permissions (or is an admin). It is the coarse "OR" gate used by the api-gateway to fast-reject callers that hold none of an endpoint's declared permissions; the precise, possibly relation-dependent check still happens in the downstream service.
func (i *Identity) CheckHasAnyPermission(perms ...Permission) *apierror.APIError {
	if len(perms) == 0 {
		return nil
	}
	if i == nil || i.Actor == nil {
		return apierror.NewAuthorizationError("You do not have permission to access this resource.")
	}
	if i.IsAdmin() {
		return nil
	}
	for _, p := range perms {
		if hasPerm, ok := i.Actor.Permissions[p.String()]; ok && hasPerm {
			return nil
		}
	}
	// Reuse the single-permission message for the first declared permission; it
	// names the domain/action the caller is missing.
	return apierror.NewAuthorizationError(i.getPermissionErrorMessage(perms[0].Domain, perms[0].Action))
}

// CheckHasRoleType passes if the identity has the given role type (or is an admin, who may do anything).
func (i *Identity) CheckHasRoleType(roleType constants.RoleType) *apierror.APIError {
	if roleType == "" {
		return nil
	}
	if i == nil || i.Actor == nil {
		return apierror.NewAuthorizationError("You do not have permission to access this resource.")
	}
	if i.IsAdmin() {
		return nil
	}
	if i.Actor.RoleType != nil && *i.Actor.RoleType == string(roleType) {
		return nil
	}
	return apierror.NewAuthorizationError("You do not have the required role to access this resource.")
}

func (i *Identity) getPermissionErrorMessage(domain PermissionDomain, action Action) string {
	if i != nil && i.Actor != nil {
		switch i.Type {
		case IdentityActorTypeUser:
			return fmt.Sprintf("User %s does not have permission to %s:%s", i.Actor.ID, domain, action)
		case IdentityActorTypeAPIKey:
			return fmt.Sprintf("This API Key does not have permission to %s:%s", domain, action)
		case IdentityActorTypeAgent:
			return fmt.Sprintf("Agent %s does not have permission to %s:%s", i.Actor.ID, domain, action)
		default:
			return fmt.Sprintf("You do not have permission to %s:%s", domain, action)
		}
	}
	return fmt.Sprintf("You do not have permission to %s:%s", domain, action)
}

// CheckIsAssignedActor verifies that the identity is authenticated and is an
// internal, customer, or supplier actor (i.e. assigned to some account).
// This mirrors the legacy dashboard's checkIsAssignedActor.
func (i *Identity) CheckIsAssignedActor() *apierror.APIError {
	if apiErr := i.CheckIsAuthenticated(); apiErr != nil {
		return apiErr
	}

	if i.Actor == nil {
		return apierror.NewAuthorizationError("You must be assigned to an account to access this resource.")
	}

	switch i.Actor.RelationType {
	case IdentityRelationTypeInternal, IdentityRelationTypeCustomer, IdentityRelationTypeSupplier:
		return nil
	default:
		return apierror.NewAuthorizationError("You must be assigned to an account to access this resource.")
	}
}

func (i *Identity) CheckIsAgent() *apierror.APIError {
	if apiErr := i.CheckIsAuthenticated(); apiErr != nil {
		return apiErr
	}

	if i.Type != IdentityActorTypeAgent {
		return apierror.NewAuthorizationError("You must be an agent to access this resource.")
	}

	return nil
}

// CheckAPIKeyAccess verifies that the identity is an internal admin with a
// target account set — the standard gate for API key CRUD operations.
func (i *Identity) CheckAPIKeyAccess() *apierror.APIError {
	if apiErr := i.CheckIsInternalActor(); apiErr != nil {
		return apiErr
	}
	if apiErr := i.CheckIsAdmin(); apiErr != nil {
		return apiErr
	}
	if apiErr := i.CheckIsTargetAccountSet(); apiErr != nil {
		return apiErr
	}
	return nil
}
