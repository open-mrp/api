package repository

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var accountRelationRepoTracer = tracing.GetTracer("core-service.account_relation_repository")

type accountRelationRepoImpl struct {
	queries *sqlc.Queries
}

func NewAccountRelationRepo(queries *sqlc.Queries) domain.AccountRelationRepo {
	return &accountRelationRepoImpl{queries: queries}
}

func (r *accountRelationRepoImpl) FindByOwnerAccountAndUserID(ctx context.Context, ownerAccountID, userID string) (*domain.AccountRelation, *apierror.APIError) {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.find_by_owner_account_and_user_id")
	defer span.End()

	row, err := r.queries.FindAccountRelationByOwnerAccountIDAndUserID(ctx, sqlc.FindAccountRelationByOwnerAccountIDAndUserIDParams{
		OwnerAccountID: ownerAccountID,
		UserID:         userID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.AccountRelation{
		ID:                    row.ID,
		CounterpartyAccountID: row.CounterpartyAccountID,
		RoleCode:              row.AccountRelationRoleCode,
	}, nil
}

func (r *accountRelationRepoImpl) FindByOwnerAccountAndAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*domain.AccountRelation, *apierror.APIError) {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.find_by_owner_account_and_api_key_id")
	defer span.End()

	row, err := r.queries.FindAccountRelationByOwnerAccountIDAndAPIKeyID(ctx, sqlc.FindAccountRelationByOwnerAccountIDAndAPIKeyIDParams{
		OwnerAccountID: ownerAccountID,
		ID:             apiKeyID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.AccountRelation{
		ID:                    row.ID,
		CounterpartyAccountID: row.CounterpartyAccountID,
		RoleCode:              row.AccountRelationRoleCode,
	}, nil
}
