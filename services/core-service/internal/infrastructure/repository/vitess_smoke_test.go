//go:build vitess_smoke

package repository

import (
	"context"
	gosql "database/sql"
	"os"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
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
	pool, err := db.NewDbPool(&db.Config{DBURI: dsn, WarmConnections: -1})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	ctx := context.Background()
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

	// --- batched reads (sqlc) ---
	t.Run("sqlc batch reads", func(t *testing.T) {
		_, err := q.GetPicksByIDs(ctx, sqlc.GetPicksByIDsParams{PickIds: pickIDs, AccountID: account})
		check("GetPicksByIDs", err)
		_, err = q.GetSalesOrderLinesForOrders(ctx, orderIDs)
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
		prefix, phrase := "PI", "ICK-00"
		start, end := "2000-01-01", "2100-01-01"
		cases := map[string]domain.ListPicksParams{
			"default ship-by":       {AccountID: account, Limit: 50},
			"created":               {AccountID: account, Limit: 50, Sort: constants.PickSortCreatedAt},
			"open ship-by":          {AccountID: account, Limit: 50, Status: &open},
			"closed created":        {AccountID: account, Limit: 50, Status: &closed, Sort: constants.PickSortCreatedAt},
			"customer":              {AccountID: account, Limit: 50, CustomerIDs: buyers},
			"customer open created": {AccountID: account, Limit: 50, CustomerIDs: buyers[:1], Status: &open, Sort: constants.PickSortCreatedAt},
			"customer group":        {AccountID: account, Limit: 50, CustomerGroupIDs: groups},
			"product line":          {AccountID: account, Limit: 50, ProductLineIDs: productLines},
			"date window":           {AccountID: account, Limit: 50, StartDate: &start, EndDate: &end},
			"number prefix":         {AccountID: account, Limit: 50, Query: &prefix},
			"phrase":                {AccountID: account, Limit: 50, Query: &phrase},
			"phrase + filters":      {AccountID: account, Limit: 50, Query: &phrase, Status: &closed, CustomerIDs: buyers, CustomerGroupIDs: groups, ProductLineIDs: productLines},
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
}
