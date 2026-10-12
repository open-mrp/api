//go:build integration

package service

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/scheduling"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/lease"
)

// delivery_order_fact is SQL end to end, so these run against a migrated database:
//
//	SQL_PREPARE_TEST_DSN='root:...@tcp(localhost:3306)/<db>?parseTime=true' \
//	    go test -tags integration ./services/core-service/internal/service/ -run DeliveryFact
//
// They seed their own account and orders and, after each edit, hold the facts to what the live query
// returns for the same window.

const (
	dlvAccount   = "ac_dlvtest_owner"
	dlvBuyer     = "ac_dlvtest_buyer"
	dlvTestLease = "delivery-fact-refresher-test"
)

var (
	dlvShipBy1 = time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	dlvShipBy2 = time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	dlvStart   = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	dlvEnd     = time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
)

func deliveryFactDB(t *testing.T) (*sql.DB, domain.RepoFactory) {
	t.Helper()
	dsn := os.Getenv("SQL_PREPARE_TEST_DSN")
	if dsn == "" {
		dsn = "root:Testing123!@tcp(localhost:3306)/openmrp?parseTime=true"
	}
	pool, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, pool.Ping())
	return pool, repository.NewRepoFactory(sqlc.New(pool), nil)
}

// seedDeliveryFixture writes an account with four orders: two the report measures (one with lines in two
// product lines, a line in none and a freight line; one fulfilled), an estimate, and an issued order with
// no ship-by date.
func seedDeliveryFixture(t *testing.T, pool *sql.DB) {
	t.Helper()
	exec := func(query string, args ...any) {
		_, err := pool.Exec(query, args...)
		require.NoError(t, err)
	}
	cleanup := func() {
		exec("DELETE FROM pick_line WHERE id LIKE 'pkl\\_dlvt\\_%'")
		exec("DELETE FROM pick WHERE account_id = ?", dlvAccount)
		exec("DELETE FROM sales_order_line WHERE id LIKE 'sol\\_dlvt\\_%'")
		exec("DELETE FROM quantity WHERE id LIKE 'qy\\_dlvt\\_%'")
		exec("DELETE FROM sales_order WHERE owner_account_id = ?", dlvAccount)
		exec("DELETE FROM product WHERE id LIKE 'pd\\_dlvt\\_%'")
		exec("DELETE FROM product_line WHERE account_id = ?", dlvAccount)
		exec("DELETE FROM account_relation WHERE owner_account_id = ?", dlvAccount)
		exec("DELETE FROM account_group WHERE owner_account_id = ?", dlvAccount)
		exec("DELETE FROM account WHERE id IN (?, ?)", dlvAccount, dlvBuyer)
		exec("DELETE FROM delivery_order_fact WHERE owner_account_id = ?", dlvAccount)
		exec("DELETE FROM delivery_fact_dirty")
		exec("DELETE FROM sales_sync WHERE name = 'delivery_fact'")
	}
	cleanup()
	t.Cleanup(cleanup)

	exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, 'Delivery Test Owner', 'company', 'active'),
	      (?, 'Delivery Test Buyer', 'company', 'active')`, dlvAccount, dlvBuyer)
	exec(`INSERT INTO account_group (id, owner_account_id, name, commission_status_code, freight_status_code, account_group_type_code)
	      VALUES ('ag_dlvt', ?, 'Delivery Test Group', 'commission_applied', 'billed_freight', 'type_group')`, dlvAccount)
	exec(`INSERT INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number, priority_code, account_group_id)
	      VALUES ('ar_dlvt', ?, ?, 'customer', 'C-1', 'normal', 'ag_dlvt')`, dlvAccount, dlvBuyer)
	exec(`INSERT INTO product_line (id, name, unit_group_id, account_id) VALUES ('pdln_dlvt_a', 'Line A', 'ungp_dlvt', ?), ('pdln_dlvt_b', 'Line B', 'ungp_dlvt', ?)`,
		dlvAccount, dlvAccount)
	exec(`INSERT INTO product (id, item_id, product_type_code, product_line_id) VALUES
	      ('pd_dlvt_a', 'it_dlvt_a', 'sale', 'pdln_dlvt_a'), ('pd_dlvt_b', 'it_dlvt_b', 'sale', 'pdln_dlvt_b'),
	      ('pd_dlvt_none', 'it_dlvt_none', 'sale', NULL), ('pd_dlvt_freight', 'it_dlvt_freight', 'shipping', 'pdln_dlvt_a')`)

	order := func(id, number, status string, shipBy any) {
		exec(`INSERT INTO sales_order (id, billing_address_id, shipping_address_id, number, priority_code, sales_order_status_code,
		      sales_order_type_code, buyer_account_id, seller_account_id, owner_account_id, created_at, updated_at,
		      ship_by_date, issued_at, first_ship_at, lead_time_days, lead_time_source_code)
		      VALUES (?, 'ad_dlvt', 'ad_dlvt', ?, 'normal', ?, 'sales_order', ?, ?, ?, NOW(3), NOW(3), ?, '2026-02-20 10:00:00', NULL, 14, 'customer')`,
			id, number, status, dlvBuyer, dlvAccount, dlvAccount, shipBy)
	}
	order("or_dlvt_1", "DLV-1", "issued", dlvShipBy1)
	order("or_dlvt_2", "DLV-2", "fulfilled", dlvShipBy2)
	order("or_dlvt_3", "DLV-3", "estimate", dlvShipBy1)
	order("or_dlvt_4", "DLV-4", "issued", nil)
	exec("INSERT INTO pick (id, number, sales_order_id, account_id) VALUES ('pk_dlvt_1', 'PK-DLV-1', 'or_dlvt_1', ?), ('pk_dlvt_2', 'PK-DLV-2', 'or_dlvt_2', ?)",
		dlvAccount, dlvAccount)

	// line inserts an order line of qty and, unless packed is empty, a pick line of packed, packed at packedAt (nil for not yet).
	line := func(id, orderID, product, qty string, packed string, packedAt any) {
		exec("INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, 'un_dlvt', NOW(3), NOW(3))", "qy_dlvt_s"+id, qty)
		exec(`INSERT INTO sales_order_line (id, product_sku, sales_order_id, quantity_id, unit_price_id, product_id, item_id)
		      VALUES (?, 'SKU', ?, ?, ?, ?, 'it_dlvt')`, "sol_dlvt_"+id, orderID, "qy_dlvt_s"+id, "rt_dlvt_"+id, product)
		if packed != "" {
			pick := "pk_dlvt_" + orderID[len(orderID)-1:]
			exec("INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, 'un_dlvt', NOW(3), NOW(3))", "qy_dlvt_p"+id, packed)
			exec("INSERT INTO pick_line (id, pick_id, quantity_id, sales_order_line_id, packed_at) VALUES (?, ?, ?, ?, ?)",
				"pkl_dlvt_"+id, pick, "qy_dlvt_p"+id, "sol_dlvt_"+id, packedAt)
		}
	}
	packedAt := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	line("1a", "or_dlvt_1", "pd_dlvt_a", "5", "3", packedAt)
	line("1b", "or_dlvt_1", "pd_dlvt_b", "7.25", "7.25", nil)
	line("1n", "or_dlvt_1", "pd_dlvt_none", "2", "", nil)
	line("1f", "or_dlvt_1", "pd_dlvt_freight", "1", "1", packedAt)
	line("2a", "or_dlvt_2", "pd_dlvt_a", "4", "4", packedAt)
	line("3a", "or_dlvt_3", "pd_dlvt_a", "9", "", nil)
	line("4a", "or_dlvt_4", "pd_dlvt_a", "11", "", nil)
}

func newDeliveryTestRefresher(pool *sql.DB, repos domain.RepoFactory) *DeliveryFactRefresher {
	return NewDeliveryFactRefresher(&DeliveryFactRefresherConfig{
		Repos:           repos,
		TxManager:       NewTransactionManager(pool, sqlc.New(pool), nil),
		Lease:           lease.New(repository.NewLeaseRepo(sqlc.New(pool))),
		LeaseName:       dlvTestLease,
		ReconcileBudget: time.Minute,
	})
}

// runDeliveryPass ticks until the reconcile pass in progress (or due) has completed.
func runDeliveryPass(t *testing.T, ctx context.Context, r *DeliveryFactRefresher, repos domain.RepoFactory) {
	t.Helper()
	for range 1000 {
		r.Tick(ctx)
		sync, apiErr := repos.NewDeliveryFactRepo().GetSync(ctx)
		require.Nil(t, apiErr)
		if sync.CursorSalesOrderID == nil && sync.LastCompletedAt != nil {
			return
		}
	}
	t.Fatal("the reconcile pass did not complete")
}

// deliveryLiveAndFacts reads the fixture's window twice: from the live tables (with the facts briefly
// marked unready) and from the facts.
func deliveryLiveAndFacts(t *testing.T, ctx context.Context, repos domain.RepoFactory) (live, facts []scheduling.DeliveryOutcome) {
	t.Helper()
	factRepo := repos.NewDeliveryFactRepo()
	sync, apiErr := factRepo.GetSync(ctx)
	require.Nil(t, apiErr)
	require.NotNil(t, sync.LastCompletedAt, "the facts are not ready")

	read := func() []scheduling.DeliveryOutcome {
		out, apiErr := repos.NewProductionScheduleInputRepo().ListDeliveryOutcomes(ctx, dlvAccount, dlvStart, dlvEnd, domain.DeliveryFilters{})
		require.Nil(t, apiErr)
		return out
	}
	facts = read()
	unready := *sync
	unready.LastCompletedAt = nil
	require.Nil(t, factRepo.SaveSync(ctx, unready))
	live = read()
	require.Nil(t, factRepo.SaveSync(ctx, *sync))
	return live, facts
}

func requireFactsMatchLive(t *testing.T, ctx context.Context, repos domain.RepoFactory) []scheduling.DeliveryOutcome {
	t.Helper()
	live, facts := deliveryLiveAndFacts(t, ctx, repos)
	require.Equal(t, live, facts)
	return facts
}

func TestDeliveryFact_BackfillThenMarksKeepItFresh(t *testing.T) {
	pool, repos := deliveryFactDB(t)
	seedDeliveryFixture(t, pool)
	ctx := context.Background()
	refresher := newDeliveryTestRefresher(pool, repos)
	marker := NewDeliveryFactMarker(repos, nil)

	runDeliveryPass(t, ctx, refresher, repos)
	got := requireFactsMatchLive(t, ctx, repos)
	require.Len(t, got, 2, "only the issued and fulfilled orders with a ship-by date are measured")
	require.Equal(t, "or_dlvt_1", got[0].SalesOrderID)
	require.InDelta(t, 14.25, got[0].QuantityOrdered, 1e-9, "sale lines only, across product lines and none")
	require.InDelta(t, 3, got[0].QuantityPacked, 1e-9, "packed pick lines only")
	require.Equal(t, []scheduling.ProductLineRef{{ID: "pdln_dlvt_a", Name: "Line A"}, {ID: "pdln_dlvt_b", Name: "Line B"}}, got[0].ProductLines)
	require.Equal(t, "Delivery Test Buyer", got[0].CustomerName)
	require.Equal(t, "ag_dlvt", got[0].CustomerGroupID)

	exec := func(query string, args ...any) {
		_, err := pool.Exec(query, args...)
		require.NoError(t, err)
	}
	refresh := func(e audit.ObservedEvent) {
		e.AccountID = dlvAccount
		marker.HandleAuditEvent(ctx, e)
		refresher.Tick(ctx)
		marks, apiErr := repos.NewDeliveryFactRepo().ListDirty(ctx, 10)
		require.Nil(t, apiErr)
		require.Empty(t, marks, "the drain clears what it refreshed")
	}

	t.Run("a pick line packed", func(t *testing.T) {
		exec("UPDATE pick_line SET packed_at = NOW(3) WHERE id = 'pkl_dlvt_1b'")
		refresh(audit.ObservedEvent{Action: constants.AuditActionUpdate, ResourceType: constants.ObjectTypePickLine, ResourceID: "pkl_dlvt_1b",
			RootResourceType: constants.ObjectTypeSalesOrder, RootResourceID: "or_dlvt_1"})
		got := requireFactsMatchLive(t, ctx, repos)
		require.InDelta(t, 10.25, got[0].QuantityPacked, 1e-9)
	})

	t.Run("a line quantity edited on its own", func(t *testing.T) {
		exec("UPDATE quantity SET value = 6 WHERE id = 'qy_dlvt_s1a'")
		refresh(audit.ObservedEvent{Action: constants.AuditActionUpdate, ResourceType: constants.ObjectTypeQuantity, ResourceID: "qy_dlvt_s1a"})
		got := requireFactsMatchLive(t, ctx, repos)
		require.InDelta(t, 15.25, got[0].QuantityOrdered, 1e-9)
	})

	t.Run("a pick quantity edited on its own", func(t *testing.T) {
		exec("UPDATE quantity SET value = 2 WHERE id = 'qy_dlvt_p2a'")
		refresh(audit.ObservedEvent{Action: constants.AuditActionUpdate, ResourceType: constants.ObjectTypeQuantity, ResourceID: "qy_dlvt_p2a"})
		got := requireFactsMatchLive(t, ctx, repos)
		require.InDelta(t, 2, got[1].QuantityPacked, 1e-9)
	})

	t.Run("a product moved to another product line", func(t *testing.T) {
		exec("UPDATE product SET product_line_id = 'pdln_dlvt_a' WHERE id = 'pd_dlvt_b'")
		refresh(audit.ObservedEvent{Action: constants.AuditActionUpdate, ResourceType: constants.ObjectTypeProduct, ResourceID: "pd_dlvt_b",
			ChangedFields: []string{"product_line_id"}})
		got := requireFactsMatchLive(t, ctx, repos)
		require.Equal(t, []scheduling.ProductLineRef{{ID: "pdln_dlvt_a", Name: "Line A"}}, got[0].ProductLines)
		facts, apiErr := repos.NewDeliveryFactRepo().GetFacts(ctx, []string{"or_dlvt_1"})
		require.Nil(t, apiErr)
		require.Len(t, facts, 2, "the old line's row is gone: line A and no line remain")
	})

	t.Run("an order's commitment moved", func(t *testing.T) {
		exec("UPDATE sales_order SET ship_by_date = '2026-03-25' WHERE id = 'or_dlvt_1'")
		refresh(audit.ObservedEvent{Action: constants.AuditActionUpdate, ResourceType: constants.ObjectTypeSalesOrder, ResourceID: "or_dlvt_1"})
		got := requireFactsMatchLive(t, ctx, repos)
		require.Equal(t, "or_dlvt_2", got[0].SalesOrderID, "the order now sorts after the other")
		facts, apiErr := repos.NewDeliveryFactRepo().GetFacts(ctx, []string{"or_dlvt_1"})
		require.Nil(t, apiErr)
		for _, f := range facts {
			require.Equal(t, "2026-03-25", f.ShipByDate.Format(time.DateOnly), "no row is left at the old date")
		}
	})

	t.Run("an order no longer measured", func(t *testing.T) {
		exec("UPDATE sales_order SET sales_order_status_code = 'estimate' WHERE id = 'or_dlvt_2'")
		refresh(audit.ObservedEvent{Action: constants.AuditActionUpdate, ResourceType: constants.ObjectTypeSalesOrder, ResourceID: "or_dlvt_2"})
		got := requireFactsMatchLive(t, ctx, repos)
		require.Len(t, got, 1)
	})

	t.Run("an order deleted without an event is removed by the reconcile pass", func(t *testing.T) {
		exec("DELETE FROM sales_order WHERE id = 'or_dlvt_1'")
		require.Nil(t, repos.NewDeliveryFactRepo().RestartReconcile(ctx))
		runDeliveryPass(t, ctx, refresher, repos)
		got := requireFactsMatchLive(t, ctx, repos)
		require.Empty(t, got)
		facts, apiErr := repos.NewDeliveryFactRepo().GetFacts(ctx, []string{"or_dlvt_1"})
		require.Nil(t, apiErr)
		require.Empty(t, facts)
	})
}

func TestDeliveryFact_UnreadyFallsBackToLive(t *testing.T) {
	pool, repos := deliveryFactDB(t)
	seedDeliveryFixture(t, pool)
	ctx := context.Background()

	// No pass has completed: the facts are empty, and the report still answers from the live tables.
	got, apiErr := repos.NewProductionScheduleInputRepo().ListDeliveryOutcomes(ctx, dlvAccount, dlvStart, dlvEnd, domain.DeliveryFilters{})
	require.Nil(t, apiErr)
	require.Len(t, got, 2)
}
