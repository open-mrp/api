package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/cache"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

type countingCoreClient struct {
	domain.AuthCoreClient

	calls        map[string]int
	owner        map[string]string
	hasAccess    bool
	permission   string
	counterparty string
}

func newCountingCoreClient() *countingCoreClient {
	return &countingCoreClient{calls: map[string]int{}, owner: map[string]string{}, hasAccess: true, permission: "orders:read"}
}

func (f *countingCoreClient) GetAccountContext(_ context.Context, accountID string) (*domain.AccountContext, *apierror.APIError) {
	f.calls["context:"+accountID]++
	out := &domain.AccountContext{AccountID: accountID, AccountMode: constants.AccountModeProduction}
	if owner, ok := f.owner[accountID]; ok {
		out.OwnerAccountID = &owner
		out.AccountMode = constants.AccountModeSandbox
	}
	return out, nil
}

func (f *countingCoreClient) GetUserAccountAccess(_ context.Context, userID, accountID string) (*domain.AccountUserAccess, bool, *apierror.APIError) {
	f.calls["access:"+userID+":"+accountID]++
	if !f.hasAccess {
		return nil, false, nil
	}
	return &domain.AccountUserAccess{AccountID: accountID, Permissions: map[string]bool{f.permission: true}}, true, nil
}

func (f *countingCoreClient) GetAccountRelationByUserID(_ context.Context, target, actor, userID string) (*domain.AuthAccountRelation, bool, *apierror.APIError) {
	f.calls["relation:"+target+":"+actor+":"+userID]++
	if f.counterparty != "" {
		return &domain.AuthAccountRelation{ID: "acre_1", CounterpartyAccountID: f.counterparty}, true, nil
	}
	return &domain.AuthAccountRelation{ID: "acre_1"}, true, nil
}

func (f *countingCoreClient) GetRolePermissions(_ context.Context, roleID string) (map[string]bool, *apierror.APIError) {
	f.calls["role:"+roleID]++
	return map[string]bool{f.permission: true}, nil
}

func (f *countingCoreClient) GetAdminRole(context.Context) (string, *apierror.APIError) {
	f.calls["admin_role"]++
	return "rl_admin", nil
}

func newTestCachedClient(t *testing.T, inner domain.AuthCoreClient, clock *time.Time) *CachedCoreClient {
	t.Helper()
	storeCfg := &cache.MemoryStoreConfig{}
	if clock != nil {
		storeCfg.Now = func() time.Time { return *clock }
	}
	store, err := cache.NewMemoryStore(storeCfg)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}
	c, err := NewCachedCoreClient(&CachedCoreClientConfig{Client: inner, Store: store})
	if err != nil {
		t.Fatalf("NewCachedCoreClient: %v", err)
	}
	return c
}

func TestCachedCoreClient_ServesRepeatLookupsFromCache(t *testing.T) {
	inner := newCountingCoreClient()
	c := newTestCachedClient(t, inner, nil)
	ctx := context.Background()

	for range 3 {
		if _, err := c.GetAccountContext(ctx, "ac_1"); err != nil {
			t.Fatal(err)
		}
		if _, found, err := c.GetUserAccountAccess(ctx, "us_1", "ac_1"); err != nil || !found {
			t.Fatalf("access: found=%v err=%v", found, err)
		}
		if _, err := c.GetRolePermissions(ctx, "rl_1"); err != nil {
			t.Fatal(err)
		}
	}

	for _, key := range []string{"context:ac_1", "access:us_1:ac_1", "role:rl_1"} {
		if inner.calls[key] != 1 {
			t.Errorf("%s: core called %d times, want 1", key, inner.calls[key])
		}
	}
}

func TestCachedCoreClient_AccountUserEventDropsThatAccountOnly(t *testing.T) {
	inner := newCountingCoreClient()
	c := newTestCachedClient(t, inner, nil)
	ctx := context.Background()

	_, _, _ = c.GetUserAccountAccess(ctx, "us_1", "ac_1")
	_, _, _ = c.GetUserAccountAccess(ctx, "us_1", "ac_2")

	c.HandleAuditEvent(ctx, audit.ObservedEvent{AccountID: "ac_1", ResourceType: constants.ObjectTypeAccountUser, ResourceID: "acus_9", Action: constants.AuditActionUpdate})
	inner.permission = "orders:write"

	access, _, _ := c.GetUserAccountAccess(ctx, "us_1", "ac_1")
	if !access.Permissions["orders:write"] {
		t.Fatal("role change in ac_1 was served stale")
	}
	_, _, _ = c.GetUserAccountAccess(ctx, "us_1", "ac_2")
	if inner.calls["access:us_1:ac_2"] != 1 {
		t.Fatal("an event in ac_1 evicted ac_2's entries")
	}
}

func TestCachedCoreClient_RoleEventDropsRolePermissionsAndAccountAccess(t *testing.T) {
	inner := newCountingCoreClient()
	c := newTestCachedClient(t, inner, nil)
	ctx := context.Background()

	_, _ = c.GetRolePermissions(ctx, "rl_1")
	_, _, _ = c.GetUserAccountAccess(ctx, "us_1", "ac_1")

	c.HandleAuditEvent(ctx, audit.ObservedEvent{AccountID: "ac_1", ResourceType: constants.ObjectTypeRole, ResourceID: "rl_1", Action: constants.AuditActionUpdate})

	_, _ = c.GetRolePermissions(ctx, "rl_1")
	_, _, _ = c.GetUserAccountAccess(ctx, "us_1", "ac_1")
	if inner.calls["role:rl_1"] != 2 || inner.calls["access:us_1:ac_1"] != 2 {
		t.Fatalf("role update left entries live: %v", inner.calls)
	}
}

func TestCachedCoreClient_OwnerEventReachesItsSandboxes(t *testing.T) {
	inner := newCountingCoreClient()
	inner.owner["ac_sandbox"] = "ac_owner"
	c := newTestCachedClient(t, inner, nil)
	ctx := context.Background()

	_, _ = c.GetAccountContext(ctx, "ac_sandbox")
	c.HandleAuditEvent(ctx, audit.ObservedEvent{AccountID: "ac_owner", ResourceType: constants.ObjectTypeAccount, ResourceID: "ac_owner", Action: constants.AuditActionUpdate})
	_, _ = c.GetAccountContext(ctx, "ac_sandbox")

	if inner.calls["context:ac_sandbox"] != 2 {
		t.Fatal("owner's billing change did not reach its sandbox's context")
	}
}

func TestCachedCoreClient_RelationDiesWithActorAccount(t *testing.T) {
	inner := newCountingCoreClient()
	c := newTestCachedClient(t, inner, nil)
	ctx := context.Background()

	_, _, _ = c.GetAccountRelationByUserID(ctx, "ac_customer", "ac_merchant", "us_1")
	c.HandleAuditEvent(ctx, audit.ObservedEvent{AccountID: "ac_merchant", ResourceType: constants.ObjectTypeAccountUser, ResourceID: "acus_1", Action: constants.AuditActionDelete})
	_, _, _ = c.GetAccountRelationByUserID(ctx, "ac_customer", "ac_merchant", "us_1")

	if inner.calls["relation:ac_customer:ac_merchant:us_1"] != 2 {
		t.Fatal("removing the user from the actor account left the owner-side relation cached")
	}
}

func TestCachedCoreClient_CustomerDeleteDropsCounterparty(t *testing.T) {
	inner := newCountingCoreClient()
	c := newTestCachedClient(t, inner, nil)
	ctx := context.Background()

	_, _, _ = c.GetUserAccountAccess(ctx, "us_portal", "ac_customer")
	c.HandleAuditEvent(ctx, audit.ObservedEvent{AccountID: "ac_merchant", ResourceType: constants.ObjectTypeCustomer, ResourceID: "ac_customer", Action: constants.AuditActionDelete})
	_, _, _ = c.GetUserAccountAccess(ctx, "us_portal", "ac_customer")

	if inner.calls["access:us_portal:ac_customer"] != 2 {
		t.Fatal("deleting a customer left its members' access cached")
	}
}

func TestCachedCoreClient_IgnoresUnrelatedEvents(t *testing.T) {
	inner := newCountingCoreClient()
	c := newTestCachedClient(t, inner, nil)
	ctx := context.Background()

	_, _, _ = c.GetUserAccountAccess(ctx, "us_1", "ac_1")
	for _, e := range []audit.ObservedEvent{
		{AccountID: "ac_1", ResourceType: constants.ObjectTypeSalesOrder, ResourceID: "so_1", Action: constants.AuditActionUpdate},
		{AccountID: "ac_1", ResourceType: constants.ObjectTypeCustomer, ResourceID: "ac_c", Action: constants.AuditActionUpdate},
	} {
		c.HandleAuditEvent(ctx, e)
	}
	_, _, _ = c.GetUserAccountAccess(ctx, "us_1", "ac_1")

	if inner.calls["access:us_1:ac_1"] != 1 {
		t.Fatal("an event no lookup depends on evicted the cache")
	}
}

func TestCachedCoreClient_NegativeAccessExpiresSooner(t *testing.T) {
	now := time.Unix(0, 0)
	inner := newCountingCoreClient()
	inner.hasAccess = false
	c := newTestCachedClient(t, inner, &now)
	ctx := context.Background()

	_, _, _ = c.GetUserAccountAccess(ctx, "us_1", "ac_1")
	now = now.Add(defaultNegativeTTL + time.Second)
	inner.hasAccess = true
	_, found, _ := c.GetUserAccountAccess(ctx, "us_1", "ac_1")

	if !found {
		t.Fatal("a stale 'no access' outlived the negative TTL")
	}
}

func TestCachedCoreClient_FlushDropsEverything(t *testing.T) {
	inner := newCountingCoreClient()
	c := newTestCachedClient(t, inner, nil)
	ctx := context.Background()

	_, _ = c.GetRolePermissions(ctx, "rl_1")
	c.Flush()
	_, _ = c.GetRolePermissions(ctx, "rl_1")

	if inner.calls["role:rl_1"] != 2 {
		t.Fatal("Flush left entries live")
	}
}

func TestCachedCoreClient_PassesThroughUncachedMethods(t *testing.T) {
	inner := newCountingCoreClient()
	c := newTestCachedClient(t, inner, nil)

	for range 2 {
		_, _ = c.GetAdminRole(context.Background())
	}
	if inner.calls["admin_role"] != 2 {
		t.Fatal("uncached method was cached")
	}
}

func TestCachedCoreClient_CounterpartyMembershipChangeReachesRelation(t *testing.T) {
	inner := newCountingCoreClient()
	inner.counterparty = "ac_customer"
	c := newTestCachedClient(t, inner, nil)
	ctx := context.Background()

	_, _, _ = c.GetAccountRelationByUserID(ctx, "ac_merchant", "", "us_portal")
	c.HandleAuditEvent(ctx, audit.ObservedEvent{AccountID: "ac_customer", ResourceType: constants.ObjectTypeAccountUser, ResourceID: "acus_portal", Action: constants.AuditActionUpdate})
	_, _, _ = c.GetAccountRelationByUserID(ctx, "ac_merchant", "", "us_portal")

	if inner.calls["relation:ac_merchant::us_portal"] != 2 {
		t.Fatal("disabling a portal user in the customer account left their relation to the merchant cached")
	}
}
