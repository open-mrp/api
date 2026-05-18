package repository

import (
	"context"
	gosql "database/sql"
	"slices"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var materialRepoTracer = tracing.GetTracer("core-service.material_repository")

type materialRepoImpl struct {
	queries *sqlc.Queries
}

func NewMaterialRepo(queries *sqlc.Queries) domain.MaterialRepo {
	return &materialRepoImpl{queries: queries}
}

func materialCreatedAt(m *domain.Material) time.Time { return m.CreatedAt }
func materialID(m *domain.Material) string           { return m.ID }

func mapGetMaterialByItemIDRow(row sqlc.GetMaterialByItemIDRow) *domain.Material {
	var description *string
	if row.ItemDescription.Valid {
		description = &row.ItemDescription.String
	}
	var notes *string
	if row.ItemNotes.Valid {
		notes = &row.ItemNotes.String
	}

	return &domain.Material{
		ID:     row.ID,
		ItemID: row.ItemID,
		Item: &domain.Item{
			ID:             row.ItemID,
			SKU:            row.Sku,
			Description:    description,
			Notes:          notes,
			ItemTypeCode:   row.ItemTypeCode,
			ItemCategoryID: row.ItemCategoryID,
			CategoryName:   row.CategoryName,
			UnitValueID:    row.UnitValueID,
			UnitCostID:     row.UnitCostID,
			BurnRateID:     row.BurnRateID,
			AccountID:      row.AccountID,
			IsDirty:        row.IsDirty,
			CreatedAt:      row.ItemCreatedAt,
			UpdatedAt:      row.ItemUpdatedAt,
			UnitValue: &domain.Rate{
				ID:                row.UnitValueRateID,
				Value:             row.UnitValueRateValue,
				NumeratorUnitID:   row.UnitValueNumeratorUnitID,
				DenominatorUnitID: row.UnitValueDenominatorUnitID,
				CreatedAt:         row.UnitValueCreatedAt,
				UpdatedAt:         row.UnitValueUpdatedAt,
			},
			UnitCost: &domain.Rate{
				ID:                row.UnitCostRateID,
				Value:             row.UnitCostRateValue,
				NumeratorUnitID:   row.UnitCostNumeratorUnitID,
				DenominatorUnitID: row.UnitCostDenominatorUnitID,
				CreatedAt:         row.UnitCostCreatedAt,
				UpdatedAt:         row.UnitCostUpdatedAt,
			},
			BurnRate: &domain.Rate{
				ID:                row.BurnRateIDJoined,
				Value:             row.BurnRateValue,
				NumeratorUnitID:   row.BurnRateNumeratorUnitID,
				DenominatorUnitID: row.BurnRateDenominatorUnitID,
				CreatedAt:         row.BurnRateCreatedAt,
				UpdatedAt:         row.BurnRateUpdatedAt,
			},
			Category: &domain.ItemCategory{
				ID:                   row.ItemCategoryID,
				Name:                 row.CategoryName,
				ItemCategoryTypeCode: row.ItemCategoryTypeCode,
				UnitGroupID:          row.CategoryUnitGroupID,
				UnitGroupName:        row.CategoryUnitGroupName,
				UnitGroupTypeCode:    row.CategoryUnitGroupType,
				UnitGroupCreatedAt:   row.CategoryUnitGroupCreatedAt,
				UnitGroupUpdatedAt:   row.CategoryUnitGroupUpdatedAt,
			},
		},
		OrderPoint: &domain.Quantity{
			ID:               row.OrderPointID,
			Value:            row.OrderPointValue,
			UnitID:           row.OrderPointUnitID,
			UnitAbbreviation: row.OrderPointUnitAbbreviation,
			UnitType:         row.OrderPointUnitType,
		},
		LeadTime: &domain.Quantity{
			ID:               row.LeadTimeID,
			Value:            row.LeadTimeValue,
			UnitID:           row.LeadTimeUnitID,
			UnitAbbreviation: row.LeadTimeUnitAbbreviation,
			UnitType:         row.LeadTimeUnitType,
		},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func mapMaterialBaseItem(itemID, sku string, description, notes gosql.NullString, itemTypeCode, itemCategoryID, categoryName, itemCategoryTypeCode, categoryUnitGroupID, unitValueID, unitCostID, burnRateID, accountID string, isDirty bool, itemCreatedAt, itemUpdatedAt, categoryCreatedAt, categoryUpdatedAt time.Time) *domain.Item {
	var descPtr *string
	if description.Valid {
		descPtr = &description.String
	}
	var notesPtr *string
	if notes.Valid {
		notesPtr = &notes.String
	}
	return &domain.Item{
		ID:             itemID,
		SKU:            sku,
		Description:    descPtr,
		Notes:          notesPtr,
		ItemTypeCode:   itemTypeCode,
		ItemCategoryID: itemCategoryID,
		CategoryName:   categoryName,
		UnitValueID:    unitValueID,
		UnitCostID:     unitCostID,
		BurnRateID:     burnRateID,
		AccountID:      accountID,
		IsDirty:        isDirty,
		CreatedAt:      itemCreatedAt,
		UpdatedAt:      itemUpdatedAt,
		Category: &domain.ItemCategory{
			ID:                   itemCategoryID,
			Name:                 categoryName,
			ItemCategoryTypeCode: itemCategoryTypeCode,
			UnitGroupID:          categoryUnitGroupID,
			CreatedAt:            categoryCreatedAt,
			UpdatedAt:            categoryUpdatedAt,
		},
	}
}

func mapMaterialForwardBaseRow(row sqlc.ListMaterialsForwardBaseRow) *domain.Material {
	return &domain.Material{
		ID:     row.ID,
		ItemID: row.ItemID,
		Item:   mapMaterialBaseItem(row.ItemID, row.Sku, row.ItemDescription, row.ItemNotes, row.ItemTypeCode, row.ItemCategoryID, row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID, row.UnitValueID, row.UnitCostID, row.BurnRateID, row.AccountID, row.IsDirty, row.ItemCreatedAt, row.ItemUpdatedAt, row.CategoryCreatedAt, row.CategoryUpdatedAt),
		OrderPoint: &domain.Quantity{
			ID:               row.OrderPointID,
			Value:            row.OrderPointValue,
			UnitID:           row.OrderPointUnitID,
			UnitAbbreviation: row.OrderPointUnitAbbreviation,
			UnitType:         row.OrderPointUnitType,
		},
		LeadTime: &domain.Quantity{
			ID:               row.LeadTimeID,
			Value:            row.LeadTimeValue,
			UnitID:           row.LeadTimeUnitID,
			UnitAbbreviation: row.LeadTimeUnitAbbreviation,
			UnitType:         row.LeadTimeUnitType,
		},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func mapMaterialBackwardBaseRow(row sqlc.ListMaterialsBackwardBaseRow) *domain.Material {
	return &domain.Material{
		ID:     row.ID,
		ItemID: row.ItemID,
		Item:   mapMaterialBaseItem(row.ItemID, row.Sku, row.ItemDescription, row.ItemNotes, row.ItemTypeCode, row.ItemCategoryID, row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID, row.UnitValueID, row.UnitCostID, row.BurnRateID, row.AccountID, row.IsDirty, row.ItemCreatedAt, row.ItemUpdatedAt, row.CategoryCreatedAt, row.CategoryUpdatedAt),
		OrderPoint: &domain.Quantity{
			ID:               row.OrderPointID,
			Value:            row.OrderPointValue,
			UnitID:           row.OrderPointUnitID,
			UnitAbbreviation: row.OrderPointUnitAbbreviation,
			UnitType:         row.OrderPointUnitType,
		},
		LeadTime: &domain.Quantity{
			ID:               row.LeadTimeID,
			Value:            row.LeadTimeValue,
			UnitID:           row.LeadTimeUnitID,
			UnitAbbreviation: row.LeadTimeUnitAbbreviation,
			UnitType:         row.LeadTimeUnitType,
		},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func mapMaterialGetByIDBaseRow(row sqlc.GetMaterialByIDBaseRow) *domain.Material {
	return &domain.Material{
		ID:     row.ID,
		ItemID: row.ItemID,
		Item:   mapMaterialBaseItem(row.ItemID, row.Sku, row.ItemDescription, row.ItemNotes, row.ItemTypeCode, row.ItemCategoryID, row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID, row.UnitValueID, row.UnitCostID, row.BurnRateID, row.AccountID, row.IsDirty, row.ItemCreatedAt, row.ItemUpdatedAt, row.CategoryCreatedAt, row.CategoryUpdatedAt),
		OrderPoint: &domain.Quantity{
			ID:               row.OrderPointID,
			Value:            row.OrderPointValue,
			UnitID:           row.OrderPointUnitID,
			UnitAbbreviation: row.OrderPointUnitAbbreviation,
			UnitType:         row.OrderPointUnitType,
		},
		LeadTime: &domain.Quantity{
			ID:               row.LeadTimeID,
			Value:            row.LeadTimeValue,
			UnitID:           row.LeadTimeUnitID,
			UnitAbbreviation: row.LeadTimeUnitAbbreviation,
			UnitType:         row.LeadTimeUnitType,
		},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func applyMaterialStitches(ctx context.Context, queries *sqlc.Queries, materials []*domain.Material, incs []string) *apierror.APIError {
	items := itemsFromMaterials(materials)
	itemIncs := extractItemIncludes(incs)
	if apiErr := stitchItemRates(ctx, queries, items, itemIncs); apiErr != nil {
		return apiErr
	}
	if apiErr := stitchItemCategoryUnitGroups(ctx, queries, items, itemIncs); apiErr != nil {
		return apiErr
	}
	if apiErr := stitchItemAttributes(ctx, queries, items, itemIncs); apiErr != nil {
		return apiErr
	}
	if slices.Contains(itemIncs, "category.properties") {
		if apiErr := enrichItemCategoryProperties(ctx, queries, items); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

func (r *materialRepoImpl) List(ctx context.Context, params domain.ListMaterialsParams) (*domain.ListMaterialsResult, *apierror.APIError) {
	ctx, span := materialRepoTracer.Start(ctx, "repository.material.list")
	defer span.End()

	catSearch := db.NewCatalogSearch(params.Query)
	searchQuery := catSearch.Contains
	searchRankEnabled := catSearch.Contains.Valid
	materialSearchRank := func(m *domain.Material) int32 {
		return db.CatalogSearchRank(m.Item.SKU, catSearch)
	}
	includeCategoryFilter := len(params.CategoryIDs) > 0
	includeAttributeFilter := len(params.AttributeIDs) > 0

	categoryIDs := params.CategoryIDs
	if categoryIDs == nil {
		categoryIDs = []string{}
	}
	attributeIDs := params.AttributeIDs
	if attributeIDs == nil {
		attributeIDs = []string{}
	}

	var startDate, endDate gosql.NullTime
	if params.StartDate != nil {
		startDate = gosql.NullTime{Time: *params.StartDate, Valid: true}
	}
	if params.EndDate != nil {
		endDate = gosql.NullTime{Time: *params.EndDate, Valid: true}
	}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListMaterialsBackwardBase(ctx, sqlc.ListMaterialsBackwardBaseParams{
				AccountID:              params.AccountID,
				SearchQuery:            searchQuery,
				SearchExact:            catSearch.Exact,
				SearchPrefix:           catSearch.Prefix,
				IncludeCategoryFilter:  includeCategoryFilter,
				CategoryIds:            categoryIDs,
				IncludeAttributeFilter: includeAttributeFilter,
				AttributeIds:           attributeIDs,
				StartDate:              startDate,
				EndDate:                endDate,
				CursorMatchTier:        db.NullTierInt64Param(cur.MatchTier),
				CursorCreatedAt:        cur.OccurredAt,
				CursorID:               cur.ID,
				Limit:                  params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			materials := make([]*domain.Material, len(rows))
			for i, row := range rows {
				materials[i] = mapMaterialBackwardBaseRow(row)
			}
			result, pageInfo := pagination.BuildPageStringWithSearchRank(materials, params.Limit, cursorDir, searchRankEnabled, materialCreatedAt, materialID, materialSearchRank)
			if apiErr := applyMaterialStitches(ctx, r.queries, result, params.Includes); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			return &domain.ListMaterialsResult{Materials: result, PageInfo: pageInfo}, nil
		}

		// Forward
		rows, err := r.queries.ListMaterialsForwardBase(ctx, sqlc.ListMaterialsForwardBaseParams{
			AccountID:              params.AccountID,
			SearchQuery:            searchQuery,
			SearchExact:            catSearch.Exact,
			SearchPrefix:           catSearch.Prefix,
			IncludeCategoryFilter:  includeCategoryFilter,
			CategoryIds:            categoryIDs,
			IncludeAttributeFilter: includeAttributeFilter,
			AttributeIds:           attributeIDs,
			StartDate:              startDate,
			EndDate:                endDate,
			CursorCreatedAt:        gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorMatchTier:        db.NullTierInt64Param(cur.MatchTier),
			CursorID:               gosql.NullString{String: cur.ID, Valid: true},
			Limit:                  params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		materials := make([]*domain.Material, len(rows))
		for i, row := range rows {
			materials[i] = mapMaterialForwardBaseRow(row)
		}
		result, pageInfo := pagination.BuildPageStringWithSearchRank(materials, params.Limit, cursorDir, searchRankEnabled, materialCreatedAt, materialID, materialSearchRank)
		if apiErr := applyMaterialStitches(ctx, r.queries, result, params.Includes); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return &domain.ListMaterialsResult{Materials: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListMaterialsForwardBase(ctx, sqlc.ListMaterialsForwardBaseParams{
		AccountID:              params.AccountID,
		SearchQuery:            searchQuery,
		SearchExact:            catSearch.Exact,
		SearchPrefix:           catSearch.Prefix,
		IncludeCategoryFilter:  includeCategoryFilter,
		CategoryIds:            categoryIDs,
		IncludeAttributeFilter: includeAttributeFilter,
		AttributeIds:           attributeIDs,
		StartDate:              startDate,
		EndDate:                endDate,
		Limit:                  params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	materials := make([]*domain.Material, len(rows))
	for i, row := range rows {
		materials[i] = mapMaterialForwardBaseRow(row)
	}
	result, pageInfo := pagination.BuildPageStringWithSearchRank(materials, params.Limit, cursorDir, searchRankEnabled, materialCreatedAt, materialID, materialSearchRank)
	if apiErr := applyMaterialStitches(ctx, r.queries, result, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.ListMaterialsResult{Materials: result, PageInfo: pageInfo}, nil
}

func (r *materialRepoImpl) GetByID(ctx context.Context, params domain.GetMaterialParams) (*domain.Material, *apierror.APIError) {
	ctx, span := materialRepoTracer.Start(ctx, "repository.material.get_by_id")
	defer span.End()

	row, err := r.queries.GetMaterialByIDBase(ctx, sqlc.GetMaterialByIDBaseParams{
		ID:        params.MaterialID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	material := mapMaterialGetByIDBaseRow(row)

	if apiErr := applyMaterialStitches(ctx, r.queries, []*domain.Material{material}, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return material, nil
}

func (r *materialRepoImpl) GetByItemID(ctx context.Context, accountID, itemID string) (*domain.Material, *apierror.APIError) {
	ctx, span := materialRepoTracer.Start(ctx, "repository.material.get_by_item_id")
	defer span.End()

	row, err := r.queries.GetMaterialByItemID(ctx, sqlc.GetMaterialByItemIDParams{
		ItemID:    itemID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	material := mapGetMaterialByItemIDRow(row)
	return material, nil
}

func (r *materialRepoImpl) Create(ctx context.Context, id string, params domain.CreateMaterialParams) *apierror.APIError {
	ctx, span := materialRepoTracer.Start(ctx, "repository.material.create")
	defer span.End()

	err := r.queries.CreateMaterial(ctx, sqlc.CreateMaterialParams{
		ID:           id,
		ItemID:       params.ItemID,
		OrderPointID: params.OrderPointID,
		LeadTimeID:   params.LeadTimeID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *materialRepoImpl) Update(ctx context.Context, params domain.UpdateMaterialParams) *apierror.APIError {
	ctx, span := materialRepoTracer.Start(ctx, "repository.material.update")
	defer span.End()

	existing, apiErr := r.GetByID(ctx, domain.GetMaterialParams{AccountID: params.AccountID, MaterialID: params.MaterialID})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	_, err := r.queries.UpdateMaterial(ctx, existing.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *materialRepoImpl) DeleteByID(ctx context.Context, accountID, materialID string) *apierror.APIError {
	ctx, span := materialRepoTracer.Start(ctx, "repository.material.delete_by_id")
	defer span.End()

	result, err := r.queries.DeleteMaterialByID(ctx, sqlc.DeleteMaterialByIDParams{
		ID:        materialID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Material not found."))
	}

	return nil
}

func (r *materialRepoImpl) DeleteByItemID(ctx context.Context, accountID, itemID string) *apierror.APIError {
	material, apiErr := r.GetByItemID(ctx, accountID, itemID)
	if apiErr != nil {
		return apiErr
	}
	return r.DeleteByID(ctx, accountID, material.ID)
}

func (r *materialRepoImpl) InsertQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError {
	ctx, span := materialRepoTracer.Start(ctx, "repository.material.insert_quantity")
	defer span.End()

	err := r.queries.MaterialInsertQuantity(ctx, sqlc.MaterialInsertQuantityParams{
		ID:     id,
		Value:  value,
		UnitID: unitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *materialRepoImpl) UpdateQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError {
	ctx, span := materialRepoTracer.Start(ctx, "repository.material.update_quantity")
	defer span.End()

	_, err := r.queries.MaterialUpdateQuantity(ctx, sqlc.MaterialUpdateQuantityParams{
		Value:  value,
		UnitID: unitID,
		ID:     id,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *materialRepoImpl) InsertRate(ctx context.Context, id, value, numeratorUnitID, denominatorUnitID string) *apierror.APIError {
	ctx, span := materialRepoTracer.Start(ctx, "repository.material.insert_rate")
	defer span.End()

	err := r.queries.MaterialInsertRate(ctx, sqlc.MaterialInsertRateParams{
		ID:                id,
		Value:             value,
		NumeratorUnitID:   numeratorUnitID,
		DenominatorUnitID: denominatorUnitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *materialRepoImpl) InsertItem(ctx context.Context, id string, params domain.CreateMaterialParams) *apierror.APIError {
	ctx, span := materialRepoTracer.Start(ctx, "repository.material.insert_item")
	defer span.End()

	err := r.queries.MaterialInsertItem(ctx, sqlc.MaterialInsertItemParams{
		ID:             id,
		Sku:            params.SKU,
		Description:    toNullString(params.Description),
		Notes:          toNullString(params.Notes),
		ItemCategoryID: params.CategoryID,
		UnitValueID:    params.UnitValueRateID,
		UnitCostID:     params.UnitCostRateID,
		BurnRateID:     params.BurnRateRateID,
		AccountID:      params.AccountID,
		IsDirty:        false,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func mapMaterialExportRow(row sqlc.ExportMaterialsWithFiltersRow) *domain.Material {
	return &domain.Material{
		ID:     row.ID,
		ItemID: row.ItemID,
		Item:   mapMaterialBaseItem(row.ItemID, row.Sku, row.ItemDescription, row.ItemNotes, row.ItemTypeCode, row.ItemCategoryID, row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID, row.UnitValueID, row.UnitCostID, row.BurnRateID, row.AccountID, row.IsDirty, row.ItemCreatedAt, row.ItemUpdatedAt, row.CategoryCreatedAt, row.CategoryUpdatedAt),
		OrderPoint: &domain.Quantity{
			ID:               row.OrderPointID,
			Value:            row.OrderPointValue,
			UnitID:           row.OrderPointUnitID,
			UnitAbbreviation: row.OrderPointUnitAbbreviation,
			UnitType:         row.OrderPointUnitType,
		},
		LeadTime: &domain.Quantity{
			ID:               row.LeadTimeID,
			Value:            row.LeadTimeValue,
			UnitID:           row.LeadTimeUnitID,
			UnitAbbreviation: row.LeadTimeUnitAbbreviation,
			UnitType:         row.LeadTimeUnitType,
		},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func (r *materialRepoImpl) Export(ctx context.Context, params domain.ExportMaterialsParams) ([]*domain.Material, *apierror.APIError) {
	ctx, span := materialRepoTracer.Start(ctx, "repository.material.export")
	defer span.End()

	catSearch := db.NewCatalogSearch(params.Query)
	searchQuery := catSearch.Contains

	includeCategoryFilter := len(params.CategoryIDs) > 0
	includeAttributeFilter := len(params.AttributeIDs) > 0

	categoryIDs := params.CategoryIDs
	if categoryIDs == nil {
		categoryIDs = []string{}
	}
	attributeIDs := params.AttributeIDs
	if attributeIDs == nil {
		attributeIDs = []string{}
	}

	var startDate, endDate gosql.NullTime
	if params.StartDate != nil {
		startDate = gosql.NullTime{Time: *params.StartDate, Valid: true}
	}
	if params.EndDate != nil {
		endDate = gosql.NullTime{Time: *params.EndDate, Valid: true}
	}

	rows, err := r.queries.ExportMaterialsWithFilters(ctx, sqlc.ExportMaterialsWithFiltersParams{
		AccountID:              params.AccountID,
		SearchQuery:            searchQuery,
		SearchExact:            catSearch.Exact,
		SearchPrefix:           catSearch.Prefix,
		IncludeCategoryFilter:  includeCategoryFilter,
		CategoryIds:            categoryIDs,
		IncludeAttributeFilter: includeAttributeFilter,
		AttributeIds:           attributeIDs,
		StartDate:              startDate,
		EndDate:                endDate,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	materials := make([]*domain.Material, len(rows))
	for i, row := range rows {
		materials[i] = mapMaterialExportRow(row)
	}

	exportIncludes := []string{"item", "item.category", "item.category.properties", "item.unit_value", "item.unit_cost", "item.attributes"}
	if apiErr := applyMaterialStitches(ctx, r.queries, materials, exportIncludes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return materials, nil
}

func (r *materialRepoImpl) UpdateItem(ctx context.Context, params domain.UpdateMaterialParams) *apierror.APIError {
	ctx, span := materialRepoTracer.Start(ctx, "repository.material.update_item")
	defer span.End()

	existing, apiErr := r.GetByID(ctx, domain.GetMaterialParams{AccountID: params.AccountID, MaterialID: params.MaterialID})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	_, err := r.queries.MaterialUpdateItem(ctx, sqlc.MaterialUpdateItemParams{
		Sku:               toNullString(params.SKU),
		UpdateDescription: params.UpdateDescription,
		Description:       toNullString(params.Description),
		UpdateNotes:       params.UpdateNotes,
		Notes:             toNullString(params.Notes),
		ID:                existing.ItemID,
		AccountID:         params.AccountID,
	})
	if apiErr = db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
