package repository

import (
	"context"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/ptrutil"
	"github.com/augno/api/shared/tracing"
)

var accountUserRepoTracer = tracing.GetTracer("auth-service.account_user_repository")

type accountUserRepoImpl struct {
	db *sqlc.Queries
}

func NewAccountUserRepo(db *sqlc.Queries) domain.AccountUserRepo {
	return &accountUserRepoImpl{db: db}
}

func (r *accountUserRepoImpl) FindByAccountAndUserID(ctx context.Context, userID, accountID string) (*domain.AccountUser, *contracts.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.findByAccountAndUserID")
	defer span.End()

	accountUserRow, err := r.db.FindAccountUserWithRoleByAccountIDAndUserID(ctx, sqlc.FindAccountUserWithRoleByAccountIDAndUserIDParams{
		AccountID: accountID,
		UserID:    userID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == contracts.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.AccountUser{
		ID:           accountUserRow.ID,
		UserID:       accountUserRow.UserID,
		DepartmentID: ptrutil.NullStringToPtr(accountUserRow.DepartmentID),
		RoleID:       &accountUserRow.RoleID.String,
		RoleTypeCode: ptrutil.NullStringToPtr(accountUserRow.RoleTypeCode),
		AccountID:    accountUserRow.AccountID,
		LastUsedAt:   ptrutil.NullTimeToPtr(accountUserRow.LastUsedAt),
		CreatedAt:    accountUserRow.CreatedAt,
		UpdatedAt:    accountUserRow.UpdatedAt,
	}, nil
}

func (r *accountUserRepoImpl) UpdateLastUsedAt(ctx context.Context, accountUserID string, lastUsedAt time.Time) *contracts.APIError {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.updateLastUsedAt")
	defer span.End()

	err := r.db.UpdateAccountUserLastUsedAt(ctx, sqlc.UpdateAccountUserLastUsedAtParams{
		ID:         accountUserID,
		LastUsedAt: db.NullTime(lastUsedAt),
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountUserRepoImpl) FindAccountAffiliationsByUserID(ctx context.Context, userID string) ([]domain.AccountAffiliation, *contracts.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.findAccountAffiliationsByUserID")
	defer span.End()

	accountAffiliationRows, err := r.db.FindAccountAffiliationsByUserID(ctx, userID)

	if err != nil {
		apiErr := db.MapSQLError(err)
		return nil, tracing.Trace(span, apiErr)
	}

	accountAffiliations := make([]domain.AccountAffiliation, len(accountAffiliationRows))
	for i, row := range accountAffiliationRows {
		accountAffiliations[i] = domain.AccountAffiliation{
			AccountID:   row.AccountID,
			AccountName: row.AccountName,
			RoleID:      row.RoleID,
			RoleName:    row.RoleName,
			LastUsedAt:  ptrutil.NullTimeToPtr(row.LastUsedAt),
		}
	}

	return accountAffiliations, nil
}

func (r *accountUserRepoImpl) FindLastUsedAccountIDByUserID(ctx context.Context, userID string) (string, *contracts.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.findLastUsedAccountIDByUserID")
	defer span.End()

	accountID, err := r.db.FindLastUsedAccountID(ctx, userID)

	if err != nil {
		apiErr := db.MapSQLError(err)
		return "", tracing.Trace(span, apiErr)
	}

	return accountID, nil
}
