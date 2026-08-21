package types

import (
	"strings"
	"testing"
)

func strPtr(s string) *string { return &s }

func userIdentity(name, roleName, roleID *string) *Identity {
	account := "acct_1"
	return &Identity{
		Type:   IdentityActorTypeUser,
		Target: &IdentityTarget{AccountID: account},
		Actor: &IdentityActor{
			RelationType: IdentityRelationTypeInternal,
			ID:           "usr_9f3c",
			Name:         name,
			AccountID:    &account,
			RoleID:       roleID,
			RoleName:     roleName,
			RoleType:     strPtr("sales_rep"),
			Permissions:  map[string]bool{},
		},
	}
}

// A denied user must be identifiable from the message alone: who they are, which role
// they hold, and which role an admin would have to edit.
func TestPermissionErrorNamesActorAndRole(t *testing.T) {
	t.Parallel()

	identity := userIdentity(strPtr("Dana Whitfield"), strPtr("Sales Rep"), strPtr("role_7b21"))
	apiErr := identity.CheckHasPermission(PermissionDomainSalesOrders, ActionRead)
	if apiErr == nil {
		t.Fatal("expected the check to fail")
	}

	for _, want := range []string{"Dana Whitfield", `"Sales Rep"`, "role_7b21", "sales_orders:read"} {
		if !strings.Contains(apiErr.PublicMessage, want) {
			t.Errorf("message %q is missing %q", apiErr.PublicMessage, want)
		}
	}
	if strings.Contains(apiErr.PublicMessage, "usr_9f3c") {
		t.Errorf("message should name the user, not their ID: %q", apiErr.PublicMessage)
	}
}

// Any-of endpoints must list every permission that would have been accepted, joined so
// the reader can tell one is enough — naming only the first understates the ways in.
func TestPermissionErrorListsEveryAcceptedPermission(t *testing.T) {
	t.Parallel()

	identity := userIdentity(strPtr("Dana Whitfield"), strPtr("Sales Rep"), strPtr("role_7b21"))
	apiErr := identity.CheckHasAnyPermission(
		Permission{Domain: PermissionDomainSalesOrders, Action: ActionRead},
		Permission{Domain: PermissionDomainProductionSchedules, Action: ActionRead},
	)
	if apiErr == nil {
		t.Fatal("expected the check to fail")
	}
	if !strings.Contains(apiErr.PublicMessage, "sales_orders:read or production_schedules:read") {
		t.Errorf("message should offer both permissions as alternatives: %q", apiErr.PublicMessage)
	}
}

// Missing name or role must degrade to something still readable rather than printing
// empty quotes or a bare "Their role \"\"".
func TestPermissionErrorFallsBackWhenNameOrRoleUnknown(t *testing.T) {
	t.Parallel()

	noName := userIdentity(nil, strPtr("Sales Rep"), strPtr("role_7b21"))
	if msg := noName.CheckHasPermission(PermissionDomainSalesOrders, ActionRead).PublicMessage; !strings.Contains(msg, "usr_9f3c") {
		t.Errorf("without a name the ID should stand in: %q", msg)
	}

	noRole := userIdentity(strPtr("Dana Whitfield"), nil, nil)
	msg := noRole.CheckHasPermission(PermissionDomainSalesOrders, ActionRead).PublicMessage
	if strings.Contains(msg, "Their role") {
		t.Errorf("a roleless actor should not have a role described: %q", msg)
	}
	if !strings.Contains(msg, "Dana Whitfield") {
		t.Errorf("the actor should still be named: %q", msg)
	}
}

// An any-of set exists so a caller holding any one of several domains gets through. Office
// and AR roles hold sales_orders:read and not production_schedules:read, and a set naming
// both must admit them — declaring only the schedules domain on a sales-facing endpoint is
// how a feature silently 403s for everyone it was built for.
func TestAnyOfSetAdmitsARoleHoldingOnlyOneDomain(t *testing.T) {
	t.Parallel()

	account := "ac_REDACTED_ACCOUNT_ID"
	officeStaff := &Identity{
		Type:   IdentityActorTypeUser,
		Target: &IdentityTarget{AccountID: account},
		Actor: &IdentityActor{
			RelationType: IdentityRelationTypeInternal,
			ID:           "usr_office",
			Name:         strPtr("Gwen Marion"),
			AccountID:    &account,
			RoleID:       strPtr("0bf9d328-d213-4f20-a1eb-239075f0cd7e"),
			RoleName:     strPtr("Office Staff"),
			RoleType:     strPtr("user"),
			Permissions:  map[string]bool{"sales_orders:read": true},
		},
	}

	if apiErr := officeStaff.CheckHasAnyPermission(
		Permission{Domain: PermissionDomainSalesOrders, Action: ActionRead},
		Permission{Domain: PermissionDomainProductionSchedules, Action: ActionRead},
	); apiErr != nil {
		t.Fatalf("sales_orders:read should satisfy the any-of set: %s", apiErr.PublicMessage)
	}

	// The schedules permission on its own must still not be satisfied by a sales-only role,
	// or the any-of set would be hiding a genuinely missing grant.
	if apiErr := officeStaff.CheckHasPermission(PermissionDomainProductionSchedules, ActionRead); apiErr == nil {
		t.Fatal("a sales-only role must still fail a production_schedules:read-only check")
	}
}
