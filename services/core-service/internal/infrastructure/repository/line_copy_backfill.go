package repository

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db/backfill"
)

// LineCopyBackfill fills the copies of order lines written before they existed, with their invoice and
// pick lines, walking order lines by id. It reads from read (a replica) and writes to write (the
// primary). A write only fills a line that still has no copy, so a line a writer copied since the
// replica read keeps the writer's newer values, and a re-run batch changes nothing.
type LineCopyBackfill struct {
	read, write sqlc.DBTX
}

func NewLineCopyBackfill(read, write *sqlc.Queries) *LineCopyBackfill {
	return &LineCopyBackfill{read: read.DB(), write: write.DB()}
}

// Batch is a backfill.Batch over sales_order_line ids.
func (b *LineCopyBackfill) Batch(ctx context.Context, m *backfill.Meter, cursor string, limit int) (string, int, bool, error) {
	var ids []string
	if err := m.Time(func() (err error) {
		ids, err = selectStrings(ctx, b.read, "SELECT id FROM sales_order_line WHERE id > ? ORDER BY id LIMIT ?", cursor, limit)
		return err
	}); err != nil {
		return cursor, 0, false, err
	}
	if len(ids) == 0 {
		return cursor, 0, true, nil
	}

	in := "IN (" + placeholders(len(ids)) + ")"
	args := stringsToAny(ids)
	for _, step := range []struct {
		t     lineCopyTable
		where string
	}{
		{orderLineCopies, "sol.id " + in},
		{invoiceLineCopies, "il.sales_order_line_id " + in},
		{pickLineCopies, "pl.sales_order_line_id " + in},
	} {
		var rows []lineCopyRow
		if err := m.Time(func() (err error) {
			rows, err = readLineCopies(ctx, b.read, step.t, step.where+" AND "+step.t.uncopiedWhere(step.t.alias), args...)
			return err
		}); err != nil {
			return cursor, 0, false, err
		}
		if err := m.Time(func() error { return fillLineCopies(ctx, b.write, step.t, rows) }); err != nil {
			return cursor, 0, false, err
		}
	}
	return ids[len(ids)-1], len(ids), len(ids) < limit, nil
}
