package repository

import (
	"context"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var accountUserRepoTracer = tracing.GetTracer("core-service.account_user_repository")

type accountUserRepoImpl struct {
	queries *sqlc.Queries
}

func NewAccountUserRepo(queries *sqlc.Queries) domain.AccountUserRepo {
	return &accountUserRepoImpl{queries: queries}
}

func (r *accountUserRepoImpl) FindByAccountAndUserID(ctx context.Context, userID, accountID string) (*domain.AccountUser, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.find_by_account_and_user_id")
	defer span.End()

	row, err := r.queries.FindAccountUserWithRoleByAccountIDAndUserID(ctx, sqlc.FindAccountUserWithRoleByAccountIDAndUserIDParams{
		AccountID: accountID,
		UserID:    userID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.AccountUser{
		ID:           row.ID,
		UserID:       row.UserID,
		DepartmentID: db.StringFromNullString(row.DepartmentID),
		RoleID:       db.StringFromNullString(row.RoleID),
		RoleTypeCode: db.StringFromNullString(row.RoleTypeCode),
		AccountID:    row.AccountID,
		LastUsedAt:   db.TimeFromNullTime(row.LastUsedAt),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

func (r *accountUserRepoImpl) FindAffiliationsByUserID(ctx context.Context, userID string) ([]domain.AccountAffiliation, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.find_affiliations_by_user_id")
	defer span.End()

	rows, err := r.queries.FindAccountAffiliationsByUserID(ctx, userID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	affiliations := make([]domain.AccountAffiliation, len(rows))
	for i, row := range rows {
		affiliations[i] = domain.AccountAffiliation{
			AccountID:   row.AccountID,
			AccountName: row.AccountName,
			RoleID:      row.RoleID,
			RoleName:    row.RoleName,
			LastUsedAt:  db.TimeFromNullTime(row.LastUsedAt),
		}
	}

	return affiliations, nil
}

func (r *accountUserRepoImpl) FindLastUsedAccountID(ctx context.Context, userID string) (string, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.find_last_used_account_id")
	defer span.End()

	accountID, err := r.queries.FindLastUsedAccountID(ctx, userID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return "", nil
		}
		return "", tracing.Trace(span, apiErr)
	}

	return accountID, nil
}

func (r *accountUserRepoImpl) UpdateLastUsedAt(ctx context.Context, accountUserID string, lastUsedAt time.Time) *apierror.APIError {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.update_last_used_at")
	defer span.End()

	err := r.queries.UpdateAccountUserLastUsedAt(ctx, sqlc.UpdateAccountUserLastUsedAtParams{
		ID:         accountUserID,
		LastUsedAt: db.NullTime(lastUsedAt),
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
