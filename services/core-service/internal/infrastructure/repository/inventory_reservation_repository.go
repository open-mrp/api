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
		allocatedSum, allocParseErr := decimal.NewFromString(decimalToString(issue.AllocatedSum))
		if allocParseErr != nil {
			return nil, apierror.NewInternalError(allocParseErr, "Invalid allocated sum for issue.")
		}

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
			if apiErr := r.allocateOpenIssue(ctx, issue.ID, take, params.AccountID, params.ItemID, issue.StorageLocationID, issue.LotID); apiErr != nil {
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

			if apiErr := r.allocateOpenIssue(ctx, newIssueID, take, params.AccountID, params.ItemID, issue.StorageLocationID, issue.LotID); apiErr != nil {
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

// allocatedSumsByReceipt returns how much each of the given receipts has already been drawn down.
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

// allocateOpenIssue performs FIFO allocation of an open issue against available receipts.
func (r *inventoryReservationRepo) allocateOpenIssue(ctx context.Context, issueID string, issueQty decimal.Decimal, accountID, itemID string, storageLocationID, lotID sql.NullString) *apierror.APIError {
	receipts, err := r.queries.FindReceiptsForAllocation(ctx, sqlc.FindReceiptsForAllocationParams{
		AccountID:         accountID,
		ItemID:            itemID,
		StorageLocationID: storageLocationID,
		LotID:             lotID,
	})
	if err != nil {
		return db.MapSQLError(err)
	}

	remaining := issueQty
	var exhaustedReceiptIDs []string

	// One aggregate for the whole candidate set. Asking per receipt put a round trip inside the walk
	// below, which an item whose older receipts are all full pays for on every allocation.
	allocatedByReceipt, apiErr := r.allocatedSumsByReceipt(ctx, receipts)
	if apiErr != nil {
		return apiErr
	}

	for _, receipt := range receipts {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}

		receiptQty, parseErr := decimal.NewFromString(receipt.QuantityValue)
		if parseErr != nil {
			return apierror.NewInternalError(parseErr, "Invalid receipt quantity value.")
		}

		available := receiptQty.Sub(allocatedByReceipt[receipt.ID])
		if available.LessThanOrEqual(decimal.Zero) {
			exhaustedReceiptIDs = append(exhaustedReceiptIDs, receipt.ID)
			continue
		}
		take := decimal.Min(available, remaining)
		// A receipt that is already over-allocated leaves available negative. IsZero let that through,
		// writing a negative allocation and growing `remaining` — so the issue drew more than it asked for.
		if take.LessThanOrEqual(decimal.Zero) {
			continue
		}

		// Create allocation record
		allocationID, genErr := id.GenID(id.InventoryAllocationIDPrefix, nil)
		if genErr != nil {
			return genErr
		}
		allocQtyID, genErr := id.GenID(id.QuantityIDPrefix, nil)
		if genErr != nil {
			return genErr
		}
		allocUnitCostID, genErr := id.GenID(id.RateIDPrefix, nil)
		if genErr != nil {
			return genErr
		}
		// A total cost is a quantity of currency, not a rate — the column points at `quantity`, and
		// the reversal paths delete it from there.
		allocTotalCostID, genErr := id.GenID(id.QuantityIDPrefix, nil)
		if genErr != nil {
			return genErr
		}

		// Insert quantity for the allocation
		if err := r.queries.InsertQuantityForInventory(ctx, sqlc.InsertQuantityForInventoryParams{
			ID:     allocQtyID,
			Value:  take.String(),
			UnitID: receipt.UnitID,
		}); err != nil {
			return db.MapSQLError(err)
		}

		// Carry the receipt's own cost onto the allocation so COGS reads the price actually paid.
		unitCost, parseErr := decimal.NewFromString(receipt.UnitCostValue)
		if parseErr != nil {
			return apierror.NewInternalError(parseErr, "Invalid receipt unit cost value.")
		}

		if err := r.queries.InsertRateForInventory(ctx, sqlc.InsertRateForInventoryParams{
			ID:                allocUnitCostID,
			Value:             receipt.UnitCostValue,
			NumeratorUnitID:   receipt.UnitCostNumeratorUnitID,
			DenominatorUnitID: receipt.UnitCostDenominatorUnitID,
		}); err != nil {
			return db.MapSQLError(err)
		}

		// Extending the rate over the allocated quantity cancels its denominator, leaving an amount
		// in the rate's numerator currency.
		if err := r.queries.InsertQuantityForInventory(ctx, sqlc.InsertQuantityForInventoryParams{
			ID:     allocTotalCostID,
			Value:  take.Mul(unitCost).String(),
			UnitID: receipt.UnitCostNumeratorUnitID,
		}); err != nil {
			return db.MapSQLError(err)
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
			return db.MapSQLError(err)
		}

		remaining = remaining.Sub(take)
	}

	if len(exhaustedReceiptIDs) > 0 {
		if err := r.queries.MarkInventoryReceiptsAllocated(ctx, exhaustedReceiptIDs); err != nil {
			return db.MapSQLError(err)
		}
	}

	if remaining.LessThanOrEqual(decimal.Zero) {
		if err := r.queries.CloseFullyAllocatedInventoryIssue(ctx, issueID); err != nil {
			return db.MapSQLError(err)
		}
	}

	return nil
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

		allocatedRaw, aErr := r.queries.GetAllocationSumForIssue(ctx, issue.ID)
		if apiErr := db.MapSQLError(aErr); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}

		var allocated decimal.Decimal
		switch v := allocatedRaw.(type) {
		case []byte:
			allocated, _ = decimal.NewFromString(string(v))
		case string:
			allocated, _ = decimal.NewFromString(v)
		default:
			allocated = decimal.Zero
		}

		issueRemaining := issueMeasure.Sub(allocated)
		if issueRemaining.LessThanOrEqual(decimal.Zero) {
			continue
		}

		if apiErr := r.allocateOpenIssue(ctx, issue.ID, issueRemaining, accountID, itemID, issue.StorageLocationID, issue.LotID); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}
