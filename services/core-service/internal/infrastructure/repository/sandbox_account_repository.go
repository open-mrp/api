package repository

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var sandboxAccountRepoTracer = tracing.GetTracer("core-service.sandbox_account_repository")

type sandboxAccountRepoImpl struct {
	queries *sqlc.Queries
}

func NewSandboxAccountRepo(queries *sqlc.Queries) domain.SandboxAccountRepo {
	return &sandboxAccountRepoImpl{queries: queries}
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
