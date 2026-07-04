package types

import (
	"testing"
)

// A relation actor must fail owner-account permission checks even when it carries
// role permissions (those apply to its own account, not the target/owner account),
// but must pass CheckHasRelationCapability for a permission it does hold.
func TestRelationActorPermissionScoping(t *testing.T) {
	t.Parallel()

	customerAccount := "acct_customer"
	relation := IdentityRelationTypeCustomer
	identity := &Identity{
		Type:   IdentityActorTypeUser,
		Target: &IdentityTarget{AccountID: "acct_merchant", RelationType: &relation},
		Actor: &IdentityActor{
			RelationType: IdentityRelationTypeCustomer,
			ID:           "usr_customer",
			AccountID:    &customerAccount,
			// Carries an own-account role permission; RoleType intentionally unset.
			Permissions: map[string]bool{"purchase_orders:create": true},
		},
	}

	// Owner-account checks: denied regardless of the carried permission.
	if apiErr := identity.CheckHasPermission(PermissionDomainPurchaseOrders, ActionCreate); apiErr == nil {
		t.Fatal("relation actor must be denied by CheckHasPermission even for a carried permission")
	}
	if apiErr := identity.CheckHasAnyPermission(
		Permission{Domain: PermissionDomainSalesOrders, Action: ActionRead},
		Permission{Domain: PermissionDomainPurchaseOrders, Action: ActionCreate},
	); apiErr == nil {
		t.Fatal("relation actor must be denied by CheckHasAnyPermission")
	}

	// Customer-side capability: allowed for a held permission, denied otherwise.
	if apiErr := identity.CheckHasRelationCapability(PermissionDomainPurchaseOrders, ActionCreate); apiErr != nil {
		t.Fatalf("relation actor should pass capability check for a held permission: %v", apiErr)
	}
	if apiErr := identity.CheckHasRelationCapability(PermissionDomainSalesOrders, ActionCreate); apiErr == nil {
		t.Fatal("relation actor must be denied a capability it does not hold")
	}
}

// An internal actor is unaffected by the relation guard: its permissions still
// authorize, and the capability check does not apply to it.
func TestInternalActorPermissionUnaffectedByRelationGuard(t *testing.T) {
	t.Parallel()

	account := "acct_internal"
	identity := &Identity{
		Type:   IdentityActorTypeUser,
		Target: &IdentityTarget{AccountID: account},
		Actor: &IdentityActor{
			RelationType: IdentityRelationTypeInternal,
			ID:           "usr_internal",
			AccountID:    &account,
			Permissions:  map[string]bool{"sales_orders:create": true},
		},
	}

	if apiErr := identity.CheckHasPermission(PermissionDomainSalesOrders, ActionCreate); apiErr != nil {
		t.Fatalf("internal actor with the permission must pass: %v", apiErr)
	}
	// The capability check is for relation actors only.
	if apiErr := identity.CheckHasRelationCapability(PermissionDomainSalesOrders, ActionCreate); apiErr == nil {
		t.Fatal("CheckHasRelationCapability must reject a non-relation actor")
	}
}
