package repository

import (
	"context"
	gosql "database/sql"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
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

	// Determine where the new line sits. Regular product lines slot in just above
	// any credit/freight lines, which are kept at the bottom of the list so the line
	// order does not look odd. Credit/freight lines themselves, and any line on an
	// order that has none, append to the end.
	lineItemNumber, apiErr := r.resolveNewLineItemNumber(ctx, params.SalesOrderID, params.ProductID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Create quantity record
	err := r.queries.CreateOrderLineQuantity(ctx, sqlc.CreateOrderLineQuantityParams{
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

// resolveNewLineItemNumber computes the line_item_number for a line being added to an order. A credit/freight (shipping) line, or any line on an order that has none, appends to the end. A regular product line slots in at the first credit/freight line's position, pushing that block down by one so credit/freight stay at the bottom. Callers run this inside the create transaction so the shift and insert are atomic.
func (r *salesOrderLineRepoImpl) resolveNewLineItemNumber(ctx context.Context, salesOrderID, productID string) (int32, *apierror.APIError) {
	nextNumber, err := r.queries.GetNextLineItemNumber(ctx, salesOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, apiErr
	}

	// Credit/freight lines always append to the end.
	if productID != "" {
		isSystem, err := r.queries.IsSystemLineProduct(ctx, productID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return 0, apiErr
		}
		if isSystem {
			return nextNumber, nil
		}
	}

	firstSystem, err := r.queries.GetFirstSystemLineNumber(ctx, salesOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, apiErr
	}
	// No credit/freight lines on the order: append to the end.
	if firstSystem <= 0 {
		return nextNumber, nil
	}

	// Open a slot at the credit/freight block by pushing it (and anything after) down.
	firstSystemLine := safeconv.Int64ToInt32(firstSystem)
	if apiErr := db.MapSQLError(r.queries.ShiftSalesOrderLineNumbersAtOrAbove(ctx, sqlc.ShiftSalesOrderLineNumbersAtOrAboveParams{
		SalesOrderID:       salesOrderID,
		FromLineItemNumber: gosql.NullInt32{Int32: firstSystemLine, Valid: true},
	})); apiErr != nil {
		return 0, apiErr
	}
	return firstSystemLine, nil
}

func (r *salesOrderLineRepoImpl) Update(ctx context.Context, params domain.UpdateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.update")
	defer span.End()

	// Update basic sales order line fields
	err := r.queries.UpdateSalesOrderLine(ctx, sqlc.UpdateSalesOrderLineParams{
		ProductSku:         toNullString(params.ProductSKU),
		ProductDescription: field.StringToNullString(params.ProductDescription),
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

// syncFulfillmentQuantityRows updates the given fulfillment quantity rows by primary
// key: the unit is always relabeled to the order line's unit, and — when syncValue is
// set — the value follows too, but only for rows still mirroring the order line's
// pre-update value (partial snapshots keep the amount that actually moved). Rows that
// changed get their owning line's updated_at bumped via touch. Per-PK updates are
// deliberate: an UPDATE ... WHERE id IN (subquery) can't use the index on the
// production-sized quantity table and locks every row it scans, which stalled the
// whole update-line transaction until the request deadline.
func (r *salesOrderLineRepoImpl) syncFulfillmentQuantityRows(
	ctx context.Context,
	quantityIDs []string,
	previousQuantityValue, quantityValue, quantityUnitID string,
	syncValue bool,
	touch func(ctx context.Context, quantityID string) error,
) *apierror.APIError {
	for _, quantityID := range quantityIDs {
		changed, err := r.queries.UpdateQuantityUnitByID(ctx, sqlc.UpdateQuantityUnitByIDParams{
			UnitID: quantityUnitID,
			ID:     quantityID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}

		if syncValue {
			affected, err := r.queries.SyncQuantityValueByID(ctx, sqlc.SyncQuantityValueByIDParams{
				Value:         quantityValue,
				ID:            quantityID,
				PreviousValue: previousQuantityValue,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return apiErr
			}
			changed += affected
		}

		if changed == 0 {
			continue
		}
		if apiErr := db.MapSQLError(touch(ctx, quantityID)); apiErr != nil {
			return apiErr
		}
	}

	return nil
}

func (r *salesOrderLineRepoImpl) SyncInvoiceLineQuantities(ctx context.Context, salesOrderLineID, previousQuantityValue, quantityValue, quantityUnitID string) *apierror.APIError {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.sync_invoice_line_quantities")
	defer span.End()

	quantityIDs, err := r.queries.GetInvoiceLineQuantityIDsBySalesOrderLine(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if apiErr := r.syncFulfillmentQuantityRows(ctx, quantityIDs, previousQuantityValue, quantityValue, quantityUnitID, true, r.queries.TouchInvoiceLineByQuantityID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *salesOrderLineRepoImpl) SyncShipmentLineQuantities(ctx context.Context, salesOrderLineID, previousQuantityValue, quantityValue, quantityUnitID string) *apierror.APIError {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.sync_shipment_line_quantities")
	defer span.End()

	quantityIDs, err := r.queries.GetShipmentLineQuantityIDsBySalesOrderLine(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if apiErr := r.syncFulfillmentQuantityRows(ctx, quantityIDs, previousQuantityValue, quantityValue, quantityUnitID, true, r.queries.TouchShipmentLineByQuantityID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *salesOrderLineRepoImpl) SyncPickLineQuantityUnits(ctx context.Context, salesOrderLineID, quantityUnitID string) *apierror.APIError {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.sync_pick_line_quantity_units")
	defer span.End()

	quantityIDs, err := r.queries.GetPickLineQuantityIDsBySalesOrderLine(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Pick line quantities are picking progress (how much has been picked), not
	// mirrors of the ordered amount — value changes are handled by the pick
	// reconciliation in the service instead. Only the unit is relabeled here.
	if apiErr := r.syncFulfillmentQuantityRows(ctx, quantityIDs, "", "", quantityUnitID, false, r.queries.TouchPickLineByQuantityID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
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

func (r *salesOrderLineRepoImpl) HasShipmentAgainstOrderLine(ctx context.Context, salesOrderLineID string) (bool, *apierror.APIError) {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.has_shipment_against_order_line")
	defer span.End()

	hasShipment, err := r.queries.HasShipmentAgainstOrderLine(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return hasShipment, nil
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

func (r *salesOrderLineRepoImpl) GetLineOrder(ctx context.Context, salesOrderID string) ([]*domain.SalesOrderLinePosition, *apierror.APIError) {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.get_line_order")
	defer span.End()

	rows, err := r.queries.GetSalesOrderLineOrder(ctx, salesOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	positions := make([]*domain.SalesOrderLinePosition, len(rows))
	for i, row := range rows {
		positions[i] = &domain.SalesOrderLinePosition{
			ID:             row.ID,
			LineItemNumber: row.LineItemNumber.Int32,
			IsSystem: row.ProductTypeCode.Valid &&
				(row.ProductTypeCode.String == string(constants.ProductTypeCodeShipping) ||
					row.ProductTypeCode.String == string(constants.ProductTypeCodeCredit)),
		}
	}

	return positions, nil
}

func (r *salesOrderLineRepoImpl) SetLineItemNumber(ctx context.Context, salesOrderLineID string, lineItemNumber int32) *apierror.APIError {
	ctx, span := salesOrderLineRepoTracer.Start(ctx, "repository.sales_order_line.set_line_item_number")
	defer span.End()

	err := r.queries.SetSalesOrderLineItemNumber(ctx, sqlc.SetSalesOrderLineItemNumberParams{
		LineItemNumber: gosql.NullInt32{Int32: lineItemNumber, Valid: true},
		ID:             salesOrderLineID,
	})
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
	if row.ProductTypeCode.Valid {
		line.ProductTypeCode = &row.ProductTypeCode.String
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
	if row.ProductTypeCode.Valid {
		line.ProductTypeCode = &row.ProductTypeCode.String
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

	return line
}
