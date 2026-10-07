package types

import "testing"

func supplierPortalIdentity() *Identity {
	supplierAccount := "acct_supplier"
	relation := IdentityRelationTypeSupplier
	return &Identity{
		Type:   IdentityActorTypeAPIKey,
		Target: &IdentityTarget{AccountID: "acct_merchant", RelationType: &relation},
		Actor: &IdentityActor{
			RelationType: IdentityRelationTypeSupplier,
			ID:           "apky_supplier",
			AccountID:    &supplierAccount,
			Permissions:  map[string]bool{"sales_orders:read": true},
		},
	}
}

func merchantTargetingCustomerIdentity() *Identity {
	merchantAccount := "acct_merchant"
	relation := IdentityRelationTypeCustomer
	return &Identity{
		Type:   IdentityActorTypeUser,
		Target: &IdentityTarget{AccountID: "acct_customer", RelationType: &relation},
		Actor: &IdentityActor{
			RelationType: IdentityRelationTypeInternal,
			ID:           "usr_internal",
			AccountID:    &merchantAccount,
			Permissions:  map[string]bool{"customers:read": true},
		},
	}
}

func TestPortalAccountID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		identity *Identity
		want     *string
	}{
		{name: "customer portal narrows to its own account", identity: customerPortalIdentity(), want: new("acct_customer")},
		{name: "supplier portal narrows to its own account", identity: supplierPortalIdentity(), want: new("acct_supplier")},
		{name: "a portal loading what its request includes is still a portal", identity: supplierPortalIdentity().ForIncludeReads(), want: new("acct_supplier")},
		{name: "seller staff read the whole account", identity: internalIdentityWithout(map[string]bool{"sales_orders:read": true}), want: nil},
		{name: "seller staff acting in a customer's account are not a portal", identity: merchantTargetingCustomerIdentity(), want: nil},
		{name: "unauthenticated", identity: GetUnauthenticatedIdentity(new("acct_merchant")), want: nil},
		{name: "no identity", identity: nil, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.identity.PortalAccountID()
			switch {
			case tt.want == nil && got != nil:
				t.Fatalf("expected no narrowing, got %q", *got)
			case tt.want != nil && got == nil:
				t.Fatalf("expected narrowing to %q, got none", *tt.want)
			case tt.want != nil && *got != *tt.want:
				t.Fatalf("expected narrowing to %q, got %q", *tt.want, *got)
			}
		})
	}
}
