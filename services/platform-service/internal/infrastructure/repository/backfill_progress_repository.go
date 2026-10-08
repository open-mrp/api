package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db/backfill"
)

type backfillProgressRepo struct {
	queries *sqlc.Queries
}

// NewBackfillProgressRepo stores backfill cursors in backfill_progress.
func NewBackfillProgressRepo(queries *sqlc.Queries) backfill.Progress {
	return &backfillProgressRepo{queries: queries}
}

func (r *backfillProgressRepo) Load(ctx context.Context, name string) (string, bool, error) {
	row, err := r.queries.GetBackfillProgress(ctx, name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return row.CursorValue, row.CompletedAt.Valid, nil
}

func (r *backfillProgressRepo) Save(ctx context.Context, name, cursor string, rows int64, completed bool) error {
	var completedAt sql.NullTime
	if completed {
		completedAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	}
	return r.queries.SaveBackfillProgress(ctx, sqlc.SaveBackfillProgressParams{
		Name:        name,
		CursorValue: cursor,
		RowsDone:    rows,
		CompletedAt: completedAt,
	})
}
