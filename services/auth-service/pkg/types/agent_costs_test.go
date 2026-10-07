package types

import (
	"strings"
	"testing"

	apierror "github.com/open-mrp/api/shared/errors"
)

func agentIdentity(roleType string, perms ...string) *Identity {
	account := "acct_merchant"
	name := "Restock Planner"
	granted := map[string]bool{}
	for _, p := range perms {
		granted[p] = true
	}
	return &Identity{
		Type:   IdentityActorTypeAgent,
		Target: &IdentityTarget{AccountID: account},
		Actor: &IdentityActor{
			RelationType: IdentityRelationTypeInternal,
			ID:           "agdf_planner",
			Name:         &name,
			AccountID:    &account,
			RoleID:       strPtr("rl_agent"),
			RoleType:     &roleType,
			Permissions:  granted,
		},
	}
}

var (
	costsReadPermission = Permission{Domain: PermissionDomainCosts, Action: ActionRead}
	itemsReadPermission = Permission{Domain: PermissionDomainItems, Action: ActionRead}
)

func TestCanReadCosts_NeverForAnAgent(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		identity *Identity
	}{
		{"agent whose role grants costs:read", agentIdentity("agent", "costs:read", "items:read")},
		{"agent carrying an admin role type", agentIdentity("admin", "costs:read")},
		{"agent loading an include", agentIdentity("agent", "costs:read").ForIncludeReads()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.identity.CanReadCosts() {
				t.Fatal("an agent must never read costs")
			}
		})
	}

	if !internalIdentityWithout(map[string]bool{"costs:read": true}).CanReadCosts() {
		t.Fatal("a user holding costs:read still reads costs")
	}
}

func TestCheckHasAnyPermission_AgentsNeverHoldCosts(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		identity *Identity
	}{
		{"agent whose role grants costs:read", agentIdentity("agent", "costs:read", "items:read")},
		{"agent carrying an admin role type", agentIdentity("admin", "costs:read", "items:read")},
		{"agent loading an include", agentIdentity("agent", "costs:read", "items:read").ForIncludeReads()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			apiErr := tc.identity.CheckHasPermission(PermissionDomainCosts, ActionRead)
			if apiErr == nil {
				t.Fatal("costs:read must never pass for an agent")
			}
			if apiErr.Code != apierror.ErrorCodeInsufficientPerms {
				t.Errorf("code = %q, want %q", apiErr.Code, apierror.ErrorCodeInsufficientPerms)
			}
			if !strings.Contains(apiErr.PublicMessage, "Restock Planner") {
				t.Errorf("the refusal must name the agent: %q", apiErr.PublicMessage)
			}

			if apiErr := tc.identity.CheckHasAnyPermission(costsReadPermission, itemsReadPermission); apiErr != nil {
				t.Fatalf("another permission the agent holds still passes beside costs:read: %v", apiErr)
			}
			if apiErr := tc.identity.CheckHasPermission(PermissionDomainItems, ActionRead); apiErr != nil {
				t.Fatalf("the agent's other permissions are untouched: %v", apiErr)
			}
		})
	}
}

func TestCheckHasAnyPermission_AgentRefusedEveryAcceptedPermissionButCosts(t *testing.T) {
	t.Parallel()

	identity := agentIdentity("agent", "costs:read")
	apiErr := identity.CheckHasAnyPermission(costsReadPermission, itemsReadPermission)
	if apiErr == nil {
		t.Fatal("an agent holding only costs:read must be refused")
	}
	if !strings.Contains(apiErr.PublicMessage, "items:read") || strings.Contains(apiErr.PublicMessage, "costs:read") {
		t.Errorf("the refusal must offer only what could let the agent through: %q", apiErr.PublicMessage)
	}
}
