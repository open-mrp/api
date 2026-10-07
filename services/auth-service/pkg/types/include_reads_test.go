package types

import (
	"testing"

	apierror "github.com/open-mrp/api/shared/errors"
)

func internalIdentityWithout(perms map[string]bool) *Identity {
	account := "acct_merchant"
	roleID := "role_viewer"
	roleType := "custom"
	return &Identity{
		Type:   IdentityActorTypeUser,
		Target: &IdentityTarget{AccountID: account},
		Actor: &IdentityActor{
			RelationType: IdentityRelationTypeInternal,
			ID:           "usr_internal",
			AccountID:    &account,
			RoleID:       &roleID,
			RoleType:     &roleType,
			Permissions:  perms,
		},
	}
}

func customerPortalIdentity() *Identity {
	customerAccount := "acct_customer"
	relation := IdentityRelationTypeCustomer
	return &Identity{
		Type:   IdentityActorTypeUser,
		Target: &IdentityTarget{AccountID: "acct_merchant", RelationType: &relation},
		Actor: &IdentityActor{
			RelationType: IdentityRelationTypeCustomer,
			ID:           "usr_customer",
			AccountID:    &customerAccount,
			Permissions:  map[string]bool{},
		},
	}
}

func TestForIncludeReads_ReturnsAFlaggedCopy(t *testing.T) {
	t.Parallel()

	original := internalIdentityWithout(map[string]bool{"sales_orders:read": true})
	flagged := original.ForIncludeReads()

	if flagged == original {
		t.Fatal("ForIncludeReads returned the identity itself")
	}
	if original.IncludeReads {
		t.Fatal("ForIncludeReads set the flag on the identity the request carries")
	}
	if !flagged.IncludeReads {
		t.Fatal("the copy does not carry the flag")
	}
	if flagged.Type != original.Type || flagged.Actor != original.Actor || flagged.Target != original.Target {
		t.Fatalf("the copy is not the same caller: %+v vs %+v", flagged, original)
	}
	if (*Identity)(nil).ForIncludeReads() != nil {
		t.Fatal("a nil identity must stay nil")
	}
}

func TestCheckHasAnyPermission_IncludeReadsPassReadsOnly(t *testing.T) {
	t.Parallel()

	read := Permission{Domain: PermissionDomainItems, Action: ActionRead}
	update := Permission{Domain: PermissionDomainItems, Action: ActionUpdate}
	create := Permission{Domain: PermissionDomainItems, Action: ActionCreate}
	del := Permission{Domain: PermissionDomainItems, Action: ActionDelete}

	for _, tc := range []struct {
		name     string
		identity *Identity
	}{
		{"internal actor", internalIdentityWithout(map[string]bool{"sales_orders:read": true})},
		{"customer portal actor", customerPortalIdentity()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if apiErr := tc.identity.CheckHasAnyPermission(read); apiErr == nil {
				t.Fatal("without the flag a read the role does not grant must be refused")
			}

			flagged := tc.identity.ForIncludeReads()
			if apiErr := flagged.CheckHasPermission(read.Domain, read.Action); apiErr != nil {
				t.Fatalf("an included read must pass: %v", apiErr)
			}
			if apiErr := flagged.CheckHasAnyPermission(update, read); apiErr != nil {
				t.Fatalf("a read among the accepted permissions must pass: %v", apiErr)
			}
			for _, write := range []Permission{create, update, del} {
				apiErr := flagged.CheckHasAnyPermission(write)
				if apiErr == nil {
					t.Fatalf("%s must never pass on the flag", write)
				}
				if apiErr.Code != apierror.ErrorCodeInsufficientPerms {
					t.Errorf("%s: code = %q, want %q", write, apiErr.Code, apierror.ErrorCodeInsufficientPerms)
				}
			}
			if apiErr := flagged.CheckHasAnyPermission(create, update); apiErr == nil {
				t.Fatal("writes alone must never pass on the flag")
			}
		})
	}
}

func TestCheckHasAnyPermission_IncludeReadsNeverStandInForCostsRead(t *testing.T) {
	t.Parallel()

	costs := Permission{Domain: PermissionDomainCosts, Action: ActionRead}
	for _, tc := range []struct {
		name     string
		identity *Identity
	}{
		{"internal actor", internalIdentityWithout(map[string]bool{"sales_orders:read": true})},
		{"customer portal actor", customerPortalIdentity()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			flagged := tc.identity.ForIncludeReads()
			if apiErr := flagged.CheckHasAnyPermission(costs); apiErr == nil {
				t.Fatal("costs:read must never pass on the flag")
			}
			if apiErr := flagged.CheckHasAnyPermission(costs, Permission{Domain: PermissionDomainItems, Action: ActionRead}); apiErr != nil {
				t.Fatalf("another read beside it still passes: %v", apiErr)
			}
		})
	}
}

func TestCheckHasAnyPermission_IncludeReadsNeverAdmitAnUnauthenticatedCaller(t *testing.T) {
	t.Parallel()

	target := "acct_merchant"
	identity := GetUnauthenticatedIdentity(&target).ForIncludeReads()

	apiErr := identity.CheckHasPermission(PermissionDomainItems, ActionRead)
	if apiErr == nil {
		t.Fatal("an unauthenticated caller must not read on the flag")
	}
	if apiErr.Code != apierror.ErrorCodeInvalidCredentials {
		t.Fatalf("code = %q, want %q", apiErr.Code, apierror.ErrorCodeInvalidCredentials)
	}
	if identity.IsIncludeRead() {
		t.Fatal("an unauthenticated caller is never an include read")
	}
	if apiErr := identity.CheckIsInternalActorForRead(); apiErr == nil {
		t.Fatal("an unauthenticated caller must not pass the internal-actor read gate")
	}
}

func TestIsIncludeRead_NeedsAnAssignedActorOfTheTarget(t *testing.T) {
	t.Parallel()

	unassigned := (&Identity{
		Type:  IdentityActorTypeUser,
		Actor: &IdentityActor{RelationType: IdentityRelationTypeUnassigned, ID: "usr_new", Permissions: map[string]bool{}},
	}).ForIncludeReads()
	if unassigned.IsIncludeRead() {
		t.Fatal("an actor with no account is never an include read")
	}
	if apiErr := unassigned.CheckHasPermission(PermissionDomainItems, ActionRead); apiErr == nil {
		t.Fatal("an actor with no account must not read on the flag")
	}

	noTarget := internalIdentityWithout(map[string]bool{}).ForIncludeReads()
	noTarget.Target = nil
	if noTarget.IsIncludeRead() {
		t.Fatal("an identity without a target account is never an include read")
	}

	if internalIdentityWithout(map[string]bool{}).IsIncludeRead() {
		t.Fatal("an identity without the flag is never an include read")
	}
	if !customerPortalIdentity().ForIncludeReads().IsIncludeRead() {
		t.Fatal("a customer portal actor loading includes is an include read")
	}
}

func TestCheckIsInternalActorForRead(t *testing.T) {
	t.Parallel()

	portal := customerPortalIdentity()
	if apiErr := portal.CheckIsInternalActorForRead(); apiErr == nil {
		t.Fatal("without the flag a portal actor is not an internal actor")
	}
	if apiErr := portal.ForIncludeReads().CheckIsInternalActorForRead(); apiErr != nil {
		t.Fatalf("a portal actor loading includes passes: %v", apiErr)
	}
	if apiErr := portal.ForIncludeReads().CheckIsInternalActor(); apiErr == nil {
		t.Fatal("CheckIsInternalActor itself must not honor the flag")
	}
	if apiErr := internalIdentityWithout(map[string]bool{}).CheckIsInternalActorForRead(); apiErr != nil {
		t.Fatalf("an internal actor of the target passes as before: %v", apiErr)
	}
}
