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
		day, yes, no := "2000-01-01", true, false
		for name, params := range map[string]domain.ListSalesOrdersParams{
			"plain":    {AccountID: account, Limit: 50},
			"customer": {AccountID: account, Limit: 50, CustomerIDs: buyers},
			"status":   {AccountID: account, Limit: 50, StatusCodes: []string{"issued", "estimate"}},
			"buyer":    {AccountID: account, Limit: 50, BuyerAccountID: &buyers[0]},
			// The optimizer hints, semijoins, and index sets the page query is built with.
			"lines":   {AccountID: account, Limit: 5, ItemIDs: []string{"it_none"}, ProductLineIDs: productLines},
			"group":   {AccountID: account, Limit: 5, CustomerGroupIDs: groups, SalesRepIDs: []string{"acus_none"}},
			"ship-by": {AccountID: account, Limit: 5, ShipByAfter: &day, CustomerIDs: buyers, SalesRepIDs: []string{"acus_none"}},
			"dates":   {AccountID: account, Limit: 5, StartDate: &day, PastDue: &yes},
			"due":     {AccountID: account, Limit: 5, EndDate: &day, PastDue: &no},
		} {
			page, apiErr := repo.List(ctx, params)
			checkAPI("ListSalesOrders/"+name, apiErr)
			if page != nil && page.PageInfo.NextCursor != nil {
				params.Cursor = page.PageInfo.NextCursor
				next, apiErr := repo.List(ctx, params)
				checkAPI("ListSalesOrders next/"+name, apiErr)
				if next != nil && next.PageInfo.PrevCursor != nil {
					params.Cursor = next.PageInfo.PrevCursor
					_, apiErr = repo.List(ctx, params)
					checkAPI("ListSalesOrders prev/"+name, apiErr)
				}
			}
		}
		_, apiErr := repo.GetByIDs(ctx, account, &buyers[0], orderIDs)
		checkAPI("SalesOrder.GetByIDs", apiErr)
		_, apiErr = repo.GetLinesForOrders(ctx, orderIDs)
		checkAPI("SalesOrder.GetLinesForOrders", apiErr)
	})

	// --- catalog and customer lists: page chosen in a derived table, keys forced, filters resolved first ---
	t.Run("catalog and customer lists", func(t *testing.T) {
		catAccount := ids("SELECT i.account_id FROM item i JOIN product p ON p.item_id = i.id GROUP BY i.account_id ORDER BY COUNT(*) DESC LIMIT 1")[0]
		categories := ids("SELECT id FROM item_category WHERE account_id = ? LIMIT 2", catAccount)
		attributes := append(ids("SELECT id FROM attribute WHERE account_id = ? LIMIT 2", catAccount), "attr_none")
		suppliers := append(ids("SELECT supplier_account_id FROM supplier_material WHERE owner_account_id = ? LIMIT 1", catAccount), "ac_none")
		customers := append(ids("SELECT counterparty_account_id FROM account_relation WHERE owner_account_id = ? AND account_relation_role_code = 'customer' LIMIT 2", catAccount), "ac_none")
		search := "a"
		from, to := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), time.Now().UTC().Add(time.Hour)

		items := NewItemRepo(q)
		// One per index hint and filter shape: the created, type, and category keys, the primary key for
		// a filter on another table, a search's tier ordering, and the subassembly probe.
		for name, p := range map[string]domain.ListItemsParams{
			"plain":       {AccountID: catAccount, Limit: 5},
			"type":        {AccountID: catAccount, Limit: 5, Types: []string{"part", "product"}},
			"category":    {AccountID: catAccount, Limit: 5, Types: []string{"product"}, CategoryIDs: append(categories, "itcg_none")},
			"other":       {AccountID: catAccount, Limit: 5, AttributeIDs: attributes, SupplierID: &suppliers[0], ProductLineIDs: productLines},
			"customer":    {AccountID: catAccount, Limit: 5, CustomerIDs: customers, ProductLineIDs: productLines},
			"search":      {AccountID: catAccount, Limit: 5, Query: &search, StartDate: &from, EndDate: &to},
			"exact":       {AccountID: catAccount, Limit: 5, Query: &search, IsExactMatch: true},
			"subassembly": {AccountID: catAccount, Limit: 5, OnlyInitialSubassemblies: true},
		} {
			page, apiErr := items.List(ctx, p)
			checkAPI("ListItems/"+name, apiErr)
			if page != nil && page.PageInfo.NextCursor != nil {
				p.Cursor = page.PageInfo.NextCursor
				next, apiErr := items.List(ctx, p)
				checkAPI("ListItems/"+name+" next", apiErr)
				if next != nil && next.PageInfo.PrevCursor != nil {
					p.Cursor = next.PageInfo.PrevCursor
					_, apiErr = items.List(ctx, p)
					checkAPI("ListItems/"+name+" prev", apiErr)
				}
			}
		}

		products := NewProductRepo(q)
		yes := true
		for name, p := range map[string]domain.ListProductsFullParams{
			"plain":    {AccountID: catAccount, Limit: 5},
			"category": {AccountID: catAccount, Limit: 5, CategoryIDs: categories, IsPortalReady: &yes},
			"lines":    {AccountID: catAccount, Limit: 5, ProductLineIDs: productLines, CustomerIDs: customers, AttributeIDs: attributes},
			"search":   {AccountID: catAccount, Limit: 5, Query: &search, StartDate: &from, EndDate: &to},
		} {
			page, apiErr := products.List(ctx, p)
			checkAPI("ListProductsFull/"+name, apiErr)
			if page != nil && page.PageInfo.NextCursor != nil {
				p.Cursor = page.PageInfo.NextCursor
				next, apiErr := products.List(ctx, p)
				checkAPI("ListProductsFull/"+name+" next", apiErr)
				if next != nil && next.PageInfo.PrevCursor != nil {
					p.Cursor = next.PageInfo.PrevCursor
					_, apiErr = products.List(ctx, p)
					checkAPI("ListProductsFull/"+name+" prev", apiErr)
				}
			}
		}
		_, apiErr := products.SearchBySKU(ctx, catAccount, "%a%")
		checkAPI("SearchProductsBySKU", apiErr)
		_, apiErr = products.ListByAccount(ctx, catAccount)
		checkAPI("ListProductsByAccount", apiErr)

		catalog := NewCatalogRepo(q)
		_, apiErr = catalog.ListProductLines(ctx, catAccount)
		checkAPI("ListCatalogProductLines", apiErr)
		_, apiErr = catalog.ListProductLinesForCustomer(ctx, catAccount, customers[0])
		checkAPI("ListCatalogProductLinesForCustomer", apiErr)
		for _, line := range productLines {
			_, apiErr = catalog.ListProducts(ctx, catAccount, line)
			checkAPI("ListCatalogProducts", apiErr)
		}

		customerRepo := NewCustomerRepo(q)
		state := "NC"
		for name, p := range map[string]domain.ListCustomersParams{
			"plain":   {AccountID: catAccount, Limit: 5},
			"filters": {AccountID: catAccount, Limit: 5, Query: &search, CarrierIDs: carriers, PaymentTermIDs: []string{"pt_none"}},
			"pricing": {AccountID: catAccount, Limit: 5, PricingGroupIDs: groups, State: &state},
		} {
			page, apiErr := customerRepo.List(ctx, p)
			checkAPI("ListCustomers/"+name, apiErr)
			if page != nil && page.PageInfo.NextCursor != nil {
				p.Cursor = page.PageInfo.NextCursor
				next, apiErr := customerRepo.List(ctx, p)
				checkAPI("ListCustomers/"+name+" next", apiErr)
				if next != nil && next.PageInfo.PrevCursor != nil {
					p.Cursor = next.PageInfo.PrevCursor
					_, apiErr = customerRepo.List(ctx, p)
					checkAPI("ListCustomers/"+name+" prev", apiErr)
				}
			}
		}

		addressAccount := ids("SELECT account_id FROM account_address GROUP BY account_id ORDER BY COUNT(*) DESC LIMIT 1")[0]
		dropShip := false
		page, apiErr := NewAddressRepo(q).List(ctx, domain.ListAddressesParams{AccountID: addressAccount, Limit: 2, Query: &search, DropShip: &dropShip})
		checkAPI("ListAddresses", apiErr)
		if page != nil && page.PageInfo.NextCursor != nil {
			_, apiErr = NewAddressRepo(q).List(ctx, domain.ListAddressesParams{AccountID: addressAccount, Limit: 2, Cursor: page.PageInfo.NextCursor})
			checkAPI("ListAddresses next", apiErr)
		}
	})

	t.Run("invoice and receivable lists", func(t *testing.T) {
		invoices := NewInvoiceRepo(q)
		search, paid, unpaid, overpaid := "1", "paid", "unpaid", "overpaid"
		from, to := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), time.Now().UTC().Add(time.Hour)
		for name, params := range map[string]domain.ListInvoicesParams{
			"plain":    {AccountID: account, Limit: 1},
			"paid":     {AccountID: account, Limit: 5, Status: &paid, StartDate: &from, EndDate: &to},
			"unpaid":   {AccountID: account, Limit: 5, Status: &unpaid, Query: &search},
			"overpaid": {AccountID: account, Limit: 5, Status: &overpaid},
			"orders": {AccountID: account, Limit: 5, CustomerIDs: buyers, CustomerGroupIDs: groups, SalesRepIDs: []string{"acus_none"},
				ItemIDs: []string{"it_none"}, ProductLineIDs: productLines},
			"customer": {AccountID: account, Limit: 5, CustomerIDs: buyers},
		} {
			page, apiErr := invoices.List(ctx, params)
			checkAPI("ListInvoices/"+name, apiErr)
			if page != nil && page.PageInfo.NextCursor != nil {
				params.Cursor = page.PageInfo.NextCursor
				next, apiErr := invoices.List(ctx, params)
				checkAPI("ListInvoices next/"+name, apiErr)
				if next != nil && next.PageInfo.PrevCursor != nil {
					params.Cursor = next.PageInfo.PrevCursor
					_, apiErr = invoices.List(ctx, params)
					checkAPI("ListInvoices prev/"+name, apiErr)
				}
			}
		}
		page, apiErr := invoices.ListByCustomer(ctx, domain.ListCustomerInvoicesParams{AccountID: account, CustomerAccountID: buyers[0], Limit: 1})
		checkAPI("ListCustomerInvoices", apiErr)
		if page != nil && page.PageInfo.NextCursor != nil {
			_, apiErr = invoices.ListByCustomer(ctx, domain.ListCustomerInvoicesParams{AccountID: account, CustomerAccountID: buyers[0], Limit: 1, Query: &search, Cursor: page.PageInfo.NextCursor})
			checkAPI("ListCustomerInvoices next", apiErr)
		}

		receivables := NewReceivableRepo(q)
		for _, cutoff := range []*time.Time{nil, &to} {
			_, apiErr = receivables.List(ctx, domain.ListReceivablesParams{AccountID: account, CutoffDate: cutoff, Query: &search, Limit: 5})
			checkAPI("ListReceivables", apiErr)
			_, apiErr = receivables.ListByCustomer(ctx, domain.ListReceivablesByCustomerParams{AccountID: account, CustomerAccountID: buyers[0], CutoffDate: cutoff, Limit: 5})
			checkAPI("ListReceivablesByCustomer", apiErr)
		}
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
		// The shapes plan through vtgate whether or not the seed holds deliveries.
		dlvAccount := ids("SELECT account_id FROM delivery LIMIT 1")
		if len(dlvAccount) == 0 {
			dlvAccount = []string{account}
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

	// Delivery performance forces the order's ship-by keys and takes customers resolved to buyers.
	t.Run("delivery performance", func(t *testing.T) {
		schedule := NewProductionScheduleInputRepo(q)
		from, to := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), time.Now().UTC().AddDate(1, 0, 0)
		for _, f := range []domain.DeliveryFilters{
			{},
			{CustomerIDs: buyers, CustomerGroupIDs: groups, ProductLineIDs: productLines, SalesRepIDs: []string{"acus_none"}},
			{CustomerIDs: buyers},
		} {
			_, apiErr := schedule.ListDeliveryOutcomes(ctx, account, from, to, f)
			checkAPI("ListDeliveryOutcomes", apiErr)
			_, apiErr = schedule.CountUncommittedOrders(ctx, account, from, to, f)
			checkAPI("CountUncommittedOrders", apiErr)
		}
		// The line-level sales and open-order analytics take the same resolved buyers; sales forces the invoice key.
		analytics := NewAnalyticsRepo(q)
		// OEE's downtime overlap forces both one-sided downtime keys.
		_, apiErr := analytics.GetOeeDowntimeIntervals(ctx, domain.GetOeeWindowParams{AccountID: account, StartDate: from, EndDate: to})
		checkAPI("GetOeeDowntimeIntervals", apiErr)
		for _, p := range []domain.AnalyzeSalesParams{
			{AccountID: account, StartDate: from, EndDate: to},
			{AccountID: account, StartDate: from, EndDate: to, CustomerIDs: buyers, CustomerGroupIDs: groups, ProductLineIDs: productLines, SalesRepIDs: []string{"acus_none"}},
		} {
			_, apiErr := analytics.GetSalesEntries(ctx, p)
			checkAPI("GetSalesEntries", apiErr)
			_, apiErr = analytics.GetOrderEntries(ctx, domain.AnalyzeOrdersParams{AccountID: account, CustomerIDs: p.CustomerIDs, CustomerGroupIDs: p.CustomerGroupIDs,
				ProductLineIDs: p.ProductLineIDs, SalesRepIDs: p.SalesRepIDs})
			checkAPI("GetOrderEntries", apiErr)
		}
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
		// A filtered page walks one filter's key per value (UNION of ordered LIMITs), after counting each
		// filter's lines to pick which; both directions seek on (invoiced_at, invoice_id).
		byCustomer := unfiltered
		byCustomer.CustomerIDs, byCustomer.ProductLineIDs = buyers, productLines
		custPage, apiErr := reports.GetInvoicePage(ctx, domain.AnalyzeSalesInvoicesParams{SalesReportFilter: byCustomer, Limit: 1})
		checkAPI("GetInvoicePage by customer", apiErr)
		if custPage != nil && custPage.PageInfo.NextCursor != nil {
			next, apiErr := reports.GetInvoicePage(ctx, domain.AnalyzeSalesInvoicesParams{SalesReportFilter: byCustomer, Limit: 1, Cursor: custPage.PageInfo.NextCursor})
			checkAPI("GetInvoicePage by customer next", apiErr)
			if next != nil && next.PageInfo.PrevCursor != nil {
				_, apiErr = reports.GetInvoicePage(ctx, domain.AnalyzeSalesInvoicesParams{SalesReportFilter: byCustomer, Limit: 1, Cursor: next.PageInfo.PrevCursor})
				checkAPI("GetInvoicePage by customer prev", apiErr)
			}
		}
		linePage, apiErr := reports.GetLinePage(ctx, domain.ListSalesLinesParams{SalesReportFilter: unfiltered, HasWindow: true, Limit: 2})
		checkAPI("GetLinePage", apiErr)
		if linePage != nil && linePage.PageInfo.NextCursor != nil {
			_, apiErr = reports.GetLinePage(ctx, domain.ListSalesLinesParams{SalesReportFilter: unfiltered, Limit: 2, Cursor: linePage.PageInfo.NextCursor})
			checkAPI("GetLinePage next", apiErr)
		}
		_, apiErr = reports.GetLinePage(ctx, domain.ListSalesLinesParams{SalesReportFilter: filter, Limit: 5})
		checkAPI("GetLinePage filtered", apiErr)
		// One per page shape: buyer by buyer, and the in-order key set.
		for name, f := range map[string]domain.SalesReportFilter{
			"buyers": {AccountID: account, CustomerIDs: buyers},
			"lines":  {AccountID: account, ProductLineIDs: productLines, SalesRepIDs: []string{"acus_none"}, ItemIDs: []string{"it_none"}},
		} {
			linePage, apiErr := reports.GetLinePage(ctx, domain.ListSalesLinesParams{SalesReportFilter: f, Limit: 1})
			checkAPI("GetLinePage "+name, apiErr)
			if linePage != nil && linePage.PageInfo.NextCursor != nil {
				_, apiErr = reports.GetLinePage(ctx, domain.ListSalesLinesParams{SalesReportFilter: f, Limit: 1, Cursor: linePage.PageInfo.NextCursor})
				checkAPI("GetLinePage next "+name, apiErr)
			}
		}

		// The rollups: a full sweep's statements, then every report shape that reads them.
		cursor := domain.SalesRollupDay{Day: salesFactSweepFloor}
		for {
			next, apiErr := facts.NextRollupDay(ctx, cursor)
			checkAPI("NextRollupDay", apiErr)
			if next == nil || apiErr != nil {
				break
			}
			checkAPI("RebuildRollupDay", facts.RebuildRollupDay(ctx, *next))
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
		// A customer filter: the customer breakdown forces the rollup's group key, the rest the lines' buyer key.
		customers := ragged
		customers.CustomerIDs = buyers
		for _, f := range []domain.SalesReportFilter{ragged, oneLine, twoLines, customers} {
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
		if len(next) > 0 {
			_, apiErr = facts.ListBuyerSummaryRefreshes(ctx, domain.SalesBuyerKey{}, &next[len(next)-1])
			checkAPI("ListBuyerSummaryRefreshes", apiErr)
			_, apiErr = facts.LatestBuyerFactRefreshes(ctx, next[0].AccountID, []string{next[0].BuyerAccountID})
			checkAPI("LatestBuyerFactRefreshes", apiErr)
		}
		checkAPI("DeleteBuyerSummaries", facts.DeleteBuyerSummaries(ctx, []domain.SalesBuyerKey{{AccountID: "ac_none", BuyerAccountID: "ac_none"}}))
		_, apiErr = facts.CountInvoiceLines(ctx, []string{"iv_none"})
		checkAPI("CountInvoiceLines", apiErr)
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

	// --- inventory, production and log lists: SQL built in Go (per-value UNION ALL arms, a page chosen
	// in a derived table and joined after, runs resolved from the batch side) and index hints ---
	t.Run("inventory, production and log lists", func(t *testing.T) {
		first := func(query string, args ...any) string {
			if v := ids(query, args...); len(v) > 0 {
				return v[0]
			}
			return "none"
		}
		from, to := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), time.Now().UTC().Add(time.Hour)
		search := "a"

		iclAccount := first("SELECT account_id FROM inventory_change_log LIMIT 1")
		item := first("SELECT item_id FROM inventory_change_log WHERE account_id = ? LIMIT 1", iclAccount)
		user := first("SELECT responsible_user_id FROM inventory_change_log WHERE account_id = ? AND responsible_user_id IS NOT NULL LIMIT 1", iclAccount)
		logs := NewInventoryChangeLogRepo(q)
		// One per page shape: the unfiltered key, a filter's key, arms per value, a long list's IN.
		for _, p := range []domain.ListInventoryChangeLogsParams{
			{AccountID: iclAccount, Limit: 5, StartDate: &from},
			{AccountID: iclAccount, Limit: 5, ItemIDs: []string{item}, ActionTypeCodes: []string{"scan", "user_correction"}},
			{AccountID: iclAccount, Limit: 5, StartDate: &from, EndDate: &to, ActionTypeCodes: []string{"scan", "user_correction", "system_action"}},
			{AccountID: iclAccount, Limit: 5, ItemIDs: []string{item, "it_none"}, ChangedByUserIDs: []string{user}},
			{AccountID: iclAccount, Limit: 5, StartDate: &from, Query: &search},
		} {
			page, apiErr := logs.List(ctx, p)
			checkAPI("inventory change logs List", apiErr)
			if page != nil && page.PageInfo.NextCursor != nil {
				p.Cursor = page.PageInfo.NextCursor
				next, apiErr := logs.List(ctx, p)
				checkAPI("inventory change logs List next", apiErr)
				if next != nil && next.PageInfo.PrevCursor != nil {
					p.Cursor = next.PageInfo.PrevCursor
					_, apiErr = logs.List(ctx, p)
					checkAPI("inventory change logs List prev", apiErr)
				}
			}
		}

		batchAccount := first("SELECT account_id FROM batch LIMIT 1")
		station := first("SELECT scanning_station_id FROM batch WHERE account_id = ? AND scanning_station_id IS NOT NULL LIMIT 1", batchAccount)
		batches := NewBatchRepo(q)
		for _, p := range []domain.ListBatchesByScanningStationParams{
			{AccountID: batchAccount, ScanningStationID: station, Limit: 5},
			{AccountID: batchAccount, ScanningStationID: station, Limit: 5, Query: &search},
		} {
			page, apiErr := batches.FindByScanningStation(ctx, p)
			checkAPI("batches FindByScanningStation", apiErr)
			if page != nil && page.PageInfo.NextCursor != nil {
				p.Cursor = page.PageInfo.NextCursor
				next, apiErr := batches.FindByScanningStation(ctx, p)
				checkAPI("batches FindByScanningStation next", apiErr)
				if next != nil && next.PageInfo.PrevCursor != nil {
					p.Cursor = next.PageInfo.PrevCursor
					_, apiErr = batches.FindByScanningStation(ctx, p)
					checkAPI("batches FindByScanningStation prev", apiErr)
				}
			}
		}
		_, apiErr := NewScanningStationRepo(q).GetByIDs(ctx, batchAccount, []string{station, "sst_none"})
		checkAPI("scanning stations GetByIDs", apiErr)

		runAccount := first("SELECT account_id FROM production_run LIMIT 1")
		run := first("SELECT id FROM production_run WHERE account_id = ? LIMIT 1", runAccount)
		machine := first("SELECT bm.B FROM _batches_machines bm JOIN batch b ON b.id = bm.A WHERE b.account_id = ? LIMIT 1", runAccount)
		runItem := first("SELECT item_id FROM batch WHERE account_id = ? AND production_run_id IS NOT NULL LIMIT 1", runAccount)
		open, day := "open", from.Format("2006-01-02")
		runs := NewProductionRunRepo(q)
		for _, p := range []domain.ListProductionRunsParams{
			{AccountID: runAccount, Limit: 5},
			{AccountID: runAccount, Limit: 5, Status: &open, ItemIDs: []string{runItem}, MachineIDs: []string{machine}, Query: &search, StartDate: &day},
			{AccountID: runAccount, Limit: 5, Query: &search},
		} {
			page, apiErr := runs.List(ctx, p)
			checkAPI("production runs List", apiErr)
			if page != nil && page.PageInfo.NextCursor != nil {
				p.Cursor = page.PageInfo.NextCursor
				next, apiErr := runs.List(ctx, p)
				checkAPI("production runs List next", apiErr)
				if next != nil && next.PageInfo.PrevCursor != nil {
					p.Cursor = next.PageInfo.PrevCursor
					_, apiErr = runs.List(ctx, p)
					checkAPI("production runs List prev", apiErr)
				}
			}
		}
		_, apiErr = runs.ListBatchesByRun(ctx, domain.ListBatchesByProductionRunParams{AccountID: runAccount, ProductionRunID: run, Limit: 5, SearchQuery: &search})
		checkAPI("production runs ListBatchesByRun", apiErr)

		dtAccount := first("SELECT account_id FROM machine_downtime_event LIMIT 1")
		dtMachine := first("SELECT machine_id FROM machine_downtime_event WHERE account_id = ? LIMIT 1", dtAccount)
		downtime := NewMachineDowntimeRepo(q)
		for _, p := range []domain.ListMachineDowntimeEventsParams{
			{AccountID: dtAccount, Limit: 5},
			{AccountID: dtAccount, Limit: 5, MachineIDs: []string{dtMachine, "mch_none"}, OpenOnly: true, Query: &search, StartDate: &from, EndDate: &to},
			{AccountID: dtAccount, Limit: 5, ReasonCodes: []string{"breakdown", "changeover"}, DepartmentIDs: []string{"dept_none"}},
		} {
			page, apiErr := downtime.List(ctx, p)
			checkAPI("downtime List", apiErr)
			if page != nil && page.PageInfo.NextCursor != nil {
				p.Cursor = page.PageInfo.NextCursor
				next, apiErr := downtime.List(ctx, p)
				checkAPI("downtime List next", apiErr)
				if next != nil && next.PageInfo.PrevCursor != nil {
					p.Cursor = next.PageInfo.PrevCursor
					_, apiErr = downtime.List(ctx, p)
					checkAPI("downtime List prev", apiErr)
				}
			}
		}

		emailAccount := first("SELECT account_id FROM email_log LIMIT 1")
		_, apiErr = NewEmailLogRepo(q).List(ctx, domain.ListEmailLogsParams{AccountID: emailAccount, Limit: 5, Query: &search})
		checkAPI("email logs List", apiErr)
	})
}
