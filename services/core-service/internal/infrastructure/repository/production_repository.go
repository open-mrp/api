package repository

import (
	"context"
	"database/sql"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var productionRepoTracer = tracing.GetTracer("core-service.production_repository")

type productionRepoImpl struct {
	queries *sqlc.Queries
}

func NewProductionRepo(queries *sqlc.Queries) domain.ProductionRepo {
	return &productionRepoImpl{queries: queries}
}

func (r *productionRepoImpl) Get(ctx context.Context, accountID, productionStepID, productionID string) (*domain.Production, *apierror.APIError) {
	ctx, span := productionRepoTracer.Start(ctx, "repository.production.get")
	defer span.End()

	row, err := r.queries.GetProductionByID(ctx, sqlc.GetProductionByIDParams{
		ID:               productionID,
		ProductionStepID: sql.NullString{String: productionStepID, Valid: true},
		AccountID:        accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.Production{
		ID:              row.ID,
		ItemID:          row.ItemID,
		ItemSKU:         row.ItemSku,
		ItemDescription: nullStringToPtr(row.ItemDescription),
		ItemTypeCode:    row.ItemTypeCode,
		Quantity: domain.Quantity{
			ID:               row.QuantityID,
			Value:            row.QuantityValue,
			UnitID:           row.QuantityUnitID,
			UnitAbbreviation: row.QuantityUnitAbbreviation,
			UnitType:         row.QuantityUnitType,
		},
		ProductionStepID: row.ProductionStepID.String,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

func (r *productionRepoImpl) UpdateItem(ctx context.Context, productionID, itemID string) *apierror.APIError {
	ctx, span := productionRepoTracer.Start(ctx, "repository.production.update_item")
	defer span.End()

	err := r.queries.UpdateProductionItem(ctx, sqlc.UpdateProductionItemParams{
		ItemID: itemID,
		ID:     productionID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionRepoImpl) UpdateQuantity(ctx context.Context, productionID, value, unitID string) *apierror.APIError {
	ctx, span := productionRepoTracer.Start(ctx, "repository.production.update_quantity")
	defer span.End()

	err := r.queries.UpdateProductionQuantity(ctx, sqlc.UpdateProductionQuantityParams{
		Value:        value,
		UnitID:       unitID,
		ProductionID: productionID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionRepoImpl) GetQuantityID(ctx context.Context, productionID string) (string, *apierror.APIError) {
	ctx, span := productionRepoTracer.Start(ctx, "repository.production.get_quantity_id")
	defer span.End()

	quantityID, err := r.queries.GetProductionQuantityID(ctx, productionID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return quantityID, nil
}
