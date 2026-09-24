package grpc

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/open-mrp/api/services/auth-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/cache"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

const (
	defaultAccessTTL         = 5 * time.Minute
	defaultAccountContextTTL = time.Minute
	defaultNegativeTTL       = 15 * time.Second
)

// CachedCoreClientConfig configures a CachedCoreClient.
type CachedCoreClientConfig struct {
	// Client (required) is the core-service client that misses are delegated to.
	Client domain.AuthCoreClient

	// Store (required) holds this replica's entries.
	Store *cache.MemoryStore

	// AccessTTL (optional; default: 5m) bounds staleness of roles, permissions, memberships and relations for the changes no audit event reports (global role migrations, purges).
	AccessTTL time.Duration

	// AccountContextTTL (optional; default: 1m) bounds staleness of an account's mode and subscription status; billing webhooks change the latter without an audit event.
	AccountContextTTL time.Duration

	// NegativeTTL (optional; default: 15s) bounds how long a "no access" or "no relation" answer is trusted; customer portal registration grants access without an audit event.
	NegativeTTL time.Duration
}

// WithDefaults returns a copy of the config with unset fields filled. It is safe to call on a nil receiver.
func (c *CachedCoreClientConfig) WithDefaults() *CachedCoreClientConfig {
	if c == nil {
		c = &CachedCoreClientConfig{}
	}
	out := *c
	if out.AccessTTL == 0 {
		out.AccessTTL = defaultAccessTTL
	}
	if out.AccountContextTTL == 0 {
		out.AccountContextTTL = defaultAccountContextTTL
	}
	if out.NegativeTTL == 0 {
		out.NegativeTTL = defaultNegativeTTL
	}
	return &out
}

func (c *CachedCoreClientConfig) validate() error {
	if c.Client == nil {
		return fmt.Errorf("cached core client: client is required")
	}
	if c.Store == nil {
		return fmt.Errorf("cached core client: store is required")
	}
	if c.AccessTTL <= 0 || c.AccountContextTTL <= 0 || c.NegativeTTL <= 0 {
		return fmt.Errorf("cached core client: TTLs must be positive")
	}
	return nil
}

type cachedAccess struct {
	Access *domain.AccountUserAccess
	Found  bool
}

type cachedRelation struct {
	Relation *domain.AuthAccountRelation
	Found    bool
}

// CachedCoreClient serves the per-request credential lookups from a per-replica cache. Entries are scoped to the accounts they describe and dropped when an audit event reports a change to one of them; TTLs bound whatever no event reports.
type CachedCoreClient struct {
	domain.AuthCoreClient

	store          *cache.MemoryStore
	accountContext *cache.Cache[*domain.AccountContext]
	access         *cache.Cache[cachedAccess]
	relation       *cache.Cache[cachedRelation]
	rolePerms      *cache.Cache[map[string]bool]

	// dependents maps an account to the accounts whose entries read its data without being scoped to it: a sandbox inherits its owner's billing, and a counterparty-side relation reads the counterparty's membership.
	mu         sync.Mutex
	dependents map[string]map[string]struct{}
}

var _ domain.AuthCoreClient = (*CachedCoreClient)(nil)

// NewCachedCoreClient wraps cfg.Client. Methods it does not override pass straight through.
func NewCachedCoreClient(cfg *CachedCoreClientConfig) (*CachedCoreClient, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	accountContext, err := cache.New[*domain.AccountContext](&cache.Config{Name: "auth.account_context", Store: cfg.Store, TTL: cfg.AccountContextTTL})
	if err != nil {
		return nil, err
	}
	access, err := cache.New(&cache.Config{Name: "auth.account_access", Store: cfg.Store, TTL: cfg.AccessTTL},
		cache.WithTTLFunc(func(v cachedAccess) time.Duration { return negativeTTL(v.Found, cfg.NegativeTTL) }))
	if err != nil {
		return nil, err
	}
	relation, err := cache.New(&cache.Config{Name: "auth.account_relation", Store: cfg.Store, TTL: cfg.AccessTTL},
		cache.WithTTLFunc(func(v cachedRelation) time.Duration { return negativeTTL(v.Found, cfg.NegativeTTL) }))
	if err != nil {
		return nil, err
	}
	rolePerms, err := cache.New[map[string]bool](&cache.Config{Name: "auth.role_permissions", Store: cfg.Store, TTL: cfg.AccessTTL})
	if err != nil {
		return nil, err
	}

	return &CachedCoreClient{
		AuthCoreClient: cfg.Client,
		store:          cfg.Store,
		accountContext: accountContext,
		access:         access,
		relation:       relation,
		rolePerms:      rolePerms,
		dependents:     make(map[string]map[string]struct{}),
	}, nil
}

func negativeTTL(found bool, ttl time.Duration) time.Duration {
	if found {
		return 0
	}
	return ttl
}

func accountScope(accountID string) string { return "account:" + accountID }

func roleScope(roleID string) string { return "role:" + roleID }

func accountScopes(accountIDs ...string) []string {
	scopes := make([]string, 0, len(accountIDs))
	for _, id := range accountIDs {
		if id != "" {
			scopes = append(scopes, accountScope(id))
		}
	}
	return scopes
}

func (c *CachedCoreClient) GetAccountContext(ctx context.Context, accountID string) (*domain.AccountContext, *apierror.APIError) {
	return c.accountContext.GetOrLoad(ctx, cache.Key{Scopes: accountScopes(accountID), ID: "context"}, func(ctx context.Context) (*domain.AccountContext, *apierror.APIError) {
		accountCtx, apiErr := c.AuthCoreClient.GetAccountContext(ctx, accountID)
		if apiErr == nil && accountCtx != nil && accountCtx.OwnerAccountID != nil && *accountCtx.OwnerAccountID != accountID {
			c.trackDependent(*accountCtx.OwnerAccountID, accountID)
		}
		return accountCtx, apiErr
	})
}

func (c *CachedCoreClient) GetUserAccountAccess(ctx context.Context, userID, accountID string) (*domain.AccountUserAccess, bool, *apierror.APIError) {
	v, apiErr := c.access.GetOrLoad(ctx, cache.Key{Scopes: accountScopes(accountID), ID: "user:" + userID}, func(ctx context.Context) (cachedAccess, *apierror.APIError) {
		access, found, apiErr := c.AuthCoreClient.GetUserAccountAccess(ctx, userID, accountID)
		return cachedAccess{Access: access, Found: found}, apiErr
	})
	return v.Access, v.Found, apiErr
}

// GetAccountRelationByUserID is scoped to the actor account as well as the target: an owner-side match depends on the user's membership there.
func (c *CachedCoreClient) GetAccountRelationByUserID(ctx context.Context, targetAccountID, actorAccountID, userID string) (*domain.AuthAccountRelation, bool, *apierror.APIError) {
	key := cache.Key{Scopes: accountScopes(targetAccountID, actorAccountID), ID: "user:" + userID + ":actor:" + actorAccountID}
	v, apiErr := c.relation.GetOrLoad(ctx, key, func(ctx context.Context) (cachedRelation, *apierror.APIError) {
		relation, found, apiErr := c.AuthCoreClient.GetAccountRelationByUserID(ctx, targetAccountID, actorAccountID, userID)
		if apiErr == nil && found && relation != nil && relation.CounterpartyAccountID != targetAccountID {
			c.trackDependent(relation.CounterpartyAccountID, targetAccountID)
		}
		return cachedRelation{Relation: relation, Found: found}, apiErr
	})
	return v.Relation, v.Found, apiErr
}

func (c *CachedCoreClient) GetAccountRelationByAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*domain.AuthAccountRelation, bool, *apierror.APIError) {
	key := cache.Key{Scopes: accountScopes(ownerAccountID), ID: "api_key:" + strconv.FormatInt(apiKeyID, 10)}
	v, apiErr := c.relation.GetOrLoad(ctx, key, func(ctx context.Context) (cachedRelation, *apierror.APIError) {
		relation, found, apiErr := c.AuthCoreClient.GetAccountRelationByAPIKeyID(ctx, ownerAccountID, apiKeyID)
		return cachedRelation{Relation: relation, Found: found}, apiErr
	})
	return v.Relation, v.Found, apiErr
}

func (c *CachedCoreClient) GetRolePermissions(ctx context.Context, roleID string) (map[string]bool, *apierror.APIError) {
	if roleID == "" {
		return c.AuthCoreClient.GetRolePermissions(ctx, roleID)
	}
	return c.rolePerms.GetOrLoad(ctx, cache.Key{Scopes: []string{roleScope(roleID)}, ID: "permissions"}, func(ctx context.Context) (map[string]bool, *apierror.APIError) {
		return c.AuthCoreClient.GetRolePermissions(ctx, roleID)
	})
}

// HandleAuditEvent drops the entries an audit event may have made stale. Events for resources no cached lookup reads are ignored, so ordinary business writes do not churn the cache.
func (c *CachedCoreClient) HandleAuditEvent(ctx context.Context, e audit.ObservedEvent) {
	var accounts []string
	var scopes []string

	switch e.ResourceType {
	case constants.ObjectTypeAccount, constants.ObjectTypeAccountUser, constants.ObjectTypeSandbox:
		accounts = append(accounts, e.AccountID)
		if e.ResourceType == constants.ObjectTypeAccount {
			accounts = append(accounts, e.ResourceID)
		}
	case constants.ObjectTypeRole:
		accounts = append(accounts, e.AccountID)
		scopes = append(scopes, roleScope(e.ResourceID))
	case constants.ObjectTypeCustomer, constants.ObjectTypeSupplier:
		// Only creates and deletes add or remove a relation or the counterparty's members; the ResourceID is the counterparty account.
		if e.Action != constants.AuditActionCreate && e.Action != constants.AuditActionDelete {
			return
		}
		accounts = append(accounts, e.AccountID, e.ResourceID)
	default:
		return
	}

	for _, accountID := range accounts {
		scopes = append(scopes, accountScopes(accountID)...)
		scopes = append(scopes, accountScopes(c.dependentsOf(accountID)...)...)
	}
	// Scope rotations on a MemoryStore cannot fail.
	_ = cache.Invalidate(ctx, c.store, scopes...)
}

// Flush drops every entry, for when this replica may have missed audit events.
func (c *CachedCoreClient) Flush() {
	c.store.Clear()
}

// trackDependent records that entries scoped to dependent read data owned by on. It runs inside the load, before the entry is stored, so no cached entry is ever missing from the index.
func (c *CachedCoreClient) trackDependent(on, dependent string) {
	if on == "" || dependent == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	set, ok := c.dependents[on]
	if !ok {
		set = make(map[string]struct{})
		c.dependents[on] = set
	}
	set[dependent] = struct{}{}
}

func (c *CachedCoreClient) dependentsOf(accountID string) []string {
	if accountID == "" {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.dependents[accountID]))
	for id := range c.dependents[accountID] {
		out = append(out, id)
	}
	return out
}
