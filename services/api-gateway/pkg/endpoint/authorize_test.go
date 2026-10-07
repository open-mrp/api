package apiendpoint

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

func ctxWithPerms(perms map[string]bool, roleType string) context.Context {
	accountID := "acct_1"
	actor := &types.IdentityActor{ID: "user_1", AccountID: &accountID, Permissions: perms}
	if roleType != "" {
		rt := roleType
		actor.RoleType = &rt
	}
	id := &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor:  actor,
	}
	return appctx.WithIdentity(context.Background(), id)
}

func ep(perms types.AnyOfPermissions, role constants.RoleType) *APIEndpoint[any, any] {
	return &APIEndpoint[any, any]{RequiredPermissions: perms, RequiredRoleType: role}
}

func TestAuthorize_NoDeclaration_Allows(t *testing.T) {
	// An endpoint that declares nothing is not gated here (downstream enforces).
	if err := ep(nil, "").authorize(context.Background()); err != nil {
		t.Errorf("undeclared endpoint should pass the gate, got %v", err)
	}
}

func TestAuthorize_Permissions_OR(t *testing.T) {
	read := types.AnyOfPermissions{{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}}
	orSet := types.AnyOfPermissions{
		{Domain: types.PermissionDomainTeamUsers, Action: types.ActionUpdate},
		{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate},
		{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
	}

	// Holds the exact permission → allowed.
	if err := ep(read, "").authorize(ctxWithPerms(map[string]bool{"customers:read": true}, "")); err != nil {
		t.Errorf("caller with customers:read should pass, got %v", err)
	}
	// Holds none of the declared → rejected fast.
	if err := ep(read, "").authorize(ctxWithPerms(map[string]bool{"products:read": true}, "")); err == nil {
		t.Error("caller without customers:read should be rejected")
	}
	// OR: holds ONE of the relation set (the customer-contact case) → allowed,
	// even though it lacks team/suppliers. The precise check happens downstream.
	if err := ep(orSet, "").authorize(ctxWithPerms(map[string]bool{"customers:update": true}, "")); err != nil {
		t.Errorf("caller with one of the OR set should pass, got %v", err)
	}
	// Holds none of the OR set → rejected.
	if err := ep(orSet, "").authorize(ctxWithPerms(map[string]bool{"products:update": true}, "")); err == nil {
		t.Error("caller with none of the OR set should be rejected")
	}
}

func TestAuthorize_AdminBypassesPermissions(t *testing.T) {
	read := types.AnyOfPermissions{{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}}
	if err := ep(read, "").authorize(ctxWithPerms(map[string]bool{}, string(constants.RoleTypeAdmin))); err != nil {
		t.Errorf("admin should bypass the permission gate, got %v", err)
	}
}

func TestAuthorize_RoleType(t *testing.T) {
	adminOnly := ep(nil, constants.RoleTypeAdmin)
	// Non-admin → rejected.
	if err := adminOnly.authorize(ctxWithPerms(map[string]bool{}, string(constants.RoleTypeCustom))); err == nil {
		t.Error("non-admin should be rejected from an admin-only endpoint")
	}
	// Admin → allowed.
	if err := adminOnly.authorize(ctxWithPerms(map[string]bool{}, string(constants.RoleTypeAdmin))); err != nil {
		t.Errorf("admin should pass an admin-only endpoint, got %v", err)
	}
}

func TestAuthorize_MissingIdentity_Rejected(t *testing.T) {
	read := types.AnyOfPermissions{{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}}
	err := ep(read, "").authorize(context.Background())
	if err == nil {
		t.Fatal("a permissioned endpoint with no identity in context should be rejected")
	}
	// No identity means unauthenticated, not under-permissioned → 401, not 403.
	if got := apierror.GetHTTPStatusCode(err.Code); got != http.StatusUnauthorized {
		t.Errorf("missing identity should yield 401, got %d", got)
	}
}

// An unauthenticated caller (no session cookie/token resolves to an
// unauthenticated identity) must get a 401 so the client knows to authenticate,
// not a generic 403 that reads as a missing permission. Regression guard for the
// flood of unattributed "You do not have permission" 403s in the request logs.
func TestAuthorize_Unauthenticated_Returns401(t *testing.T) {
	read := types.AnyOfPermissions{{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}}
	ctx := appctx.WithIdentity(context.Background(), types.GetUnauthenticatedIdentity(nil))
	err := ep(read, "").authorize(ctx)
	if err == nil {
		t.Fatal("unauthenticated caller should be rejected from a permissioned endpoint")
	}
	if got := apierror.GetHTTPStatusCode(err.Code); got != http.StatusUnauthorized {
		t.Errorf("unauthenticated caller should yield 401, got %d (%q)", got, err.PublicMessage)
	}
}

// ctxWithRelationActor builds an identity for an actor reaching the target
// account through the given account relation (customer/supplier). Such actors
// carry no permission set; their access is authorized downstream.
func ctxWithRelationActor(relation types.IdentityRelationType) context.Context {
	accountID := "acct_vendor"
	actorAccountID := "acct_customer"
	id := &types.Identity{
		Type:   types.IdentityActorTypeAPIKey,
		Target: &types.IdentityTarget{AccountID: accountID, RelationType: &relation},
		Actor: &types.IdentityActor{
			ID:           "apky_1",
			AccountID:    &actorAccountID,
			RelationType: relation,
			Permissions:  map[string]bool{},
		},
	}
	return appctx.WithIdentity(context.Background(), id)
}

// Customer- and supplier-relation actors hold no permissions; the coarse gateway
// gate must let them through so the downstream service can make the precise,
// relation-scoped decision (regression guard for the customer-portal 403s).
func TestAuthorize_RelationActor_BypassesGate(t *testing.T) {
	read := types.AnyOfPermissions{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead}}
	for _, relation := range []types.IdentityRelationType{
		types.IdentityRelationTypeCustomer,
		types.IdentityRelationTypeSupplier,
	} {
		if err := ep(read, "").authorize(ctxWithRelationActor(relation)); err != nil {
			t.Errorf("%s relation actor should bypass the gateway permission gate, got: %v", relation, err)
		}
	}
}

// Internal actors with none of the declared permissions are still rejected at the
// gate — the bypass is scoped to customer/supplier relations only.
func TestAuthorize_InternalActorWithoutPermission_Rejected(t *testing.T) {
	read := types.AnyOfPermissions{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead}}
	if err := ep(read, "").authorize(ctxWithRelationActor(types.IdentityRelationTypeInternal)); err == nil {
		t.Error("an internal actor holding none of the declared permissions should be rejected")
	}
}

// A user may act on their own record without the permission the endpoint asks of anyone acting on
// someone else's, when the endpoint names the path parameter that identifies them.
func TestActsOnSelf(t *testing.T) {
	update := types.AnyOfPermissions{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionUpdate}}
	request := func(ctx context.Context, pathID string) *http.Request {
		r, _ := http.NewRequestWithContext(appctx.WithPathParams(ctx, map[string]string{"id": pathID}), http.MethodPatch, "/v1/identity/users/"+pathID, nil)
		return r
	}
	withoutPerms := ctxWithPerms(map[string]bool{}, "")

	self := &APIEndpoint[any, any]{RequiredPermissions: update, SelfPathParam: "id"}
	if !self.actsOnSelf(request(withoutPerms, "user_1")) {
		t.Error("the caller's own ID in the path should count as acting on themselves")
	}
	if self.actsOnSelf(request(withoutPerms, "user_2")) {
		t.Error("another user's ID in the path must not count as acting on themselves")
	}
	if ep(update, "").actsOnSelf(request(withoutPerms, "user_1")) {
		t.Error("an endpoint that names no self parameter must not exempt anyone")
	}

	apiKey := ctxWithRelationActor(types.IdentityRelationTypeInternal)
	if self.actsOnSelf(request(apiKey, "apky_1")) {
		t.Error("only a signed-in user acts on themselves; an API key's ID in the path must not count")
	}
	unauthenticated := appctx.WithIdentity(context.Background(), types.GetUnauthenticatedIdentity(nil))
	if self.actsOnSelf(request(unauthenticated, "")) {
		t.Error("an unauthenticated caller must not count as acting on themselves")
	}
}

func TestAuthorize_RequiresAllPermissions(t *testing.T) {
	both := &APIEndpoint[any, any]{
		RequiredPermissions: types.AnyOfPermissions{
			{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionDelete},
		},
		RequiresAllPermissions: true,
	}

	for _, perms := range []map[string]bool{{"customers:update": true}, {"customers:delete": true}} {
		if err := both.authorize(ctxWithPerms(perms, "")); err == nil || err.Code != apierror.ErrorCodeInsufficientPerms {
			t.Errorf("caller with only %v should be refused, got %v", perms, err)
		}
	}
	if err := both.authorize(ctxWithPerms(map[string]bool{"customers:update": true, "customers:delete": true}, "")); err != nil {
		t.Errorf("caller with both should pass, got %v", err)
	}
	if err := both.authorize(ctxWithPerms(map[string]bool{}, string(constants.RoleTypeAdmin))); err != nil {
		t.Errorf("admin should bypass the permission gate, got %v", err)
	}
}

// ctxActingIn builds an internal caller holding perms that acts in an account it relates to as relation, or in its own account when relation is empty.
func ctxActingIn(relation types.IdentityRelationType, perms map[string]bool, roleType string) context.Context {
	ctx := ctxWithPerms(perms, roleType)
	if relation == "" {
		return ctx
	}
	id, _ := appctx.GetIdentityFromContext(ctx)
	id.Target = &types.IdentityTarget{AccountID: "acct_counterparty", RelationType: &relation}
	return ctx
}

// A counterparty-routed endpoint asks for exactly one permission per request: its own in the caller's account, and the customers or suppliers permission when acting in one of those accounts.
func TestAuthorize_CounterpartyPermissions(t *testing.T) {
	readOrders := &APIEndpoint[any, any]{
		RequiredPermissions:     types.AnyOfPermissions{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead}},
		CounterpartyPermissions: Counterparties(types.ActionRead),
	}
	createMaterials := &APIEndpoint[any, any]{
		RequiredPermissions:     types.AnyOfPermissions{{Domain: types.PermissionDomainMaterials, Action: types.ActionCreate}},
		CounterpartyPermissions: Counterparties(types.ActionUpdate),
	}
	poLines := &APIEndpoint[any, any]{
		RequiredPermissions:     types.AnyOfPermissions{{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionUpdate}},
		CounterpartyPermissions: CounterpartyPermissions{Supplier: types.Permission{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}},
	}
	own, customer, supplier := types.IdentityRelationType(""), types.IdentityRelationTypeCustomer, types.IdentityRelationTypeSupplier

	cases := []struct {
		name     string
		endpoint *APIEndpoint[any, any]
		relation types.IdentityRelationType
		holds    string
		allowed  bool
		names    string
	}{
		{"own account, own permission", readOrders, own, "sales_orders:read", true, ""},
		{"own account, customers permission", readOrders, own, "customers:read", false, "sales_orders:read"},
		{"own account, suppliers permission", readOrders, own, "suppliers:read", false, "sales_orders:read"},
		{"customer account, customers permission", readOrders, customer, "customers:read", true, ""},
		{"customer account, own permission", readOrders, customer, "sales_orders:read", false, "customers:read"},
		{"customer account, suppliers permission", readOrders, customer, "suppliers:read", false, "customers:read"},
		{"supplier account, suppliers permission", readOrders, supplier, "suppliers:read", true, ""},
		{"supplier account, customers permission", readOrders, supplier, "customers:read", false, "suppliers:read"},
		{"create in own account", createMaterials, own, "materials:create", true, ""},
		{"create in own account with customers:update", createMaterials, own, "customers:update", false, "materials:create"},
		{"create in a customer's account takes customers:update", createMaterials, customer, "customers:update", true, ""},
		{"create in a customer's account refuses customers:create", createMaterials, customer, "customers:create", false, "customers:update"},
		{"create in a supplier's account takes suppliers:update", createMaterials, supplier, "suppliers:update", true, ""},
		{"no customer entry keeps the own permission", poLines, customer, "purchase_orders:update", true, ""},
		{"no customer entry refuses customers:update", poLines, customer, "customers:update", false, "purchase_orders:update"},
		{"supplier entry", poLines, supplier, "suppliers:update", true, ""},
		{"supplier entry refuses the own permission", poLines, supplier, "purchase_orders:update", false, "suppliers:update"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.endpoint.authorize(ctxActingIn(tc.relation, map[string]bool{tc.holds: true}, ""))
			if tc.allowed {
				if err != nil {
					t.Fatalf("holding %s should pass, got %v", tc.holds, err)
				}
				return
			}
			if err == nil || err.Code != apierror.ErrorCodeInsufficientPerms {
				t.Fatalf("holding %s should be refused, got %v", tc.holds, err)
			}
			if !strings.Contains(err.PublicMessage, tc.names) {
				t.Errorf("refusal %q should name %s", err.PublicMessage, tc.names)
			}
		})
	}

	for _, relation := range []types.IdentityRelationType{own, customer, supplier} {
		if err := readOrders.authorize(ctxActingIn(relation, map[string]bool{}, string(constants.RoleTypeAdmin))); err != nil {
			t.Errorf("admin acting in a %q account should pass, got %v", relation, err)
		}
	}
	if err := readOrders.authorize(ctxWithRelationActor(customer)); err != nil {
		t.Errorf("a customer-relation actor is authorized downstream, got %v", err)
	}
}
