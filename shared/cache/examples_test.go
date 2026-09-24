package cache_test

import (
	"context"
	"time"

	"github.com/open-mrp/api/shared/cache"
	apierror "github.com/open-mrp/api/shared/errors"
)

// ExampleNew caches a lookup per account and drops it when the account changes.
func ExampleNew() {
	store, err := cache.NewMemoryStore(nil)
	if err != nil {
		panic(err)
	}
	plans, err := cache.New[string](&cache.Config{Name: "billing.plan", Store: store, TTL: time.Minute})
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	key := cache.Key{Scopes: []string{"account:acct_123"}, ID: "plan"}
	plan, apiErr := plans.GetOrLoad(ctx, key, func(context.Context) (string, *apierror.APIError) {
		return "growth", nil
	})
	if apiErr != nil {
		panic(apiErr)
	}
	_ = plan

	_ = cache.Invalidate(ctx, store, "account:acct_123")
}
