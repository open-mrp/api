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

var pickLineRepoTracer = tracing.GetTracer("core-service.pick_line_repository")

type pickLineRepoImpl struct {
	queries *sqlc.Queries
}

func NewPickLineRepo(queries *sqlc.Queries) domain.PickLineRepo {
	return &pickLineRepoImpl{queries: queries}
}

func mapGetPickLineRow(row sqlc.GetPickLineRow) *domain.PickLine {
	var packedAt *time.Time
	if row.PackedAt.Valid {
		packedAt = &row.PackedAt.Time
	}

	var lineItemNumber int32
	if row.LineItemNumber.Valid {
		lineItemNumber = row.LineItemNumber.Int32
	}

	var productDescription *string
	if row.ProductDescription.Valid {
		productDescription = &row.ProductDescription.String
	}

	return &domain.PickLine{
		ID:                        row.ID,
		PickID:                    row.PickID,
		SalesOrderLineID:          row.SalesOrderLineID,
		QuantityID:                row.QuantityID,
		QuantityValue:             row.QuantityValue,
		QuantityUnitID:            row.QuantityUnitID,
		QuantityUnitName:          row.QuantityUnitName,
		QuantityUnitAbbreviation:  row.QuantityUnitAbbreviation,
		PackedAt:                  packedAt,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
		OrderLineItemNumber:       lineItemNumber,
		OrderLineSKU:              row.ProductSku,
		OrderLineDescription:      productDescription,
		OrderedQuantityValue:      row.OrderedQuantityValue,
		OrderedQuantityUnitID:     row.OrderedQuantityUnitID,
		OrderedQuantityUnitName:   row.OrderedQuantityUnitName,
		OrderedQuantityUnitAbbrev: row.OrderedQuantityUnitAbbreviation,
	}
}

func (r *pickLineRepoImpl) Get(ctx context.Context, pickLineID string) (*domain.PickLine, *apierror.APIError) {
	ctx, span := pickLineRepoTracer.Start(ctx, "repository.pick_line.get")
	defer span.End()

	row, err := r.queries.GetPickLine(ctx, pickLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetPickLineRow(row), nil
}

func (r *pickLineRepoImpl) UpdateQuantity(ctx context.Context, pickLineID, quantityValue string) *apierror.APIError {
	ctx, span := pickLineRepoTracer.Start(ctx, "repository.pick_line.update_quantity")
	defer span.End()

	if err := r.queries.UpdatePickLineQuantity(ctx, sqlc.UpdatePickLineQuantityParams{
		Value:      quantityValue,
		PickLineID: pickLineID,
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}

	return nil
}

func (r *pickLineRepoImpl) PickRemainingQuantity(ctx context.Context, pickLineID string) *apierror.APIError {
	ctx, span := pickLineRepoTracer.Start(ctx, "repository.pick_line.pick_remaining_quantity")
	defer span.End()

	if err := r.queries.PickRemainingQuantityForLine(ctx, pickLineID); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}

	return nil
}

func (r *pickLineRepoImpl) VoidLine(ctx context.Context, pickLineID string) *apierror.APIError {
	ctx, span := pickLineRepoTracer.Start(ctx, "repository.pick_line.void_line")
	defer span.End()

	if err := r.queries.VoidPickLine(ctx, pickLineID); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}

	return nil
}

func (r *pickLineRepoImpl) IsInPick(ctx context.Context, pickLineID, pickID string) (bool, *apierror.APIError) {
	ctx, span := pickLineRepoTracer.Start(ctx, "repository.pick_line.is_in_pick")
	defer span.End()

	result, err := r.queries.IsPickLineInPick(ctx, sqlc.IsPickLineInPickParams{
		PickLineID: pickLineID,
		PickID:     pickID,
	})
	if err != nil {
		return false, tracing.Trace(span, db.MapSQLError(err))
	}

	return result, nil
}

func (r *pickLineRepoImpl) CreateForRemaining(ctx context.Context, id, quantityID, pickID, orderLineID string) *apierror.APIError {
	ctx, span := pickLineRepoTracer.Start(ctx, "repository.pick_line.create_for_remaining")
	defer span.End()

	if err := r.queries.CreatePickLine(ctx, sqlc.CreatePickLineParams{
		ID:               id,
		PickID:           pickID,
		QuantityID:       quantityID,
		SalesOrderLineID: orderLineID,
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}

	return nil
}

func (r *pickLineRepoImpl) CalculateRemainingForOrderLine(ctx context.Context, orderLineID string) (string, string, *apierror.APIError) {
	ctx, span := pickLineRepoTracer.Start(ctx, "repository.pick_line.calculate_remaining")
	defer span.End()

	row, err := r.queries.CalculateRemainingForOrderLine(ctx, orderLineID)
	if err != nil {
		return "", "", tracing.Trace(span, db.MapSQLError(err))
	}

	// RemainingValue is sqlc-typed interface{} (the GREATEST/arithmetic result has no
	// single column type), so the MySQL driver scans it as []byte. fmt.Sprintf("%v", …)
	// on a []byte prints the byte-value array ("[53 46 …]"), which then fails ParseFloat
	// in createPickLineForRemainingQuantity and silently suppresses the remainder pick
	// line. decimalToString decodes the bytes back to the decimal string.
	remainingValue := decimalToString(row.RemainingValue)

	return remainingValue, row.UnitID, nil
}

func (r *pickLineRepoImpl) GetOrderLinePackProgress(ctx context.Context, orderLineID string) (string, string, string, *apierror.APIError) {
	ctx, span := pickLineRepoTracer.Start(ctx, "repository.pick_line.get_pack_progress")
	defer span.End()

	row, err := r.queries.GetOrderLinePackProgress(ctx, orderLineID)
	if err != nil {
		return "", "", "", tracing.Trace(span, db.MapSQLError(err))
	}

	// PackedValue is a SUM(CASE ...) result, sqlc-typed interface{} and scanned as
	// []byte by the driver — decode it explicitly (see CalculateRemainingForOrderLine).
	return row.OrderedValue, decimalToString(row.PackedValue), row.UnitID, nil
}

func (r *pickLineRepoImpl) DeleteUnpackedForOrderLine(ctx context.Context, orderLineID string) *apierror.APIError {
	ctx, span := pickLineRepoTracer.Start(ctx, "repository.pick_line.delete_unpacked_for_order_line")
	defer span.End()

	// Delete the quantity rows first (they are joined through the pick lines we are about to remove), then the pick lines.
	if err := r.queries.DeleteQuantitiesByUnpackedPickLinesForLine(ctx, orderLineID); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	if err := r.queries.DeleteUnpackedPickLinesForLine(ctx, orderLineID); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *pickLineRepoImpl) HasUnpackedPickLineForOrderLine(ctx context.Context, orderLineID string) (bool, *apierror.APIError) {
	ctx, span := pickLineRepoTracer.Start(ctx, "repository.pick_line.has_unpacked_for_order_line")
	defer span.End()

	hasUnpacked, err := r.queries.HasUnpackedPickLineForOrderLine(ctx, orderLineID)
	if err != nil {
		return false, tracing.Trace(span, db.MapSQLError(err))
	}

	return hasUnpacked, nil
}

func (r *pickLineRepoImpl) UnpackByShipment(ctx context.Context, shipmentID string) *apierror.APIError {
	ctx, span := pickLineRepoTracer.Start(ctx, "repository.pick_line.unpack_by_shipment")
	defer span.End()

	err := r.queries.UnpackPickLinesByShipment(ctx, shipmentID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
