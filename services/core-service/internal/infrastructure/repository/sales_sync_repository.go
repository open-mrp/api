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

// getSalesSync returns a pass's row; found is false before the pass has run.
func (r *salesFactRepoImpl) getSalesSync(ctx context.Context, name string) (row salesSyncRow, found bool, err error) {
	err = r.queries.DB().QueryRowContext(ctx, `SELECT cursor_created_at, cursor_invoice_id, cursor_account_id, cursor_day, cursor_buyer_account_id,
    facts_since, pass_started_at, last_completed_at FROM sales_sync WHERE name = ?`, name).Scan(
		&row.cursorCreatedAt, &row.cursorInvoiceID, &row.cursorAccountID, &row.cursorDay, &row.cursorBuyerAccountID,
		&row.factsSince, &row.passStartedAt, &row.lastCompletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return salesSyncRow{}, false, nil
	}
	return row, err == nil, err
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
