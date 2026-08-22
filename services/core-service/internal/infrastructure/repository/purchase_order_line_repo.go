package repository

import (
	"context"
	gosql "database/sql"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/tracing"
)

var purchaseOrderLineRepoTracer = tracing.GetTracer("core-service.purchase_order_line_repository")

type purchaseOrderLineRepoImpl struct {
	queries *sqlc.Queries
}

func NewPurchaseOrderLineRepo(queries *sqlc.Queries) domain.PurchaseOrderLineRepo {
	return &purchaseOrderLineRepoImpl{queries: queries}
}

func (r *purchaseOrderLineRepoImpl) Get(ctx context.Context, salesOrderLineID, salesOrderID string) (*domain.PurchaseOrderLine, *apierror.APIError) {
	ctx, span := purchaseOrderLineRepoTracer.Start(ctx, "repository.purchase_order_line.get")
	defer span.End()

	row, err := r.queries.GetPurchaseOrderLine(ctx, sqlc.GetPurchaseOrderLineParams{
		SalesOrderLineID: salesOrderLineID,
		SalesOrderID:     salesOrderID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapPurchaseOrderLineRow(row), nil
}

func (r *purchaseOrderLineRepoImpl) Create(ctx context.Context, lineID string, params domain.CreatePurchaseOrderLineParams) (*domain.PurchaseOrderLine, *apierror.APIError) {
	ctx, span := purchaseOrderLineRepoTracer.Start(ctx, "repository.purchase_order_line.create")
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
	lineItemNumber, err := r.queries.GetNextPurchaseOrderLineItemNumber(ctx, params.SalesOrderID)
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

	// Create the purchase order line
	err = r.queries.CreatePurchaseOrderLine(ctx, sqlc.CreatePurchaseOrderLineParams{
		ID:                 lineID,
		ProductSku:         params.ProductSKU,
		ProductDescription: toNullString(params.ProductDescription),
		EdiLineItemID:      gosql.NullString{},
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
	return r.Get(ctx, lineID, params.SalesOrderID)
}

func (r *purchaseOrderLineRepoImpl) Update(ctx context.Context, params domain.UpdatePurchaseOrderLineParams) (*domain.PurchaseOrderLine, *apierror.APIError) {
	ctx, span := purchaseOrderLineRepoTracer.Start(ctx, "repository.purchase_order_line.update")
	defer span.End()

	// Update basic purchase order line fields
	err := r.queries.UpdatePurchaseOrderLine(ctx, sqlc.UpdatePurchaseOrderLineParams{
		ProductSku:         toNullString(params.ProductSKU),
		ProductDescription: toNullString(params.ProductDescription),
		ProductID:          toNullString(params.ProductID),
		ItemID:             toNullString(params.ItemID),
		EdiLineItemID:      gosql.NullString{},
		ID:                 params.PurchaseOrderLineID,
		SalesOrderID:       params.SalesOrderID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// If quantity fields need updating, fetch the line to get the quantity ID
	if params.QuantityValue != nil || params.QuantityUnitID != nil {
		row, err := r.queries.GetPurchaseOrderLine(ctx, sqlc.GetPurchaseOrderLineParams{
			SalesOrderLineID: params.PurchaseOrderLineID,
			SalesOrderID:     params.SalesOrderID,
		})
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
		row, err := r.queries.GetPurchaseOrderLine(ctx, sqlc.GetPurchaseOrderLineParams{
			SalesOrderLineID: params.PurchaseOrderLineID,
			SalesOrderID:     params.SalesOrderID,
		})
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
		row, err := r.queries.GetPurchaseOrderLine(ctx, sqlc.GetPurchaseOrderLineParams{
			SalesOrderLineID: params.PurchaseOrderLineID,
			SalesOrderID:     params.SalesOrderID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		if row.UnitCostID.Valid {
			err = r.queries.UpdateOrderLineRateValue(ctx, sqlc.UpdateOrderLineRateValueParams{
				Value:             toNullString(params.UnitCostValue),
				NumeratorUnitID:   toNullString(params.UnitCostNumeratorUnitID),
				DenominatorUnitID: toNullString(params.UnitCostDenominatorUnitID),
				ID:                row.UnitCostID.String,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}

	// Re-fetch the updated line
	return r.Get(ctx, params.PurchaseOrderLineID, params.SalesOrderID)
}

func (r *purchaseOrderLineRepoImpl) Delete(ctx context.Context, salesOrderLineID, salesOrderID string) *apierror.APIError {
	ctx, span := purchaseOrderLineRepoTracer.Start(ctx, "repository.purchase_order_line.delete")
	defer span.End()

	err := r.queries.DeletePurchaseOrderLine(ctx, sqlc.DeletePurchaseOrderLineParams{
		ID:           salesOrderLineID,
		SalesOrderID: salesOrderID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *purchaseOrderLineRepoImpl) IsInOrder(ctx context.Context, salesOrderLineID, salesOrderID string) (bool, *apierror.APIError) {
	ctx, span := purchaseOrderLineRepoTracer.Start(ctx, "repository.purchase_order_line.is_in_order")
	defer span.End()

	exists, err := r.queries.IsLineInPurchaseOrder(ctx, sqlc.IsLineInPurchaseOrderParams{
		SalesOrderLineID: salesOrderLineID,
		SalesOrderID:     salesOrderID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exists, nil
}

func (r *purchaseOrderLineRepoImpl) GetNextLineItemNumber(ctx context.Context, salesOrderID string) (int32, *apierror.APIError) {
	ctx, span := purchaseOrderLineRepoTracer.Start(ctx, "repository.purchase_order_line.get_next_line_item_number")
	defer span.End()

	nextNumber, err := r.queries.GetNextPurchaseOrderLineItemNumber(ctx, salesOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return nextNumber, nil
}

func (r *purchaseOrderLineRepoImpl) DeleteCascade(ctx context.Context, salesOrderLineID string) *apierror.APIError {
	ctx, span := purchaseOrderLineRepoTracer.Start(ctx, "repository.purchase_order_line.delete_cascade")
	defer span.End()

	// Delete quantity records for receiving order lines associated with this purchase order line
	err := r.queries.DeleteQuantitiesByReceivingOrderLinesForLine(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Delete receiving order lines for this purchase order line
	err = r.queries.DeleteReceivingOrderLinesByPurchaseOrderLine(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Delete the purchase order line itself
	// Note: We use the sales_order_line table directly; the line ID is sufficient
	err = r.queries.DeleteSalesOrderLine(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *purchaseOrderLineRepoImpl) CreateQuantity(ctx context.Context, quantityID, value, unitID string) *apierror.APIError {
	ctx, span := purchaseOrderLineRepoTracer.Start(ctx, "repository.purchase_order_line.create_quantity")
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

func (r *purchaseOrderLineRepoImpl) CreateRate(ctx context.Context, rateID, value, numeratorUnitID, denominatorUnitID string) *apierror.APIError {
	ctx, span := purchaseOrderLineRepoTracer.Start(ctx, "repository.purchase_order_line.create_rate")
	defer span.End()

	err := r.queries.CreateOrderLineRate(ctx, sqlc.CreateOrderLineRateParams{
		ID:                rateID,
		Value:             value,
		NumeratorUnitID:   numeratorUnitID,
		DenominatorUnitID: denominatorUnitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *purchaseOrderLineRepoImpl) UpdateQuantityValue(ctx context.Context, quantityID, value string) *apierror.APIError {
	ctx, span := purchaseOrderLineRepoTracer.Start(ctx, "repository.purchase_order_line.update_quantity_value")
	defer span.End()

	err := r.queries.UpdateOrderLineQuantityValue(ctx, sqlc.UpdateOrderLineQuantityValueParams{
		Value: value,
		ID:    quantityID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *purchaseOrderLineRepoImpl) UpdateRateValue(ctx context.Context, rateID, value string, numeratorUnitID, denominatorUnitID *string) *apierror.APIError {
	ctx, span := purchaseOrderLineRepoTracer.Start(ctx, "repository.purchase_order_line.update_rate_value")
	defer span.End()

	err := r.queries.UpdateOrderLineRateValue(ctx, sqlc.UpdateOrderLineRateValueParams{
		Value:             gosql.NullString{String: value, Valid: true},
		NumeratorUnitID:   toNullString(numeratorUnitID),
		DenominatorUnitID: toNullString(denominatorUnitID),
		ID:                rateID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// Mapping helpers

func mapPurchaseOrderLineRow(row sqlc.GetPurchaseOrderLineRow) *domain.PurchaseOrderLine {
	line := &domain.PurchaseOrderLine{
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

	// Quantity received
	receivedVal := decimalToString(row.QuantityReceivedValue)
	line.QuantityReceivedValue = &receivedVal

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

	return line
}
