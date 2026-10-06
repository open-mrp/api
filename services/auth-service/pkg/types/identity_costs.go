package types

var costsRead = Permission{Domain: PermissionDomainCosts, Action: ActionRead}.String()

// CanReadCosts reports whether the caller may see the seller's cost and margin figures: an internal admin, or an internal actor whose role grants costs:read.
//
// Customer and supplier portal actors never may, whatever their own role carries. It reads the role directly rather than through CheckHasAnyPermission so no request-scoped relaxation of read checks can widen it.
func (i *Identity) CanReadCosts() bool {
	if !i.IsInternalActor() {
		return false
	}
	if i.IsAdmin() {
		return true
	}
	return i.Actor.Permissions[costsRead]
}
