package repository

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/ledger"
	"github.com/open-mrp/api/shared/tracing"
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
			// Full consumption: change status to open and allocate.
			//
			// Guarded on `reserved` and checked for rows affected. The reservation read at the top of
			// this function may have been deleted by an order edit since, and the unguarded UPDATE this
			// replaces matched nothing and said nothing — so the walk below drew receipts down to cover
			// an issue that no longer exists. There is no foreign key to object afterwards.
			claimed, err := r.queries.ClaimReservedInventoryIssueAsOpen(ctx, sqlc.ClaimReservedInventoryIssueAsOpenParams{
				ID:      issue.ID,
				BatchID: producedBatchID,
			})
			if err != nil {
				return nil, db.MapSQLError(err)
			}
			affected, err := claimed.RowsAffected()
			if err != nil {
				return nil, db.MapSQLError(err)
			}
			if affected == 0 {
				// Deleted or already consumed in between. Nothing to cover.
				continue
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

// allocationMeasure is one allocation row as the ledger's arithmetic needs it: a value and the unit
// it was stamped in. Allocations against a single receipt can carry different units — the row records
// whatever unit the code that wrote it chose — so they are summed through their own ratios rather
// than added raw.
type allocationMeasure struct {
	unitID string
	value  string
}

// sumInBase totals allocation rows in base units, each through its own unit's ratio.
func (r *inventoryReservationRepo) sumInBase(ctx context.Context, rows []allocationMeasure) (decimal.Decimal, *apierror.APIError) {
	if len(rows) == 0 {
		return decimal.Zero, nil
	}

	unitIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		unitIDs = append(unitIDs, row.unitID)
	}
	ratios, apiErr := r.unitRatios(ctx, unitIDs)
	if apiErr != nil {
		return decimal.Zero, apiErr
	}

	total := decimal.Zero
	for _, row := range rows {
		value, parseErr := decimal.NewFromString(row.value)
		if parseErr != nil {
			return decimal.Zero, apierror.NewInternalError(parseErr, "Invalid allocation quantity value.")
		}
		total = total.Add(value.Mul(ratios[row.unitID]))
	}
	return total, nil
}

// drawnBaseForReceipt reports what a receipt has actually been drawn, right now, in base units.
//
// This is a current read, and that is the whole point of it. A locking read sees the latest committed
// version of every row it touches regardless of when this transaction's snapshot opened, so the
// number it returns is true even for a transaction that has been queued on a receipt lock while
// somebody else committed against it. The plain sum it replaces is what produced the 2026-08-26
// over-draws: a transaction woke from the receipt lock and computed what was left from a view frozen
// before the winner committed.
//
// It costs a round trip per receipt this issue actually draws on, typically one to three, and it
// replaces the single batched sum over the whole candidate set rather than adding to it.
func (r *inventoryReservationRepo) drawnBaseForReceipt(ctx context.Context, receiptID string) (decimal.Decimal, *apierror.APIError) {
	rows, err := r.queries.ReadReceiptAllocationsForUpdate(ctx, receiptID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return decimal.Zero, apiErr
	}

	measures := make([]allocationMeasure, 0, len(rows))
	for _, row := range rows {
		measures = append(measures, allocationMeasure{unitID: row.UnitID, value: row.Value})
	}
	return r.sumInBase(ctx, measures)
}

// coveredBaseForIssue reports what an issue has actually been covered by, right now, in base units.
// Same mechanism and same reason as drawnBaseForReceipt, on the demand side: this is the number that
// decides whether the issue closes.
func (r *inventoryReservationRepo) coveredBaseForIssue(ctx context.Context, issueID string) (decimal.Decimal, *apierror.APIError) {
	rows, err := r.queries.ReadIssueCoverageForUpdate(ctx, issueID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return decimal.Zero, apiErr
	}

	measures := make([]allocationMeasure, 0, len(rows))
	for _, row := range rows {
		measures = append(measures, allocationMeasure{unitID: row.UnitID, value: row.Value})
	}
	return r.sumInBase(ctx, measures)
}

// ledgerReceiptOverdrawnSkipped reports a receipt that was already drawn past its capacity before this
// transaction touched it. Somebody else's breach, so it is a page rather than a failure: see the
// delta rule in drawFromReceipts.
func ledgerReceiptOverdrawnSkipped(ctx context.Context, receiptID string, drawn, capacity decimal.Decimal) {
	slog.ErrorContext(ctx, "inventory receipt is already over-drawn; skipping it",
		"receipt_id", receiptID, "drawn_base", drawn.String(), "capacity_base", capacity.String())
}

// ledgerConcurrentOverdrawDetected reports a receipt that went over capacity between this
// transaction's own measurement and its own insert. Our draw fitted the free space we measured while
// holding the receipt's lock, so the writer that broke it held no lock — which today means the
// dashboard's Prisma allocator.
func ledgerConcurrentOverdrawDetected(ctx context.Context, receiptID string, before, after, capacity decimal.Decimal) {
	slog.ErrorContext(ctx, "inventory receipt was over-drawn by an unlocked concurrent writer",
		"receipt_id", receiptID, "drawn_before_base", before.String(), "drawn_after_base", after.String(),
		"capacity_base", capacity.String())
}

// allocateOpenIssue draws down receipts to cover an issue, returning how much it managed to cover.
//
// Demand and coverage are both in the issue's own unit, `demandRatio`. The receipts covering it are
// recorded in whatever unit their own source used, so each one is converted before it is compared.
// The reconcile page produces exactly that pair — a receipt in grams from setting a level, an issue
// in pounds from adjusting one — and comparing the raw values allocated 40 g against demand for
// 40 lbs and closed the issue as covered.
func (r *inventoryReservationRepo) allocateOpenIssue(ctx context.Context, issueID string, demand, demandRatio decimal.Decimal, accountID, itemID string, storageLocationID, lotID sql.NullString) (decimal.Decimal, *apierror.APIError) {
	receipts, apiErr := r.lockReceiptsForAllocation(ctx, accountID, itemID, storageLocationID, lotID)
	if apiErr != nil {
		return decimal.Zero, apiErr
	}
	if len(receipts) == 0 {
		return decimal.Zero, nil
	}

	ratios, apiErr := r.receiptUnitRatios(ctx, receipts)
	if apiErr != nil {
		return decimal.Zero, apiErr
	}
	return r.drawFromReceipts(ctx, issueID, demand, demandRatio, receipts, ratios)
}

// lockReceiptsForAllocation takes the item's candidate receipts under FOR UPDATE. Kept separate from
// the walk so callers can take this lock before any plain read of their own: see the statement
// ordering rule on drawFromReceipts.
func (r *inventoryReservationRepo) lockReceiptsForAllocation(ctx context.Context, accountID, itemID string, storageLocationID, lotID sql.NullString) ([]sqlc.FindReceiptsForAllocationRow, *apierror.APIError) {
	receipts, err := r.queries.FindReceiptsForAllocation(ctx, sqlc.FindReceiptsForAllocationParams{
		AccountID:         accountID,
		ItemID:            itemID,
		StorageLocationID: storageLocationID,
		LotID:             lotID,
	})
	if err != nil {
		return nil, db.MapSQLError(err)
	}
	return receipts, nil
}

// receiptUnitRatios resolves the ratios a draw needs: each receipt's own unit, and the denominator
// unit of its unit cost, which the total-cost conversion is expressed in.
func (r *inventoryReservationRepo) receiptUnitRatios(ctx context.Context, receipts []sqlc.FindReceiptsForAllocationRow, extra ...string) (map[string]decimal.Decimal, *apierror.APIError) {
	unitIDs := make([]string, 0, len(receipts)*2+len(extra))
	unitIDs = append(unitIDs, extra...)
	for _, receipt := range receipts {
		unitIDs = append(unitIDs, receipt.UnitID, receipt.UnitCostDenominatorUnitID)
	}
	return r.unitRatios(ctx, unitIDs)
}

// drawFromReceipts covers `demand` from already-locked receipts, returning how much it managed.
//
// STATEMENT ORDER IS LOAD-BEARING. Every locking read a draw depends on — the receipts here, the
// issue's own row and coverage at the caller — runs before the first plain read. InnoDB opens the
// REPEATABLE READ view on the first CONSISTENT read, and a locking read is not one, so the ratios and
// sums below are read from a view no older than the locks this transaction already holds.
//
// What a receipt has left is then read again currently, per receipt, rather than taken from that
// view at all. The two mechanisms answer different populations: ordering makes the arithmetic right
// against every writer that takes the same locks, and the current read makes it right against the one
// that does not — dashboard/apps/api's Prisma allocator, which writes these same rows on live request
// paths with no locking read anywhere.
func (r *inventoryReservationRepo) drawFromReceipts(ctx context.Context, issueID string, demand, demandRatio decimal.Decimal, receipts []sqlc.FindReceiptsForAllocationRow, ratios map[string]decimal.Decimal) (decimal.Decimal, *apierror.APIError) {
	remaining := demand
	var exhaustedReceiptIDs []string

	for _, receipt := range receipts {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}

		receiptQty, parseErr := decimal.NewFromString(receipt.QuantityValue)
		if parseErr != nil {
			return decimal.Zero, apierror.NewInternalError(parseErr, "Invalid receipt quantity value.")
		}

		receiptRatio := ratios[receipt.UnitID]
		capacity := convertMeasure(receiptQty, receiptRatio, decimal.NewFromInt(1))

		// Read currently, while this transaction already holds the receipt's X lock, so what is left is
		// what is actually committed rather than what this transaction's snapshot remembers.
		drawnBefore, apiErr := r.drawnBaseForReceipt(ctx, receipt.ID)
		if apiErr != nil {
			return decimal.Zero, apiErr
		}

		// A receipt already drawn past its capacity when we found it is somebody else's breach — the
		// dashboard's, or one of the rows the 2026-08-26 over-draws left behind. Skipping and alarming
		// is deliberate: failing here would charge this transaction for a row it did not write, and the
		// issue behind it would then fail on every pass forever, which is a permanent inbox failure for
		// a corruption somebody else committed.
		free := capacity.Sub(drawnBefore)
		if free.LessThanOrEqual(ledger.Epsilon) {
			ledgerReceiptOverdrawnSkipped(ctx, receipt.ID, drawnBefore, capacity)
			// Nothing left to give; it should not be offered to the next issue either.
			exhaustedReceiptIDs = append(exhaustedReceiptIDs, receipt.ID)
			continue
		}

		// What is left of the receipt is worked out in the receipt's own units, where it is exact; only
		// the comparison against the demand needs converting.
		receiptLeft := convertMeasure(free, decimal.NewFromInt(1), receiptRatio)
		available := convertMeasure(receiptLeft, receiptRatio, demandRatio)
		take := decimal.Min(available, remaining)
		if take.LessThanOrEqual(decimal.Zero) {
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

		// Our own arithmetic, checked against the free space we measured while holding the lock. This
		// is the half of the rule that is ours to fail on: `free` came from a current read, so a draw
		// that exceeds it is this transaction's bug and nothing else's, and the transaction dies.
		ourBase := convertMeasure(allocQty, receiptRatio, decimal.NewFromInt(1))
		if drawnBefore.Add(ourBase).Sub(capacity).GreaterThan(ledger.Epsilon) {
			return decimal.Zero, apierror.NewInvariantViolationError(
				"Allocation would draw receipt " + receipt.ID + " to " + drawnBefore.Add(ourBase).String() +
					" against a capacity of " + capacity.String() + ".")
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

		// Read again, currently, now that our own row is committed to the transaction. Our draw fitted
		// the free space we measured while holding this receipt's X lock, so anything that pushed it
		// over in between was written by somebody holding no lock at all — which today means the
		// dashboard. Alarm, never abort: our arithmetic was right and rolling it back would not undo
		// theirs.
		drawnAfter, apiErr := r.drawnBaseForReceipt(ctx, receipt.ID)
		if apiErr != nil {
			return decimal.Zero, apiErr
		}
		if drawnAfter.Sub(capacity).GreaterThan(ledger.Epsilon) {
			ledgerConcurrentOverdrawDetected(ctx, receipt.ID, drawnBefore, drawnAfter, capacity)
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

	// The ratios are resolved per issue, after that issue's receipt lock, rather than for the whole
	// set here. Resolving them here is a plain read, and a plain read before the locking ones opens
	// this transaction's view early — which is the defect, not an optimisation.
	for _, issue := range issues {
		if apiErr := r.allocateOneOpenIssue(ctx, accountID, itemID, issue.ID, issue.QuantityValue, issue.UnitID,
			issue.StorageLocationID, issue.LotID); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

var openIssueCursorEpoch = time.Unix(0, 0).UTC()

func (r *inventoryReservationRepo) AllocateOpenIssuesForItemPage(ctx context.Context, accountID, itemID string, afterCreatedAt time.Time, afterID string, limit int32) (time.Time, string, int, *apierror.APIError) {
	ctx, span := inventoryReservationRepoTracer.Start(ctx, "repository.inventory_reservation.allocate_open_issues_for_item_page")
	defer span.End()

	cursorCreatedAt := afterCreatedAt
	if afterID == "" {
		cursorCreatedAt = openIssueCursorEpoch
	}

	issues, err := r.queries.FindOpenIssuesForItemPaged(ctx, sqlc.FindOpenIssuesForItemPagedParams{
		AccountID:       accountID,
		ItemID:          itemID,
		CursorCreatedAt: cursorCreatedAt,
		CursorID:        afterID,
		Limit:           limit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return time.Time{}, "", 0, tracing.Trace(span, apiErr)
	}

	// See AllocateOpenIssuesForItem: the unit ratios are resolved per issue, after its locks.
	lastCreatedAt := afterCreatedAt
	lastID := afterID
	for _, issue := range issues {
		if apiErr := r.allocateOneOpenIssue(ctx, accountID, itemID, issue.ID, issue.QuantityValue, issue.UnitID,
			issue.StorageLocationID, issue.LotID); apiErr != nil {
			return time.Time{}, "", 0, tracing.Trace(span, apiErr)
		}
		lastCreatedAt = issue.CreatedAt
		lastID = issue.ID
	}

	return lastCreatedAt, lastID, len(issues), nil
}

// allocateOneOpenIssue covers one open issue, whatever is left of it, from the item's receipts.
//
// STATEMENT ORDER IS LOAD-BEARING, and this is where the ordering rule is applied:
//
//	1 FindReceiptsForAllocation FOR UPDATE   (locking read — no read view yet)
//	2 ReadIssueCoverageForUpdate FOR UPDATE  (locking read — still no read view)
//	3 GetUnitRatios                          (plain — THE VIEW OPENS HERE)
//
// InnoDB opens the REPEATABLE READ view on the first CONSISTENT read, and a locking read is not one.
// So by the time anything is read for arithmetic, this transaction holds every candidate receipt and
// every allocation already against the issue, and its view is strictly newer than every conforming
// writer that released them.
//
// That is the guarantee 7044443a's comment claimed and never delivered. Its locking read was on the
// open-issue page, but unitRatios ran next and opened the view before any receipt lock was held, so a
// transaction that queued on a receipt lock still computed what was left from a pre-lock snapshot.
// The unit ratios are resolved here, after the locks, rather than for the whole page by the caller,
// which is what that plain read was.
func (r *inventoryReservationRepo) allocateOneOpenIssue(ctx context.Context, accountID, itemID, issueID, quantityValue, issueUnitID string, storageLocationID, lotID sql.NullString) *apierror.APIError {
	issueMeasure, pErr := decimal.NewFromString(quantityValue)
	if pErr != nil {
		return apierror.NewInternalError(pErr, "Failed to parse issue quantity.")
	}

	receipts, apiErr := r.lockReceiptsForAllocation(ctx, accountID, itemID, storageLocationID, lotID)
	if apiErr != nil {
		return apiErr
	}

	// A current read, and it is what decides the close. The plain sum it replaces read from a view
	// frozen before the receipt locks above were held, so an issue already covered by a sibling
	// transaction still looked untouched and was allocated in full a second time.
	coveredBase, apiErr := r.coveredBaseForIssue(ctx, issueID)
	if apiErr != nil {
		return apiErr
	}

	ratios, apiErr := r.receiptUnitRatios(ctx, receipts, issueUnitID)
	if apiErr != nil {
		return apiErr
	}
	issueRatio := ratios[issueUnitID]
	if issueRatio.LessThanOrEqual(decimal.Zero) {
		return apierror.NewInvariantViolationError("Missing unit ratio for the unit an inventory issue was recorded in.")
	}

	issueRemaining := issueMeasure.Sub(convertMeasure(coveredBase, decimal.NewFromInt(1), issueRatio))
	if issueRemaining.LessThanOrEqual(decimal.Zero) {
		// Already covered, and left open by an allocation that predates the closing below.
		if err := r.queries.CloseFullyAllocatedInventoryIssue(ctx, issueID); err != nil {
			return db.MapSQLError(err)
		}
		return nil
	}
	if len(receipts) == 0 {
		return nil
	}

	covered, apiErr := r.drawFromReceipts(ctx, issueID, issueRemaining, issueRatio, receipts, ratios)
	if apiErr != nil {
		return apiErr
	}

	// Demand that is now covered stops being demand. An issue left open reads as unfilled for the
	// rest of its life and is re-examined by every later allocation for the item.
	if covered.GreaterThanOrEqual(issueRemaining) {
		if err := r.queries.CloseFullyAllocatedInventoryIssue(ctx, issueID); err != nil {
			return db.MapSQLError(err)
		}
	}

	return nil
}
