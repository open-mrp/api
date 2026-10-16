package repository

import (
	"context"
	gosql "database/sql"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/apikit/db/backfill"
)

// ManufacturingMethodBackfillName is the backfill's entry in backfill_progress.
const ManufacturingMethodBackfillName = "manufacturing_methods_from_steps"

// ManufacturingMethodBackfill makes every production step written before manufacturing methods the single operation of
// an active method for the item it produces, with its consumptions as materials and its machines. It walks steps by id
// from the replica and adopts each in its own short transaction on the primary; an adopted step is only re-synced, so
// a re-run batch changes nothing.
type ManufacturingMethodBackfill struct {
	read  *sqlc.Queries
	write *gosql.DB
}

func NewManufacturingMethodBackfill(read *sqlc.Queries, write *gosql.DB) *ManufacturingMethodBackfill {
	return &ManufacturingMethodBackfill{read: read, write: write}
}

// Batch is a backfill.Batch over production step ids.
func (b *ManufacturingMethodBackfill) Batch(ctx context.Context, m *backfill.Meter, cursor string, limit int) (string, int, bool, error) {
	var ids []string
	if err := m.Time(func() error {
		var apiErr error
		ids, apiErr = b.read.ListUnadoptedStepIDsAfter(ctx, sqlc.ListUnadoptedStepIDsAfterParams{AfterID: cursor, Limit: int32(limit)}) // #nosec G115 -- the runner's batch size
		return apiErr
	}); err != nil {
		return cursor, 0, false, err
	}
	if len(ids) == 0 {
		return cursor, 0, true, nil
	}
	for _, stepID := range ids {
		if err := m.Time(func() error { return b.adopt(ctx, stepID) }); err != nil {
			return cursor, 0, false, err
		}
	}
	return ids[len(ids)-1], len(ids), len(ids) < limit, nil
}

func (b *ManufacturingMethodBackfill) adopt(ctx context.Context, stepID string) error {
	tx, err := b.write.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if apiErr := NewManufacturingMethodRepo(sqlc.New(tx)).SyncLegacyStep(ctx, stepID); apiErr != nil {
		return apiErr
	}
	return tx.Commit()
}
