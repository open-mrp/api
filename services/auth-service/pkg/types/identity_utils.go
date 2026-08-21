package types

import (
	"fmt"
	"strings"

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
//
// Relation actors (customer/supplier) always fail this check: their carried role
// permissions apply to their OWN account, not the target/owner account this check
// authorizes against, so they hold no permissions here. Customer/supplier-side
// capabilities (e.g. a portal order) are authorized separately via
// CheckHasRelationCapability, which reads those same carried permissions.
func (i *Identity) CheckHasAnyPermission(perms ...Permission) *apierror.APIError {
	if len(perms) == 0 {
		return nil
	}
	// An unauthenticated caller isn't missing a permission — it isn't signed in.
	// Return a 401 so the message names the real problem (and the client knows to
	// authenticate) instead of the generic "no permission" 403.
	if !i.IsAuthenticated() {
		return i.CheckIsAuthenticated()
	}
	if i.IsRelationActor() {
		return apierror.NewAuthorizationError(i.getAnyOfPermissionErrorMessage(perms))
	}
	if i.IsAdmin() {
		return nil
	}
	if i.Actor != nil {
		for _, p := range perms {
			if hasPerm, ok := i.Actor.Permissions[p.String()]; ok && hasPerm {
				return nil
			}
		}
	}
	return apierror.NewAuthorizationError(i.getAnyOfPermissionErrorMessage(perms))
}

// CheckHasRelationCapability reports whether a customer/supplier relation actor holds
// the given permission in their OWN account (from their carried role). Use this ONLY
// for explicitly customer/supplier-side authorization (e.g. a portal order create);
// never for operations scoped to the target/owner account — for those a relation actor
// holds no permissions (see CheckHasAnyPermission). Non-relation actors are rejected so
// callers don't accidentally use it as a general permission check.
func (i *Identity) CheckHasRelationCapability(domain PermissionDomain, action Action) *apierror.APIError {
	if i == nil || i.Actor == nil || !i.IsRelationActor() {
		return apierror.NewAuthorizationError(i.getPermissionErrorMessage(domain, action))
	}
	if hasPerm, ok := i.Actor.Permissions[Permission{Domain: domain, Action: action}.String()]; ok && hasPerm {
		return nil
	}
	return apierror.NewAuthorizationError(i.getPermissionErrorMessage(domain, action))
}

// CheckHasRoleType passes if the identity has the given role type (or is an admin, who may do anything).
func (i *Identity) CheckHasRoleType(roleType constants.RoleType) *apierror.APIError {
	if roleType == "" {
		return nil
	}
	// Unauthenticated callers get a 401, not a role-based 403 — they aren't signed
	// in, so the required-role message would be misleading.
	if !i.IsAuthenticated() {
		return i.CheckIsAuthenticated()
	}
	if i.IsAdmin() {
		return nil
	}
	if i.Actor != nil && i.Actor.RoleType != nil && *i.Actor.RoleType == string(roleType) {
		return nil
	}
	return apierror.NewAuthorizationError("You do not have the required role to access this resource.")
}

func (i *Identity) getPermissionErrorMessage(domain PermissionDomain, action Action) string {
	return i.getAnyOfPermissionErrorMessage([]Permission{{Domain: domain, Action: action}})
}

// getAnyOfPermissionErrorMessage explains an authorization failure to the person who hit
// it. Whoever reads this — a support engineer, an admin deciding what to grant — needs to
// know who was denied, what they hold, and what would have let them through, so the
// message names the actor, names their role, and lists every permission that would have
// been accepted rather than only the first declared one. The role ID comes along because
// it is what an admin edits; the role name is what they recognize.
func (i *Identity) getAnyOfPermissionErrorMessage(perms []Permission) string {
	required := describePermissions(perms)

	if i == nil || i.Actor == nil {
		return fmt.Sprintf("You do not have permission to %s.", required)
	}

	switch i.Type {
	case IdentityActorTypeUser:
		return fmt.Sprintf("%s does not have permission to %s.%s", i.actorLabel("This user"), required, i.roleClause())
	case IdentityActorTypeAPIKey:
		return fmt.Sprintf("API key %s does not have permission to %s.%s", i.actorLabel("(unnamed)"), required, i.roleClause())
	case IdentityActorTypeAgent:
		return fmt.Sprintf("Agent %s does not have permission to %s.%s", i.actorLabel("(unnamed)"), required, i.roleClause())
	default:
		return fmt.Sprintf("You do not have permission to %s.", required)
	}
}

// actorLabel prefers the actor's name over its ID, since the ID means nothing to a reader.
// Falls back to the ID only when the name is unset, and to fallback when neither is known.
func (i *Identity) actorLabel(fallback string) string {
	if i.Actor.Name != nil && *i.Actor.Name != "" {
		return *i.Actor.Name
	}
	if i.Actor.ID != "" {
		return i.Actor.ID
	}
	return fallback
}

// roleClause names the role the actor holds, so the reader knows which role to change.
// Returns "" when no role is carried — relation actors hold none, and saying so would
// misdescribe why they were denied.
func (i *Identity) roleClause() string {
	hasName := i.Actor.RoleName != nil && *i.Actor.RoleName != ""
	hasID := i.Actor.RoleID != nil && *i.Actor.RoleID != ""
	switch {
	case hasName && hasID:
		return fmt.Sprintf(" Their role %q (ID %s) does not grant it.", *i.Actor.RoleName, *i.Actor.RoleID)
	case hasName:
		return fmt.Sprintf(" Their role %q does not grant it.", *i.Actor.RoleName)
	case hasID:
		return fmt.Sprintf(" Their role (ID %s) does not grant it.", *i.Actor.RoleID)
	default:
		return ""
	}
}

// describePermissions renders the accepted set as "a", "a or b", or "a, b or c" — any one
// of them is enough, and a bare comma-separated list reads as though all are required.
func describePermissions(perms []Permission) string {
	codes := make([]string, 0, len(perms))
	for _, p := range perms {
		codes = append(codes, fmt.Sprintf("%s:%s", p.Domain, p.Action))
	}
	switch len(codes) {
	case 0:
		return "perform this action"
	case 1:
		return codes[0]
	default:
		return strings.Join(codes[:len(codes)-1], ", ") + " or " + codes[len(codes)-1]
	}
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
