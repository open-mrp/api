package repository

import (
	"context"
	"database/sql"
	"errors"
)

// salesSyncRow is one pass's row of sales_sync; each pass reads the cursor columns that fit its order.
type salesSyncRow struct {
	cursorCreatedAt      sql.NullTime
	cursorInvoiceID      sql.NullString
	cursorAccountID      sql.NullString
	cursorDay            sql.NullTime
	cursorBuyerAccountID sql.NullString
	factsSince           sql.NullTime
	passStartedAt        sql.NullTime
	lastCompletedAt      sql.NullTime
}

// legacySalesSync is where a pass kept its state before sales_sync, read once to seed its row there.
// TODO(sales-sync): drop with sales_fact_sync and sales_rollup_sync once every deployment has moved.
var legacySalesSync = map[string]string{
	salesFactSyncName:   `SELECT cursor_created_at, cursor_invoice_id, NULL, NULL, pass_started_at, last_completed_at FROM sales_fact_sync WHERE name = ?`,
	salesRollupSyncName: `SELECT NULL, NULL, cursor_account_id, cursor_day, pass_started_at, last_completed_at FROM sales_rollup_sync WHERE name = ?`,
}

// getSalesSync returns a pass's row; found is false before the pass has run. A pass with no row yet whose
// state is still in its old table is carried over first, so a deploy resumes each pass where it stood.
func (r *salesFactRepoImpl) getSalesSync(ctx context.Context, name string) (row salesSyncRow, found bool, err error) {
	err = r.queries.DB().QueryRowContext(ctx, `SELECT cursor_created_at, cursor_invoice_id, cursor_account_id, cursor_day, cursor_buyer_account_id,
    facts_since, pass_started_at, last_completed_at FROM sales_sync WHERE name = ?`, name).Scan(
		&row.cursorCreatedAt, &row.cursorInvoiceID, &row.cursorAccountID, &row.cursorDay, &row.cursorBuyerAccountID,
		&row.factsSince, &row.passStartedAt, &row.lastCompletedAt)
	if err == nil {
		return row, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return row, false, err
	}
	legacy, ok := legacySalesSync[name]
	if !ok {
		return salesSyncRow{}, false, nil
	}
	err = r.queries.DB().QueryRowContext(ctx, legacy, name).Scan(
		&row.cursorCreatedAt, &row.cursorInvoiceID, &row.cursorAccountID, &row.cursorDay, &row.passStartedAt, &row.lastCompletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return salesSyncRow{}, false, nil
	}
	if err != nil {
		return row, false, err
	}
	// INSERT IGNORE: if another pod carried it over first, its row stands.
	if _, err := r.queries.DB().ExecContext(ctx, `INSERT IGNORE INTO sales_sync (name, cursor_created_at, cursor_invoice_id, cursor_account_id, cursor_day,
    pass_started_at, last_completed_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, NOW(3))`,
		name, row.cursorCreatedAt, row.cursorInvoiceID, row.cursorAccountID, row.cursorDay, row.passStartedAt, row.lastCompletedAt); err != nil {
		return row, false, err
	}
	return row, true, nil
}

func (r *salesFactRepoImpl) saveSalesSync(ctx context.Context, name string, row salesSyncRow) error {
	_, err := r.queries.DB().ExecContext(ctx, `INSERT INTO sales_sync (name, cursor_created_at, cursor_invoice_id, cursor_account_id, cursor_day,
    cursor_buyer_account_id, facts_since, pass_started_at, last_completed_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3))
ON DUPLICATE KEY UPDATE cursor_created_at = VALUES(cursor_created_at), cursor_invoice_id = VALUES(cursor_invoice_id),
    cursor_account_id = VALUES(cursor_account_id), cursor_day = VALUES(cursor_day), cursor_buyer_account_id = VALUES(cursor_buyer_account_id),
    facts_since = VALUES(facts_since), pass_started_at = VALUES(pass_started_at), last_completed_at = VALUES(last_completed_at),
    updated_at = VALUES(updated_at)`,
		name, row.cursorCreatedAt, row.cursorInvoiceID, row.cursorAccountID, row.cursorDay,
		row.cursorBuyerAccountID, row.factsSince, row.passStartedAt, row.lastCompletedAt)
	return err
}
