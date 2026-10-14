// Package authz is the gateway's coarse access gate: an endpoint's Auth policy, and the authorizer apikit's endpoint
// runs against it before binding a request. It never rejects a caller who could be authorized; the precise, possibly
// relation-dependent check still runs in the downstream service. Importing the package registers the authorizer.
package authz

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/transport"
)

// Policy is an endpoint's access policy, set as its Auth. An endpoint that declares neither Permissions nor RoleType is unrestricted here; authorization happens downstream.
type Policy struct {
	// Permissions is the any-of permission set the endpoint requires, using typed domain/action constants (e.g. {types.PermissionDomainCustomers, types.ActionRead}) so typos are caught by the compiler. Holding one is enough to reach the handler. Agent tools and OpenAPI docs surface the same declaration.
	Permissions types.AnyOfPermissions
	// RequireAll makes Permissions all-of, for an endpoint whose service checks the permissions together: the caller must hold every one.
	RequireAll bool
	// Counterparty (optional) is what a request acting in a customer's or supplier's account needs in place of Permissions, which then govern only the caller's own account.
	Counterparty CounterpartyPermissions
	// SelfPathParam (optional) names the path parameter that holds a user ID, on endpoints where a user may always act on their own record. A signed-in user whose ID it is passes the gate without holding Permissions or RoleType; the downstream service still decides.
	SelfPathParam string
	// RoleType (optional) is a role type the caller must have (e.g. constants.RoleTypeAdmin), instead of or in addition to a permission.
	RoleType constants.RoleType
}

// Of returns the Policy an endpoint's Auth holds, and false when it has none.
func Of(auth any) (Policy, bool) {
	switch p := auth.(type) {
	case Policy:
		return p, true
	case *Policy:
		if p != nil {
			return *p, true
		}
	}
	return Policy{}, false
}

func init() {
	apiendpoint.SetAuthorizer(Authorize)
}

// Authorize enforces policy's RoleType and Permissions against the caller in r. Permissions are any-of unless RequireAll is set, and a request acting in a customer's or supplier's account needs the matching Counterparty entry instead when the policy declares one. Customer- and supplier-relation actors pass (they are authorized by relation downstream), as does a signed-in user acting on their own record through SelfPathParam. It returns nil when the request may proceed.
func Authorize(r *http.Request, auth any) *apierror.APIError {
	p, ok := Of(auth)
	if !ok {
		return nil
	}
	apiErr := p.check(r.Context())
	if apiErr != nil && p.actsOnSelf(r) {
		return nil
	}
	return apiErr
}

func (p Policy) check(ctx context.Context) *apierror.APIError {
	if p.RoleType == "" && len(p.Permissions) == 0 {
		return nil
	}
	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return apierror.NewAuthenticationError("You must be authenticated to access this resource.")
	}
	// An unauthenticated caller gets a 401 here rather than being carried into the permission check, where a nil actor would surface as a misleading 403.
	if apiErr := identity.CheckIsAuthenticated(); apiErr != nil {
		return apiErr
	}
	// Customer- and supplier-relation actors carry no permission set and no role type; their access is relation-scoped and authorized entirely downstream, so rejecting them here would refuse callers who could be authorized.
	if identity.IsRelationActor() {
		return nil
	}
	if apiErr := identity.CheckHasRoleType(p.RoleType); apiErr != nil {
		return apiErr
	}
	if perm, ok := p.Counterparty.For(identity); ok {
		return identity.CheckHasPermission(perm.Domain, perm.Action)
	}
	if p.RequireAll {
		for _, perm := range p.Permissions {
			if apiErr := identity.CheckHasPermission(perm.Domain, perm.Action); apiErr != nil {
				return apiErr
			}
		}
		return nil
	}
	return identity.CheckHasAnyPermission(p.Permissions...)
}

// actsOnSelf reports whether the caller is the signed-in user named by the policy's SelfPathParam.
func (p Policy) actsOnSelf(r *http.Request) bool {
	if p.SelfPathParam == "" {
		return false
	}
	identity, ok := appctx.Identity[*types.Identity](r.Context())
	if !ok || identity == nil || !identity.HasUserActor() {
		return false
	}
	return identity.Actor.ID == transport.PathExtractor(r)(p.SelfPathParam)
}
