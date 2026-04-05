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

var supplierMaterialRepoTracer = tracing.GetTracer("core-service.supplier_material_repository")

type supplierMaterialRepoImpl struct {
	queries *sqlc.Queries
}

func NewSupplierMaterialRepo(queries *sqlc.Queries) domain.SupplierMaterialRepo {
	return &supplierMaterialRepoImpl{queries: queries}
}

func supplierMaterialCreatedAt(sm *domain.SupplierMaterial) time.Time { return sm.CreatedAt }
func supplierMaterialID(sm *domain.SupplierMaterial) string           { return sm.ID }

func mapSupplierMaterialForwardRow(row sqlc.ListSupplierMaterialsForwardRow) *domain.SupplierMaterial {
	return mapSupplierMaterialFromRow(
		row.ID, row.MaterialID, row.SupplierAccountID, row.SupplierPartNumber,
		row.SupplierDescription, row.IsActive, row.OwnerAccountID, row.CreatedAt, row.UpdatedAt,
		row.MaterialTypeID, row.ItemID, row.OrderPointID, row.LeadTimeID,
		row.MaterialCreatedAt, row.MaterialUpdatedAt,
		row.Sku, row.ItemDescription, row.ItemNotes, row.ItemTypeCode,
		row.ItemCategoryID, row.UnitValueID, row.UnitCostID, row.BurnRateID,
		row.ItemAccountID, row.IsDirty, row.ItemCreatedAt, row.ItemUpdatedAt,
		row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID,
		row.OrderPointValue, row.OrderPointUnitID, row.OrderPointUnitAbbreviation, row.OrderPointUnitType,
		row.LeadTimeValue, row.LeadTimeUnitID, row.LeadTimeUnitAbbreviation, row.LeadTimeUnitType,
		row.UnitValueRateID, row.UnitValueRateValue, row.UnitValueNumeratorUnitID, row.UnitValueDenominatorUnitID,
		row.UnitValueCreatedAt, row.UnitValueUpdatedAt,
		row.UnitCostRateID, row.UnitCostRateValue, row.UnitCostNumeratorUnitID, row.UnitCostDenominatorUnitID,
		row.UnitCostCreatedAt, row.UnitCostUpdatedAt,
		row.BurnRateIDJoined, row.BurnRateValue, row.BurnRateNumeratorUnitID, row.BurnRateDenominatorUnitID,
		row.BurnRateCreatedAt, row.BurnRateUpdatedAt,
	)
}

func mapSupplierMaterialBackwardRow(row sqlc.ListSupplierMaterialsBackwardRow) *domain.SupplierMaterial {
	return mapSupplierMaterialFromRow(
		row.ID, row.MaterialID, row.SupplierAccountID, row.SupplierPartNumber,
		row.SupplierDescription, row.IsActive, row.OwnerAccountID, row.CreatedAt, row.UpdatedAt,
		row.MaterialTypeID, row.ItemID, row.OrderPointID, row.LeadTimeID,
		row.MaterialCreatedAt, row.MaterialUpdatedAt,
		row.Sku, row.ItemDescription, row.ItemNotes, row.ItemTypeCode,
		row.ItemCategoryID, row.UnitValueID, row.UnitCostID, row.BurnRateID,
		row.ItemAccountID, row.IsDirty, row.ItemCreatedAt, row.ItemUpdatedAt,
		row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID,
		row.OrderPointValue, row.OrderPointUnitID, row.OrderPointUnitAbbreviation, row.OrderPointUnitType,
		row.LeadTimeValue, row.LeadTimeUnitID, row.LeadTimeUnitAbbreviation, row.LeadTimeUnitType,
		row.UnitValueRateID, row.UnitValueRateValue, row.UnitValueNumeratorUnitID, row.UnitValueDenominatorUnitID,
		row.UnitValueCreatedAt, row.UnitValueUpdatedAt,
		row.UnitCostRateID, row.UnitCostRateValue, row.UnitCostNumeratorUnitID, row.UnitCostDenominatorUnitID,
		row.UnitCostCreatedAt, row.UnitCostUpdatedAt,
		row.BurnRateIDJoined, row.BurnRateValue, row.BurnRateNumeratorUnitID, row.BurnRateDenominatorUnitID,
		row.BurnRateCreatedAt, row.BurnRateUpdatedAt,
	)
}

func mapSupplierMaterialGetRow(row sqlc.GetSupplierMaterialBySupplierAndItemIDRow) *domain.SupplierMaterial {
	return mapSupplierMaterialFromRow(
		row.ID, row.MaterialID, row.SupplierAccountID, row.SupplierPartNumber,
		row.SupplierDescription, row.IsActive, row.OwnerAccountID, row.CreatedAt, row.UpdatedAt,
		row.MaterialTypeID, row.ItemID, row.OrderPointID, row.LeadTimeID,
		row.MaterialCreatedAt, row.MaterialUpdatedAt,
		row.Sku, row.ItemDescription, row.ItemNotes, row.ItemTypeCode,
		row.ItemCategoryID, row.UnitValueID, row.UnitCostID, row.BurnRateID,
		row.ItemAccountID, row.IsDirty, row.ItemCreatedAt, row.ItemUpdatedAt,
		row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID,
		row.OrderPointValue, row.OrderPointUnitID, row.OrderPointUnitAbbreviation, row.OrderPointUnitType,
		row.LeadTimeValue, row.LeadTimeUnitID, row.LeadTimeUnitAbbreviation, row.LeadTimeUnitType,
		row.UnitValueRateID, row.UnitValueRateValue, row.UnitValueNumeratorUnitID, row.UnitValueDenominatorUnitID,
		row.UnitValueCreatedAt, row.UnitValueUpdatedAt,
		row.UnitCostRateID, row.UnitCostRateValue, row.UnitCostNumeratorUnitID, row.UnitCostDenominatorUnitID,
		row.UnitCostCreatedAt, row.UnitCostUpdatedAt,
		row.BurnRateIDJoined, row.BurnRateValue, row.BurnRateNumeratorUnitID, row.BurnRateDenominatorUnitID,
		row.BurnRateCreatedAt, row.BurnRateUpdatedAt,
	)
}

func mapSupplierMaterialFromRow(
	id, materialID, supplierAccountID, supplierPartNumber string,
	supplierDescription gosql.NullString, isActive bool, ownerAccountID string,
	createdAt, updatedAt time.Time,
	materialTypeID, itemID, orderPointID, leadTimeID string,
	materialCreatedAt, materialUpdatedAt time.Time,
	sku string, itemDescription, itemNotes gosql.NullString, itemTypeCode string,
	itemCategoryID, unitValueID, unitCostID, burnRateID, itemAccountID string,
	isDirty bool, itemCreatedAt, itemUpdatedAt time.Time,
	categoryName, itemCategoryTypeCode, categoryUnitGroupID string,
	orderPointValue, orderPointUnitID, orderPointUnitAbbreviation, orderPointUnitType gosql.NullString,
	leadTimeValue, leadTimeUnitID, leadTimeUnitAbbreviation, leadTimeUnitType gosql.NullString,
	unitValueRateID, unitValueRateValue, unitValueNumeratorUnitID, unitValueDenominatorUnitID string,
	unitValueCreatedAt, unitValueUpdatedAt time.Time,
	unitCostRateID, unitCostRateValue, unitCostNumeratorUnitID, unitCostDenominatorUnitID string,
	unitCostCreatedAt, unitCostUpdatedAt time.Time,
	burnRateIDJoined, burnRateValue, burnRateNumeratorUnitID, burnRateDenominatorUnitID string,
	burnRateCreatedAt, burnRateUpdatedAt time.Time,
) *domain.SupplierMaterial {
	var desc *string
	if supplierDescription.Valid {
		desc = &supplierDescription.String
	}

	var itemDesc *string
	if itemDescription.Valid {
		itemDesc = &itemDescription.String
	}

	var itemNotesPtr *string
	if itemNotes.Valid {
		itemNotesPtr = &itemNotes.String
	}

	var orderPoint *domain.Quantity
	if orderPointValue.Valid {
		orderPoint = &domain.Quantity{
			ID:               orderPointID,
			Value:            orderPointValue.String,
			UnitID:           orderPointUnitID.String,
			UnitAbbreviation: orderPointUnitAbbreviation.String,
			UnitType:         orderPointUnitType.String,
		}
	}

	var leadTime *domain.Quantity
	if leadTimeValue.Valid {
		leadTime = &domain.Quantity{
			ID:               leadTimeID,
			Value:            leadTimeValue.String,
			UnitID:           leadTimeUnitID.String,
			UnitAbbreviation: leadTimeUnitAbbreviation.String,
			UnitType:         leadTimeUnitType.String,
		}
	}

	return &domain.SupplierMaterial{
		ID:                  id,
		MaterialID:          materialID,
		SupplierAccountID:   supplierAccountID,
		SupplierPartNumber:  supplierPartNumber,
		SupplierDescription: desc,
		IsActive:            isActive,
		OwnerAccountID:      ownerAccountID,
		CreatedAt:           createdAt,
		UpdatedAt:           updatedAt,
		Material: &domain.Material{
			ID:         materialTypeID,
			ItemID:     itemID,
			CreatedAt:  materialCreatedAt,
			UpdatedAt:  materialUpdatedAt,
			OrderPoint: orderPoint,
			LeadTime:   leadTime,
			Item: &domain.Item{
				ID:             itemID,
				SKU:            sku,
				Description:    itemDesc,
				Notes:          itemNotesPtr,
				ItemTypeCode:   itemTypeCode,
				ItemCategoryID: itemCategoryID,
				CategoryName:   categoryName,
				UnitValueID:    unitValueID,
				UnitCostID:     unitCostID,
				BurnRateID:     burnRateID,
				AccountID:      itemAccountID,
				IsDirty:        isDirty,
				CreatedAt:      itemCreatedAt,
				UpdatedAt:      itemUpdatedAt,
				UnitValue: &domain.Rate{
					ID:                unitValueRateID,
					Value:             unitValueRateValue,
					NumeratorUnitID:   unitValueNumeratorUnitID,
					DenominatorUnitID: unitValueDenominatorUnitID,
					CreatedAt:         unitValueCreatedAt,
					UpdatedAt:         unitValueUpdatedAt,
				},
				UnitCost: &domain.Rate{
					ID:                unitCostRateID,
					Value:             unitCostRateValue,
					NumeratorUnitID:   unitCostNumeratorUnitID,
					DenominatorUnitID: unitCostDenominatorUnitID,
					CreatedAt:         unitCostCreatedAt,
					UpdatedAt:         unitCostUpdatedAt,
				},
				BurnRate: &domain.Rate{
					ID:                burnRateIDJoined,
					Value:             burnRateValue,
					NumeratorUnitID:   burnRateNumeratorUnitID,
					DenominatorUnitID: burnRateDenominatorUnitID,
					CreatedAt:         burnRateCreatedAt,
					UpdatedAt:         burnRateUpdatedAt,
				},
				Category: &domain.ItemCategory{
					ID:                   itemCategoryID,
					Name:                 categoryName,
					ItemCategoryTypeCode: itemCategoryTypeCode,
					UnitGroupID:          categoryUnitGroupID,
				},
			},
		},
	}
}

func (r *supplierMaterialRepoImpl) List(ctx context.Context, params domain.ListSupplierMaterialsParams) (*domain.ListSupplierMaterialsResult, *apierror.APIError) {
	ctx, span := supplierMaterialRepoTracer.Start(ctx, "repository.supplier_material.list")
	defer span.End()

	searchQuery := gosql.NullString{}
	if params.Query != nil && *params.Query != "" {
		searchQuery = gosql.NullString{String: "%" + *params.Query + "%", Valid: true}
	}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListSupplierMaterialsBackward(ctx, sqlc.ListSupplierMaterialsBackwardParams{
				SupplierAccountID: params.SupplierAccountID,
				OwnerAccountID:    params.OwnerAccountID,
				SearchQuery:       searchQuery,
				CursorCreatedAt:   cur.OccurredAt,
				CursorID:          cur.ID,
				Limit:             params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			items := make([]*domain.SupplierMaterial, len(rows))
			for i, row := range rows {
				items[i] = mapSupplierMaterialBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, supplierMaterialCreatedAt, supplierMaterialID)
			return &domain.ListSupplierMaterialsResult{SupplierMaterials: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListSupplierMaterialsForward(ctx, sqlc.ListSupplierMaterialsForwardParams{
			SupplierAccountID: params.SupplierAccountID,
			OwnerAccountID:    params.OwnerAccountID,
			SearchQuery:       searchQuery,
			CursorCreatedAt:   gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:          gosql.NullString{String: cur.ID, Valid: true},
			Limit:             params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items := make([]*domain.SupplierMaterial, len(rows))
		for i, row := range rows {
			items[i] = mapSupplierMaterialForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, supplierMaterialCreatedAt, supplierMaterialID)
		return &domain.ListSupplierMaterialsResult{SupplierMaterials: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListSupplierMaterialsForward(ctx, sqlc.ListSupplierMaterialsForwardParams{
		SupplierAccountID: params.SupplierAccountID,
		OwnerAccountID:    params.OwnerAccountID,
		SearchQuery:       searchQuery,
		Limit:             params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]*domain.SupplierMaterial, len(rows))
	for i, row := range rows {
		items[i] = mapSupplierMaterialForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, supplierMaterialCreatedAt, supplierMaterialID)
	return &domain.ListSupplierMaterialsResult{SupplierMaterials: result, PageInfo: pageInfo}, nil
}

func (r *supplierMaterialRepoImpl) GetBySupplierAndItemID(ctx context.Context, ownerAccountID, supplierAccountID, itemID string) (*domain.SupplierMaterial, *apierror.APIError) {
	ctx, span := supplierMaterialRepoTracer.Start(ctx, "repository.supplier_material.get_by_supplier_and_item_id")
	defer span.End()

	row, err := r.queries.GetSupplierMaterialBySupplierAndItemID(ctx, sqlc.GetSupplierMaterialBySupplierAndItemIDParams{
		SupplierAccountID: supplierAccountID,
		ItemID:            itemID,
		OwnerAccountID:    ownerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapSupplierMaterialGetRow(row), nil
}

func (r *supplierMaterialRepoImpl) Create(ctx context.Context, id string, params domain.CreateSupplierMaterialParams) (*domain.SupplierMaterial, *apierror.APIError) {
	ctx, span := supplierMaterialRepoTracer.Start(ctx, "repository.supplier_material.create")
	defer span.End()

	err := r.queries.CreateSupplierMaterial(ctx, sqlc.CreateSupplierMaterialParams{
		ID:                  id,
		MaterialID:          params.MaterialID,
		SupplierAccountID:   params.SupplierAccountID,
		SupplierPartNumber:  params.SupplierPartNumber,
		SupplierDescription: toNullString(params.SupplierDescription),
		IsActive:            params.IsActive,
		OwnerAccountID:      params.OwnerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Look up item_id from material_id so we can re-fetch the full record.
	itemID, err := r.queries.GetItemIDByMaterialID(ctx, params.MaterialID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.GetBySupplierAndItemID(ctx, params.OwnerAccountID, params.SupplierAccountID, itemID)
}

func (r *supplierMaterialRepoImpl) Update(ctx context.Context, params domain.UpdateSupplierMaterialParams) (*domain.SupplierMaterial, *apierror.APIError) {
	ctx, span := supplierMaterialRepoTracer.Start(ctx, "repository.supplier_material.update")
	defer span.End()

	// Look up the existing supplier material to get the sm.ID for the update query.
	existing, apiErr := r.GetBySupplierAndItemID(ctx, params.OwnerAccountID, params.SupplierAccountID, params.ItemID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	updateDescription := gosql.NullString{}
	if params.UpdateDescription {
		updateDescription = gosql.NullString{String: "1", Valid: true}
	}

	result, err := r.queries.UpdateSupplierMaterial(ctx, sqlc.UpdateSupplierMaterialParams{
		SupplierPartNumber:  toNullString(params.SupplierPartNumber),
		UpdateDescription:   updateDescription,
		SupplierDescription: toNullString(params.SupplierDescription),
		IsActive:            toNullBool(params.IsActive),
		ID:                  existing.ID,
		OwnerAccountID:      params.OwnerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Supplier material not found."))
	}

	return r.GetBySupplierAndItemID(ctx, params.OwnerAccountID, params.SupplierAccountID, params.ItemID)
}

func (r *supplierMaterialRepoImpl) Delete(ctx context.Context, params domain.DeleteSupplierMaterialParams) (*domain.SupplierMaterial, *apierror.APIError) {
	ctx, span := supplierMaterialRepoTracer.Start(ctx, "repository.supplier_material.delete")
	defer span.End()

	// Fetch the full record before deleting so we can return it.
	existing, apiErr := r.GetBySupplierAndItemID(ctx, params.OwnerAccountID, params.SupplierAccountID, params.ItemID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, err := r.queries.DeleteSupplierMaterial(ctx, sqlc.DeleteSupplierMaterialParams{
		ID:             existing.ID,
		OwnerAccountID: params.OwnerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Supplier material not found."))
	}

	return existing, nil
}

func (r *supplierMaterialRepoImpl) ExistsByMaterialAndSupplier(ctx context.Context, ownerAccountID, materialID, supplierAccountID string) (bool, *apierror.APIError) {
	ctx, span := supplierMaterialRepoTracer.Start(ctx, "repository.supplier_material.exists_by_material_and_supplier")
	defer span.End()

	count, err := r.queries.ExistsSupplierMaterialByMaterialAndSupplier(ctx, sqlc.ExistsSupplierMaterialByMaterialAndSupplierParams{
		MaterialID:        materialID,
		SupplierAccountID: supplierAccountID,
		OwnerAccountID:    ownerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}
