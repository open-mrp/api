//go:build vitess_smoke

package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/querytag"
)

// Runs queries through a real vtgate (vitess/vttestserver) with the production pool settings, so each
// statement reaches Vitess exactly as it will in production: interpolated, planned by vtgate,
// executed by vttablet. Run it with `make vitess-smoke`, which applies the migrations and seeds
// through that vtgate first. Add a case here when a change adds SQL that vtparse cannot see (SQL
// built in Go) or that plain MySQL accepts but vtgate may plan differently.
func TestVitessSmoke(t *testing.T) {
	dsn := os.Getenv("VITESS_SMOKE_DSN")
	if dsn == "" {
		t.Skip("VITESS_SMOKE_DSN is not set")
	}
	pool, err := db.NewDbPool(&db.Config{DBURI: dsn, WarmConnections: -1, Application: "vitess-smoke"})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	// Tagged, so every statement below also proves vtgate accepts the trailing SQLCommenter comment.
	ctx := querytag.With(context.Background(), querytag.Job, "vitess-smoke")
	q := sqlc.New(pool)

	ids := func(query string, args ...any) []string {
		t.Helper()
		rows, err := pool.QueryContext(ctx, query, args...)
		if err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			out = append(out, id)
		}
		return out
	}

	account := ids("SELECT p.account_id FROM pick p LIMIT 1")[0]
	pickIDs := ids("SELECT id FROM pick WHERE account_id = ?", account)
	orderIDs := ids("SELECT id FROM sales_order WHERE owner_account_id = ?", account)
	buyers := ids("SELECT DISTINCT buyer_account_id FROM sales_order WHERE owner_account_id = ?", account)
	groups := append(ids("SELECT id FROM account_group LIMIT 3"), "ag_none")
	productLines := append(ids("SELECT id FROM product_line LIMIT 3"), "pl_none")
	unitGroups := ids("SELECT id FROM unit_group LIMIT 5")
	carriers := append(ids("SELECT id FROM carrier LIMIT 5"), "ca_none")
	if len(pickIDs) == 0 || len(orderIDs) == 0 || len(buyers) == 0 {
		t.Fatal("seed data is missing picks or orders")
	}

	check := func(name string, apiErrOrErr any) {
		t.Helper()
		switch e := apiErrOrErr.(type) {
		case nil:
		case error:
			if e != nil {
				t.Errorf("%s: %v", name, e)
			}
		}
	}
	checkAPI := func(name string, apiErr *apierror.APIError) {
		t.Helper()
		if apiErr != nil {
			t.Errorf("%s: %s", name, apierror.Describe(apiErr))
		}
	}

	// --- report pools stop their queries on the database side ---
	t.Run("report query timeout", func(t *testing.T) {
		reports, err := db.NewDbPool(&db.Config{DBURI: dsn, WarmConnections: -1, Application: "vitess-smoke", MaxQueryTime: 500 * time.Millisecond})
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		defer reports.Close()
		var one int
		if err := reports.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil || one != 1 {
			t.Fatalf("a quick select under the limit: %v", err)
		}
		// Far more work than half a second: vtgate must stop it, not merely stop waiting for it.
		started := time.Now()
		var n float64
		err = reports.QueryRowContext(ctx, "SELECT SUM(a.value * b.value + c.value) FROM quantity a, quantity b, quantity c").Scan(&n)
		if err == nil {
			t.Fatalf("the query ran to completion (sum %v) instead of being stopped", n)
		}
		if took := time.Since(started); took > 5*time.Second {
			t.Errorf("stopped after %s, want about the 500ms limit", took)
		}
		t.Logf("stopped after %s: %v", time.Since(started).Round(time.Millisecond), err)
		if apiErr := db.MapSQLError(err); apiErr.Code != apierror.ErrorCodeRequestTimeout {
			t.Errorf("maps to %s, want request_timeout", apiErr.Code)
		}

		// vtgate's own directive alone, as when it fires before MySQL's: it too must stop the query and map to a timeout.
		started = time.Now()
		err = pool.QueryRowContext(ctx, "SELECT /*vt+ QUERY_TIMEOUT_MS=500 */ SUM(a.value * b.value + c.value) FROM quantity a, quantity b, quantity c").Scan(&n)
		if err == nil {
			t.Fatal("vtgate's QUERY_TIMEOUT_MS did not stop the query")
		}
		t.Logf("vtgate stopped it after %s: %v", time.Since(started).Round(time.Millisecond), err)
		if apiErr := db.MapSQLError(err); apiErr.Code != apierror.ErrorCodeRequestTimeout {
			t.Errorf("vtgate's timeout maps to %s, want request_timeout", apiErr.Code)
		}
	})

	// --- batched reads (sqlc) ---
	t.Run("sqlc batch reads", func(t *testing.T) {
		_, err := q.GetPicksByIDs(ctx, sqlc.GetPicksByIDsParams{PickIds: pickIDs, AccountID: account})
		check("GetPicksByIDs", err)
		_, err = q.GetSalesOrderLinesForOrders(ctx, sqlc.GetSalesOrderLinesForOrdersParams{SalesOrderIds: orderIDs})
		check("GetSalesOrderLinesForOrders", err)
		_, err = q.GetShipmentIDsForSalesOrders(ctx, orderIDs)
		check("GetShipmentIDsForSalesOrders", err)
		_, err = q.GetInvoiceIDsForSalesOrders(ctx, orderIDs)
		check("GetInvoiceIDsForSalesOrders", err)
		_, err = q.GetSalesOrdersByIDs(ctx, sqlc.GetSalesOrdersByIDsParams{SalesOrderIds: orderIDs, AccountID: account})
		check("GetSalesOrdersByIDs", err)
		_, err = q.GetCustomersByIDs(ctx, sqlc.GetCustomersByIDsParams{OwnerAccountID: account, CounterpartyAccountIds: buyers})
		check("GetCustomersByIDs", err)
		_, err = q.GetPurchaseOrdersByIDs(ctx, sqlc.GetPurchaseOrdersByIDsParams{SalesOrderIds: orderIDs, AccountID: account})
		check("GetPurchaseOrdersByIDs", err)
		_, err = q.ListRelatedCounterpartyIDs(ctx, sqlc.ListRelatedCounterpartyIDsParams{OwnerAccountID: account, CounterpartyAccountIds: append(buyers, "ac_stranger")})
		check("ListRelatedCounterpartyIDs", err)
		_, err = q.ListCarrierOptionsByCarrierIDs(ctx, sqlc.ListCarrierOptionsByCarrierIDsParams{CarrierIds: carriers, AccountID: gosql.NullString{String: account, Valid: true}})
		check("ListCarrierOptionsByCarrierIDs", err)
		if len(unitGroups) > 0 {
			_, err = q.GetUnitGroupsForProductLinesByIDs(ctx, sqlc.GetUnitGroupsForProductLinesByIDsParams{Ids: unitGroups, AccountID: gosql.NullString{String: account, Valid: true}})
			check("GetUnitGroupsForProductLinesByIDs", err)
			_, err = q.GetUnitGroupsForCategoriesByIDs(ctx, unitGroups)
			check("GetUnitGroupsForCategoriesByIDs", err)
		}
	})

	// --- repository paths that assemble SQL or chain queries ---
	t.Run("sales order list", func(t *testing.T) {
		repo := NewSalesOrderRepo(q)
		for name, params := range map[string]domain.ListSalesOrdersParams{
			"plain":    {AccountID: account, Limit: 50},
			"customer": {AccountID: account, Limit: 50, CustomerIDs: buyers},
			"status":   {AccountID: account, Limit: 50, StatusCodes: []string{"issued", "estimate"}},
			"buyer":    {AccountID: account, Limit: 50, BuyerAccountID: &buyers[0]},
		} {
			_, apiErr := repo.List(ctx, params)
			checkAPI("ListSalesOrders/"+name, apiErr)
		}
		_, apiErr := repo.GetByIDs(ctx, account, &buyers[0], orderIDs)
		checkAPI("SalesOrder.GetByIDs", apiErr)
		_, apiErr = repo.GetLinesForOrders(ctx, orderIDs)
		checkAPI("SalesOrder.GetLinesForOrders", apiErr)
	})

	t.Run("pick list", func(t *testing.T) {
		repo := NewPickRepo(q)
		open, closed := "open", "closed"
		// More customers than are merged, so the list counts the set's picks to choose its index.
		manyCustomers := append([]string{}, buyers...)
		for i := 0; len(manyCustomers) <= pickBuyerMergeMax; i++ {
			manyCustomers = append(manyCustomers, fmt.Sprintf("ac_smoke_%03d", i))
		}
		prefix, phrase := "PI", "ICK-00"
		start, end := "2000-01-01", "2100-01-01"
		cases := map[string]domain.ListPicksParams{
			"default ship-by":        {AccountID: account, Limit: 50},
			"created":                {AccountID: account, Limit: 50, Sort: constants.PickSortCreatedAt},
			"open ship-by":           {AccountID: account, Limit: 50, Status: &open},
			"closed created":         {AccountID: account, Limit: 50, Status: &closed, Sort: constants.PickSortCreatedAt},
			"customer":               {AccountID: account, Limit: 50, CustomerIDs: buyers},
			"one customer open":      {AccountID: account, Limit: 50, CustomerIDs: buyers[:1], Status: &open},
			"customers merged open":  {AccountID: account, Limit: 50, CustomerIDs: append([]string{"ac_merge_other"}, buyers...), Status: &open},
			"many customers counted": {AccountID: account, Limit: 50, CustomerIDs: manyCustomers},
			"group and customer":     {AccountID: account, Limit: 50, CustomerIDs: buyers, CustomerGroupIDs: groups},
			"customer open created":  {AccountID: account, Limit: 50, CustomerIDs: buyers[:1], Status: &open, Sort: constants.PickSortCreatedAt},
			"customer group":         {AccountID: account, Limit: 50, CustomerGroupIDs: groups},
			"product line":           {AccountID: account, Limit: 50, ProductLineIDs: productLines},
			"date window":            {AccountID: account, Limit: 50, StartDate: &start, EndDate: &end},
			"number prefix":          {AccountID: account, Limit: 50, Query: &prefix},
			"phrase":                 {AccountID: account, Limit: 50, Query: &phrase},
			"phrase + filters":       {AccountID: account, Limit: 50, Query: &phrase, Status: &closed, CustomerIDs: buyers, CustomerGroupIDs: groups, ProductLineIDs: productLines},
		}
		for name, params := range cases {
			result, apiErr := repo.List(ctx, params)
			checkAPI("ListPicks/"+name, apiErr)
			if apiErr != nil {
				continue
			}
			// Page forward then back through a one-row page to run both keyset directions.
			paged := params
			paged.Limit = 1
			first, apiErr := repo.List(ctx, paged)
			checkAPI("ListPicks/"+name+"/first page", apiErr)
			if apiErr == nil && first.PageInfo.NextCursor != nil {
				paged.Cursor = first.PageInfo.NextCursor
				second, apiErr := repo.List(ctx, paged)
				checkAPI("ListPicks/"+name+"/next page", apiErr)
				if apiErr == nil && second.PageInfo.PrevCursor != nil {
					paged.Cursor = second.PageInfo.PrevCursor
					_, apiErr = repo.List(ctx, paged)
					checkAPI("ListPicks/"+name+"/prev page", apiErr)
				}
			}
			_ = result
		}
	})

	// pageBothWays lists params, then pages forward and back through one-row pages.
	pageBothWays := func(name string, list func(cursor *string, limit int32) (next, prev *string, apiErr *apierror.APIError)) {
		next, _, apiErr := list(nil, 50)
		checkAPI(name, apiErr)
		next, _, apiErr = list(nil, 1)
		checkAPI(name+"/first page", apiErr)
		if apiErr != nil || next == nil {
			return
		}
		_, prev, apiErr := list(next, 1)
		checkAPI(name+"/next page", apiErr)
		if apiErr == nil && prev != nil {
			_, _, apiErr = list(prev, 1)
			checkAPI(name+"/prev page", apiErr)
		}
	}

	// The shipment list builds its SQL in Go: one case per read it chooses (a list-order walk, a
	// customer set's buyer ranges, an item's or product line's matched shipments) and per count.
	t.Run("shipment list", func(t *testing.T) {
		repo := NewShipmentRepo(q)
		items := append(ids("SELECT DISTINCT sol.item_id FROM shipment_line sl JOIN sales_order_line sol ON sol.id = sl.sales_order_line_id LIMIT 2"), "it_none")
		reps := append(ids("SELECT DISTINCT default_sales_rep_id FROM account_relation WHERE owner_account_id = ? AND default_sales_rep_id IS NOT NULL", account), "acus_none")
		status, search := "shipped", "SH"
		start, end := "2000-01-01", "2100-01-01"
		for name, params := range map[string]domain.ListShipmentsParams{
			"unfiltered":                 {AccountID: account},
			"status":                     {AccountID: account, Status: &status},
			"one customer":               {AccountID: account, CustomerIDs: buyers[:1]},
			"customers":                  {AccountID: account, CustomerIDs: append([]string{"ac_none"}, buyers...)},
			"group and sales rep":        {AccountID: account, CustomerGroupIDs: groups, SalesRepIDs: reps},
			"items":                      {AccountID: account, ItemIDs: items},
			"product lines":              {AccountID: account, ProductLineIDs: productLines},
			"items and product lines":    {AccountID: account, ItemIDs: items, ProductLineIDs: productLines},
			"search, window and filters": {AccountID: account, Query: &search, Status: &status, CustomerIDs: buyers, StartDate: &start, EndDate: &end},
		} {
			pageBothWays("ListShipments/"+name, func(cursor *string, limit int32) (*string, *string, *apierror.APIError) {
				params.Cursor, params.Limit = cursor, limit
				result, apiErr := repo.List(ctx, params)
				if apiErr != nil {
					return nil, nil, apiErr
				}
				return result.PageInfo.NextCursor, result.PageInfo.PrevCursor, nil
			})
		}
	})

	t.Run("delivery list", func(t *testing.T) {
		repo := NewDeliveryRepo(q)
		dlvAccount := ids("SELECT account_id FROM delivery LIMIT 1")
		if len(dlvAccount) == 0 {
			t.Fatal("seed data has no deliveries")
		}
		suppliers := append(ids("SELECT DISTINCT so.seller_account_id FROM delivery d JOIN sales_order so ON so.id = d.sales_order_id"), "ac_none")
		items := append(ids("SELECT DISTINCT sol.item_id FROM delivery_line dl JOIN receiving_order_line rol ON rol.id = dl.receiving_order_line_id JOIN sales_order_line sol ON sol.id = rol.sales_order_line_id"), "it_none")
		status, search := "accepted", "DLV"
		from, to := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), time.Now().UTC().Add(time.Hour)
		for name, params := range map[string]domain.ListDeliveriesParams{
			"unfiltered":        {AccountID: dlvAccount[0]},
			"status":            {AccountID: dlvAccount[0], Status: &status},
			"suppliers":         {AccountID: dlvAccount[0], SupplierIDs: suppliers},
			"items":             {AccountID: dlvAccount[0], ItemIDs: items},
			"search and window": {AccountID: dlvAccount[0], Query: &search, SupplierIDs: suppliers, ItemIDs: items, StartDate: &from, EndDate: &to},
		} {
			pageBothWays("ListDeliveries/"+name, func(cursor *string, limit int32) (*string, *string, *apierror.APIError) {
				params.Cursor, params.Limit = cursor, limit
				result, apiErr := repo.List(ctx, params)
				if apiErr != nil {
					return nil, nil, apiErr
				}
				return result.PageInfo.NextCursor, result.PageInfo.PrevCursor, nil
			})
		}
	})

	// --- writes, rolled back ---
	t.Run("writes", func(t *testing.T) {
		tx, err := pool.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback() }()
		txq := q.WithTx(tx)

		check("MarkSalesOrderFreightPending", txq.MarkSalesOrderFreightPending(ctx, sqlc.MarkSalesOrderFreightPendingParams{SalesOrderID: orderIDs[0], AccountID: account}))
		since, err := txq.LockSalesOrderFreightPendingSince(ctx, sqlc.LockSalesOrderFreightPendingSinceParams{SalesOrderID: orderIDs[0], AccountID: account})
		check("LockSalesOrderFreightPendingSince", err)
		if err == nil && !since.Valid {
			t.Error("freight_pending_since was not set")
		}
		_, err = txq.GetSalesOrderFreightPendingSince(ctx, sqlc.GetSalesOrderFreightPendingSinceParams{SalesOrderID: orderIDs[0], AccountID: account})
		check("GetSalesOrderFreightPendingSince", err)
		check("ClearSalesOrderFreightPending", txq.ClearSalesOrderFreightPending(ctx, sqlc.ClearSalesOrderFreightPendingParams{SalesOrderID: orderIDs[0], AccountID: account}))

		check("MergeCustomerPicks", txq.MergeCustomerPicks(ctx, sqlc.MergeCustomerPicksParams{OwnerAccountID: account, TargetAccountID: buyers[0]}))
		check("MergeCustomerShipmentBuyers", txq.MergeCustomerShipmentBuyers(ctx, sqlc.MergeCustomerShipmentBuyersParams{OwnerAccountID: account, TargetAccountID: buyers[0]}))

		carrier := ids("SELECT id FROM carrier LIMIT 1")
		address := ids("SELECT shipping_address_id FROM sales_order WHERE id = ?", orderIDs[0])
		if len(carrier) == 0 || len(address) == 0 {
			t.Fatal("seed data has no carrier or order address")
		}
		check("CreateShipment", txq.CreateShipment(ctx, sqlc.CreateShipmentParams{
			ID: "sh_vitess_smoke", Number: "SMOKE-SH-1", SalesOrderID: orderIDs[0], ShipmentStatusCode: "packed", AccountID: account,
			CarrierID:         gosql.NullString{String: carrier[0], Valid: true},
			ShippingAddressID: gosql.NullString{String: address[0], Valid: true},
		}))
		var shipmentBuyer gosql.NullString
		if err := tx.QueryRowContext(ctx, "SELECT buyer_account_id FROM shipment WHERE id = ?", "sh_vitess_smoke").Scan(&shipmentBuyer); err != nil || !shipmentBuyer.Valid {
			t.Errorf("CreateShipment did not copy the buyer: %v %v", shipmentBuyer, err)
		}

		// CreatePick needs an order without a pick (pick.sales_order_id is unique).
		free := ids("SELECT so.id FROM sales_order so LEFT JOIN pick p ON p.sales_order_id = so.id WHERE so.owner_account_id = ? AND p.id IS NULL LIMIT 1", account)
		if len(free) > 0 {
			check("CreatePick", txq.CreatePick(ctx, sqlc.CreatePickParams{ID: "pk_vitess_smoke", Number: "SMOKE-1", SalesOrderID: free[0], AccountID: account}))
			var buyer gosql.NullString
			if err := tx.QueryRowContext(ctx, "SELECT buyer_account_id FROM pick WHERE id = ?", "pk_vitess_smoke").Scan(&buyer); err != nil || !buyer.Valid {
				t.Errorf("CreatePick did not copy the buyer: %v %v", buyer, err)
			}
		} else {
			t.Log("no pick-less order in the seed; CreatePick not exercised")
		}
	})

	// sales_line_fact maintenance and the sales reports build SQL in Go, which vtparse cannot see.
	// --- payments: list SQL built in Go, keyset both ways, and the flag recompute's row locks ---
	t.Run("payments", func(t *testing.T) {
		txAccount := ids("SELECT account_id FROM transaction LIMIT 1")
		if len(txAccount) == 0 {
			t.Fatal("seed data has no transactions")
		}
		acct := txAccount[0]
		search, status := "TXN", "unallocated"
		from, to := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), time.Now().UTC().Add(time.Hour)

		txs := NewTransactionRepo(q)
		for _, p := range []domain.ListTransactionsParams{
			{AccountID: acct, Limit: 1},
			{AccountID: acct, Limit: 5, Query: &search, Status: &status, TypeCodes: []string{"payment"}, MethodCodes: []string{"check"},
				AdjustmentTypeCodes: []string{"x"}, CustomerIDs: buyers, CustomerGroupIDs: groups, StartDate: &from, EndDate: &to},
			// One per index hint the list sends: every list-order key, the funds key, and the funds and customer keys.
			{AccountID: acct, Limit: 5, Status: &status, TypeCodes: []string{"payment"}, MethodCodes: []string{"check"}},
			{AccountID: acct, Limit: 5, StartDate: &from, EndDate: &to},
			{AccountID: acct, Limit: 5, StartDate: &from, CustomerIDs: buyers},
			{AccountID: acct, Limit: 5, CustomerGroupIDs: groups},
		} {
			page, apiErr := txs.List(ctx, p)
			checkAPI("transactions List", apiErr)
			if page != nil && page.PageInfo.NextCursor != nil {
				p.Cursor = page.PageInfo.NextCursor
				next, apiErr := txs.List(ctx, p)
				checkAPI("transactions List next", apiErr)
				if next != nil && next.PageInfo.PrevCursor != nil {
					p.Cursor = next.PageInfo.PrevCursor
					_, apiErr = txs.List(ctx, p)
					checkAPI("transactions List prev", apiErr)
				}
			}
		}
		customers := ids("SELECT customer_account_id FROM transaction WHERE account_id = ? LIMIT 1", acct)
		if len(customers) > 0 {
			payment := "payment"
			for _, p := range []domain.ListAccountTransactionsParams{
				{AccountID: acct, CustomerAccountID: customers[0], Limit: 5, Status: &status, Query: &search, WithAllocations: true},
				// One per index hint: the customer key alone, and with the status and type keys.
				{AccountID: acct, CustomerAccountID: customers[0], Limit: 1, WithAllocations: true},
				{AccountID: acct, CustomerAccountID: customers[0], Limit: 5, Status: &status, Type: &payment, WithAllocations: true},
			} {
				page, apiErr := txs.ListByCustomer(ctx, p)
				checkAPI("transactions ListByCustomer", apiErr)
				if page != nil && page.PageInfo.NextCursor != nil {
					p.Cursor = page.PageInfo.NextCursor
					_, apiErr = txs.ListByCustomer(ctx, p)
					checkAPI("transactions ListByCustomer next", apiErr)
				}
			}
		}
		if one := ids("SELECT id FROM transaction WHERE account_id = ? LIMIT 1", acct); len(one) > 0 {
			_, apiErr := txs.Get(ctx, acct, one[0])
			checkAPI("transactions Get", apiErr)
		}

		settlements := NewSettlementRepo(q)
		for _, p := range []domain.ListSettlementsParams{
			{AccountID: acct, Limit: 1},
			{AccountID: acct, Limit: 5, Query: &search, TransactionIDs: []string{"tx_none"}, InvoiceIDs: []string{"iv_none"}, StartDate: &from, EndDate: &to},
			// Filters resolved to settlements that exist, so the page reads them by id.
			{AccountID: acct, Limit: 5, TransactionIDs: ids("SELECT transaction_id FROM transaction_allocation WHERE account_id = ? LIMIT 3", acct)},
			{AccountID: acct, Limit: 5, InvoiceIDs: ids("SELECT invoice_id FROM transaction_allocation WHERE account_id = ? LIMIT 3", acct), StartDate: &from},
		} {
			page, apiErr := settlements.List(ctx, p)
			checkAPI("settlements List", apiErr)
			if page != nil && page.PageInfo.NextCursor != nil {
				p.Cursor = page.PageInfo.NextCursor
				_, apiErr = settlements.List(ctx, p)
				checkAPI("settlements List next", apiErr)
			}
		}

		allocations := NewTransactionAllocationRepo(q)
		credits := domain.ListOpenCreditsParams{AccountID: acct, Limit: 1}
		page, apiErr := allocations.ListOpenCredits(ctx, credits)
		checkAPI("ListOpenCredits", apiErr)
		if page != nil && page.PageInfo.NextCursor != nil {
			credits.Cursor = page.PageInfo.NextCursor
			next, apiErr := allocations.ListOpenCredits(ctx, credits)
			checkAPI("ListOpenCredits next", apiErr)
			if next != nil && next.PageInfo.PrevCursor != nil {
				credits.Cursor = next.PageInfo.PrevCursor
				_, apiErr = allocations.ListOpenCredits(ctx, credits)
				checkAPI("ListOpenCredits prev", apiErr)
			}
		}
		_, apiErr = allocations.ListOpenCredits(ctx, domain.ListOpenCreditsParams{AccountID: acct, Limit: 5, CustomerIDs: buyers, SearchQuery: &search, StartDate: &from, EndDate: &to})
		checkAPI("ListOpenCredits filtered", apiErr)
		entries, apiErr := allocations.ListEntries(ctx, domain.ListAllocationEntriesParams{AccountID: acct, Limit: 1, Query: &search, StartDate: &from, EndDate: &to})
		checkAPI("ListEntries", apiErr)
		if entries != nil && entries.PageInfo.NextCursor != nil {
			_, apiErr = allocations.ListEntries(ctx, domain.ListAllocationEntriesParams{AccountID: acct, Limit: 1, Query: &search, Cursor: entries.PageInfo.NextCursor})
			checkAPI("ListEntries next", apiErr)
		}
		// One per shape the entry list builds: each list key, a search resolved to transactions and
		// invoices, and a search past the resolve limit matched row by row.
		payment := "payment"
		entryParams := []domain.ListAllocationEntriesParams{
			{AccountID: acct, Limit: 1},
			{AccountID: acct, Limit: 1, TransactionType: &payment, StartDate: &from, EndDate: &to},
		}
		for _, number := range ids("SELECT t.number FROM transaction_allocation ta JOIN transaction t ON t.id = ta.transaction_id WHERE ta.account_id = ? LIMIT 1", acct) {
			entryParams = append(entryParams, domain.ListAllocationEntriesParams{AccountID: acct, Limit: 1, Query: &number})
		}
		for _, number := range ids("SELECT i.number FROM transaction_allocation ta JOIN invoice i ON i.id = ta.invoice_id WHERE ta.account_id = ? LIMIT 1", acct) {
			entryParams = append(entryParams, domain.ListAllocationEntriesParams{AccountID: acct, Limit: 1, Query: &number, TransactionType: &payment})
		}
		for _, p := range entryParams {
			page, apiErr := allocations.ListEntries(ctx, p)
			checkAPI("ListEntries", apiErr)
			if page != nil && page.PageInfo.NextCursor != nil {
				p.Cursor = page.PageInfo.NextCursor
				next, apiErr := allocations.ListEntries(ctx, p)
				checkAPI("ListEntries next", apiErr)
				if next != nil && next.PageInfo.PrevCursor != nil {
					p.Cursor = next.PageInfo.PrevCursor
					_, apiErr = allocations.ListEntries(ctx, p)
					checkAPI("ListEntries prev", apiErr)
				}
			}
		}
		for _, name := range ids("SELECT a.name FROM transaction_allocation ta JOIN transaction t ON t.id = ta.transaction_id JOIN account a ON a.id = t.customer_account_id WHERE ta.account_id = ? LIMIT 1", acct) {
			limit := allocationSearchResolveLimit
			allocationSearchResolveLimit = 0
			_, apiErr = allocations.ListEntries(ctx, domain.ListAllocationEntriesParams{AccountID: acct, Limit: 1, Query: &name})
			allocationSearchResolveLimit = limit
			checkAPI("ListEntries row-by-row search", apiErr)
		}

		tx, err := pool.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback() }()
		locked := NewSettlementRepo(q.WithTx(tx))
		txIDs := ids("SELECT id FROM transaction WHERE account_id = ? LIMIT 3", acct)
		invIDs := ids("SELECT id FROM invoice WHERE account_id = ? LIMIT 3", acct)
		checkAPI("LockPaymentFlagRows", locked.LockPaymentFlagRows(ctx, acct, txIDs, invIDs))
		_, apiErr = locked.GetTransactionAllocationTotals(ctx, acct, txIDs)
		checkAPI("GetTransactionAllocationTotals", apiErr)
		_, apiErr = locked.GetInvoicePaymentTotals(ctx, acct, invIDs)
		checkAPI("GetInvoicePaymentTotals", apiErr)
		// Rolled back with the transaction above.
		checkAPI("MarkTransactionsCreatedBySettlement", locked.MarkTransactionsCreatedBySettlement(ctx, acct, "sl_smoke", txIDs))
		checkAPI("DeleteSettlementOwnedTransactions", locked.DeleteSettlementOwnedTransactions(ctx, acct, "sl_smoke"))
	})

	t.Run("sales facts and reports", func(t *testing.T) {
		facts := NewSalesFactRepo(q)
		invoices := ids("SELECT id FROM invoice")
		computed, apiErr := facts.ComputeFacts(ctx, invoices)
		checkAPI("ComputeFacts", apiErr)
		if len(computed) == 0 {
			t.Fatal("seed data has no invoiced sales")
		}
		checkAPI("UpsertFacts", facts.UpsertFacts(ctx, computed))
		checkAPI("UpsertFacts again", facts.UpsertFacts(ctx, computed))
		stored, apiErr := facts.GetFacts(ctx, invoices)
		checkAPI("GetFacts", apiErr)
		if len(stored) != len(computed) {
			t.Errorf("stored %d facts, computed %d", len(stored), len(computed))
		}
		_, apiErr = facts.ListInvoicesAfter(ctx, nil, 10)
		checkAPI("ListInvoicesAfter", apiErr)
		_, apiErr = facts.ListFactInvoiceIDsAfter(ctx, "", 10)
		checkAPI("ListFactInvoiceIDsAfter", apiErr)
		_, apiErr = facts.FilterExistingInvoiceIDs(ctx, invoices)
		checkAPI("FilterExistingInvoiceIDs", apiErr)
		for _, scope := range []domain.SalesFactScope{
			domain.SalesFactScopeSalesOrder, domain.SalesFactScopeSalesOrderLine, domain.SalesFactScopeProduct,
			domain.SalesFactScopeQuantity, domain.SalesFactScopeRate, domain.SalesFactScopeItem, domain.SalesFactScopeBuyer,
		} {
			_, apiErr = facts.ResolveInvoiceIDs(ctx, account, scope, []string{"x_none"})
			checkAPI("ResolveInvoiceIDs "+string(scope), apiErr)
		}
		checkAPI("MarkDirty", facts.MarkDirty(ctx, domain.SalesFactScopeInvoice, invoices[0], account))
		marks, apiErr := facts.ListDirty(ctx, 10)
		checkAPI("ListDirty", apiErr)
		for _, m := range marks {
			checkAPI("ClearDirty", facts.ClearDirty(ctx, []domain.SalesFactDirtyMark{m}))
		}
		checkAPI("SaveSync", facts.SaveSync(ctx, domain.SalesFactSync{}))
		_, apiErr = facts.GetSync(ctx)
		checkAPI("GetSync", apiErr)
		checkAPI("DeleteFacts", facts.DeleteFacts(ctx, []string{"ivln_none"}))
		checkAPI("MarkInvoicesDirty", facts.MarkInvoicesDirty(ctx, account, []string{"iv_smoke_a", "iv_smoke_b"}))
		checkAPI("RestartReconcile", facts.RestartReconcile(ctx))
		smokeDay := domain.SalesRollupDay{AccountID: account, Day: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)}
		checkAPI("MarkRollupDays", facts.MarkRollupDays(ctx, []domain.SalesRollupDay{smokeDay, smokeDay}))
		rollupMarks, apiErr := facts.ListRollupDirty(ctx, 10)
		checkAPI("ListRollupDirty", apiErr)
		for _, m := range rollupMarks {
			checkAPI("ClearRollupDirty", facts.ClearRollupDirty(ctx, []domain.SalesRollupDirtyMark{m}))
		}

		reports := NewSalesReportRepo(q)
		start, end := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), time.Now().UTC().Add(time.Hour)
		filter := domain.SalesReportFilter{
			AccountID: account, StartsAt: start, EndsAt: end,
			ComparisonStartsAt: &start, ComparisonEndsAt: &end,
			CustomerIDs: buyers, CustomerGroupIDs: groups, ProductLineIDs: productLines,
			SalesRepIDs: []string{"acus_none"}, ItemIDs: []string{"it_none"},
		}
		_, apiErr = reports.GetSummary(ctx, domain.AnalyzeSalesSummaryParams{SalesReportFilter: filter, TZOffsetMinutes: -300}, true)
		checkAPI("GetSummary", apiErr)
		unfiltered := domain.SalesReportFilter{AccountID: account, StartsAt: start, EndsAt: end, ComparisonStartsAt: &start, ComparisonEndsAt: &end}
		for _, groupBy := range constants.SalesBreakdownGroupBy("").EnumValues() {
			for _, f := range []domain.SalesReportFilter{filter, unfiltered} {
				page, apiErr := reports.GetBreakdown(ctx, domain.AnalyzeSalesBreakdownParams{SalesReportFilter: f, GroupBy: constants.SalesBreakdownGroupBy(groupBy), Limit: 1}, true)
				checkAPI("GetBreakdown "+groupBy, apiErr)
				if page != nil && page.PageInfo.NextCursor != nil {
					next, apiErr := reports.GetBreakdown(ctx, domain.AnalyzeSalesBreakdownParams{SalesReportFilter: f, GroupBy: constants.SalesBreakdownGroupBy(groupBy), Limit: 1, Cursor: page.PageInfo.NextCursor}, true)
					checkAPI("GetBreakdown next "+groupBy, apiErr)
					if next != nil && next.PageInfo.PrevCursor != nil {
						_, apiErr = reports.GetBreakdown(ctx, domain.AnalyzeSalesBreakdownParams{SalesReportFilter: f, GroupBy: constants.SalesBreakdownGroupBy(groupBy), Limit: 1, Cursor: next.PageInfo.PrevCursor}, true)
						checkAPI("GetBreakdown prev "+groupBy, apiErr)
					}
				}
			}
		}
		invPage, apiErr := reports.GetInvoicePage(ctx, domain.AnalyzeSalesInvoicesParams{SalesReportFilter: unfiltered, Limit: 1})
		checkAPI("GetInvoicePage", apiErr)
		if invPage != nil && invPage.PageInfo.NextCursor != nil {
			next, apiErr := reports.GetInvoicePage(ctx, domain.AnalyzeSalesInvoicesParams{SalesReportFilter: unfiltered, Limit: 1, Cursor: invPage.PageInfo.NextCursor})
			checkAPI("GetInvoicePage next", apiErr)
			if next != nil && next.PageInfo.PrevCursor != nil {
				_, apiErr = reports.GetInvoicePage(ctx, domain.AnalyzeSalesInvoicesParams{SalesReportFilter: unfiltered, Limit: 1, Cursor: next.PageInfo.PrevCursor})
				checkAPI("GetInvoicePage prev", apiErr)
			}
		}
		_, apiErr = reports.GetInvoicePage(ctx, domain.AnalyzeSalesInvoicesParams{SalesReportFilter: filter, Limit: 5})
		checkAPI("GetInvoicePage filtered", apiErr)
		linePage, apiErr := reports.GetLinePage(ctx, domain.ListSalesLinesParams{SalesReportFilter: unfiltered, HasWindow: true, Limit: 2})
		checkAPI("GetLinePage", apiErr)
		if linePage != nil && linePage.PageInfo.NextCursor != nil {
			_, apiErr = reports.GetLinePage(ctx, domain.ListSalesLinesParams{SalesReportFilter: unfiltered, Limit: 2, Cursor: linePage.PageInfo.NextCursor})
			checkAPI("GetLinePage next", apiErr)
		}
		_, apiErr = reports.GetLinePage(ctx, domain.ListSalesLinesParams{SalesReportFilter: filter, Limit: 5})
		checkAPI("GetLinePage filtered", apiErr)

		// The rollups: a full sweep's statements, then every report shape that reads them.
		cursor := domain.SalesRollupDay{Day: salesFactSweepFloor}
		for {
			next, apiErr := facts.NextRollupDay(ctx, cursor)
			checkAPI("NextRollupDay", apiErr)
			if next == nil || apiErr != nil {
				break
			}
			checkAPI("RebuildRollupDay", facts.RebuildRollupDay(ctx, *next))
			checkAPI("RebuildRollupMonth", facts.RebuildRollupMonth(ctx, next.AccountID, next.Day))
			cursor = domain.SalesRollupDay{AccountID: next.AccountID, Day: next.Day.AddDate(0, 0, 1)}
		}
		checkAPI("SaveRollupSync", facts.SaveRollupSync(ctx, domain.SalesRollupSync{Cursor: &cursor}))
		_, apiErr = facts.GetRollupSync(ctx)
		checkAPI("GetRollupSync", apiErr)
		salesRollupsReady.Store(true)
		defer salesRollupsReady.Store(false)
		ragged := domain.SalesReportFilter{AccountID: account, StartsAt: start.Add(90 * time.Minute), EndsAt: end, ComparisonStartsAt: &start, ComparisonEndsAt: &end}
		oneLine := ragged
		oneLine.ProductLineIDs, oneLine.SalesRepIDs = productLines[:1], []string{"acus_none"}
		// Several product lines: summed per-line rows plus an exact invoice count from the facts.
		twoLines := ragged
		twoLines.ProductLineIDs = productLines[:2]
		for _, f := range []domain.SalesReportFilter{ragged, oneLine, twoLines} {
			_, apiErr = reports.GetSummary(ctx, domain.AnalyzeSalesSummaryParams{SalesReportFilter: f, TZOffsetMinutes: -300}, true)
			checkAPI("GetSummary from rollups", apiErr)
			for _, groupBy := range constants.SalesBreakdownGroupBy("").EnumValues() {
				_, apiErr = reports.GetBreakdown(ctx, domain.AnalyzeSalesBreakdownParams{SalesReportFilter: f, GroupBy: constants.SalesBreakdownGroupBy(groupBy), Limit: 5}, true)
				checkAPI("GetBreakdown from rollups "+groupBy, apiErr)
			}
		}

		// The buyer summaries and the new-customers report that reads them.
		buyer := domain.SalesBuyerKey{AccountID: account, BuyerAccountID: buyers[0]}
		checkAPI("MarkBuyers", facts.MarkBuyers(ctx, []domain.SalesBuyerKey{buyer, buyer}))
		buyerMarks, apiErr := facts.ListBuyerDirty(ctx, 10)
		checkAPI("ListBuyerDirty", apiErr)
		for _, m := range buyerMarks {
			checkAPI("ClearBuyerDirty", facts.ClearBuyerDirty(ctx, []domain.SalesBuyerDirtyMark{m}))
		}
		next, apiErr := facts.NextBuyers(ctx, domain.SalesBuyerKey{}, 50)
		checkAPI("NextBuyers", apiErr)
		for _, k := range next {
			checkAPI("RebuildBuyerSummaries", facts.RebuildBuyerSummaries(ctx, k.AccountID, []string{k.BuyerAccountID}))
		}
		checkAPI("DeleteBuyerSummariesRefreshedBefore", facts.DeleteBuyerSummariesRefreshedBefore(ctx, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)))
		completed := time.Now().UTC()
		checkAPI("SaveBuyerSummarySync", facts.SaveBuyerSummarySync(ctx, domain.SalesBuyerSummarySync{Cursor: &buyer, FactsSince: &completed, LastCompletedAt: &completed}))
		_, apiErr = facts.GetBuyerSummarySync(ctx)
		checkAPI("GetBuyerSummarySync", apiErr)
		_, apiErr = reports.BuyerSummariesReady(ctx)
		checkAPI("BuyerSummariesReady", apiErr)
		for _, p := range []domain.ListNewCustomersParams{
			{AccountID: account, StartsAt: start, EndsAt: end, Limit: 1},
			{AccountID: account, StartsAt: start, EndsAt: end, Limit: 5, CustomerGroupIDs: groups, SalesRepIDs: []string{"acus_none"}},
		} {
			page, apiErr := reports.GetNewCustomers(ctx, p)
			checkAPI("GetNewCustomers", apiErr)
			if page != nil && page.PageInfo.NextCursor != nil {
				p.Cursor = page.PageInfo.NextCursor
				back, apiErr := reports.GetNewCustomers(ctx, p)
				checkAPI("GetNewCustomers next", apiErr)
				if back != nil && back.PageInfo.PrevCursor != nil {
					p.Cursor = back.PageInfo.PrevCursor
					_, apiErr = reports.GetNewCustomers(ctx, p)
					checkAPI("GetNewCustomers prev", apiErr)
				}
			}
		}
	})
}
