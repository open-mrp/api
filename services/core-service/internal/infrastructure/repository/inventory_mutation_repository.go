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

		// Create a zero unit cost rate for the receipt.
		rateID, apiErr := id.GenID(id.RateIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		if err := r.queries.InsertRateForInventory(ctx, sqlc.InsertRateForInventoryParams{
			ID:                rateID,
			Value:             "0",
			NumeratorUnitID:   params.UnitID,
			DenominatorUnitID: params.UnitID,
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
