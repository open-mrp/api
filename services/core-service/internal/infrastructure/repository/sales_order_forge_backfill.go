package repository

import (
	"context"
	gosql "database/sql"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/apikit/db/backfill"
)

// Backfill names for the sales order copies forge.1 reads (00117).
const (
	SalesOrderBillToBackfillName = "sales_order_bill_to_snapshot"
	SalesOrderShipToBackfillName = "sales_order_ship_to_snapshot"
	DocumentPartyBackfillName    = "sales_order_document_party"
)

// DocumentPartyBackfill gives each order whose owner kept instructions (the legacy note) or metadata on the shared
// row a party row for that owner holding them. Party IDs derive from the order and the owner, and existing rows are
// left alone, so a batch can rerun safely and never overwrites what a write has stored since.
type DocumentPartyBackfill struct {
	read, write sqlc.DBTX
}

// NewDocumentPartyBackfill reads from read (a replica) and writes to write (the primary).
func NewDocumentPartyBackfill(read, write *sqlc.Queries) *DocumentPartyBackfill {
	return &DocumentPartyBackfill{read: read.DB(), write: write.DB()}
}

// Batch is a backfill.Batch over sales_order ids.
func (b *DocumentPartyBackfill) Batch(ctx context.Context, m *backfill.Meter, cursor string, limit int) (string, int, bool, error) {
	var ids []string
	if err := m.Time(func() (err error) {
		ids, err = selectStrings(ctx, b.read, "SELECT id FROM sales_order WHERE id > ? ORDER BY id LIMIT ?", cursor, limit)
		return err
	}); err != nil {
		return cursor, 0, false, err
	}
	if len(ids) == 0 {
		return cursor, 0, true, nil
	}

	type party struct {
		orderID, accountID string
		instructions       gosql.NullString
		metadata           gosql.NullString
	}
	var parties []party
	if err := m.Time(func() error {
		rows, err := b.read.QueryContext(ctx, "SELECT id, owner_account_id, note, CAST(metadata AS CHAR) FROM sales_order WHERE id IN ("+
			placeholders(len(ids))+") AND (note IS NOT NULL OR (metadata IS NOT NULL AND JSON_LENGTH(metadata) > 0))", stringsToAny(ids)...)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var p party
			if err := rows.Scan(&p.orderID, &p.accountID, &p.instructions, &p.metadata); err != nil {
				return err
			}
			parties = append(parties, p)
		}
		return rows.Err()
	}); err != nil {
		return cursor, 0, false, err
	}

	if len(parties) > 0 {
		values := make([]string, len(parties))
		args := make([]any, 0, len(parties)*5)
		for i, p := range parties {
			values[i] = "(?, ?, ?, ?, ?, NOW(3), NOW(3))"
			args = append(args, documentPartyID(p.orderID, p.accountID), p.orderID, p.accountID, p.instructions, p.metadata)
		}
		query := "INSERT IGNORE INTO document_party (id, document_id, account_id, instructions, metadata, created_at, updated_at) VALUES " +
			strings.Join(values, ", ")
		if err := m.Time(func() error {
			_, err := b.write.ExecContext(ctx, query, args...)
			return err
		}); err != nil {
			return cursor, 0, false, err
		}
	}
	return ids[len(ids)-1], len(ids), len(ids) < limit, nil
}
