package service

import (
	"context"
	"log/slog"

	"github.com/open-mrp/api/shared/cache"
)

// InvalidateDelivery drops the accounts' cached delivery reports. Called after delivery_order_fact changes,
// since a report cached between an edit's audit event and the fact refresh was built from the old facts.
func (c *AnalyticsCache) InvalidateDelivery(ctx context.Context, accountIDs []string) {
	if c == nil {
		return
	}
	scopes := make([]string, len(accountIDs))
	for i, id := range accountIDs {
		scopes[i] = analyticsScope(id, analyticsFamilyDelivery)
	}
	if err := cache.Invalidate(ctx, c.cfg.Store, scopes...); err != nil {
		slog.WarnContext(ctx, "analytics cache: delivery invalidation failed; entries expire by TTL", "accounts", len(accountIDs), "error", err)
	}
}
