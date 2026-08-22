package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	apierror "github.com/open-mrp/api/shared/errors"
)

type orderQueryRepo struct {
	queries *sqlc.Queries
}

func NewOrderQueryRepo(queries *sqlc.Queries) domain.OrderQueryRepo {
	return &orderQueryRepo{queries: queries}
}

func (r *orderQueryRepo) FindIDByProductionRun(ctx context.Context, accountID, productionRunID string) (*string, *apierror.APIError) {
	id, err := r.queries.FindOrderIDByProductionRun(ctx, sqlc.FindOrderIDByProductionRunParams{
		AccountID:       accountID,
		ProductionRunID: sql.NullString{String: productionRunID, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, apierror.NewInternalError(err, "Failed to find order by production run.")
	}
	return &id, nil
}
