package service

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"go.uber.org/mock/gomock"
)

const grantAccountID = "ac_grant"

func grantIdentity(roleType constants.RoleType, perms ...string) *types.Identity {
	rt := string(roleType)
	acct := grantAccountID
	held := map[string]bool{}
	for _, p := range perms {
		held[p] = true
	}
	return &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: grantAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "us_granter",
			AccountID:    &acct,
			RoleType:     &rt,
			Permissions:  held,
		},
	}
}

func requireForbidden(t *testing.T, apiErr *apierror.APIError, param, wantMsg string) {
	t.Helper()
	if apiErr == nil {
		t.Fatal("expected a 403, got nil")
	}
	if got := apierror.GetHTTPStatusCode(apiErr.Code); got != http.StatusForbidden {
		t.Fatalf("status = %d (%s), want 403", got, apiErr.PublicMessage)
	}
	if apiErr.Param != param {
		t.Errorf("param = %q, want %q", apiErr.Param, param)
	}
	if !strings.Contains(apiErr.PublicMessage, wantMsg) {
		t.Errorf("message %q does not contain %q", apiErr.PublicMessage, wantMsg)
	}
}

type grantRoleSvc struct {
	svc      domain.RoleSvc
	roleRepo *repositorymock.MockRoleRepo
	permRepo *repositorymock.MockRolePermissionRepo
}

func newGrantRoleSvc(t *testing.T) grantRoleSvc {
	ctrl := gomock.NewController(t)
	roleRepo := repositorymock.NewMockRoleRepo(ctrl)
	permRepo := repositorymock.NewMockRolePermissionRepo(ctrl)
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewRoleRepo().Return(roleRepo).AnyTimes()
	factory.EXPECT().NewRolePermissionRepo().Return(permRepo).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()
	idem := mediatormock.NewMockIdempotencyMed(ctrl)
	idem.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).Return(&domain.IdempotencyKey{TypeID: "idk_grant", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil).AnyTimes()
	idem.EXPECT().CacheErrorResponse(gomock.Any(), "idk_grant", gomock.Any()).DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError {
		return apiErr
	}).AnyTimes()
	idem.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_grant", gomock.Any()).Return(nil).AnyTimes()
	svc := NewRoleSvc(&RoleSvcConfig{
		Repos:           factory,
		MediatorFactory: &roleTestMediatorFactory{mediators: domain.Mediators{Idempotency: idem}},
		TxManager:       &stubTxManager{factory: factory},
	})
	return grantRoleSvc{svc: svc, roleRepo: roleRepo, permRepo: permRepo}
}

func grantCtx(identity *types.Identity) context.Context {
	return roleIdempotencyCtx(appctx.WithIdentity(context.Background(), identity))
}

func TestCreateRoleRejectsPermissionsTheCallerLacks(t *testing.T) {
	g := newGrantRoleSvc(t)
	_, apiErr := g.svc.CreateRole(grantCtx(grantIdentity(constants.RoleTypeCustom, "roles:create", "products:read")), domain.CreateRoleParams{
		Name: "Pricing",
		Permissions: []domain.CreateRolePermissionInput{
			{PermissionCode: "products", Read: true, Update: true},
			{PermissionCode: "users", Create: true},
		},
	})
	requireForbidden(t, apiErr, "permissions", "products:update, users:create")
	if strings.Contains(apiErr.PublicMessage, "products:read") {
		t.Errorf("only missing permissions are named: %q", apiErr.PublicMessage)
	}
}

func TestCreateRoleWithinTheCallersGrantProceeds(t *testing.T) {
	for _, identity := range []*types.Identity{
		grantIdentity(constants.RoleTypeCustom, "roles:create", "products:read", "products:update"),
		grantIdentity(constants.RoleTypeAdmin),
	} {
		g := newGrantRoleSvc(t)
		g.roleRepo.EXPECT().ExistsByName(gomock.Any(), grantAccountID, "Pricing", nil).Return(true, nil)
		_, apiErr := g.svc.CreateRole(grantCtx(identity), domain.CreateRoleParams{
			Name:        "Pricing",
			Permissions: []domain.CreateRolePermissionInput{{PermissionCode: "products", Read: true, Update: true}},
		})
		if apiErr == nil || apiErr.Code == apierror.ErrorCodeInsufficientPerms {
			t.Fatalf("expected to pass the grant check and stop at the name conflict, got %+v", apiErr)
		}
	}
}

func grantExistingRole(g grantRoleSvc, roleType constants.RoleType, current ...*domain.RolePermission) {
	acct := grantAccountID
	g.roleRepo.EXPECT().Get(gomock.Any(), "rl_team", grantAccountID).Return(&domain.Role{ID: "rl_team", Name: "Team", RoleType: string(roleType), AccountID: &acct}, nil).AnyTimes()
	g.permRepo.EXPECT().ListByRoleID(gomock.Any(), "rl_team").Return(current, nil).AnyTimes()
}

func TestUpdateRoleRejectsAddingPermissionsTheCallerLacks(t *testing.T) {
	g := newGrantRoleSvc(t)
	// The role already grants users:create, which the caller lacks; keeping it is fine, adding orders:delete is not.
	grantExistingRole(g, constants.RoleTypeCustom, &domain.RolePermission{PermissionCode: "users", Create: true})
	perms := []domain.CreateRolePermissionInput{{PermissionCode: "users", Create: true}, {PermissionCode: "orders", Delete: true}}
	_, apiErr := g.svc.UpdateRole(grantCtx(grantIdentity(constants.RoleTypeCustom, "roles:update", "orders:read")), domain.UpdateRoleParams{RoleID: "rl_team", Permissions: &perms})
	requireForbidden(t, apiErr, "permissions", "orders:delete")
	if strings.Contains(apiErr.PublicMessage, "users:create") {
		t.Errorf("permissions the role already grants are not newly granted: %q", apiErr.PublicMessage)
	}
}

func TestUpdateRoleAllowsRemovingPermissions(t *testing.T) {
	g := newGrantRoleSvc(t)
	grantExistingRole(g, constants.RoleTypeCustom,
		&domain.RolePermission{PermissionCode: "users", Create: true},
		&domain.RolePermission{PermissionCode: "orders", Read: true},
	)
	g.permRepo.EXPECT().DeleteByRoleID(gomock.Any(), "rl_team").Return(nil)
	g.permRepo.EXPECT().Create(gomock.Any(), gomock.Any(), "rl_team", domain.CreateRolePermissionInput{PermissionCode: "orders", Read: true}).Return(nil)
	perms := []domain.CreateRolePermissionInput{{PermissionCode: "orders", Read: true}}

	_, apiErr := g.svc.UpdateRole(grantCtx(grantIdentity(constants.RoleTypeCustom, "roles:update")), domain.UpdateRoleParams{RoleID: "rl_team", Permissions: &perms})
	if apiErr != nil {
		t.Fatalf("removing permissions is always allowed: %+v", apiErr)
	}
}

func TestUpdateRoleRejectsANonAdminChangingAnAdminRole(t *testing.T) {
	g := newGrantRoleSvc(t)
	grantExistingRole(g, constants.RoleTypeAdmin)
	name := "Renamed"
	_, apiErr := g.svc.UpdateRole(grantCtx(grantIdentity(constants.RoleTypeCustom, "roles:update")), domain.UpdateRoleParams{RoleID: "rl_team", Name: &name})
	requireForbidden(t, apiErr, "", "admin role")
}

func TestCheckRoleAssignable(t *testing.T) {
	tests := []struct {
		name      string
		identity  *types.Identity
		roleType  constants.RoleType
		rolePerms map[string]bool
		existing  bool
		wantMsg   string
	}{
		{
			name:      "subset role is assignable",
			identity:  grantIdentity(constants.RoleTypeCustom, "account_users:create", "products:read"),
			roleType:  constants.RoleTypeCustom,
			rolePerms: map[string]bool{"products:read": true},
		},
		{
			name:      "wider role is not",
			identity:  grantIdentity(constants.RoleTypeCustom, "account_users:create", "products:read"),
			roleType:  constants.RoleTypeCustom,
			rolePerms: map[string]bool{"products:read": true, "roles:update": true},
			wantMsg:   "The requested role grants permissions you do not hold: roles:update",
		},
		{
			name:     "admin role is not, for a non-admin",
			identity: grantIdentity(constants.RoleTypeCustom, "account_users:update"),
			roleType: constants.RoleTypeAdmin,
			wantMsg:  "admin role",
		},
		{
			name:      "a member's wider current role cannot be changed",
			identity:  grantIdentity(constants.RoleTypeCustom, "account_users:update"),
			roleType:  constants.RoleTypeCustom,
			rolePerms: map[string]bool{"roles:update": true},
			existing:  true,
			wantMsg:   "This member's current role",
		},
		{
			name:     "admin assigns anything",
			identity: grantIdentity(constants.RoleTypeAdmin),
			roleType: constants.RoleTypeAdmin,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			permRepo := repositorymock.NewMockRolePermissionRepo(ctrl)
			if tt.rolePerms != nil {
				permRepo.EXPECT().FindByRoleID(gomock.Any(), "rl_x").Return(tt.rolePerms, nil)
			}
			apiErr := checkRoleAssignable(context.Background(), tt.identity, permRepo, "rl_x", string(tt.roleType), tt.existing)
			if tt.wantMsg == "" {
				if apiErr != nil {
					t.Fatalf("unexpected error: %s", apiErr.PublicMessage)
				}
				return
			}
			requireForbidden(t, apiErr, "role_id", tt.wantMsg)
		})
	}
}
