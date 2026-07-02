package apiendpoint

import (
	"context"
	"testing"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
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
	if err := ep(read, "").authorize(context.Background()); err == nil {
		t.Error("a permissioned endpoint with no identity in context should be rejected")
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
