//go:build integration

package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// These run against the dev database (see testDB) and clean up after themselves.

func TestRollupDirtyMarksClearOnlyWhenUnchanged(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t)
	repo := NewSalesFactRepo(sqlc.New(pool))
	day := domain.SalesRollupDay{AccountID: "ac_marks_test", Day: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)}
	t.Cleanup(func() { _, _ = pool.Exec(`DELETE FROM sales_fact_dirty WHERE account_id = 'ac_marks_test'`) })

	require.Nil(t, repo.MarkRollupDays(ctx, []domain.SalesRollupDay{day, day}))
	first := findRollupMark(t, ctx, repo, day)
	require.NotNil(t, first, "a marked day is listed")

	// A re-mark while the rebuild ran must survive clearing the mark that was read before it.
	time.Sleep(5 * time.Millisecond)
	require.Nil(t, repo.MarkRollupDays(ctx, []domain.SalesRollupDay{day}))
	require.Nil(t, repo.ClearRollupDirty(ctx, *first))
	second := findRollupMark(t, ctx, repo, day)
	require.NotNil(t, second, "the re-marked day is still listed")
	require.True(t, second.MarkedAt.After(first.MarkedAt))

	require.Nil(t, repo.ClearRollupDirty(ctx, *second))
	require.Nil(t, findRollupMark(t, ctx, repo, day))
}

func findRollupMark(t *testing.T, ctx context.Context, repo domain.SalesFactRepo, day domain.SalesRollupDay) *domain.SalesRollupDirtyMark {
	t.Helper()
	marks, apiErr := repo.ListRollupDirty(ctx, 10_000)
	require.Nil(t, apiErr)
	for _, m := range marks {
		if m.Day == day {
			return &m
		}
	}
	return nil
}

func TestMarkInvoicesDirtyWritesOneInvoiceMarkEach(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t)
	repo := NewSalesFactRepo(sqlc.New(pool))
	t.Cleanup(func() { _, _ = pool.Exec(`DELETE FROM sales_fact_dirty WHERE account_id = 'ac_marks_test'`) })

	ids := make([]string, 1203) // spans the 500-row insert batches
	for i := range ids {
		ids[i] = "iv_marks_" + time.Duration(i).String()
	}
	require.Nil(t, repo.MarkInvoicesDirty(ctx, "ac_marks_test", ids))
	require.Nil(t, repo.MarkInvoicesDirty(ctx, "ac_marks_test", ids[:10]), "re-marking is idempotent")

	var n int
	require.NoError(t, pool.QueryRow(`SELECT COUNT(*) FROM sales_fact_dirty WHERE account_id = 'ac_marks_test' AND scope_type = 'invoice'`).Scan(&n))
	require.Equal(t, len(ids), n)
}

// Every scope resolves the invoices whose facts it feeds, using a real invoice line from the dev data.
func TestResolveInvoiceIDsForEveryScope(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t)
	repo := NewSalesFactRepo(sqlc.New(pool))

	var invoiceID, quantityID, priceID, costID, itemID, buyerID, accountID string
	err := pool.QueryRow(`SELECT il.invoice_id, il.quantity_id, sol.unit_price_id, COALESCE(sol.unit_cost_id, ''), p.item_id, f.buyer_account_id, f.account_id
FROM sales_line_fact f
JOIN invoice_line il ON il.id = f.invoice_line_id
JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
JOIN product p ON p.id = sol.product_id
WHERE sol.unit_price_id IS NOT NULL
LIMIT 1`).Scan(&invoiceID, &quantityID, &priceID, &costID, &itemID, &buyerID, &accountID)
	if errors.Is(err, sql.ErrNoRows) {
		t.Skip("the dev database holds no sales facts")
	}
	require.NoError(t, err)

	cases := map[domain.SalesFactScope][]string{
		domain.SalesFactScopeQuantity: {quantityID},
		domain.SalesFactScopeRate:     {priceID},
		domain.SalesFactScopeItem:     {itemID},
		domain.SalesFactScopeBuyer:    {buyerID},
	}
	if costID != "" {
		cases[domain.SalesFactScopeRate] = append(cases[domain.SalesFactScopeRate], costID)
	}
	for scope, ids := range cases {
		got, apiErr := repo.ResolveInvoiceIDs(ctx, accountID, scope, ids)
		require.Nil(t, apiErr, "%s", scope)
		require.Contains(t, got, invoiceID, "%s scope resolves the invoice", scope)
	}

	// A buyer is resolved within the account only.
	got, apiErr := repo.ResolveInvoiceIDs(ctx, "ac_someone_else", domain.SalesFactScopeBuyer, []string{buyerID})
	require.Nil(t, apiErr)
	require.Empty(t, got)
}

func TestRestartReconcileRewindsACompletedPass(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t)
	repo := NewSalesFactRepo(sqlc.New(pool))

	before, apiErr := repo.GetSync(ctx)
	require.Nil(t, apiErr)
	if before.LastCompletedAt == nil {
		t.Skip("the dev database has not finished its sales fact backfill")
	}
	t.Cleanup(func() { _ = repo.SaveSync(ctx, *before) })

	require.Nil(t, repo.RestartReconcile(ctx))
	after, apiErr := repo.GetSync(ctx)
	require.Nil(t, apiErr)
	require.NotNil(t, after.Cursor, "a pass is in progress")
	require.True(t, after.Cursor.CreatedAt.Before(time.Date(1971, 1, 1, 0, 0, 0, 0, time.UTC)), "from the oldest invoice")
	require.NotNil(t, after.LastCompletedAt, "reports stay ready while the pass reruns")
}

func TestLockPaymentFlagRowsInsideATransaction(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t)

	var accountID, transactionID, invoiceID string
	err := pool.QueryRow(`SELECT t.account_id, t.id, ta.invoice_id FROM transaction_allocation ta JOIN transaction t ON t.id = ta.transaction_id LIMIT 1`).
		Scan(&accountID, &transactionID, &invoiceID)
	if errors.Is(err, sql.ErrNoRows) {
		t.Skip("the dev database holds no allocations")
	}
	require.NoError(t, err)

	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	repo := NewSettlementRepo(sqlc.New(pool).WithTx(tx))
	require.Nil(t, repo.LockPaymentFlagRows(ctx, accountID, []string{transactionID}, []string{invoiceID}))

	// Another transaction cannot take the same invoice row until this one ends.
	other, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = other.Rollback() }()
	_, err = other.ExecContext(ctx, `SET SESSION innodb_lock_wait_timeout = 1`)
	require.NoError(t, err)
	_, err = other.ExecContext(ctx, `SELECT id FROM invoice WHERE id = ? FOR UPDATE`, invoiceID)
	require.Error(t, err, "the row is locked")
}
