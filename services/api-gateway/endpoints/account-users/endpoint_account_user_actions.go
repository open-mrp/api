package accountuserep

import (
	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	"github.com/open-mrp/api/services/api-gateway/pkg/lifecycle"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

func accountUserPolicy(action types.Action) authz.Policy {
	return authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: action}}, Counterparty: authz.Counterparties(action)}
}

func withAccountUserIncludes(e *apiendpoint.APIEndpoint[*lifecycle.Request, *apiresource.AccountUser]) *apiendpoint.APIEndpoint[*lifecycle.Request, *apiresource.AccountUser] {
	e.IncludeConfig = accountUserIncludes()
	return e
}

// Activates an account user, moving them from `disabled` or `removed` to `active`.
//
// A member of your own team takes a seat again, so this is refused at your plan's seat limit; a customer's or supplier's contacts take none. An `active` member is left as they are.
type ActivateAccountUserEndpoint struct{}

func (*ActivateAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*lifecycle.Request, *apiresource.AccountUser] {
	return withAccountUserIncludes(lifecycle.Action("Activate Account User", "/v1/identity/account-users/{id}/actions/activate", constants.ObjectTypeAccountUser,
		accountUserPolicy(types.ActionUpdate),
		func(svc any) apiendpoint.ServiceHandler[*lifecycle.Request, *apiresource.AccountUser] {
			return svc.(AccountUserSvc).ActivateAccountUser
		}))
}

// Disables an account user, moving them from `active` to `disabled`.
//
// They're signed out everywhere and can't sign in until activated. Admins and your own membership can't be disabled. A `removed` member is a 409 `invalid_state_transition`.
type DisableAccountUserEndpoint struct{}

func (*DisableAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*lifecycle.Request, *apiresource.AccountUser] {
	return withAccountUserIncludes(lifecycle.Action("Disable Account User", "/v1/identity/account-users/{id}/actions/disable", constants.ObjectTypeAccountUser,
		accountUserPolicy(types.ActionUpdate),
		func(svc any) apiendpoint.ServiceHandler[*lifecycle.Request, *apiresource.AccountUser] {
			return svc.(AccountUserSvc).DisableAccountUser
		}))
}

// Removes an account user, moving them to `removed`.
//
// They lose access to this account and free their seat, and records that name them keep naming them. Their profile and their other accounts are untouched. Activate them to restore the membership.
type RemoveAccountUserEndpoint struct{}

func (*RemoveAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*lifecycle.Request, *apiresource.AccountUser] {
	return withAccountUserIncludes(lifecycle.Action("Remove Account User", "/v1/identity/account-users/{id}/actions/remove", constants.ObjectTypeAccountUser,
		accountUserPolicy(types.ActionDelete),
		func(svc any) apiendpoint.ServiceHandler[*lifecycle.Request, *apiresource.AccountUser] {
			return svc.(AccountUserSvc).RemoveAccountUser
		}))
}

// Invites an account user, emailing them a sign-in link.
//
// The link works for 72 hours; inviting again sends a new one. `portal_access` becomes `invited` until they sign in. Only an `active` member can be invited (409 `invalid_state_transition` otherwise). A member with no email, or a supplier's contact (suppliers have no portal yet), is a 422.
type InviteAccountUserEndpoint struct{}

func (*InviteAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*lifecycle.Request, *apiresource.AccountUser] {
	return withAccountUserIncludes(lifecycle.Action("Invite Account User", "/v1/identity/account-users/{id}/actions/invite", constants.ObjectTypeAccountUser,
		accountUserPolicy(types.ActionUpdate),
		func(svc any) apiendpoint.ServiceHandler[*lifecycle.Request, *apiresource.AccountUser] {
			return svc.(AccountUserSvc).InviteAccountUser
		}))
}
