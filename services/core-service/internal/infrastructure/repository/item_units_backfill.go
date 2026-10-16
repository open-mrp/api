package repository

import (
	"context"
	"database/sql"

	"github.com/open-mrp/apikit/db/backfill"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// ItemUnitsBackfillName is the item-units backfill's row in backfill_progress.
const ItemUnitsBackfillName = "item_order_units"

// ItemUnitsBackfill stores the units of items written before items stored their own (00099), walking items by id. It
// pages from read (a replica) and derives and writes on write (the primary), one item at a time, and only onto an
// item that still has none stored, so an item a writer stored since keeps the writer's units and a re-run batch
// changes nothing.
type ItemUnitsBackfill struct {
	read  *sqlc.Queries
	write *sql.DB
}

// NewItemUnitsBackfill builds the backfill over a replica and the primary.
func NewItemUnitsBackfill(read *sqlc.Queries, write *sql.DB) *ItemUnitsBackfill {
	return &ItemUnitsBackfill{read: read, write: write}
}

// Batch is a backfill.Batch over item ids.
func (b *ItemUnitsBackfill) Batch(ctx context.Context, m *backfill.Meter, cursor string, limit int) (string, int, bool, error) {
	var page []sqlc.ListUnstoredItemIDsAfterRow
	if err := m.Time(func() (err error) {
		page, err = b.read.ListUnstoredItemIDsAfter(ctx, sqlc.ListUnstoredItemIDsAfterParams{Cursor: cursor, Limit: int32(limit)}) //nolint:gosec // the runner caps the batch size
		return err
	}); err != nil {
		return cursor, 0, false, err
	}
	if len(page) == 0 {
		return cursor, 0, true, nil
	}
	for _, row := range page {
		if !row.Unstored {
			continue
		}
		if err := m.Time(func() error { return b.store(ctx, row.ID) }); err != nil {
			return cursor, 0, false, err
		}
	}
	return page[len(page)-1].ID, len(page), len(page) < limit, nil
}

// store derives and writes one item's units in a short transaction of its own.
func (b *ItemUnitsBackfill) store(ctx context.Context, itemID string) error {
	tx, err := b.write.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	repo := &itemUnitsRepo{queries: sqlc.New(tx)}
	if apiErr := repo.Materialize(ctx, []string{itemID}); apiErr != nil {
		return apiErr
	}
	return tx.Commit()
}
