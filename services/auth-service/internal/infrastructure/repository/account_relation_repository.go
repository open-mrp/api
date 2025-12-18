package repository

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/tracing"
)

var accountRelationRepoTracer = tracing.GetTracer("auth-service.account_relation_repository")

type accountRelationRepoImpl struct {
	db *sqlc.Queries
}

func NewAccountRelationRepo(db *sqlc.Queries) domain.AccountRelationRepo {
	return &accountRelationRepoImpl{db: db}
}

func (r *accountRelationRepoImpl) FindByOwnerAccountAndUserID(ctx context.Context, ownerAccountID, userID string) (*domain.AuthAccountRelation, *contracts.APIError) {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.findByOwnerAccountAndUserID")
	defer span.End()

	accountRelationModel, err := r.db.FindAccountRelationByOwnerAccountIDAndUserID(ctx, sqlc.FindAccountRelationByOwnerAccountIDAndUserIDParams{
		OwnerAccountID: ownerAccountID,
		UserID:         userID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == contracts.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	roleCode, ok := types.ParseIdentityActorType(accountRelationModel.AccountRelationRoleCode)
	if !ok {
		err := fmt.Errorf("invalid account relation role code: %s", accountRelationModel.AccountRelationRoleCode)
		return nil, tracing.Trace(span, contracts.NewInternalError(err, "Failed to map account relation role code."))
	}

	return &domain.AuthAccountRelation{
		ID:                      accountRelationModel.ID,
		CounterpartyAccountID:   accountRelationModel.CounterpartyAccountID,
		AccountRelationRoleCode: roleCode,
	}, nil
}

func (r *accountRelationRepoImpl) FindByOwnerAccountAndAPIKeyID(ctx context.Context, ownerAccountID, apiKeyID string) (*domain.AuthAccountRelation, *contracts.APIError) {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.findByOwnerAccountAndAPIKeyID")
	defer span.End()

	accountRelationModel, err := r.db.FindAccountRelationByOwnerAccountIDAndAPIKeyID(ctx, sqlc.FindAccountRelationByOwnerAccountIDAndAPIKeyIDParams{
		OwnerAccountID: ownerAccountID,
		ID:             apiKeyID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == contracts.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	roleCode, ok := types.ParseIdentityActorType(accountRelationModel.AccountRelationRoleCode)
	if !ok {
		err := fmt.Errorf("invalid account relation role code: %s", accountRelationModel.AccountRelationRoleCode)
		return nil, tracing.Trace(span, contracts.NewInternalError(err, "Failed to map account relation role code."))
	}

	return &domain.AuthAccountRelation{
		ID:                      accountRelationModel.ID,
		CounterpartyAccountID:   accountRelationModel.CounterpartyAccountID,
		AccountRelationRoleCode: roleCode,
	}, nil
}
