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

var shipmentLineRepoTracer = tracing.GetTracer("core-service.shipment_line_repository")

type shipmentLineRepoImpl struct {
	queries *sqlc.Queries
}

func NewShipmentLineRepo(queries *sqlc.Queries) domain.ShipmentLineRepo {
	return &shipmentLineRepoImpl{queries: queries}
}

var shipmentLineCreatedAt = func(d *domain.ShipmentLine) time.Time { return d.CreatedAt }
var shipmentLineID = func(d *domain.ShipmentLine) string { return d.ID }

func optionalShipmentLineSearch(q *string) any {
	if q == nil || *q == "" {
		return nil
	}
	return *q
}

func (r *shipmentLineRepoImpl) List(ctx context.Context, params domain.ListShipmentLinesParams) (*domain.ListShipmentLinesResult, *apierror.APIError) {
	ctx, span := shipmentLineRepoTracer.Start(ctx, "repository.shipment_line.list")
	defer span.End()

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListShipmentLinesBackward(ctx, sqlc.ListShipmentLinesBackwardParams{
				ShipmentID:      params.ShipmentID,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			lines := make([]*domain.ShipmentLine, len(rows))
			for i, row := range rows {
				lines[i] = mapBackwardShipmentLineRow(row)
			}
			result, pageInfo := pagination.BuildPageString(lines, params.Limit, cursorDir, shipmentLineCreatedAt, shipmentLineID)
			return &domain.ListShipmentLinesResult{Lines: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListShipmentLinesForward(ctx, sqlc.ListShipmentLinesForwardParams{
			ShipmentID:      params.ShipmentID,
			Search:          optionalShipmentLineSearch(params.Query),
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		lines := make([]*domain.ShipmentLine, len(rows))
		for i, row := range rows {
			lines[i] = mapForwardShipmentLineRow(row)
		}
		result, pageInfo := pagination.BuildPageString(lines, params.Limit, cursorDir, shipmentLineCreatedAt, shipmentLineID)
		return &domain.ListShipmentLinesResult{Lines: result, PageInfo: pageInfo}, nil
	}

	// No cursor - forward from beginning
	rows, err := r.queries.ListShipmentLinesForward(ctx, sqlc.ListShipmentLinesForwardParams{
		ShipmentID:      params.ShipmentID,
		Search:          optionalShipmentLineSearch(params.Query),
		CursorCreatedAt: gosql.NullTime{},
		CursorID:        gosql.NullString{},
		Limit:           params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	lines := make([]*domain.ShipmentLine, len(rows))
	for i, row := range rows {
		lines[i] = mapForwardShipmentLineRow(row)
	}
	result, pageInfo := pagination.BuildPageString(lines, params.Limit, cursorDir, shipmentLineCreatedAt, shipmentLineID)
	return &domain.ListShipmentLinesResult{Lines: result, PageInfo: pageInfo}, nil
}

func (r *shipmentLineRepoImpl) Get(ctx context.Context, shipmentLineID string) (*domain.ShipmentLine, *apierror.APIError) {
	ctx, span := shipmentLineRepoTracer.Start(ctx, "repository.shipment_line.get")
	defer span.End()

	row, err := r.queries.GetShipmentLine(ctx, shipmentLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetShipmentLineRow(row), nil
}

func (r *shipmentLineRepoImpl) Create(ctx context.Context, id, quantityID string, params domain.CreateShipmentLineEndpointParams) (*domain.ShipmentLine, *apierror.APIError) {
	ctx, span := shipmentLineRepoTracer.Start(ctx, "repository.shipment_line.create")
	defer span.End()

	// Create the quantity first
	if err := r.queries.CreateShipmentLineQuantity(ctx, sqlc.CreateShipmentLineQuantityParams{
		ID:     quantityID,
		Value:  params.QuantityValue,
		UnitID: params.QuantityUnitID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Create the shipment line
	if err := r.queries.CreateShipmentLine(ctx, sqlc.CreateShipmentLineParams{
		ID:               id,
		ShipmentID:       params.ShipmentID,
		SalesOrderLineID: params.SalesOrderLineID,
		QuantityID:       quantityID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return r.Get(ctx, id)
}

func (r *shipmentLineRepoImpl) Update(ctx context.Context, params domain.UpdateShipmentLineEndpointParams) (*domain.ShipmentLine, *apierror.APIError) {
	ctx, span := shipmentLineRepoTracer.Start(ctx, "repository.shipment_line.update")
	defer span.End()

	updateParams := sqlc.UpdateShipmentLineParams{
		ShipmentLineID: params.ShipmentLineID,
	}
	if params.QuantityValue != nil {
		updateParams.Value = gosql.NullString{String: *params.QuantityValue, Valid: true}
	}
	if params.QuantityUnitID != nil {
		updateParams.UnitID = gosql.NullString{String: *params.QuantityUnitID, Valid: true}
	}

	result, err := r.queries.UpdateShipmentLine(ctx, updateParams)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Shipment line not found."))
	}

	return r.Get(ctx, params.ShipmentLineID)
}

func (r *shipmentLineRepoImpl) Delete(ctx context.Context, shipmentLineID string) *apierror.APIError {
	ctx, span := shipmentLineRepoTracer.Start(ctx, "repository.shipment_line.delete")
	defer span.End()

	// Delete the quantity first (references shipment_line)
	if err := r.queries.DeleteShipmentLineQuantity(ctx, shipmentLineID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	if err := r.queries.DeleteShipmentLine(ctx, shipmentLineID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func (r *shipmentLineRepoImpl) IsInShipment(ctx context.Context, shipmentLineID, shipmentID string) (bool, *apierror.APIError) {
	ctx, span := shipmentLineRepoTracer.Start(ctx, "repository.shipment_line.is_in_shipment")
	defer span.End()

	exists, err := r.queries.IsShipmentLineInShipment(ctx, sqlc.IsShipmentLineInShipmentParams{
		ShipmentLineID: shipmentLineID,
		ShipmentID:     shipmentID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exists, nil
}

func (r *shipmentLineRepoImpl) ListByShipment(ctx context.Context, shipmentID string) ([]*domain.ShipmentLine, *apierror.APIError) {
	ctx, span := shipmentLineRepoTracer.Start(ctx, "repository.shipment_line.list_by_shipment")
	defer span.End()

	rows, err := r.queries.ListShipmentLinesByShipment(ctx, shipmentID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lines := make([]*domain.ShipmentLine, len(rows))
	for i, row := range rows {
		lines[i] = mapListByShipmentRow(row)
	}

	return lines, nil
}

func (r *shipmentLineRepoImpl) DeleteByShipment(ctx context.Context, shipmentID string) *apierror.APIError {
	ctx, span := shipmentLineRepoTracer.Start(ctx, "repository.shipment_line.delete_by_shipment")
	defer span.End()

	// Delete quantities first (subquery references shipment_line)
	if err := r.queries.DeleteShipmentLineQuantitiesByShipment(ctx, shipmentID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	if err := r.queries.DeleteShipmentLinesByShipment(ctx, shipmentID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func mapForwardShipmentLineRow(row sqlc.ListShipmentLinesForwardRow) *domain.ShipmentLine {
	l := &domain.ShipmentLine{
		ID:                       row.ID,
		ShipmentID:               row.ShipmentID,
		SalesOrderLineID:         row.SalesOrderLineID,
		OrderLineSKU:             row.ProductSku,
		QuantityID:               row.QuantityID,
		QuantityValue:            row.QuantityValue,
		QuantityUnitID:           row.QuantityUnitID,
		QuantityUnitName:         row.QuantityUnitName,
		QuantityUnitAbbreviation: row.QuantityUnitAbbreviation,
		QuantityUnitType:         row.QuantityUnitType,
		CreatedAt:                row.CreatedAt,
		UpdatedAt:                row.UpdatedAt,
	}
	if row.ProductDescription.Valid {
		l.OrderLineDesc = &row.ProductDescription.String
	}
	return l
}

func mapBackwardShipmentLineRow(row sqlc.ListShipmentLinesBackwardRow) *domain.ShipmentLine {
	l := &domain.ShipmentLine{
		ID:                       row.ID,
		ShipmentID:               row.ShipmentID,
		SalesOrderLineID:         row.SalesOrderLineID,
		OrderLineSKU:             row.ProductSku,
		QuantityID:               row.QuantityID,
		QuantityValue:            row.QuantityValue,
		QuantityUnitID:           row.QuantityUnitID,
		QuantityUnitName:         row.QuantityUnitName,
		QuantityUnitAbbreviation: row.QuantityUnitAbbreviation,
		QuantityUnitType:         row.QuantityUnitType,
		CreatedAt:                row.CreatedAt,
		UpdatedAt:                row.UpdatedAt,
	}
	if row.ProductDescription.Valid {
		l.OrderLineDesc = &row.ProductDescription.String
	}
	return l
}

func mapGetShipmentLineRow(row sqlc.GetShipmentLineRow) *domain.ShipmentLine {
	l := &domain.ShipmentLine{
		ID:                       row.ID,
		ShipmentID:               row.ShipmentID,
		SalesOrderLineID:         row.SalesOrderLineID,
		OrderLineSKU:             row.ProductSku,
		QuantityID:               row.QuantityID,
		QuantityValue:            row.QuantityValue,
		QuantityUnitID:           row.QuantityUnitID,
		QuantityUnitName:         row.QuantityUnitName,
		QuantityUnitAbbreviation: row.QuantityUnitAbbreviation,
		QuantityUnitType:         row.QuantityUnitType,
		CreatedAt:                row.CreatedAt,
		UpdatedAt:                row.UpdatedAt,
	}
	if row.ProductDescription.Valid {
		l.OrderLineDesc = &row.ProductDescription.String
	}
	return l
}

func mapListByShipmentRow(row sqlc.ListShipmentLinesByShipmentRow) *domain.ShipmentLine {
	l := &domain.ShipmentLine{
		ID:                       row.ID,
		ShipmentID:               row.ShipmentID,
		SalesOrderLineID:         row.SalesOrderLineID,
		OrderLineSKU:             row.ProductSku,
		QuantityID:               row.QuantityID,
		QuantityValue:            row.QuantityValue,
		QuantityUnitID:           row.QuantityUnitID,
		QuantityUnitName:         row.QuantityUnitName,
		QuantityUnitAbbreviation: row.QuantityUnitAbbreviation,
		QuantityUnitType:         row.QuantityUnitType,
		CreatedAt:                row.CreatedAt,
		UpdatedAt:                row.UpdatedAt,
	}
	if row.ProductDescription.Valid {
		l.OrderLineDesc = &row.ProductDescription.String
	}
	if row.OrderLineItemID.Valid {
		l.OrderLineItemID = &row.OrderLineItemID.String
	}
	return l
}
