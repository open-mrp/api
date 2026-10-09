package service

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	clientmock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/client"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"go.uber.org/mock/gomock"
)

const testAccountID = "ac_roles"

func roleTestIdentity(roleType constants.RoleType, perms ...string) *types.Identity {
	rt := string(roleType)
	acct := testAccountID
	granted := map[string]bool{}
	for _, p := range perms {
		granted[p] = true
	}
	return &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: testAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_caller",
			AccountID:    &acct,
			RoleType:     &rt,
			Permissions:  granted,
		},
	}
}

func TestCheckAssignableRole(t *testing.T) {
	otherAccount := "ac_other"
	ownAccount := testAccountID

	tests := []struct {
		name       string
		identity   *types.Identity
		role       *domain.RoleInfo
		rolePerms  map[string]bool
		wantStatus int
		wantMsg    string
	}{
		{
			name:      "subset role is allowed",
			identity:  roleTestIdentity(constants.RoleTypeCustom, "agents:create", "products:read", "products:update"),
			role:      &domain.RoleInfo{ID: "rl_sub", RoleType: string(constants.RoleTypeCustom), AccountID: &ownAccount},
			rolePerms: map[string]bool{"products:read": true, "products:update": true},
		},
		{
			name:       "role with a permission the caller lacks is rejected",
			identity:   roleTestIdentity(constants.RoleTypeCustom, "agents:create", "products:read"),
			role:       &domain.RoleInfo{ID: "rl_wide", RoleType: string(constants.RoleTypeCustom), AccountID: &ownAccount},
			rolePerms:  map[string]bool{"products:read": true, "roles:update": true, "users:create": true},
			wantStatus: http.StatusForbidden,
			wantMsg:    "roles:update, users:create",
		},
		{
			name:       "admin role is rejected for a non-admin",
			identity:   roleTestIdentity(constants.RoleTypeCustom, "agents:create"),
			role:       &domain.RoleInfo{ID: "rl_admin", RoleType: string(constants.RoleTypeAdmin)},
			wantStatus: http.StatusForbidden,
			wantMsg:    "admin role",
		},
		{
			name:     "admin may attach the admin role",
			identity: roleTestIdentity(constants.RoleTypeAdmin),
			role:     &domain.RoleInfo{ID: "rl_admin", RoleType: string(constants.RoleTypeAdmin)},
		},
		{
			name:     "admin may attach any role",
			identity: roleTestIdentity(constants.RoleTypeAdmin),
			role:     &domain.RoleInfo{ID: "rl_wide", RoleType: string(constants.RoleTypeCustom), AccountID: &ownAccount},
		},
		{
			name:       "another account's role is not found, even for an admin",
			identity:   roleTestIdentity(constants.RoleTypeAdmin),
			role:       &domain.RoleInfo{ID: "rl_foreign", RoleType: string(constants.RoleTypeCustom), AccountID: &otherAccount},
			wantStatus: http.StatusBadRequest,
			wantMsg:    "Role not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			core := clientmock.NewMockCoreClient(ctrl)
			core.EXPECT().GetRoleInfo(gomock.Any(), tt.role.ID).Return(tt.role, nil)
			if tt.rolePerms != nil {
				core.EXPECT().GetRolePermissions(gomock.Any(), tt.role.ID).Return(tt.rolePerms, nil)
			}
			s := &agentDefSvcImpl{coreClient: core}

			apiErr := s.checkAssignableRole(context.Background(), tt.identity, tt.role.ID, false)
			if tt.wantStatus == 0 {
				if apiErr != nil {
					t.Fatalf("unexpected error: %v", apiErr.PublicMessage)
				}
				return
			}
			if apiErr == nil {
				t.Fatal("expected an error, got nil")
			}
			if got := apierror.GetHTTPStatusCode(apiErr.Code); got != tt.wantStatus {
				t.Errorf("status = %d, want %d", got, tt.wantStatus)
			}
			if apiErr.Param != "role_id" {
				t.Errorf("param = %q, want role_id", apiErr.Param)
			}
			if !strings.Contains(apiErr.PublicMessage, tt.wantMsg) {
				t.Errorf("message %q does not contain %q", apiErr.PublicMessage, tt.wantMsg)
			}
		})
	}
}

func TestCheckAssignableRoleUnknownRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	core := clientmock.NewMockCoreClient(ctrl)
	core.EXPECT().GetRoleInfo(gomock.Any(), "rl_missing").Return(nil, apierror.NewResourceNotFoundError("role not found"))
	s := &agentDefSvcImpl{coreClient: core}

	apiErr := s.checkAssignableRole(context.Background(), roleTestIdentity(constants.RoleTypeAdmin), "rl_missing", false)
	if apiErr == nil || apierror.GetHTTPStatusCode(apiErr.Code) != http.StatusBadRequest || apiErr.Param != "role_id" {
		t.Fatalf("expected a 400 on role_id, got %+v", apiErr)
	}
}

func TestCheckAssignableRoleExistingRoleMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	core := clientmock.NewMockCoreClient(ctrl)
	acct := testAccountID
	core.EXPECT().GetRoleInfo(gomock.Any(), "rl_wide").Return(&domain.RoleInfo{ID: "rl_wide", RoleType: string(constants.RoleTypeCustom), AccountID: &acct}, nil)
	core.EXPECT().GetRolePermissions(gomock.Any(), "rl_wide").Return(map[string]bool{"roles:update": true}, nil)
	s := &agentDefSvcImpl{coreClient: core}

	apiErr := s.checkAssignableRole(context.Background(), roleTestIdentity(constants.RoleTypeCustom, "agents:update"), "rl_wide", true)
	if apiErr == nil || !strings.HasPrefix(apiErr.PublicMessage, "This agent's role") {
		t.Fatalf("expected the existing-role message, got %+v", apiErr)
	}
}

func TestCheckAssignableRoleExistingRoleThatNoLongerExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	core := clientmock.NewMockCoreClient(ctrl)
	core.EXPECT().GetRoleInfo(gomock.Any(), "rl_gone").Return(nil, apierror.NewResourceNotFoundError("role not found"))
	s := &agentDefSvcImpl{coreClient: core}

	if apiErr := s.checkAssignableRole(context.Background(), roleTestIdentity(constants.RoleTypeCustom, "agents:update"), "rl_gone", true); apiErr != nil {
		t.Fatalf("a dangling role grants nothing, so editing the agent must pass: %v", apiErr.PublicMessage)
	}
}
