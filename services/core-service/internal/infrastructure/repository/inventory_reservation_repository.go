package repository

import (
	"context"
	"database/sql"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
)

var inventoryReservationRepoTracer = tracing.GetTracer("core-service.inventory_reservation_repository")

type inventoryReservationRepo struct {
	queries *sqlc.Queries
}

func NewInventoryReservationRepo(queries *sqlc.Queries) domain.InventoryReservationRepo {
	return &inventoryReservationRepo{queries: queries}
}

// CreateMaterialReservation creates a reserved inventory issue for a material demand linked to an order.
// firstValidBatchID prefers the batch that consumed the reservation, falling back to whatever tag the reservation already carried.
func firstValidBatchID(preferred, fallback sql.NullString) sql.NullString {
	if preferred.Valid {
		return preferred
	}
	return fallback
}

func (r *inventoryReservationRepo) CreateMaterialReservation(ctx context.Context, params domain.CreateMaterialReservationParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, inventoryReservationRepoTracer, "repository.inventory_reservation.create_material_reservation")
	defer span.End()

	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if err := r.queries.InsertQuantityForInventory(ctx, sqlc.InsertQuantityForInventoryParams{
		ID:     quantityID,
		Value:  params.Measure.String(),
		UnitID: params.UnitID,
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}

	issueID, apiErr := id.GenID(id.InventoryIssueIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if err := r.queries.InsertInventoryIssueForReservation(ctx, sqlc.InsertInventoryIssueForReservationParams{
		ID:         issueID,
		AccountID:  params.AccountID,
		ItemID:     params.ItemID,
		QuantityID: quantityID,
		StatusCode: "reserved",
		OrderID:    sql.NullString{String: params.OrderID, Valid: true},
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}

	return nil
}

// ReduceReservedForOrderItem reduces reserved quantity for an order item using FIFO order. It deletes or reduces reserved inventory issue records to release the specified measure.
func (r *inventoryReservationRepo) ReduceReservedForOrderItem(ctx context.Context, params domain.OrderReservationReductionParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, inventoryReservationRepoTracer, "repository.inventory_reservation.reduce_reserved_for_order_item")
	defer span.End()

	issues, err := r.queries.FindReservedIssuesByOrderItem(ctx, sqlc.FindReservedIssuesByOrderItemParams{
		OrderID:   sql.NullString{String: params.OrderID, Valid: true},
		AccountID: params.AccountID,
		ItemID:    params.ItemID,
	})
	if err != nil {
		return db.MapSQLError(err)
	}

	remaining := params.Measure

	for _, issue := range issues {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}

		issueQty, parseErr := decimal.NewFromString(issue.QuantityValue)
		if parseErr != nil {
			return apierror.NewInternalError(parseErr, "Invalid issue quantity value.")
		}

		take := decimal.Min(issueQty, remaining)
		if take.IsZero() {
			continue
		}

		if take.Equal(issueQty) {
			// Full deletion: remove the issue and its quantity record
			if err := r.queries.DeleteInventoryIssueByID(ctx, issue.ID); err != nil {
				return db.MapSQLError(err)
			}
			if err := r.queries.DeleteQuantityByID(ctx, issue.QuantityID); err != nil {
				return db.MapSQLError(err)
			}
		} else {
			// Partial reduction: decrease the quantity
			newValue := issueQty.Sub(take)
			if err := r.queries.UpdateQuantityValue(ctx, sqlc.UpdateQuantityValueParams{
				Value: newValue.String(),
				ID:    issue.QuantityID,
			}); err != nil {
				return db.MapSQLError(err)
			}
		}

		remaining = remaining.Sub(take)
	}

	return nil
}

// ReduceReservedForOrderMaterials reduces reserved quantities for multiple materials.
func (r *inventoryReservationRepo) ReduceReservedForOrderMaterials(ctx context.Context, orderID, accountID string, demands []domain.MaterialDemandItem) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, inventoryReservationRepoTracer, "repository.inventory_reservation.reduce_reserved_for_order_materials")
	defer span.End()

	for _, demand := range demands {
		if apiErr := r.ReduceReservedForOrderItem(ctx, domain.OrderReservationReductionParams{
			OrderID:   orderID,
			AccountID: accountID,
			ItemID:    demand.ItemID,
			Measure:   demand.Measure,
			UnitID:    demand.UnitID,
		}); apiErr != nil {
			return apiErr
		}
	}

	return nil
}

// AllocateReservationsForConsumption converts reserved inventory issues to open issues, performing FIFO allocation against receipts. Returns the remaining quantity that could not be allocated from reservations.
func (r *inventoryReservationRepo) AllocateReservationsForConsumption(ctx context.Context, params domain.ConsumptionAllocationParams) (*domain.ConsumptionAllocationResult, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, inventoryReservationRepoTracer, "repository.inventory_reservation.allocate_reservations_for_consumption")
	defer span.End()

	issues, err := r.queries.FindReservedIssuesWithAllocationSums(ctx, sqlc.FindReservedIssuesWithAllocationSumsParams{
		OrderID:   sql.NullString{String: params.OrderID, Valid: true},
		AccountID: params.AccountID,
		ItemID:    params.ItemID,
	})
	if err != nil {
		return nil, db.MapSQLError(err)
	}

	remainingToIssue := params.Measure

	// Tagging the issues with the batch that consumed them is what makes the consumption reversible:
	// deleting the batch looks its ledger rows up by this column.
	producedBatchID := sql.NullString{}
	if params.ProducedBatchID != "" {
		producedBatchID = sql.NullString{String: params.ProducedBatchID, Valid: true}
	}

	for _, issue := range issues {
		if remainingToIssue.LessThanOrEqual(decimal.Zero) {
			break
		}

		issueQty, parseErr := decimal.NewFromString(issue.QuantityValue)
		if parseErr != nil {
			return nil, apierror.NewInternalError(parseErr, "Invalid issue quantity value.")
		}

		// SUM() over a DECIMAL comes back as interface{}; decimalToString covers every shape the driver
		// may hand back. Reading it as []byte alone left allocatedSum at zero for any other type, which
		// reads as "nothing allocated yet" and allocates the issue a second time.
		allocatedRef, allocParseErr := decimal.NewFromString(decimalToString(issue.AllocatedSum))
		if allocParseErr != nil {
			return nil, apierror.NewInternalError(allocParseErr, "Invalid allocated sum for issue.")
		}

		// The sum came through each allocation's own ratio; the rest of this loop works in the issue's
		// unit, since that is what the split below writes back into the issue's quantity row.
		issueRatio, ratioErr := decimal.NewFromString(issue.UnitRatio)
		if ratioErr != nil {
			return nil, apierror.NewInternalError(ratioErr, "Invalid unit ratio on issue quantity.")
		}
		allocatedSum := convertMeasure(allocatedRef, decimal.NewFromInt(1), issueRatio)

		available := issueQty.Sub(allocatedSum)
		take := decimal.Min(available, remainingToIssue)
		if take.LessThanOrEqual(decimal.Zero) {
			continue
		}

		if take.Equal(available) {
			// Full consumption: change status to open and allocate
			if err := r.queries.UpdateInventoryIssueStatusToOpen(ctx, sqlc.UpdateInventoryIssueStatusToOpenParams{
				ID:      issue.ID,
				BatchID: producedBatchID,
			}); err != nil {
				return nil, db.MapSQLError(err)
			}
			// take, not issueQty: the portion already allocated has been drawn from receipts once
			// already, and re-issuing the whole quantity consumes it a second time.
			if _, apiErr := r.allocateOpenIssue(ctx, issue.ID, take, issueRatio, params.AccountID, params.ItemID,
				issue.StorageLocationID, issue.LotID); apiErr != nil {
				return nil, apiErr
			}
		} else {
			// Partial consumption: split the issue
			// Reduce original issue quantity
			newOrigQty := issueQty.Sub(take)
			if err := r.queries.UpdateQuantityValue(ctx, sqlc.UpdateQuantityValueParams{
				Value: newOrigQty.String(),
				ID:    issue.QuantityID,
			}); err != nil {
				return nil, db.MapSQLError(err)
			}

			// Create new open issue with the taken quantity
			newIssueID, genErr := id.GenID(id.InventoryIssueIDPrefix, nil)
			if genErr != nil {
				return nil, genErr
			}
			newQtyID, genErr := id.GenID(id.QuantityIDPrefix, nil)
			if genErr != nil {
				return nil, genErr
			}

			if err := r.queries.InsertQuantityForInventory(ctx, sqlc.InsertQuantityForInventoryParams{
				ID:     newQtyID,
				Value:  take.String(),
				UnitID: issue.UnitID,
			}); err != nil {
				return nil, db.MapSQLError(err)
			}

			if err := r.queries.InsertInventoryIssueForReservation(ctx, sqlc.InsertInventoryIssueForReservationParams{
				ID:                newIssueID,
				AccountID:         params.AccountID,
				ItemID:            params.ItemID,
				QuantityID:        newQtyID,
				StatusCode:        "open",
				OrderID:           sql.NullString{String: params.OrderID, Valid: true},
				BatchID:           firstValidBatchID(producedBatchID, issue.BatchID),
				StorageLocationID: issue.StorageLocationID,
				LotID:             issue.LotID,
			}); err != nil {
				return nil, db.MapSQLError(err)
			}

			if _, apiErr := r.allocateOpenIssue(ctx, newIssueID, take, issueRatio, params.AccountID, params.ItemID,
				issue.StorageLocationID, issue.LotID); apiErr != nil {
				return nil, apiErr
			}
		}

		remainingToIssue = remainingToIssue.Sub(take)
	}

	return &domain.ConsumptionAllocationResult{
		RemainingMeasure: remainingToIssue,
		RemainingUnitID:  params.UnitID,
	}, nil
}

// convertMeasure expresses a value given in `from` units in `to` units.
//
// Units with the same ratio are the same unit as far as arithmetic goes, and handing the value back
// untouched keeps that case exact. Ratios arrive rounded to thirty places, so multiplying by one and
// dividing by it again is not guaranteed to land where it started.
func convertMeasure(value, from, to decimal.Decimal) decimal.Decimal {
	if from.Equal(to) {
		return value
	}
	return value.Mul(from).Div(to)
}

// unitRatios looks up the given units' ratios, deduplicated.
//
// Every unit carries its ratio against the same reference for its dimension, so any two convert
// directly: `value * ratio_from / ratio_to`. A unit id read off a quantity row always resolves, so a
// missing one is a broken row rather than something to default past.
func (r *inventoryReservationRepo) unitRatios(ctx context.Context, unitIDs []string) (map[string]decimal.Decimal, *apierror.APIError) {
	wanted := make(map[string]struct{}, len(unitIDs))
	ids := make([]string, 0, len(unitIDs))
	for _, unitID := range unitIDs {
		if unitID == "" {
			continue
		}
		if _, seen := wanted[unitID]; seen {
			continue
		}
		wanted[unitID] = struct{}{}
		ids = append(ids, unitID)
	}

	ratios := make(map[string]decimal.Decimal, len(ids))
	if len(ids) == 0 {
		return ratios, nil
	}

	rows, err := r.queries.GetUnitRatios(ctx, ids)
	if err != nil {
		return nil, db.MapSQLError(err)
	}

	for _, row := range rows {
		ratio, parseErr := decimal.NewFromString(row.Ratio)
		if parseErr != nil {
			return nil, apierror.NewInternalError(parseErr, "Invalid unit ratio.")
		}
		// Zero would divide by zero below, and no unit is worth nothing of its own reference.
		if ratio.LessThanOrEqual(decimal.Zero) {
			ratio = decimal.NewFromInt(1)
		}
		ratios[row.ID] = ratio
	}

	for _, unitID := range ids {
		if _, ok := ratios[unitID]; !ok {
			return nil, apierror.NewInvariantViolationError("Unknown unit on an inventory quantity: " + unitID)
		}
	}

	return ratios, nil
}

// allocatedSumsByReceipt returns how much each of the given receipts has already been drawn down,
// each allocation taken through its own unit's ratio.
//
// A receipt with no allocations is absent from the result rather than zero; the zero value of the map
// is what the caller wants for it anyway. An unhandled driver type used to leave the sum at zero,
// which reads as "untouched" and let a receipt be drawn twice, so an unparseable value is an error
// here rather than a default.
func (r *inventoryReservationRepo) allocatedSumsByReceipt(ctx context.Context, receipts []sqlc.FindReceiptsForAllocationRow) (map[string]decimal.Decimal, *apierror.APIError) {
	sums := make(map[string]decimal.Decimal, len(receipts))
	if len(receipts) == 0 {
		return sums, nil
	}

	ids := make([]string, 0, len(receipts))
	for _, receipt := range receipts {
		ids = append(ids, receipt.ID)
	}

	rows, err := r.queries.GetAllocationSumsForReceipts(ctx, ids)
	if err != nil {
		return nil, db.MapSQLError(err)
	}

	for _, row := range rows {
		allocated, parseErr := decimal.NewFromString(decimalToString(row.TotalAllocated))
		if parseErr != nil {
			return nil, apierror.NewInternalError(parseErr, "Invalid allocated sum for receipt.")
		}
		sums[row.InventoryReceiptID] = allocated
	}

	return sums, nil
}

// allocateOpenIssue draws down receipts to cover an issue, returning how much it managed to cover.
//
// Demand and coverage are both in the issue's own unit, `demandRatio`. The receipts covering it are
// recorded in whatever unit their own source used, so each one is converted before it is compared.
// The reconcile page produces exactly that pair — a receipt in grams from setting a level, an issue
// in pounds from adjusting one — and comparing the raw values allocated 40 g against demand for
// 40 lbs and closed the issue as covered.
func (r *inventoryReservationRepo) allocateOpenIssue(ctx context.Context, issueID string, demand, demandRatio decimal.Decimal, accountID, itemID string, storageLocationID, lotID sql.NullString) (decimal.Decimal, *apierror.APIError) {
	receipts, err := r.queries.FindReceiptsForAllocation(ctx, sqlc.FindReceiptsForAllocationParams{
		AccountID:         accountID,
		ItemID:            itemID,
		StorageLocationID: storageLocationID,
		LotID:             lotID,
	})
	if err != nil {
		return decimal.Zero, db.MapSQLError(err)
	}
	if len(receipts) == 0 {
		return decimal.Zero, nil
	}

	unitIDs := make([]string, 0, len(receipts)*2)
	for _, receipt := range receipts {
		unitIDs = append(unitIDs, receipt.UnitID, receipt.UnitCostDenominatorUnitID)
	}
	ratios, apiErr := r.unitRatios(ctx, unitIDs)
	if apiErr != nil {
		return decimal.Zero, apiErr
	}

	remaining := demand
	var exhaustedReceiptIDs []string

	// One aggregate for the whole candidate set. Asking per receipt put a round trip inside the walk
	// below, which an item whose older receipts are all full pays for on every allocation.
	allocatedByReceipt, apiErr := r.allocatedSumsByReceipt(ctx, receipts)
	if apiErr != nil {
		return decimal.Zero, apiErr
	}

	for _, receipt := range receipts {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}

		receiptQty, parseErr := decimal.NewFromString(receipt.QuantityValue)
		if parseErr != nil {
			return decimal.Zero, apierror.NewInternalError(parseErr, "Invalid receipt quantity value.")
		}

		// What is left of the receipt is worked out in the receipt's own units, where it is exact; only
		// the comparison against the demand needs converting.
		receiptRatio := ratios[receipt.UnitID]
		receiptLeft := receiptQty.Sub(convertMeasure(allocatedByReceipt[receipt.ID], decimal.NewFromInt(1), receiptRatio))
		available := convertMeasure(receiptLeft, receiptRatio, demandRatio)
		take := decimal.Min(available, remaining)
		// A receipt that is already over-allocated leaves available negative. IsZero let that through,
		// writing a negative allocation and growing `remaining` — so the issue drew more than it asked for.
		if take.LessThanOrEqual(decimal.Zero) {
			// Nothing left to give; it should not be offered to the next issue either.
			exhaustedReceiptIDs = append(exhaustedReceiptIDs, receipt.ID)
			continue
		}

		// Create allocation record
		allocationID, genErr := id.GenID(id.InventoryAllocationIDPrefix, nil)
		if genErr != nil {
			return decimal.Zero, genErr
		}
		allocQtyID, genErr := id.GenID(id.QuantityIDPrefix, nil)
		if genErr != nil {
			return decimal.Zero, genErr
		}
		allocUnitCostID, genErr := id.GenID(id.RateIDPrefix, nil)
		if genErr != nil {
			return decimal.Zero, genErr
		}
		allocTotalCostID, genErr := id.GenID(id.QuantityIDPrefix, nil)
		if genErr != nil {
			return decimal.Zero, genErr
		}

		// Recorded against the receipt it draws on, in that receipt's unit. Converting the demand into
		// it can round, so a take that empties the receipt is written as the exact remainder instead —
		// a receipt left a hair short of covered is re-read and re-locked by every later pass.
		allocQty := convertMeasure(take, demandRatio, receiptRatio)
		if take.Equal(available) {
			allocQty = receiptLeft
		}
		if err := r.queries.InsertQuantityForInventory(ctx, sqlc.InsertQuantityForInventoryParams{
			ID:     allocQtyID,
			Value:  allocQty.String(),
			UnitID: receipt.UnitID,
		}); err != nil {
			return decimal.Zero, db.MapSQLError(err)
		}

		// The allocation is costed at what the stock it draws on was received at, copied off the
		// receipt. Writing zero here left the ledger saying every allocated unit cost nothing.
		unitCost, parseErr := decimal.NewFromString(receipt.UnitCostValue)
		if parseErr != nil {
			return decimal.Zero, apierror.NewInternalError(parseErr, "Invalid receipt unit cost value.")
		}
		if err := r.queries.InsertRateForInventory(ctx, sqlc.InsertRateForInventoryParams{
			ID:                allocUnitCostID,
			Value:             unitCost.String(),
			NumeratorUnitID:   receipt.UnitCostNumeratorUnitID,
			DenominatorUnitID: receipt.UnitCostDenominatorUnitID,
		}); err != nil {
			return decimal.Zero, db.MapSQLError(err)
		}

		// Total cost is a quantity of money — the rate's numerator — not a rate. It was being written
		// to `rate`, so the id on the allocation pointed at a row `quantity` does not have and every
		// reader joining it came back empty.
		//
		// A rate is priced per its denominator unit, so the quantity has to be expressed in that unit
		// before it is multiplied: $6/lb against 9,071 grams is $54,431 otherwise.
		costQty := convertMeasure(allocQty, receiptRatio, ratios[receipt.UnitCostDenominatorUnitID])
		if err := r.queries.InsertQuantityForInventory(ctx, sqlc.InsertQuantityForInventoryParams{
			ID:     allocTotalCostID,
			Value:  costQty.Mul(unitCost).String(),
			UnitID: receipt.UnitCostNumeratorUnitID,
		}); err != nil {
			return decimal.Zero, db.MapSQLError(err)
		}

		// Insert the allocation
		if err := r.queries.InsertInventoryAllocation(ctx, sqlc.InsertInventoryAllocationParams{
			ID:                 allocationID,
			InventoryReceiptID: receipt.ID,
			InventoryIssueID:   issueID,
			QuantityID:         allocQtyID,
			UnitCostID:         allocUnitCostID,
			TotalCostID:        allocTotalCostID,
		}); err != nil {
			return decimal.Zero, db.MapSQLError(err)
		}

		remaining = remaining.Sub(take)

		if take.Equal(available) {
			exhaustedReceiptIDs = append(exhaustedReceiptIDs, receipt.ID)
		}
	}

	// One statement for the whole walk rather than an UPDATE per receipt.
	if len(exhaustedReceiptIDs) > 0 {
		if err := r.queries.MarkInventoryReceiptsAllocated(ctx, exhaustedReceiptIDs); err != nil {
			return decimal.Zero, db.MapSQLError(err)
		}
	}

	return demand.Sub(remaining), nil
}

// AllocateOpenIssuesForItem performs FIFO allocation of all open inventory issues for the given item against available receipts.
func (r *inventoryReservationRepo) AllocateOpenIssuesForItem(ctx context.Context, accountID, itemID string) *apierror.APIError {
	ctx, span := inventoryReservationRepoTracer.Start(ctx, "repository.inventory_reservation.allocate_open_issues_for_item")
	defer span.End()

	issues, err := r.queries.FindOpenIssuesForItem(ctx, sqlc.FindOpenIssuesForItemParams{
		AccountID: accountID,
		ItemID:    itemID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	for _, issue := range issues {
		issueMeasure, pErr := decimal.NewFromString(issue.QuantityValue)
		if pErr != nil {
			return tracing.Trace(span, apierror.NewInternalError(pErr, "Failed to parse issue quantity."))
		}

		// The allocated sum below is taken through each allocation's own ratio; this is what reads it
		// back in the unit the issue was recorded in.
		issueRatio, rErr := decimal.NewFromString(issue.UnitRatio)
		if rErr != nil {
			return tracing.Trace(span, apierror.NewInternalError(rErr, "Invalid unit ratio on issue quantity."))
		}

		allocatedRaw, aErr := r.queries.GetAllocationSumForIssue(ctx, issue.ID)
		if apiErr := db.MapSQLError(aErr); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}

		// SUM() over a DECIMAL arrives as interface{} and the driver picks the shape. Reading only
		// []byte and string and falling back to zero for anything else — an int64 for a whole number,
		// a float64 — reads as "nothing allocated yet" for an issue that is already covered, and
		// allocates the whole of it a second time. That is where negative shortages come from: the
		// issue then has more allocated against it than it ever asked for.
		allocated, parseErr := decimal.NewFromString(decimalToString(allocatedRaw))
		if parseErr != nil {
			return tracing.Trace(span, apierror.NewInternalError(parseErr, "Invalid allocated sum for issue."))
		}

		issueRemaining := issueMeasure.Sub(convertMeasure(allocated, decimal.NewFromInt(1), issueRatio))
		if issueRemaining.LessThanOrEqual(decimal.Zero) {
			// Already covered, and left open by an allocation that predates the closing below.
			if err := r.queries.CloseFullyAllocatedInventoryIssue(ctx, issue.ID); err != nil {
				return tracing.Trace(span, db.MapSQLError(err))
			}
			continue
		}

		covered, apiErr := r.allocateOpenIssue(ctx, issue.ID, issueRemaining, issueRatio, accountID, itemID,
			issue.StorageLocationID, issue.LotID)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}

		// Demand that is now covered stops being demand. An issue left open reads as unfilled for the
		// rest of its life and is re-examined by every later allocation for the item.
		if covered.GreaterThanOrEqual(issueRemaining) {
			if err := r.queries.CloseFullyAllocatedInventoryIssue(ctx, issue.ID); err != nil {
				return tracing.Trace(span, db.MapSQLError(err))
			}
		}
	}

	return nil
}
