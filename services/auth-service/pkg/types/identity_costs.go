package types

import apierror "github.com/open-mrp/api/shared/errors"

var costsRead = Permission{Domain: PermissionDomainCosts, Action: ActionRead}.String()

// CanReadCosts reports whether the caller may see the seller's cost and margin figures: an internal admin, or an internal actor whose role grants costs:read.
//
// Customer and supplier portal actors never may, whatever their own role carries, and neither may an agent: what its tools return is kept on the run, where any member of the account can read it. It reads the role directly rather than through CheckHasAnyPermission so no request-scoped relaxation of read checks can widen it.
func (i *Identity) CanReadCosts() bool {
	if !i.IsInternalActor() || i.IsAgent() {
		return false
	}
	if i.IsAdmin() {
		return true
	}
	return i.Actor.Permissions[costsRead]
}

// agentCostsRefusal names the agent refused, so whoever reads the run sees why the tool failed.
func (i *Identity) agentCostsRefusal() *apierror.APIError {
	who := "An agent"
	if i.Actor != nil {
		who = "Agent " + i.actorLabel("(unnamed)")
	}
	return apierror.NewAuthorizationError(who + " may not read costs: agents never receive cost or margin data, whatever their role grants.")
}

// withoutCosts drops the costs domain from perms, for a caller who may never be granted it.
func withoutCosts(perms []Permission) []Permission {
	kept := make([]Permission, 0, len(perms))
	for _, p := range perms {
		if p.Domain != PermissionDomainCosts {
			kept = append(kept, p)
		}
	}
	return kept
}
