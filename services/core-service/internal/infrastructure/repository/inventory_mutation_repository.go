package repository

import (
	"context"
	"database/sql"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/tracing"
)

var inventoryMutationRepoTracer = tracing.GetTracer("core-service.inventory_mutation_repository")

type inventoryMutationRepo struct {
	queries *sqlc.Queries
}

func NewInventoryMutationRepo(queries *sqlc.Queries) domain.InventoryMutationRepo {
	return &inventoryMutationRepo{queries: queries}
}

func (r *inventoryMutationRepo) UpdateInventory(ctx context.Context, params domain.InventoryUpdateParams) *apierror.APIError {
	if params.Measure.IsZero() {
		return nil
	}

	// Create a quantity record for the inventory change.
	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	absMeasure := params.Measure.Abs()

	if err := r.queries.InsertQuantityForInventory(ctx, sqlc.InsertQuantityForInventoryParams{
		ID:     quantityID,
		Value:  absMeasure.String(),
		UnitID: params.UnitID,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to insert quantity for inventory.")
	}

	if params.Measure.GreaterThan(decimal.Zero) {
		// Positive measure = receipt (inventory increase).
		receiptID, apiErr := id.GenID(id.InventoryReceiptIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}

		// The receipt carries a copy of the item's unit cost as it stood when the stock landed, which
		// is what inventory is later valued at — a receipt written at zero values everything a scan
		// produced at nothing, and the weighted average it feeds with it.
		costRow, err := r.queries.GetItemUnitCost(ctx, sqlc.GetItemUnitCostParams{
			ItemID:    params.ItemID,
			AccountID: params.AccountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}

		rateID, apiErr := id.GenID(id.RateIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		if err := r.queries.InsertRateForInventory(ctx, sqlc.InsertRateForInventoryParams{
			ID:                rateID,
			Value:             costRow.Value,
			NumeratorUnitID:   costRow.NumeratorUnitID,
			DenominatorUnitID: costRow.DenominatorUnitID,
		}); err != nil {
			return apierror.NewInternalError(err, "Failed to insert unit cost for inventory receipt.")
		}

		batchID := sql.NullString{}
		if params.BatchID != nil {
			batchID = sql.NullString{String: *params.BatchID, Valid: true}
		}

		if err := r.queries.InsertInventoryReceipt(ctx, sqlc.InsertInventoryReceiptParams{
			ID:              receiptID,
			OwnerAccountID:  params.AccountID,
			HolderAccountID: params.AccountID,
			ItemID:          params.ItemID,
			QuantityID:      quantityID,
			UnitCostID:      rateID,
			BatchID:         batchID,
		}); err != nil {
			return apierror.NewInternalError(err, "Failed to insert inventory receipt.")
		}
	} else {
		// Negative measure = issue (inventory decrease).
		issueID, apiErr := id.GenID(id.InventoryIssueIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}

		batchID := sql.NullString{}
		if params.BatchID != nil {
			batchID = sql.NullString{String: *params.BatchID, Valid: true}
		}

		if err := r.queries.InsertInventoryIssue(ctx, sqlc.InsertInventoryIssueParams{
			ID:         issueID,
			AccountID:  params.AccountID,
			ItemID:     params.ItemID,
			QuantityID: quantityID,
			BatchID:    batchID,
		}); err != nil {
			return apierror.NewInternalError(err, "Failed to insert inventory issue.")
		}
	}

	return nil
}

func (r *inventoryMutationRepo) CreateInventoryReceipt(ctx context.Context, params domain.CreateInventoryReceiptParams) *apierror.APIError {
	ctx, span := inventoryMutationRepoTracer.Start(ctx, "repository.inventory_mutation.create_inventory_receipt")
	defer span.End()

	absMeasure := params.Measure.Abs()

	// Create a quantity record for the receipt.
	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if err := r.queries.InsertQuantityForInventory(ctx, sqlc.InsertQuantityForInventoryParams{
		ID:     quantityID,
		Value:  absMeasure.String(),
		UnitID: params.UnitID,
	}); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to insert quantity for inventory receipt."))
	}

	// Look up the item's current unit cost to associate with the receipt.
	costRow, err := r.queries.GetItemUnitCost(ctx, sqlc.GetItemUnitCostParams{
		ItemID:    params.ItemID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Create a rate record cloning the item's unit cost.
	rateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if err := r.queries.InsertRateForInventory(ctx, sqlc.InsertRateForInventoryParams{
		ID:                rateID,
		Value:             costRow.Value,
		NumeratorUnitID:   costRow.NumeratorUnitID,
		DenominatorUnitID: costRow.DenominatorUnitID,
	}); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to insert rate for inventory receipt."))
	}

	// Create the inventory receipt.
	receiptID, apiErr := id.GenID(id.InventoryReceiptIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	ownerAccountID := params.AccountID
	if params.OwnerAccountID != "" {
		ownerAccountID = params.OwnerAccountID
	}
	holderAccountID := params.AccountID
	if params.HolderAccountID != "" {
		holderAccountID = params.HolderAccountID
	}

	locationID := sql.NullString{}
	if params.LocationID != nil {
		locationID = sql.NullString{String: *params.LocationID, Valid: true}
	}
	lotID := sql.NullString{}
	if params.LotID != nil {
		lotID = sql.NullString{String: *params.LotID, Valid: true}
	}

	if err := r.queries.InsertInventoryReceipt(ctx, sqlc.InsertInventoryReceiptParams{
		ID:                receiptID,
		OwnerAccountID:    ownerAccountID,
		HolderAccountID:   holderAccountID,
		ItemID:            params.ItemID,
		QuantityID:        quantityID,
		UnitCostID:        rateID,
		StorageLocationID: locationID,
		LotID:             lotID,
	}); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to insert inventory receipt."))
	}

	return nil
}

func (r *inventoryMutationRepo) CreateInventoryIssue(ctx context.Context, params domain.CreateInventoryIssueParams) *apierror.APIError {
	ctx, span := inventoryMutationRepoTracer.Start(ctx, "repository.inventory_mutation.create_inventory_issue")
	defer span.End()

	absMeasure := params.Measure.Abs()

	// Create a quantity record for the issue.
	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if err := r.queries.InsertQuantityForInventory(ctx, sqlc.InsertQuantityForInventoryParams{
		ID:     quantityID,
		Value:  absMeasure.String(),
		UnitID: params.UnitID,
	}); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to insert quantity for inventory issue."))
	}

	// Create the inventory issue.
	issueID, apiErr := id.GenID(id.InventoryIssueIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	locationID := sql.NullString{}
	if params.LocationID != nil {
		locationID = sql.NullString{String: *params.LocationID, Valid: true}
	}
	lotID := sql.NullString{}
	if params.LotID != nil {
		lotID = sql.NullString{String: *params.LotID, Valid: true}
	}

	if err := r.queries.InsertInventoryIssue(ctx, sqlc.InsertInventoryIssueParams{
		ID:                issueID,
		AccountID:         params.AccountID,
		ItemID:            params.ItemID,
		QuantityID:        quantityID,
		StorageLocationID: locationID,
		LotID:             lotID,
	}); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to insert inventory issue."))
	}

	return nil
}

func (r *inventoryMutationRepo) CreateInventoryLog(ctx context.Context, params domain.CreateInventoryLogParams) *apierror.APIError {
	ctx, span := inventoryMutationRepoTracer.Start(ctx, "repository.inventory_mutation.create_inventory_log")
	defer span.End()

	// Create a quantity record for the log snapshot.
	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if err := r.queries.InsertQuantityForInventory(ctx, sqlc.InsertQuantityForInventoryParams{
		ID:     quantityID,
		Value:  params.Measure.String(),
		UnitID: params.UnitID,
	}); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to insert quantity for inventory log."))
	}

	// Create the inventory log.
	logID, apiErr := id.GenID(id.InventoryLogIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if err := r.queries.InsertInventoryLog(ctx, sqlc.InsertInventoryLogParams{
		ID:         logID,
		ItemID:     params.ItemID,
		QuantityID: quantityID,
		AccountID:  params.AccountID,
	}); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to insert inventory log."))
	}

	return nil
}

func (r *inventoryMutationRepo) CreateInventoryChangeLog(ctx context.Context, params domain.CreateInventoryChangeLogParams) *apierror.APIError {
	ctx, span := inventoryMutationRepoTracer.Start(ctx, "repository.inventory_mutation.create_inventory_change_log")
	defer span.End()

	// Create a quantity record for the change log.
	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if err := r.queries.InsertQuantityForInventory(ctx, sqlc.InsertQuantityForInventoryParams{
		ID:     quantityID,
		Value:  params.Measure.String(),
		UnitID: params.UnitID,
	}); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to insert quantity for inventory change log."))
	}

	// Create the inventory change log.
	changeLogID, apiErr := id.GenID(id.InventoryChangeLogIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	scanningStationID := sql.NullString{}
	if params.ScanningStationID != nil {
		scanningStationID = sql.NullString{String: *params.ScanningStationID, Valid: true}
	}

	responsibleUserID := sql.NullString{}
	if params.ResponsibleUserID != nil {
		responsibleUserID = sql.NullString{String: *params.ResponsibleUserID, Valid: true}
	}

	inventoryLogID := sql.NullString{}
	if params.InventoryLogID != nil {
		inventoryLogID = sql.NullString{String: *params.InventoryLogID, Valid: true}
	}

	if err := r.queries.InsertInventoryChangeLog(ctx, sqlc.InsertInventoryChangeLogParams{
		ID:                changeLogID,
		ItemID:            params.ItemID,
		QuantityID:        quantityID,
		ActionTypeCode:    params.ActionType,
		ScanningStationID: scanningStationID,
		AccountID:         params.AccountID,
		InventoryLogID:    inventoryLogID,
		ResponsibleUserID: responsibleUserID,
	}); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to insert inventory change log."))
	}

	return nil
}

func (r *inventoryMutationRepo) CountAllocatedReceiptsForBatch(ctx context.Context, accountID, batchID string) (int64, *apierror.APIError) {
	ctx, span := inventoryMutationRepoTracer.Start(ctx, "repository.inventory_mutation.count_allocated_receipts_for_batch")
	defer span.End()

	count, err := r.queries.CountAllocatedReceiptsForBatch(ctx, sqlc.CountAllocatedReceiptsForBatchParams{
		BatchID:   sql.NullString{String: batchID, Valid: true},
		AccountID: accountID,
	})
	if err != nil {
		return 0, tracing.Trace(span, db.MapSQLError(err))
	}

	return count, nil
}

// ReverseInventoryForBatch removes the rows a scan wrote and hands back what they were holding.
//
// Current inventory is derived from the receipt and issue rows, so removing them is the correction —
// nothing recalculates a stored level. An issue that came out of an order's reservation goes back to
// `reserved` rather than being deleted; one issued against free stock is deleted outright.
func (r *inventoryMutationRepo) ReverseInventoryForBatch(ctx context.Context, params domain.ReverseInventoryForBatchParams) ([]domain.InventoryReversalDelta, *apierror.APIError) {
	ctx, span := inventoryMutationRepoTracer.Start(ctx, "repository.inventory_mutation.reverse_inventory_for_batch")
	defer span.End()

	batchID := sql.NullString{String: params.BatchID, Valid: true}

	receipts, err := r.queries.FindReceiptsForBatchReversal(ctx, sqlc.FindReceiptsForBatchReversalParams{
		BatchID:   batchID,
		AccountID: params.AccountID,
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	for _, receipt := range receipts {
		if receipt.AllocationCount > 0 {
			return nil, tracing.Trace(span, apierror.NewValidationError("Inventory produced by this batch has already been used and cannot be reversed."))
		}
	}

	issues, err := r.queries.FindIssuesForBatchReversal(ctx, sqlc.FindIssuesForBatchReversalParams{
		BatchID:   batchID,
		AccountID: params.AccountID,
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	if len(receipts) == 0 && len(issues) == 0 {
		return nil, nil
	}

	issueIDs := make([]string, 0, len(issues))
	for _, issue := range issues {
		issueIDs = append(issueIDs, issue.ID)
	}

	// Dropping an allocation hands the quantity back to the receipt it came from.
	if len(issueIDs) > 0 {
		allocations, err := r.queries.FindAllocationsByIssueIDs(ctx, issueIDs)
		if err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}

		if len(allocations) > 0 {
			allocationIDs := make([]string, 0, len(allocations))
			quantityIDs := make([]string, 0, len(allocations)*2)
			rateIDs := make([]string, 0, len(allocations))
			receiptIDs := make([]string, 0, len(allocations))
			for _, allocation := range allocations {
				allocationIDs = append(allocationIDs, allocation.ID)
				quantityIDs = append(quantityIDs, allocation.QuantityID, allocation.TotalCostID)
				rateIDs = append(rateIDs, allocation.UnitCostID)
				receiptIDs = append(receiptIDs, allocation.InventoryReceiptID)
			}

			if err := r.queries.DeleteAllocationsByIDs(ctx, allocationIDs); err != nil {
				return nil, tracing.Trace(span, db.MapSQLError(err))
			}
			if err := r.queries.DeleteQuantitiesByIDs(ctx, quantityIDs); err != nil {
				return nil, tracing.Trace(span, db.MapSQLError(err))
			}
			if err := r.queries.DeleteRatesByIDs(ctx, rateIDs); err != nil {
				return nil, tracing.Trace(span, db.MapSQLError(err))
			}
			// Runs after the deletes so it weighs only the allocations that survived.
			if err := r.queries.FreeReleasedReceipts(ctx, receiptIDs); err != nil {
				return nil, tracing.Trace(span, db.MapSQLError(err))
			}
		}
	}

	var restoreIDs, deleteIssueIDs, deleteIssueQuantityIDs []string
	for _, issue := range issues {
		if issue.OrderID.Valid && issue.OrderID.String != "" {
			restoreIDs = append(restoreIDs, issue.ID)
			continue
		}
		deleteIssueIDs = append(deleteIssueIDs, issue.ID)
		deleteIssueQuantityIDs = append(deleteIssueQuantityIDs, issue.QuantityID)
	}

	if len(restoreIDs) > 0 {
		if err := r.queries.RestoreIssuesToReserved(ctx, restoreIDs); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
	}

	if len(deleteIssueIDs) > 0 {
		if err := r.queries.DeleteInventoryIssuesByIDs(ctx, deleteIssueIDs); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		if err := r.queries.DeleteQuantitiesByIDs(ctx, deleteIssueQuantityIDs); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
	}

	if len(receipts) > 0 {
		receiptIDs := make([]string, 0, len(receipts))
		quantityIDs := make([]string, 0, len(receipts))
		rateIDs := make([]string, 0, len(receipts))
		for _, receipt := range receipts {
			receiptIDs = append(receiptIDs, receipt.ID)
			quantityIDs = append(quantityIDs, receipt.QuantityID)
			rateIDs = append(rateIDs, receipt.UnitCostID)
		}

		if err := r.queries.DeleteInventoryReceiptsByIDs(ctx, receiptIDs); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		if err := r.queries.DeleteQuantitiesByIDs(ctx, quantityIDs); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		if err := r.queries.DeleteRatesByIDs(ctx, rateIDs); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
	}

	// One correction per reversed row, signed the opposite way from the scan.
	deltas := make([]domain.InventoryReversalDelta, 0, len(receipts)+len(issues))
	for _, receipt := range receipts {
		measure, parseErr := decimal.NewFromString(receipt.QuantityValue)
		if parseErr != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(parseErr, "Invalid receipt quantity value."))
		}
		deltas = append(deltas, domain.InventoryReversalDelta{
			ItemID:  receipt.ItemID,
			Measure: measure.Neg(),
			UnitID:  receipt.UnitID,
		})
	}
	for _, issue := range issues {
		measure, parseErr := decimal.NewFromString(issue.QuantityValue)
		if parseErr != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(parseErr, "Invalid issue quantity value."))
		}
		deltas = append(deltas, domain.InventoryReversalDelta{
			ItemID:  issue.ItemID,
			Measure: measure,
			UnitID:  issue.UnitID,
		})
	}

	return deltas, nil
}

// Gives a consumed measure back to the order's reservation, undoing the issues that took it newest
// first: their allocations drop, releasing the receipts, and an overshooting issue is split.
func (r *inventoryMutationRepo) ReverseInventoryForOrderItem(ctx context.Context, accountID, orderID, itemID string, measure decimal.Decimal) *apierror.APIError {
	ctx, span := inventoryMutationRepoTracer.Start(ctx, "repository.inventory_mutation.reverse_inventory_for_order_item")
	defer span.End()

	if measure.LessThanOrEqual(decimal.Zero) {
		return nil
	}

	issues, err := r.queries.FindOpenIssuesForOrderItemReversal(ctx, sqlc.FindOpenIssuesForOrderItemReversalParams{
		OrderID:   sql.NullString{String: orderID, Valid: true},
		AccountID: accountID,
		ItemID:    itemID,
	})
	if err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}

	remaining := measure
	issueIDs := make([]string, 0, len(issues))

	// Only the last issue taken can overshoot, and it is reversed down to the budget with the balance
	// re-issued as a fresh open row.
	var split *sqlc.FindOpenIssuesForOrderItemReversalRow
	var splitReversed, splitResidual decimal.Decimal

	for i := range issues {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}

		issueQty, parseErr := decimal.NewFromString(issues[i].QuantityValue)
		if parseErr != nil {
			return tracing.Trace(span, apierror.NewInternalError(parseErr, "Invalid issue quantity value."))
		}
		if !issueQty.IsPositive() {
			continue
		}

		issueIDs = append(issueIDs, issues[i].ID)

		if issueQty.GreaterThan(remaining) {
			split = &issues[i]
			splitReversed = remaining
			splitResidual = issueQty.Sub(remaining)
			break
		}

		remaining = remaining.Sub(issueQty)
	}

	if len(issueIDs) == 0 {
		return nil
	}

	// Dropping an allocation hands the quantity back to the receipt it came from.
	allocations, err := r.queries.FindAllocationsByIssueIDs(ctx, issueIDs)
	if err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}

	if len(allocations) > 0 {
		allocationIDs := make([]string, 0, len(allocations))
		quantityIDs := make([]string, 0, len(allocations)*2)
		rateIDs := make([]string, 0, len(allocations))
		receiptIDs := make([]string, 0, len(allocations))
		for _, allocation := range allocations {
			allocationIDs = append(allocationIDs, allocation.ID)
			quantityIDs = append(quantityIDs, allocation.QuantityID, allocation.TotalCostID)
			rateIDs = append(rateIDs, allocation.UnitCostID)
			receiptIDs = append(receiptIDs, allocation.InventoryReceiptID)
		}

		if err := r.queries.DeleteAllocationsByIDs(ctx, allocationIDs); err != nil {
			return tracing.Trace(span, db.MapSQLError(err))
		}
		if err := r.queries.DeleteQuantitiesByIDs(ctx, quantityIDs); err != nil {
			return tracing.Trace(span, db.MapSQLError(err))
		}
		if err := r.queries.DeleteRatesByIDs(ctx, rateIDs); err != nil {
			return tracing.Trace(span, db.MapSQLError(err))
		}
		// Runs after the deletes so it weighs only the allocations that survived.
		if err := r.queries.FreeReleasedReceipts(ctx, receiptIDs); err != nil {
			return tracing.Trace(span, db.MapSQLError(err))
		}
	}

	if err := r.queries.RestoreIssuesToReserved(ctx, issueIDs); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}

	if split != nil {
		if err := r.queries.UpdateQuantityValue(ctx, sqlc.UpdateQuantityValueParams{
			Value: splitReversed.String(),
			ID:    split.QuantityID,
		}); err != nil {
			return tracing.Trace(span, db.MapSQLError(err))
		}

		residualIssueID, apiErr := id.GenID(id.InventoryIssueIDPrefix, nil)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		residualQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}

		if err := r.queries.InsertQuantityForInventory(ctx, sqlc.InsertQuantityForInventoryParams{
			ID:     residualQuantityID,
			Value:  splitResidual.String(),
			UnitID: split.UnitID,
		}); err != nil {
			return tracing.Trace(span, db.MapSQLError(err))
		}

		// Carries no allocations of its own: the caller's FIFO pass covers it alongside the other
		// issues the freed receipts can now fill.
		if err := r.queries.InsertInventoryIssueForReservation(ctx, sqlc.InsertInventoryIssueForReservationParams{
			ID:                residualIssueID,
			AccountID:         accountID,
			ItemID:            itemID,
			QuantityID:        residualQuantityID,
			StatusCode:        "open",
			OrderID:           sql.NullString{String: orderID, Valid: true},
			BatchID:           split.BatchID,
			StorageLocationID: split.StorageLocationID,
			LotID:             split.LotID,
		}); err != nil {
			return tracing.Trace(span, db.MapSQLError(err))
		}
	}

	return nil
}

func (r *inventoryMutationRepo) CreateQuantityForInventory(ctx context.Context, quantityID, value, unitID string) *apierror.APIError {
	if err := r.queries.InsertQuantityForInventory(ctx, sqlc.InsertQuantityForInventoryParams{
		ID:     quantityID,
		Value:  value,
		UnitID: unitID,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to insert quantity for inventory.")
	}
	return nil
}

func (r *inventoryMutationRepo) CreateRateForInventory(ctx context.Context, rateID, value, numeratorUnitID, denominatorUnitID string) *apierror.APIError {
	if err := r.queries.InsertRateForInventory(ctx, sqlc.InsertRateForInventoryParams{
		ID:                rateID,
		Value:             value,
		NumeratorUnitID:   numeratorUnitID,
		DenominatorUnitID: denominatorUnitID,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to insert rate for inventory.")
	}
	return nil
}
