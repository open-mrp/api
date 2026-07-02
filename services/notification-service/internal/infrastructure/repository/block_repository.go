package repository

import (
	"context"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var blockRepoTracer = tracing.GetTracer("notification-service.block_repository")

type blockRepoImpl struct {
	db *sqlc.Queries
}

func NewBlockRepo(db *sqlc.Queries) domain.BlockRepo {
	return &blockRepoImpl{db: db}
}

func (r *blockRepoImpl) Create(ctx context.Context, id, accountID, blockerAccountUserID, blockedAccountUserID string) (*domain.MessageBlock, *apierror.APIError) {
	ctx, span := blockRepoTracer.Start(ctx, "repository.block.create")
	defer span.End()
	err := r.db.CreateBlock(ctx, sqlc.CreateBlockParams{
		ID:                   id,
		AccountID:            accountID,
		BlockerAccountUserID: blockerAccountUserID,
		BlockedAccountUserID: blockedAccountUserID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// CreateBlock is idempotent, so read the row back to return the canonical block (the original id and created_at win on a re-block).
	row, err := r.db.GetBlockByPair(ctx, sqlc.GetBlockByPairParams{
		BlockerAccountUserID: blockerAccountUserID,
		BlockedAccountUserID: blockedAccountUserID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.MessageBlock{
		ID:                   row.ID,
		AccountID:            row.AccountID,
		BlockerAccountUserID: row.BlockerAccountUserID,
		BlockedAccountUserID: row.BlockedAccountUserID,
		CreatedAt:            row.CreatedAt,
	}, nil
}

func (r *blockRepoImpl) Delete(ctx context.Context, blockerAccountUserID, blockedAccountUserID string) *apierror.APIError {
	ctx, span := blockRepoTracer.Start(ctx, "repository.block.delete")
	defer span.End()
	err := r.db.DeleteBlock(ctx, sqlc.DeleteBlockParams{
		BlockerAccountUserID: blockerAccountUserID,
		BlockedAccountUserID: blockedAccountUserID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *blockRepoImpl) ExistsBetween(ctx context.Context, a, b string) (bool, *apierror.APIError) {
	ctx, span := blockRepoTracer.Start(ctx, "repository.block.exists_between")
	defer span.End()
	blocked, err := r.db.BlockExistsBetween(ctx, sqlc.BlockExistsBetweenParams{A: a, B: b})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return blocked, nil
}

func (r *blockRepoImpl) List(ctx context.Context, blockerAccountUserID string) ([]*domain.MessageBlock, *apierror.APIError) {
	ctx, span := blockRepoTracer.Start(ctx, "repository.block.list")
	defer span.End()
	rows, err := r.db.ListBlocks(ctx, blockerAccountUserID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	blocks := make([]*domain.MessageBlock, 0, len(rows))
	for _, row := range rows {
		blocks = append(blocks, &domain.MessageBlock{
			ID:                   row.ID,
			AccountID:            row.AccountID,
			BlockerAccountUserID: row.BlockerAccountUserID,
			BlockedAccountUserID: row.BlockedAccountUserID,
			CreatedAt:            row.CreatedAt,
		})
	}
	return blocks, nil
}
