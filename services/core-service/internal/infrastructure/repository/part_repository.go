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
	"github.com/augno/api/shared/patch"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var partRepoTracer = tracing.GetTracer("core-service.part_repository")

type partRepoImpl struct {
	queries *sqlc.Queries
}

func NewPartRepo(queries *sqlc.Queries) domain.PartRepo {
	return &partRepoImpl{queries: queries}
}

func partItemCreatedAt(p *domain.Part) time.Time { return p.Item.CreatedAt }
func partItemID(p *domain.Part) string           { return p.ItemID }

func mapPartBaseItem(id, sku string, description, notes gosql.NullString, itemTypeCode, itemCategoryID, categoryName, itemCategoryTypeCode, categoryUnitGroupID, unitValueID, unitCostID, burnRateID, accountID string, isDirty bool, createdAt, updatedAt, categoryCreatedAt, categoryUpdatedAt time.Time) *domain.Item {
	var descPtr *string
	if description.Valid {
		descPtr = &description.String
	}
	var notesPtr *string
	if notes.Valid {
		notesPtr = &notes.String
	}
	return &domain.Item{
		ID:             id,
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
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
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

func mapPartGetBaseRow(row sqlc.GetPartBaseRow) *domain.Part {
	return &domain.Part{
		ID:        row.PartID,
		ItemID:    row.ID,
		CreatedAt: row.PartCreatedAt,
		UpdatedAt: row.PartUpdatedAt,
		Item:      mapPartBaseItem(row.ID, row.Sku, row.Description, row.Notes, row.ItemTypeCode, row.ItemCategoryID, row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID, row.UnitValueID, row.UnitCostID, row.BurnRateID, row.AccountID, row.IsDirty, row.CreatedAt, row.UpdatedAt, row.CategoryCreatedAt, row.CategoryUpdatedAt),
	}
}

func mapPartForwardBaseRow(row sqlc.ListPartsForwardBaseRow) *domain.Part {
	return &domain.Part{
		ID:        row.PartID,
		ItemID:    row.ID,
		CreatedAt: row.PartCreatedAt,
		UpdatedAt: row.PartUpdatedAt,
		Item:      mapPartBaseItem(row.ID, row.Sku, row.Description, row.Notes, row.ItemTypeCode, row.ItemCategoryID, row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID, row.UnitValueID, row.UnitCostID, row.BurnRateID, row.AccountID, row.IsDirty, row.CreatedAt, row.UpdatedAt, row.CategoryCreatedAt, row.CategoryUpdatedAt),
	}
}

func mapPartBackwardBaseRow(row sqlc.ListPartsBackwardBaseRow) *domain.Part {
	return &domain.Part{
		ID:        row.PartID,
		ItemID:    row.ID,
		CreatedAt: row.PartCreatedAt,
		UpdatedAt: row.PartUpdatedAt,
		Item:      mapPartBaseItem(row.ID, row.Sku, row.Description, row.Notes, row.ItemTypeCode, row.ItemCategoryID, row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID, row.UnitValueID, row.UnitCostID, row.BurnRateID, row.AccountID, row.IsDirty, row.CreatedAt, row.UpdatedAt, row.CategoryCreatedAt, row.CategoryUpdatedAt),
	}
}

func stitchPartAttributes(ctx context.Context, queries *sqlc.Queries, parts []*domain.Part, incs []string) *apierror.APIError {
	if !slices.Contains(incs, "attributes") {
		return nil
	}

	seen := make(map[string]struct{})
	var itemIDs []string
	for _, part := range parts {
		if part.Item == nil {
			continue
		}
		if _, ok := seen[part.ItemID]; !ok {
			seen[part.ItemID] = struct{}{}
			itemIDs = append(itemIDs, part.ItemID)
		}
	}
	if len(itemIDs) == 0 {
		return nil
	}

	rows, err := queries.GetItemAttributesByItemIDs(ctx, itemIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}

	byItemID := make(map[string][]*domain.ItemAttribute, len(parts))
	for _, row := range rows {
		var colorCode *string
		if row.ColorCode != "" {
			colorCode = &row.ColorCode
		}
		byItemID[row.ItemID] = append(byItemID[row.ItemID], &domain.ItemAttribute{
			ID:         row.ID,
			Value:      row.Text,
			ColorCode:  colorCode,
			Order:      row.Order,
			PropertyID: row.PropertyID,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		})
	}

	for _, part := range parts {
		if part.Item == nil {
			continue
		}
		if attrs, ok := byItemID[part.ItemID]; ok {
			part.Item.Attributes = attrs
		} else {
			part.Item.Attributes = []*domain.ItemAttribute{}
		}
	}
	return nil
}

func applyPartStitches(ctx context.Context, queries *sqlc.Queries, parts []*domain.Part, incs []string) *apierror.APIError {
	items := itemsFromParts(parts)
	itemIncs := extractItemIncludes(incs)
	if apiErr := stitchItemRates(ctx, queries, items, itemIncs); apiErr != nil {
		return apiErr
	}
	if apiErr := stitchItemCategoryUnitGroups(ctx, queries, items, itemIncs); apiErr != nil {
		return apiErr
	}
	if apiErr := stitchPartAttributes(ctx, queries, parts, itemIncs); apiErr != nil {
		return apiErr
	}
	if slices.Contains(itemIncs, "category.properties") {
		if apiErr := enrichItemCategoryProperties(ctx, queries, items); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

func (r *partRepoImpl) Create(ctx context.Context, partID, itemID string, params domain.CreatePartParams) (*domain.Part, *apierror.APIError) {
	ctx, span := partRepoTracer.Start(ctx, "repository.part.create")
	defer span.End()

	err := r.queries.InsertPart(ctx, sqlc.InsertPartParams{
		ID:     partID,
		ItemID: itemID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetPartParams{
		AccountID: params.AccountID,
		PartID:    partID,
		Includes:  params.Includes,
	})
}

func (r *partRepoImpl) Get(ctx context.Context, params domain.GetPartParams) (*domain.Part, *apierror.APIError) {
	ctx, span := partRepoTracer.Start(ctx, "repository.part.get")
	defer span.End()

	row, err := r.queries.GetPartBase(ctx, sqlc.GetPartBaseParams{
		PartID:    params.PartID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	part := mapPartGetBaseRow(row)

	if apiErr := applyPartStitches(ctx, r.queries, []*domain.Part{part}, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return part, nil
}

func (r *partRepoImpl) List(ctx context.Context, params domain.ListPartsParams) (*domain.ListPartsResult, *apierror.APIError) {
	ctx, span := partRepoTracer.Start(ctx, "repository.part.list")
	defer span.End()

	catSearch := db.NewCatalogSearch(params.Query)
	searchQuery := catSearch.Contains
	searchRankEnabled := catSearch.Contains.Valid
	partSearchRank := func(p *domain.Part) int32 {
		return db.CatalogSearchRank(p.Item.SKU, catSearch)
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
			rows, err := r.queries.ListPartsBackwardBase(ctx, sqlc.ListPartsBackwardBaseParams{
				AccountID:              params.AccountID,
				IncludeCategoryFilter:  includeCategoryFilter,
				CategoryIds:            categoryIDs,
				IncludeAttributeFilter: includeAttributeFilter,
				AttributeIds:           attributeIDs,
				StartDate:              startDate,
				EndDate:                endDate,
				SearchQuery:            searchQuery,
				SearchExact:            catSearch.Exact,
				SearchPrefix:           catSearch.Prefix,
				CursorMatchTier:        db.NullTierInt64Param(cur.MatchTier),
				CursorCreatedAt:        cur.OccurredAt,
				CursorID:               cur.ID,
				Limit:                  params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			parts := make([]*domain.Part, len(rows))
			for i, row := range rows {
				parts[i] = mapPartBackwardBaseRow(row)
			}
			result, pageInfo := pagination.BuildPageStringWithSearchRank(parts, params.Limit, cursorDir, searchRankEnabled, partItemCreatedAt, partItemID, partSearchRank)
			if apiErr := applyPartStitches(ctx, r.queries, result, params.Includes); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			return &domain.ListPartsResult{Parts: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListPartsForwardBase(ctx, sqlc.ListPartsForwardBaseParams{
			AccountID:              params.AccountID,
			IncludeCategoryFilter:  includeCategoryFilter,
			CategoryIds:            categoryIDs,
			IncludeAttributeFilter: includeAttributeFilter,
			AttributeIds:           attributeIDs,
			StartDate:              startDate,
			EndDate:                endDate,
			SearchQuery:            searchQuery,
			SearchExact:            catSearch.Exact,
			SearchPrefix:           catSearch.Prefix,
			CursorCreatedAt:        gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorMatchTier:        db.NullTierInt64Param(cur.MatchTier),
			CursorID:               gosql.NullString{String: cur.ID, Valid: true},
			Limit:                  params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		parts := make([]*domain.Part, len(rows))
		for i, row := range rows {
			parts[i] = mapPartForwardBaseRow(row)
		}
		result, pageInfo := pagination.BuildPageStringWithSearchRank(parts, params.Limit, cursorDir, searchRankEnabled, partItemCreatedAt, partItemID, partSearchRank)
		if apiErr := applyPartStitches(ctx, r.queries, result, params.Includes); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return &domain.ListPartsResult{Parts: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListPartsForwardBase(ctx, sqlc.ListPartsForwardBaseParams{
		AccountID:              params.AccountID,
		IncludeCategoryFilter:  includeCategoryFilter,
		CategoryIds:            categoryIDs,
		IncludeAttributeFilter: includeAttributeFilter,
		AttributeIds:           attributeIDs,
		StartDate:              startDate,
		EndDate:                endDate,
		SearchQuery:            searchQuery,
		SearchExact:            catSearch.Exact,
		SearchPrefix:           catSearch.Prefix,
		Limit:                  params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	parts := make([]*domain.Part, len(rows))
	for i, row := range rows {
		parts[i] = mapPartForwardBaseRow(row)
	}
	result, pageInfo := pagination.BuildPageStringWithSearchRank(parts, params.Limit, cursorDir, searchRankEnabled, partItemCreatedAt, partItemID, partSearchRank)
	if apiErr := applyPartStitches(ctx, r.queries, result, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.ListPartsResult{Parts: result, PageInfo: pageInfo}, nil
}

func (r *partRepoImpl) Delete(ctx context.Context, params domain.DeletePartParams) *apierror.APIError {
	ctx, span := partRepoTracer.Start(ctx, "repository.part.delete")
	defer span.End()

	err := r.queries.SoftDeletePart(ctx, sqlc.SoftDeletePartParams{
		PartID:    params.PartID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *partRepoImpl) ExistsBySKU(ctx context.Context, accountID, sku string, excludeItemID *string) (bool, *apierror.APIError) {
	ctx, span := partRepoTracer.Start(ctx, "repository.part.exists_by_sku")
	defer span.End()

	excludeID := ""
	if excludeItemID != nil {
		excludeID = *excludeItemID
	}

	exists, err := r.queries.CheckPartSKUExists(ctx, sqlc.CheckPartSKUExistsParams{
		Sku:       sku,
		AccountID: accountID,
		ExcludeID: excludeID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exists, nil
}

func (r *partRepoImpl) InsertRate(ctx context.Context, id, value, numeratorUnitID, denominatorUnitID string) *apierror.APIError {
	ctx, span := partRepoTracer.Start(ctx, "repository.part.insert_rate")
	defer span.End()

	err := r.queries.InsertRateForPart(ctx, sqlc.InsertRateForPartParams{
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

func (r *partRepoImpl) UpdateItem(ctx context.Context, params domain.PartUpdateItemParams) *apierror.APIError {
	ctx, span := partRepoTracer.Start(ctx, "repository.part.update_item")
	defer span.End()

	_, err := r.queries.PartUpdateItem(ctx, sqlc.PartUpdateItemParams{
		Sku:         toNullString(params.SKU),
		Description: patch.StringToNullString(params.Description),
		Notes:       patch.StringToNullString(params.Notes),
		ID:          params.ItemID,
		AccountID:   params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *partRepoImpl) TouchUpdatedAt(ctx context.Context, partID string) *apierror.APIError {
	ctx, span := partRepoTracer.Start(ctx, "repository.part.touch_updated_at")
	defer span.End()

	err := r.queries.TouchPartUpdatedAt(ctx, partID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func mapPartExportRow(row sqlc.ExportPartsWithFiltersRow) *domain.Part {
	return &domain.Part{
		ID:        row.PartID,
		ItemID:    row.ID,
		CreatedAt: row.PartCreatedAt,
		UpdatedAt: row.PartUpdatedAt,
		Item:      mapPartBaseItem(row.ID, row.Sku, row.Description, row.Notes, row.ItemTypeCode, row.ItemCategoryID, row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID, row.UnitValueID, row.UnitCostID, row.BurnRateID, row.AccountID, row.IsDirty, row.CreatedAt, row.UpdatedAt, row.CategoryCreatedAt, row.CategoryUpdatedAt),
	}
}

func (r *partRepoImpl) Export(ctx context.Context, params domain.ExportPartsParams) ([]*domain.Part, *apierror.APIError) {
	ctx, span := partRepoTracer.Start(ctx, "repository.part.export")
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

	rows, err := r.queries.ExportPartsWithFilters(ctx, sqlc.ExportPartsWithFiltersParams{
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

	parts := make([]*domain.Part, len(rows))
	for i, row := range rows {
		parts[i] = mapPartExportRow(row)
	}

	exportIncludes := []string{"item", "item.category", "item.category.properties", "item.unit_value", "item.unit_cost", "item.attributes"}
	if apiErr := applyPartStitches(ctx, r.queries, parts, exportIncludes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return parts, nil
}

func (r *partRepoImpl) InsertItem(ctx context.Context, itemID string, params domain.CreatePartParams, unitValueID, burnRateID, unitCostID string) *apierror.APIError {
	ctx, span := partRepoTracer.Start(ctx, "repository.part.insert_item")
	defer span.End()

	description := gosql.NullString{}
	if params.Description != nil {
		description = gosql.NullString{String: *params.Description, Valid: true}
	}
	notes := gosql.NullString{}
	if params.Notes != nil {
		notes = gosql.NullString{String: *params.Notes, Valid: true}
	}

	err := r.queries.InsertItemForPart(ctx, sqlc.InsertItemForPartParams{
		ID:             itemID,
		Sku:            params.SKU,
		Description:    description,
		Notes:          notes,
		UnitValueID:    unitValueID,
		BurnRateID:     burnRateID,
		AccountID:      params.AccountID,
		UnitCostID:     unitCostID,
		ItemCategoryID: params.CategoryID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
