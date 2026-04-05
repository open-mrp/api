package repository

import (
	"context"
	gosql "database/sql"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
)

var salesOrderLineRepoTracer = tracing.GetTracer("core-service.sales_order_line_repository")

type salesOrderLineRepoImpl struct {
	queries *sqlc.Queries
}

func NewSalesOrderLineRepo(queries *sqlc.Queries) domain.SalesOrderLineRepo {
	return &salesOrderLineRepoImpl{queries: queries}
}

func (r *salesOrderLineRepoImpl) List(ctx context.Context, salesOrderID string) ([]*domain.SalesOrderLine, *apierror.APIError) {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.list")
	defer span.End()

	rows, err := r.queries.GetSalesOrderLines(ctx, salesOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lines := make([]*domain.SalesOrderLine, len(rows))
	for i, row := range rows {
		lines[i] = mapSalesOrderLinesRow(row)
	}

	return lines, nil
}

func (r *salesOrderLineRepoImpl) Get(ctx context.Context, salesOrderLineID string) (*domain.SalesOrderLine, *apierror.APIError) {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.get")
	defer span.End()

	row, err := r.queries.GetSalesOrderLine(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapSalesOrderLineRow(row), nil
}

func (r *salesOrderLineRepoImpl) Create(ctx context.Context, lineID string, params domain.CreateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.create")
	defer span.End()

	// Generate quantity ID
	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Generate unit price rate ID
	unitPriceID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Optionally generate unit cost rate ID
	var unitCostID gosql.NullString
	if params.UnitCostValue != nil {
		costID, apiErr := id.GenID(id.RateIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		unitCostID = gosql.NullString{String: costID, Valid: true}
	}

	// Get next line item number
	lineItemNumber, err := r.queries.GetNextLineItemNumber(ctx, params.SalesOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Create quantity record
	err = r.queries.CreateOrderLineQuantity(ctx, sqlc.CreateOrderLineQuantityParams{
		ID:     quantityID,
		Value:  params.QuantityValue,
		UnitID: params.QuantityUnitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Create unit price rate record
	err = r.queries.CreateOrderLineRate(ctx, sqlc.CreateOrderLineRateParams{
		ID:                unitPriceID,
		Value:             params.UnitPriceValue,
		NumeratorUnitID:   params.UnitPriceNumeratorUnitID,
		DenominatorUnitID: params.UnitPriceDenominatorUnitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Create unit cost rate record if provided
	if unitCostID.Valid {
		err = r.queries.CreateOrderLineRate(ctx, sqlc.CreateOrderLineRateParams{
			ID:                unitCostID.String,
			Value:             *params.UnitCostValue,
			NumeratorUnitID:   *params.UnitCostNumeratorUnitID,
			DenominatorUnitID: *params.UnitCostDenominatorUnitID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Create the sales order line
	err = r.queries.CreateSalesOrderLine(ctx, sqlc.CreateSalesOrderLineParams{
		ID:                 lineID,
		ProductSku:         params.ProductSKU,
		ProductDescription: toNullString(params.ProductDescription),
		EdiLineItemID:      toNullString(params.EdiLineItemID),
		LineItemNumber:     gosql.NullInt32{Int32: lineItemNumber, Valid: true},
		ProductID:          gosql.NullString{String: params.ProductID, Valid: params.ProductID != ""},
		ItemID:             toNullString(params.ItemID),
		SalesOrderID:       params.SalesOrderID,
		QuantityID:         quantityID,
		UnitPriceID:        unitPriceID,
		UnitCostID:         unitCostID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Re-fetch the created line
	return r.Get(ctx, lineID)
}

func (r *salesOrderLineRepoImpl) Update(ctx context.Context, params domain.UpdateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.update")
	defer span.End()

	// Update basic sales order line fields
	err := r.queries.UpdateSalesOrderLine(ctx, sqlc.UpdateSalesOrderLineParams{
		ProductSku:         toNullString(params.ProductSKU),
		ProductDescription: toNullString(params.ProductDescription),
		ProductID:          toNullString(params.ProductID),
		ItemID:             toNullString(params.ItemID),
		EdiLineItemID:      toNullString(params.EdiLineItemID),
		ID:                 params.SalesOrderLineID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// If quantity fields need updating, fetch the line to get the quantity ID
	if params.QuantityValue != nil || params.QuantityUnitID != nil {
		row, err := r.queries.GetSalesOrderLine(ctx, params.SalesOrderLineID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		updateParams := sqlc.UpdateOrderLineQuantityValueParams{
			ID: row.QuantityID,
		}
		if params.QuantityValue != nil {
			updateParams.Value = *params.QuantityValue
		} else {
			updateParams.Value = row.QuantityValue
		}
		if params.QuantityUnitID != nil {
			updateParams.UnitID = gosql.NullString{String: *params.QuantityUnitID, Valid: true}
		}

		err = r.queries.UpdateOrderLineQuantityValue(ctx, updateParams)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// If unit price fields need updating
	if params.UnitPriceValue != nil || params.UnitPriceNumeratorUnitID != nil || params.UnitPriceDenominatorUnitID != nil {
		row, err := r.queries.GetSalesOrderLine(ctx, params.SalesOrderLineID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		err = r.queries.UpdateOrderLineRateValue(ctx, sqlc.UpdateOrderLineRateValueParams{
			Value:             toNullString(params.UnitPriceValue),
			NumeratorUnitID:   toNullString(params.UnitPriceNumeratorUnitID),
			DenominatorUnitID: toNullString(params.UnitPriceDenominatorUnitID),
			ID:                row.UnitPriceID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// If unit cost fields need updating
	if params.UnitCostValue != nil || params.UnitCostNumeratorUnitID != nil || params.UnitCostDenominatorUnitID != nil {
		row, err := r.queries.GetSalesOrderLine(ctx, params.SalesOrderLineID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		if row.UnitCostID.Valid {
			// Update existing unit cost rate
			err = r.queries.UpdateOrderLineRateValue(ctx, sqlc.UpdateOrderLineRateValueParams{
				Value:             toNullString(params.UnitCostValue),
				NumeratorUnitID:   toNullString(params.UnitCostNumeratorUnitID),
				DenominatorUnitID: toNullString(params.UnitCostDenominatorUnitID),
				ID:                row.UnitCostID.String,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		} else if params.UnitCostValue != nil && params.UnitCostNumeratorUnitID != nil && params.UnitCostDenominatorUnitID != nil {
			// Create a new unit cost rate when one doesn't exist (matches Dashboard behavior)
			costID, apiErr := id.GenID(id.RateIDPrefix, nil)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}

			err = r.queries.CreateOrderLineRate(ctx, sqlc.CreateOrderLineRateParams{
				ID:                costID,
				Value:             *params.UnitCostValue,
				NumeratorUnitID:   *params.UnitCostNumeratorUnitID,
				DenominatorUnitID: *params.UnitCostDenominatorUnitID,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}

			err = r.queries.SetSalesOrderLineUnitCost(ctx, sqlc.SetSalesOrderLineUnitCostParams{
				UnitCostID: gosql.NullString{String: costID, Valid: true},
				ID:         params.SalesOrderLineID,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}

	// Re-fetch the updated line
	return r.Get(ctx, params.SalesOrderLineID)
}

func (r *salesOrderLineRepoImpl) Delete(ctx context.Context, salesOrderLineID string) *apierror.APIError {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.delete")
	defer span.End()

	err := r.queries.DeleteSalesOrderLine(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *salesOrderLineRepoImpl) IsInOrder(ctx context.Context, salesOrderLineID, salesOrderID, accountID string) (bool, *apierror.APIError) {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.is_in_order")
	defer span.End()

	exists, err := r.queries.IsLineInOrder(ctx, sqlc.IsLineInOrderParams{
		SalesOrderLineID: salesOrderLineID,
		SalesOrderID:     salesOrderID,
		AccountID:        accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exists, nil
}

func (r *salesOrderLineRepoImpl) GetNextLineItemNumber(ctx context.Context, salesOrderID string) (int32, *apierror.APIError) {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.get_next_line_item_number")
	defer span.End()

	nextNumber, err := r.queries.GetNextLineItemNumber(ctx, salesOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return nextNumber, nil
}

func (r *salesOrderLineRepoImpl) DeleteCascade(ctx context.Context, salesOrderLineID string) *apierror.APIError {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.delete_cascade")
	defer span.End()

	// Delete quantity records for pick lines associated with this sales order line
	err := r.queries.DeleteQuantitiesByPickLinesForLine(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Delete pick lines
	err = r.queries.DeletePickLinesBySalesOrderLine(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Delete shipment lines
	err = r.queries.DeleteShipmentLinesBySalesOrderLine(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Delete invoice lines
	err = r.queries.DeleteInvoiceLinesBySalesOrderLine(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Delete the sales order line itself
	err = r.queries.DeleteSalesOrderLine(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *salesOrderLineRepoImpl) CreateQuantity(ctx context.Context, quantityID, value, unitID string) *apierror.APIError {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.create_quantity")
	defer span.End()

	err := r.queries.CreateOrderLineQuantity(ctx, sqlc.CreateOrderLineQuantityParams{
		ID:     quantityID,
		Value:  value,
		UnitID: unitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// mapSalesOrderLineRow maps a GetSalesOrderLineRow to a domain.SalesOrderLine.
func mapSalesOrderLineRow(row sqlc.GetSalesOrderLineRow) *domain.SalesOrderLine {
	line := &domain.SalesOrderLine{
		ID:                           row.ID,
		ProductSKU:                   row.ProductSku,
		QuantityID:                   row.QuantityID,
		QuantityValue:                row.QuantityValue,
		QuantityUnitID:               row.QuantityUnitID,
		QuantityUnitName:             row.QuantityUnitName,
		QuantityUnitAbbreviation:     row.QuantityUnitAbbreviation,
		QuantityUnitType:             row.QuantityUnitType,
		UnitPriceID:                  row.UnitPriceID,
		UnitPriceValue:               row.UnitPriceValue,
		UnitPriceNumeratorUnitID:     row.UnitPriceNumeratorUnitID,
		UnitPriceNumeratorUnitAbbr:   row.UnitPriceNumeratorUnitAbbreviation,
		UnitPriceDenominatorUnitID:   row.UnitPriceDenominatorUnitID,
		UnitPriceDenominatorUnitAbbr: row.UnitPriceDenominatorUnitAbbreviation,
		CreatedAt:                    row.CreatedAt,
		UpdatedAt:                    row.UpdatedAt,
	}

	if row.LineItemNumber.Valid {
		line.LineItemNumber = row.LineItemNumber.Int32
	}
	if row.ProductDescription.Valid {
		line.ProductDescription = &row.ProductDescription.String
	}
	if row.ProductID.Valid {
		line.ProductID = &row.ProductID.String
	}
	if row.ItemID.Valid {
		line.ItemID = &row.ItemID.String
	}
	if row.ItemSku.Valid {
		line.ItemSKU = &row.ItemSku.String
	}
	if row.EdiLineItemID.Valid {
		line.EdiLineItemID = &row.EdiLineItemID.String
	}

	// Aggregated quantity values
	pickedVal := decimalToString(row.QuantityPickedValue)
	line.QuantityPickedValue = &pickedVal

	packedVal := decimalToString(row.QuantityPackedValue)
	line.QuantityPackedValue = &packedVal

	invoicedVal := decimalToString(row.QuantityInvoicedValue)
	line.QuantityInvoicedValue = &invoicedVal

	// Unit cost (nullable)
	if row.UnitCostID.Valid {
		line.UnitCostID = &row.UnitCostID.String
	}
	if row.UnitCostValue.Valid {
		line.UnitCostValue = &row.UnitCostValue.String
	}
	if row.UnitCostNumeratorUnitID.Valid {
		line.UnitCostNumeratorUnitID = &row.UnitCostNumeratorUnitID.String
	}
	if row.UnitCostNumeratorUnitAbbreviation.Valid {
		line.UnitCostNumeratorUnitAbbr = &row.UnitCostNumeratorUnitAbbreviation.String
	}
	if row.UnitCostDenominatorUnitID.Valid {
		line.UnitCostDenominatorUnitID = &row.UnitCostDenominatorUnitID.String
	}
	if row.UnitCostDenominatorUnitAbbreviation.Valid {
		line.UnitCostDenominatorUnitAbbr = &row.UnitCostDenominatorUnitAbbreviation.String
	}

	if row.CompletedAt.Valid {
		line.CompletedAt = &row.CompletedAt.Time
	}

	return line
}

// mapSalesOrderLinesRow maps a GetSalesOrderLinesRow to a domain.SalesOrderLine.
func mapSalesOrderLinesRow(row sqlc.GetSalesOrderLinesRow) *domain.SalesOrderLine {
	line := &domain.SalesOrderLine{
		ID:                           row.ID,
		ProductSKU:                   row.ProductSku,
		QuantityID:                   row.QuantityID,
		QuantityValue:                row.QuantityValue,
		QuantityUnitID:               row.QuantityUnitID,
		QuantityUnitName:             row.QuantityUnitName,
		QuantityUnitAbbreviation:     row.QuantityUnitAbbreviation,
		QuantityUnitType:             row.QuantityUnitType,
		UnitPriceID:                  row.UnitPriceID,
		UnitPriceValue:               row.UnitPriceValue,
		UnitPriceNumeratorUnitID:     row.UnitPriceNumeratorUnitID,
		UnitPriceNumeratorUnitAbbr:   row.UnitPriceNumeratorUnitAbbreviation,
		UnitPriceDenominatorUnitID:   row.UnitPriceDenominatorUnitID,
		UnitPriceDenominatorUnitAbbr: row.UnitPriceDenominatorUnitAbbreviation,
		CreatedAt:                    row.CreatedAt,
		UpdatedAt:                    row.UpdatedAt,
	}

	if row.LineItemNumber.Valid {
		line.LineItemNumber = row.LineItemNumber.Int32
	}
	if row.ProductDescription.Valid {
		line.ProductDescription = &row.ProductDescription.String
	}
	if row.ProductID.Valid {
		line.ProductID = &row.ProductID.String
	}
	if row.ItemID.Valid {
		line.ItemID = &row.ItemID.String
	}
	if row.ItemSku.Valid {
		line.ItemSKU = &row.ItemSku.String
	}
	if row.EdiLineItemID.Valid {
		line.EdiLineItemID = &row.EdiLineItemID.String
	}

	// Aggregated quantity values
	pickedVal := decimalToString(row.QuantityPickedValue)
	line.QuantityPickedValue = &pickedVal

	packedVal := decimalToString(row.QuantityPackedValue)
	line.QuantityPackedValue = &packedVal

	invoicedVal := decimalToString(row.QuantityInvoicedValue)
	line.QuantityInvoicedValue = &invoicedVal

	// Unit cost (nullable)
	if row.UnitCostID.Valid {
		line.UnitCostID = &row.UnitCostID.String
	}
	if row.UnitCostValue.Valid {
		line.UnitCostValue = &row.UnitCostValue.String
	}
	if row.UnitCostNumeratorUnitID.Valid {
		line.UnitCostNumeratorUnitID = &row.UnitCostNumeratorUnitID.String
	}
	if row.UnitCostNumeratorUnitAbbreviation.Valid {
		line.UnitCostNumeratorUnitAbbr = &row.UnitCostNumeratorUnitAbbreviation.String
	}
	if row.UnitCostDenominatorUnitID.Valid {
		line.UnitCostDenominatorUnitID = &row.UnitCostDenominatorUnitID.String
	}
	if row.UnitCostDenominatorUnitAbbreviation.Valid {
		line.UnitCostDenominatorUnitAbbr = &row.UnitCostDenominatorUnitAbbreviation.String
	}

	if row.CompletedAt.Valid {
		line.CompletedAt = &row.CompletedAt.Time
	}

	return line
}
