package types

import (
	"testing"
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

func TestCanReadCosts_AgentFollowsItsRole(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		identity *Identity
		want     bool
	}{
		{"agent whose role grants costs:read", agentIdentity("agent", "costs:read", "items:read"), true},
		{"agent whose role does not", agentIdentity("agent", "items:read"), false},
		{"agent without costs:read loading an include", agentIdentity("agent", "items:read").ForIncludeReads(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.identity.CanReadCosts(); got != tc.want {
				t.Fatalf("CanReadCosts() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCheckHasPermission_AgentCostsFollowItsRole(t *testing.T) {
	t.Parallel()

	if apiErr := agentIdentity("agent", "costs:read").CheckHasPermission(PermissionDomainCosts, ActionRead); apiErr != nil {
		t.Fatalf("an agent whose role grants costs:read passes: %v", apiErr)
	}
	if apiErr := agentIdentity("agent", "items:read").CheckHasPermission(PermissionDomainCosts, ActionRead); apiErr == nil {
		t.Fatal("an agent whose role lacks costs:read is refused")
	}
	if apiErr := agentIdentity("agent", "items:read").ForIncludeReads().CheckHasPermission(PermissionDomainCosts, ActionRead); apiErr == nil {
		t.Fatal("an include never stands in for costs:read")
	}
}
