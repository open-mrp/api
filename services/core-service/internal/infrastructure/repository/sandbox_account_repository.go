package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var sandboxAccountRepoTracer = tracing.GetTracer("core-service.sandbox_account_repository")

type sandboxAccountRepoImpl struct {
	queries *sqlc.Queries
}

func NewSandboxAccountRepo(queries *sqlc.Queries) domain.SandboxAccountRepo {
	return &sandboxAccountRepoImpl{queries: queries}
}

func sandboxCreatedAt(s *domain.SandboxAccount) time.Time { return s.CreatedAt }
func sandboxID(s *domain.SandboxAccount) int64            { return s.ID }

func mapSandboxForwardRow(row sqlc.ListSandboxAccountsForwardRow) *domain.SandboxAccount {
	return &domain.SandboxAccount{
		ID:             row.ID,
		TypeID:         row.TypeID,
		OwnerAccountID: row.OwnerAccountID,
		AccountID:      row.AccountID,
		Name:           row.Name,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func mapSandboxBackwardRow(row sqlc.ListSandboxAccountsBackwardRow) *domain.SandboxAccount {
	return &domain.SandboxAccount{
		ID:             row.ID,
		TypeID:         row.TypeID,
		OwnerAccountID: row.OwnerAccountID,
		AccountID:      row.AccountID,
		Name:           row.Name,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func (r *sandboxAccountRepoImpl) List(ctx context.Context, ownerAccountID string, cursor *string, limit int32) (*domain.ListSandboxAccountsResult, *apierror.APIError) {
	ctx, span := sandboxAccountRepoTracer.Start(ctx, "repository.sandbox_account.list")
	defer span.End()

	var cursorDir *pagination.Direction

	if cursor != nil {
		cur, err := pagination.DecodeCursor(*cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListSandboxAccountsBackward(ctx, sqlc.ListSandboxAccountsBackwardParams{
				OwnerAccountID:  ownerAccountID,
				CursorCreatedAt: cur.CreatedAt,
				CursorID:        cur.ID,
				Limit:           limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			sandboxes := make([]*domain.SandboxAccount, len(rows))
			for i, row := range rows {
				sandboxes[i] = mapSandboxBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPage(sandboxes, limit, cursorDir, sandboxCreatedAt, sandboxID)
			return &domain.ListSandboxAccountsResult{Sandboxes: result, PageInfo: pageInfo}, nil
		}

		// Forward
		rows, err := r.queries.ListSandboxAccountsForward(ctx, sqlc.ListSandboxAccountsForwardParams{
			OwnerAccountID:  ownerAccountID,
			CursorCreatedAt: gosql.NullTime{Time: cur.CreatedAt, Valid: true},
			CursorID:        gosql.NullInt64{Int64: cur.ID, Valid: true},
			Limit:           limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		sandboxes := make([]*domain.SandboxAccount, len(rows))
		for i, row := range rows {
			sandboxes[i] = mapSandboxForwardRow(row)
		}
		result, pageInfo := pagination.BuildPage(sandboxes, limit, cursorDir, sandboxCreatedAt, sandboxID)
		return &domain.ListSandboxAccountsResult{Sandboxes: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListSandboxAccountsForward(ctx, sqlc.ListSandboxAccountsForwardParams{
		OwnerAccountID: ownerAccountID,
		Limit:          limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	sandboxes := make([]*domain.SandboxAccount, len(rows))
	for i, row := range rows {
		sandboxes[i] = mapSandboxForwardRow(row)
	}
	result, pageInfo := pagination.BuildPage(sandboxes, limit, cursorDir, sandboxCreatedAt, sandboxID)
	return &domain.ListSandboxAccountsResult{Sandboxes: result, PageInfo: pageInfo}, nil
}

func (r *sandboxAccountRepoImpl) Create(ctx context.Context, typeID, ownerAccountID, accountID string) *apierror.APIError {
	ctx, span := sandboxAccountRepoTracer.Start(ctx, "repository.sandbox_account.create")
	defer span.End()

	_, err := r.queries.CreateSandboxAccount(ctx, sqlc.CreateSandboxAccountParams{
		TypeID:         typeID,
		OwnerAccountID: ownerAccountID,
		AccountID:      accountID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *sandboxAccountRepoImpl) CountByOwnerAccountID(ctx context.Context, ownerAccountID string) (int64, *apierror.APIError) {
	ctx, span := sandboxAccountRepoTracer.Start(ctx, "repository.sandbox_account.count_by_owner_account_id")
	defer span.End()

	count, err := r.queries.CountSandboxAccounts(ctx, ownerAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return count, nil
}

func (r *sandboxAccountRepoImpl) FindByTypeID(ctx context.Context, typeID string) (*domain.SandboxAccount, *apierror.APIError) {
	ctx, span := sandboxAccountRepoTracer.Start(ctx, "repository.sandbox_account.find_by_type_id")
	defer span.End()

	row, err := r.queries.FindSandboxAccountByTypeID(ctx, typeID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.SandboxAccount{
		ID:             row.ID,
		TypeID:         row.TypeID,
		OwnerAccountID: row.OwnerAccountID,
		AccountID:      row.AccountID,
		Name:           row.Name,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

func (r *sandboxAccountRepoImpl) DeleteByID(ctx context.Context, id int64) *apierror.APIError {
	ctx, span := sandboxAccountRepoTracer.Start(ctx, "repository.sandbox_account.delete_by_id")
	defer span.End()

	err := r.queries.DeleteSandboxAccountByID(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *sandboxAccountRepoImpl) FindFirstByOwnerAccountID(ctx context.Context, ownerAccountID string) (string, *apierror.APIError) {
	ctx, span := sandboxAccountRepoTracer.Start(ctx, "repository.sandbox_account.find_first_by_owner_account_id")
	defer span.End()

	accountID, err := r.queries.FindFirstSandboxAccountByOwnerAccountID(ctx, ownerAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return accountID, nil
}
