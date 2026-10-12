//go:build integration

package service

import (
	"context"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
)

// TestARateEventRefreshesTheOpenOrderSnapshot proves a dirty mark alone brings an open line's copy up to
// date: the pod's snapshot pass ran at startup and is not due again within the test.
func TestARateEventRefreshesTheOpenOrderSnapshot(t *testing.T) {
	ctx := context.Background()
	pool, repos := salesFactDB(t)
	pauseLiveRefresher(t, pool)
	pod := startEventPod(t, pool, repos, "open-order-pod")
	settle(t, pod)

	var orderID, lineID, rateID string
	if err := pool.QueryRowContext(ctx, `SELECT so.id, sol.id, sol.unit_price_id FROM open_order_line sol
JOIN open_order so ON so.owner_account_id = sol.account_id AND so.id = sol.sales_order_id
WHERE so.owner_account_id = ? ORDER BY sol.id LIMIT 1`, seedAccountID).Scan(&orderID, &lineID, &rateID); err != nil {
		t.Fatalf("the startup pass copied no open line of the seed account: %v", err)
	}
	if _, err := pool.ExecContext(ctx, `UPDATE rate SET value = value + 1 WHERE id = ?`, rateID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.ExecContext(context.Background(), `UPDATE rate SET value = value - 1 WHERE id = ?`, rateID)
		_, _, _ = repos.NewOpenOrderSnapshotRepo().RefreshOrders(context.Background(), []string{orderID})
	})

	publish(ctx, audit.ObservedEvent{AccountID: seedAccountID, Action: constants.AuditActionUpdate, ResourceType: constants.ObjectTypeRate, ResourceID: rateID}, pod)

	deadline := time.Now().Add(10 * time.Second)
	for {
		var stale int
		if err := pool.QueryRowContext(ctx, `SELECT COUNT(*) FROM open_order_line sol JOIN rate r ON r.id = sol.unit_price_id
WHERE sol.id = ? AND NOT (sol.price_value <=> r.value)`, lineID).Scan(&stale); err != nil {
			t.Fatal(err)
		}
		if stale == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the open line's price was not refreshed by its rate's event")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
