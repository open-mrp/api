package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/apikit/db/backfill"
)

type backfillProgressRepo struct {
	queries *sqlc.Queries
}

// NewBackfillProgressRepo is where agent-service's paced backfills keep their place.
func NewBackfillProgressRepo(queries *sqlc.Queries) backfill.Progress {
	return &backfillProgressRepo{queries: queries}
}

func (r *backfillProgressRepo) Load(ctx context.Context, name string) (string, bool, error) {
	row, err := r.queries.GetBackfillProgress(ctx, name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return row.CursorValue, row.CompletedAt.Valid, nil
}

func (r *backfillProgressRepo) Save(ctx context.Context, name, cursor string, rows int64, completed bool) error {
	var completedAt pgtype.Timestamptz
	if completed {
		completedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	}
	return r.queries.SaveBackfillProgress(ctx, sqlc.SaveBackfillProgressParams{
		Name:        name,
		CursorValue: cursor,
		RowsDone:    rows,
		CompletedAt: completedAt,
	})
}

// ListAgentMemoriesAfter pages every account's memories by ID.
func ListAgentMemoriesAfter(ctx context.Context, queries *sqlc.Queries, afterID string, limit int) ([]sqlc.ListAgentMemoriesAfterRow, error) {
	return queries.ListAgentMemoriesAfter(ctx, sqlc.ListAgentMemoriesAfterParams{AfterID: afterID, RowLimit: int32(limit)})
}
