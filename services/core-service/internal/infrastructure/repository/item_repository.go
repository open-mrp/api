package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var itemRepoTracer = tracing.GetTracer("core-service.item_repository")

type itemRepoImpl struct {
	queries *sqlc.Queries
}

func NewItemRepo(queries *sqlc.Queries) domain.ItemRepo {
	return &itemRepoImpl{queries: queries}
}

func itemCreatedAt(i *domain.Item) time.Time { return i.CreatedAt }
func itemID(i *domain.Item) string           { return i.ID }

// mapItemBaseRow maps a lightweight base query row (no rates, no unit group details).
func mapItemBaseRow(id, sku string, description, notes gosql.NullString, itemTypeCode, itemCategoryID, categoryName, itemCategoryTypeCode, categoryUnitGroupID, unitValueID, unitCostID, burnRateID, accountID string, isDirty bool, createdAt, updatedAt, categoryCreatedAt, categoryUpdatedAt time.Time) *domain.Item {
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

func mapItemForwardBaseRow(row sqlc.ListItemsForwardBaseRow) *domain.Item {
	return mapItemBaseRow(
		row.ID, row.Sku, row.Description, row.Notes,
		row.ItemTypeCode, row.ItemCategoryID, row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID,
		row.UnitValueID, row.UnitCostID, row.BurnRateID, row.AccountID,
		row.IsDirty, row.CreatedAt, row.UpdatedAt, row.CategoryCreatedAt, row.CategoryUpdatedAt,
	)
}

func mapItemBackwardBaseRow(row sqlc.ListItemsBackwardBaseRow) *domain.Item {
	return mapItemBaseRow(
		row.ID, row.Sku, row.Description, row.Notes,
		row.ItemTypeCode, row.ItemCategoryID, row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID,
		row.UnitValueID, row.UnitCostID, row.BurnRateID, row.AccountID,
		row.IsDirty, row.CreatedAt, row.UpdatedAt, row.CategoryCreatedAt, row.CategoryUpdatedAt,
	)
}

func mapGetItemBaseRow(row sqlc.GetItemBaseRow) *domain.Item {
	return mapItemBaseRow(
		row.ID, row.Sku, row.Description, row.Notes,
		row.ItemTypeCode, row.ItemCategoryID, row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID,
		row.UnitValueID, row.UnitCostID, row.BurnRateID, row.AccountID,
		row.IsDirty, row.CreatedAt, row.UpdatedAt, row.CategoryCreatedAt, row.CategoryUpdatedAt,
	)
}

func mapRateRow(row sqlc.GetRatesByIDsRow) *domain.Rate {
	return &domain.Rate{
		ID:                               row.ID,
		Value:                            row.Value,
		NumeratorUnitID:                  row.NumeratorUnitID,
		NumeratorUnitName:                row.NumeratorUnitName,
		NumeratorUnitAbbreviation:        row.NumeratorUnitAbbreviation,
		NumeratorUnitType:                row.NumeratorUnitType,
		NumeratorUnitRatioNumerator:      row.NumeratorUnitRatioNumerator,
		NumeratorUnitRatioDenominator:    row.NumeratorUnitRatioDenominator,
		NumeratorUnitOffsetNumerator:     row.NumeratorUnitOffsetNumerator,
		NumeratorUnitOffsetDenominator:   row.NumeratorUnitOffsetDenominator,
		NumeratorUnitCreatedAt:           row.NumeratorUnitCreatedAt,
		NumeratorUnitUpdatedAt:           row.NumeratorUnitUpdatedAt,
		DenominatorUnitID:                row.DenominatorUnitID,
		DenominatorUnitName:              row.DenominatorUnitName,
		DenominatorUnitAbbreviation:      row.DenominatorUnitAbbreviation,
		DenominatorUnitType:              row.DenominatorUnitType,
		DenominatorUnitRatioNumerator:    row.DenominatorUnitRatioNumerator,
		DenominatorUnitRatioDenominator:  row.DenominatorUnitRatioDenominator,
		DenominatorUnitOffsetNumerator:   row.DenominatorUnitOffsetNumerator,
		DenominatorUnitOffsetDenominator: row.DenominatorUnitOffsetDenominator,
		DenominatorUnitCreatedAt:         row.DenominatorUnitCreatedAt,
		DenominatorUnitUpdatedAt:         row.DenominatorUnitUpdatedAt,
		CreatedAt:                        row.CreatedAt,
		UpdatedAt:                        row.UpdatedAt,
	}
}

// extractItemIncludes strips an "item." prefix from each include key and returns
// the resulting slice. This lets material/part/product stitch functions pass
// includes like "item.burn_rate" to the shared item stitch helpers that expect
// bare keys like "burn_rate".
func extractItemIncludes(incs []string) []string {
	const prefix = "item."
	out := make([]string, 0, len(incs))
	for _, inc := range incs {
		if after, ok := strings.CutPrefix(inc, prefix); ok {
			out = append(out, after)
		}
	}
	return out
}

// stitchItemRates fetches rates for the given items and attaches them when the
// caller has requested at least one rate include (unit_value, unit_cost, burn_rate).
func stitchItemRates(ctx context.Context, queries *sqlc.Queries, items []*domain.Item, incs []string) *apierror.APIError {
	wantUnitValue := slices.Contains(incs, "unit_value")
	wantUnitCost := slices.Contains(incs, "unit_cost")
	wantBurnRate := slices.Contains(incs, "burn_rate")

	if !wantUnitValue && !wantUnitCost && !wantBurnRate {
		return nil
	}

	// Collect the rate IDs we need.
	seen := make(map[string]struct{})
	var rateIDs []string
	for _, item := range items {
		if wantUnitValue {
			if _, ok := seen[item.UnitValueID]; !ok {
				seen[item.UnitValueID] = struct{}{}
				rateIDs = append(rateIDs, item.UnitValueID)
			}
		}
		if wantUnitCost {
			if _, ok := seen[item.UnitCostID]; !ok {
				seen[item.UnitCostID] = struct{}{}
				rateIDs = append(rateIDs, item.UnitCostID)
			}
		}
		if wantBurnRate {
			if _, ok := seen[item.BurnRateID]; !ok {
				seen[item.BurnRateID] = struct{}{}
				rateIDs = append(rateIDs, item.BurnRateID)
			}
		}
	}

	if len(rateIDs) == 0 {
		return nil
	}

	rows, err := queries.GetRatesByIDs(ctx, rateIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	rateMap := make(map[string]*domain.Rate, len(rows))
	for _, row := range rows {
		rateMap[row.ID] = mapRateRow(row)
	}

	for _, item := range items {
		if wantUnitValue {
			item.UnitValue = rateMap[item.UnitValueID]
		}
		if wantUnitCost {
			item.UnitCost = rateMap[item.UnitCostID]
		}
		if wantBurnRate {
			item.BurnRate = rateMap[item.BurnRateID]
		}
	}
	return nil
}

// wantsUnitGroup returns true when the includes contain "category.unit_group"
// or any deeper path starting with "category.unit_group.".
func wantsUnitGroup(incs []string) bool {
	for _, inc := range incs {
		if inc == "category.unit_group" || strings.HasPrefix(inc, "category.unit_group.") {
			return true
		}
	}
	return false
}

// stitchItemCategoryUnitGroups fetches unit group details and attaches them to
// item categories when the caller has requested the category.unit_group include
// or any sub-include under it.
func stitchItemCategoryUnitGroups(ctx context.Context, queries *sqlc.Queries, items []*domain.Item, incs []string) *apierror.APIError {
	if !wantsUnitGroup(incs) {
		return nil
	}

	seen := make(map[string]struct{})
	var ugIDs []string
	for _, item := range items {
		if item.Category == nil {
			continue
		}
		if _, ok := seen[item.Category.UnitGroupID]; !ok {
			seen[item.Category.UnitGroupID] = struct{}{}
			ugIDs = append(ugIDs, item.Category.UnitGroupID)
		}
	}

	if len(ugIDs) == 0 {
		return nil
	}

	rows, err := queries.GetUnitGroupsByIDs(ctx, ugIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	ugMap := make(map[string]sqlc.GetUnitGroupsByIDsRow, len(rows))
	for _, row := range rows {
		ugMap[row.ID] = row
	}

	for _, item := range items {
		if item.Category == nil {
			continue
		}
		if ug, ok := ugMap[item.Category.UnitGroupID]; ok {
			item.Category.UnitGroupName = ug.Name
			item.Category.UnitGroupTypeCode = ug.UnitTypeCode
			item.Category.UnitGroupBaseUnitID = ug.BaseUnitID
			item.Category.UnitGroupCreatedAt = ug.CreatedAt
			item.Category.UnitGroupUpdatedAt = ug.UpdatedAt
		}
	}

	wantBaseUnit := slices.Contains(incs, "category.unit_group.base_unit")
	wantAssocUnits := slices.Contains(incs, "category.unit_group.associated_units")

	if wantBaseUnit {
		if apiErr := stitchItemCategoryUnitGroupBaseUnits(ctx, queries, items); apiErr != nil {
			return apiErr
		}
	}

	if wantAssocUnits {
		if apiErr := stitchItemCategoryAssociatedUnits(ctx, queries, items, incs); apiErr != nil {
			return apiErr
		}
	}

	return nil
}

// stitchItemCategoryUnitGroupBaseUnits batch-fetches the base unit for each
// item's category unit group and populates UnitGroupBaseUnit.
func stitchItemCategoryUnitGroupBaseUnits(ctx context.Context, queries *sqlc.Queries, items []*domain.Item) *apierror.APIError {
	seen := make(map[string]struct{})
	var baseUnitIDs []string
	for _, item := range items {
		if item.Category == nil || item.Category.UnitGroupBaseUnitID == "" {
			continue
		}
		if _, ok := seen[item.Category.UnitGroupBaseUnitID]; !ok {
			seen[item.Category.UnitGroupBaseUnitID] = struct{}{}
			baseUnitIDs = append(baseUnitIDs, item.Category.UnitGroupBaseUnitID)
		}
	}
	if len(baseUnitIDs) == 0 {
		return nil
	}

	rows, err := queries.GetUnitsByIDs(ctx, baseUnitIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}

	unitMap := make(map[string]domain.LightUnit, len(rows))
	for _, row := range rows {
		lu := mapGetUnitsByIDsRowToLightUnit(row)
		unitMap[lu.ID] = lu
	}

	for _, item := range items {
		if item.Category == nil {
			continue
		}
		if lu, ok := unitMap[item.Category.UnitGroupBaseUnitID]; ok {
			item.Category.UnitGroupBaseUnit = &lu
		}
	}
	return nil
}

// mapUnitGroupUnitsByUnitGroupIDsRow converts a sqlc row into a domain.UnitGroupUnit
// with its Unit sub-resource already populated (the query already joins unit).
func mapUnitGroupUnitsByUnitGroupIDsRow(row sqlc.ListUnitGroupUnitsByUnitGroupIDsRow) *domain.UnitGroupUnit {
	var acctID *string
	if row.UnitAccountID.Valid {
		acctID = &row.UnitAccountID.String
	}
	return &domain.UnitGroupUnit{
		ID:                 row.ID,
		UnitID:             row.UnitID,
		UnitGroupID:        row.UnitGroupID,
		DiscountPercentage: row.DiscountPercentage,
		DiscountFixed:      row.DiscountFixed,
		IsVisible:          row.IsVisible,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
		Unit: domain.LightUnit{
			ID:                row.UnitID,
			Name:              row.UnitName,
			Abbreviation:      row.UnitAbbreviation,
			Type:              row.UnitType,
			RatioNumerator:    row.UnitRatioNumerator,
			RatioDenominator:  row.UnitRatioDenominator,
			OffsetNumerator:   row.UnitOffsetNumerator,
			OffsetDenominator: row.UnitOffsetDenominator,
			IsBaseUnit:        row.UnitIsBaseUnit,
			AccountID:         acctID,
			CreatedAt:         row.UnitCreatedAt,
			UpdatedAt:         row.UnitUpdatedAt,
		},
	}
}

// stitchItemCategoryAssociatedUnits batch-fetches unit group units for all
// unique unit group IDs across the items and populates UnitGroupAssociatedUnits.
func stitchItemCategoryAssociatedUnits(ctx context.Context, queries *sqlc.Queries, items []*domain.Item, incs []string) *apierror.APIError {
	seen := make(map[string]struct{})
	var ugIDs []string
	for _, item := range items {
		if item.Category == nil || item.Category.UnitGroupID == "" {
			continue
		}
		if _, ok := seen[item.Category.UnitGroupID]; !ok {
			seen[item.Category.UnitGroupID] = struct{}{}
			ugIDs = append(ugIDs, item.Category.UnitGroupID)
		}
	}
	if len(ugIDs) == 0 {
		return nil
	}

	rows, err := queries.ListUnitGroupUnitsByUnitGroupIDs(ctx, ugIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}

	byUGID := make(map[string][]*domain.UnitGroupUnit)
	for _, row := range rows {
		ugu := mapUnitGroupUnitsByUnitGroupIDsRow(row)
		byUGID[row.UnitGroupID] = append(byUGID[row.UnitGroupID], ugu)
	}

	for _, item := range items {
		if item.Category == nil {
			continue
		}
		item.Category.UnitGroupAssociatedUnits = byUGID[item.Category.UnitGroupID]
	}
	return nil
}

func loadItemAttributes(ctx context.Context, queries *sqlc.Queries, item *domain.Item) *apierror.APIError {
	rows, err := queries.GetItemAttributes(ctx, item.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	attrs := make([]*domain.ItemAttribute, len(rows))
	for i, row := range rows {
		var colorCode *string
		if row.ColorCode != "" {
			colorCode = &row.ColorCode
		}
		attrs[i] = &domain.ItemAttribute{
			ID:         row.ID,
			Value:      row.Text,
			ColorCode:  colorCode,
			Order:      row.Order,
			PropertyID: row.PropertyID,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		}
	}
	item.Attributes = attrs
	return nil
}

// stitchItemAttributes batch-loads attributes for all items in a single query
// when the caller has requested the attributes include.
func stitchItemAttributes(ctx context.Context, queries *sqlc.Queries, items []*domain.Item, incs []string) *apierror.APIError {
	if !slices.Contains(incs, "attributes") {
		return nil
	}

	seen := make(map[string]struct{})
	var itemIDs []string
	for _, item := range items {
		if _, ok := seen[item.ID]; !ok {
			seen[item.ID] = struct{}{}
			itemIDs = append(itemIDs, item.ID)
		}
	}
	if len(itemIDs) == 0 {
		return nil
	}

	rows, err := queries.GetItemAttributesByItemIDs(ctx, itemIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}

	byItemID := make(map[string][]*domain.ItemAttribute, len(items))
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

	for _, item := range items {
		if attrs, ok := byItemID[item.ID]; ok {
			item.Attributes = attrs
		} else {
			item.Attributes = []*domain.ItemAttribute{}
		}
	}
	return nil
}

// applyItemStitches runs all conditional stitch queries for the given includes.
func applyItemStitches(ctx context.Context, queries *sqlc.Queries, items []*domain.Item, incs []string) *apierror.APIError {
	if apiErr := stitchItemRates(ctx, queries, items, incs); apiErr != nil {
		return apiErr
	}
	if apiErr := stitchItemCategoryUnitGroups(ctx, queries, items, incs); apiErr != nil {
		return apiErr
	}
	if apiErr := stitchItemAttributes(ctx, queries, items, incs); apiErr != nil {
		return apiErr
	}
	if slices.Contains(incs, "category.properties") {
		if apiErr := enrichItemCategoryProperties(ctx, queries, items); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

func (r *itemRepoImpl) List(ctx context.Context, params domain.ListItemsParams) (*domain.ListItemsResult, *apierror.APIError) {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.list")
	defer span.End()

	catSearch := db.NewCatalogSearch(params.Query)
	searchQuery := catSearch.Contains
	searchExact := catSearch.Exact
	searchRankEnabled := catSearch.Contains.Valid
	itemSearchRank := func(it *domain.Item) int32 {
		return db.CatalogSearchRank(it.SKU, catSearch)
	}
	includeTypeFilter := len(params.Types) > 0
	includeCategoryFilter := len(params.CategoryIDs) > 0
	includeAttributeFilter := len(params.AttributeIDs) > 0
	includeProductLineFilter := len(params.ProductLineIDs) > 0
	includeCustomerFilter := len(params.CustomerIDs) > 0

	types := params.Types
	if types == nil {
		types = []string{}
	}
	categoryIDs := params.CategoryIDs
	if categoryIDs == nil {
		categoryIDs = []string{}
	}
	attributeIDs := params.AttributeIDs
	if attributeIDs == nil {
		attributeIDs = []string{}
	}
	// product_line_ids column is nullable so sqlc generates []sql.NullString for this slice.
	productLineIDs := make([]gosql.NullString, len(params.ProductLineIDs))
	for i, id := range params.ProductLineIDs {
		productLineIDs[i] = gosql.NullString{String: id, Valid: true}
	}
	customerIDs := params.CustomerIDs
	if customerIDs == nil {
		customerIDs = []string{}
	}

	var startDate, endDate gosql.NullTime
	if params.StartDate != nil {
		startDate = gosql.NullTime{Time: *params.StartDate, Valid: true}
	}
	if params.EndDate != nil {
		endDate = gosql.NullTime{Time: *params.EndDate, Valid: true}
	}

	supplierID := gosql.NullString{}
	if params.SupplierID != nil {
		supplierID = gosql.NullString{String: *params.SupplierID, Valid: true}
	}

	var cursorDir *pagination.Direction
	var items []*domain.Item

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListItemsBackwardBase(ctx, sqlc.ListItemsBackwardBaseParams{
				AccountID:                params.AccountID,
				IncludeTypeFilter:        includeTypeFilter,
				ItemTypeCodes:            types,
				IncludeCategoryFilter:    includeCategoryFilter,
				CategoryIds:              categoryIDs,
				IncludeAttributeFilter:   includeAttributeFilter,
				AttributeIds:             attributeIDs,
				SupplierID:               supplierID,
				StartDate:                startDate,
				EndDate:                  endDate,
				SearchQuery:              searchQuery,
				SearchExact:              searchExact,
				SkuExactForMatch:         searchExact,
				SearchPrefix:             catSearch.Prefix,
				IsExactMatch:             params.IsExactMatch,
				OnlyInitialSubassemblies: params.OnlyInitialSubassemblies,
				IncludeProductLineFilter: includeProductLineFilter,
				ProductLineIds:           productLineIDs,
				IncludeCustomerFilter:    includeCustomerFilter,
				CustomerIds:              customerIDs,
				CursorMatchTier:          db.NullTierInt64Param(cur.MatchTier),
				CursorCreatedAt:          cur.OccurredAt,
				CursorID:                 cur.ID,
				Limit:                    params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			items = make([]*domain.Item, len(rows))
			for i, row := range rows {
				items[i] = mapItemBackwardBaseRow(row)
			}
		} else {
			// Forward with cursor
			rows, err := r.queries.ListItemsForwardBase(ctx, sqlc.ListItemsForwardBaseParams{
				AccountID:                params.AccountID,
				IncludeTypeFilter:        includeTypeFilter,
				ItemTypeCodes:            types,
				IncludeCategoryFilter:    includeCategoryFilter,
				CategoryIds:              categoryIDs,
				IncludeAttributeFilter:   includeAttributeFilter,
				AttributeIds:             attributeIDs,
				SupplierID:               supplierID,
				StartDate:                startDate,
				EndDate:                  endDate,
				SearchQuery:              searchQuery,
				SearchExact:              searchExact,
				SkuExactForMatch:         searchExact,
				SearchPrefix:             catSearch.Prefix,
				IsExactMatch:             params.IsExactMatch,
				OnlyInitialSubassemblies: params.OnlyInitialSubassemblies,
				IncludeProductLineFilter: includeProductLineFilter,
				ProductLineIds:           productLineIDs,
				IncludeCustomerFilter:    includeCustomerFilter,
				CustomerIds:              customerIDs,
				CursorCreatedAt:          gosql.NullTime{Time: cur.OccurredAt, Valid: true},
				CursorMatchTier:          db.NullTierInt64Param(cur.MatchTier),
				CursorID:                 gosql.NullString{String: cur.ID, Valid: true},
				Limit:                    params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			items = make([]*domain.Item, len(rows))
			for i, row := range rows {
				items[i] = mapItemForwardBaseRow(row)
			}
		}
	} else {
		// No cursor — first page
		rows, err := r.queries.ListItemsForwardBase(ctx, sqlc.ListItemsForwardBaseParams{
			AccountID:                params.AccountID,
			IncludeTypeFilter:        includeTypeFilter,
			ItemTypeCodes:            types,
			IncludeCategoryFilter:    includeCategoryFilter,
			CategoryIds:              categoryIDs,
			IncludeAttributeFilter:   includeAttributeFilter,
			AttributeIds:             attributeIDs,
			SupplierID:               supplierID,
			StartDate:                startDate,
			EndDate:                  endDate,
			SearchQuery:              searchQuery,
			SearchExact:              searchExact,
			SkuExactForMatch:         searchExact,
			SearchPrefix:             catSearch.Prefix,
			IsExactMatch:             params.IsExactMatch,
			OnlyInitialSubassemblies: params.OnlyInitialSubassemblies,
			IncludeProductLineFilter: includeProductLineFilter,
			ProductLineIds:           productLineIDs,
			IncludeCustomerFilter:    includeCustomerFilter,
			CustomerIds:              customerIDs,
			Limit:                    params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items = make([]*domain.Item, len(rows))
		for i, row := range rows {
			items[i] = mapItemForwardBaseRow(row)
		}
	}

	result, pageInfo := pagination.BuildPageStringWithSearchRank(items, params.Limit, cursorDir, searchRankEnabled, itemCreatedAt, itemID, itemSearchRank)

	if apiErr := applyItemStitches(ctx, r.queries, result, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.ListItemsResult{Items: result, PageInfo: pageInfo}, nil
}

func (r *itemRepoImpl) LoadAttributes(ctx context.Context, item *domain.Item) *apierror.APIError {
	return loadItemAttributes(ctx, r.queries, item)
}

func (r *itemRepoImpl) Get(ctx context.Context, params domain.GetItemParams) (*domain.Item, *apierror.APIError) {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.get")
	defer span.End()

	row, err := r.queries.GetItemBase(ctx, sqlc.GetItemBaseParams{
		ID:        params.ItemID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	item := mapGetItemBaseRow(row)

	if apiErr := applyItemStitches(ctx, r.queries, []*domain.Item{item}, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return item, nil
}

func (r *itemRepoImpl) GetInventory(ctx context.Context, accountID, itemID string) (*domain.ItemInventory, *apierror.APIError) {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.get_inventory")
	defer span.End()

	row, err := r.queries.GetItemInventory(ctx, sqlc.GetItemInventoryParams{
		ItemID:    itemID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.ItemInventory{
		OnHand:             formatDecimal(row.OnHand),
		OnHandUnitID:       row.UnitID,
		Reserved:           formatDecimal(row.Reserved),
		ReservedUnitID:     row.UnitID,
		AvailableToPromise: row.AvailableToPromise,
		ATPUnitID:          row.UnitID,
		Short:              formatDecimal(row.Short),
		ShortUnitID:        row.UnitID,
		UnitAbbreviation:   row.UnitAbbreviation,
		UnitType:           row.UnitType,
	}, nil
}

func (r *itemRepoImpl) GetCostFlowConsumptions(ctx context.Context, stepID string) ([]domain.CostFlowConsumption, *apierror.APIError) {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.get_cost_flow_consumptions")
	defer span.End()

	rows, err := r.queries.GetCostFlowStepConsumptions(ctx, gosql.NullString{String: stepID, Valid: stepID != ""})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := make([]domain.CostFlowConsumption, 0, len(rows))
	for _, row := range rows {
		consQty, parseErr := decimal.NewFromString(row.ConsumptionQuantityValue)
		if parseErr != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(parseErr, "Failed to parse consumption quantity."))
		}
		wasteQty, parseErr := decimal.NewFromString(row.WasteQuantityValue)
		if parseErr != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(parseErr, "Failed to parse waste quantity."))
		}
		unitCost, parseErr := decimal.NewFromString(formatDecimal(row.ConsumedItemUnitCost))
		if parseErr != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(parseErr, "Failed to parse item unit cost."))
		}

		result = append(result, domain.CostFlowConsumption{
			ConsumedItemType:    row.ConsumedItemType,
			ConsumptionQuantity: consQty,
			WasteQuantity:       wasteQty,
			UnitCost:            unitCost,
		})
	}

	return result, nil
}

func (r *itemRepoImpl) UpdateUnitCost(ctx context.Context, accountID, itemID string, cost decimal.Decimal, denominatorUnitID string) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.update_unit_cost")
	defer span.End()

	err := r.queries.UpdateItemUnitCostRate(ctx, sqlc.UpdateItemUnitCostRateParams{
		Value:             cost.StringFixed(30),
		DenominatorUnitID: denominatorUnitID,
		ItemID:            itemID,
		AccountID:         accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	err = r.queries.ClearItemDirtyFlag(ctx, sqlc.ClearItemDirtyFlagParams{
		ItemID:    itemID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *itemRepoImpl) GetTrends(ctx context.Context, accountID, itemID, trendType string) (*domain.ItemTrends, *apierror.APIError) {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.get_trends")
	defer span.End()

	rows, err := r.queries.GetItemTrends(ctx, sqlc.GetItemTrendsParams{
		ItemID:    itemID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Deduplicate by day: keep only the earliest log entry per calendar day.
	// This matches the Dashboard behavior which iterates ASC-ordered logs and
	// takes the first entry it encounters for each unique day.
	seen := make(map[string]struct{})
	var points []*domain.ItemTrend
	for _, row := range rows {
		dayKey := row.Date.Format("2006-01-02")
		if _, exists := seen[dayKey]; exists {
			continue
		}
		seen[dayKey] = struct{}{}
		points = append(points, &domain.ItemTrend{
			Date:  row.Date,
			Value: row.Value,
		})
	}

	return &domain.ItemTrends{
		TrendType: trendType,
		Points:    points,
	}, nil
}

func (r *itemRepoImpl) ExportWithInventory(ctx context.Context, accountID string) (*domain.ExportItemsResult, *apierror.APIError) {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.export_with_inventory")
	defer span.End()

	rows, err := r.queries.ExportItemsWithInventory(ctx, sqlc.ExportItemsWithInventoryParams{
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]*domain.ExportItem, len(rows))
	for i, row := range rows {
		var description *string
		if row.Description.Valid {
			description = &row.Description.String
		}
		var notes *string
		if row.Notes.Valid {
			notes = &row.Notes.String
		}
		items[i] = &domain.ExportItem{
			Item: domain.Item{
				ID:             row.ID,
				SKU:            row.Sku,
				Description:    description,
				Notes:          notes,
				ItemTypeCode:   row.ItemTypeCode,
				ItemCategoryID: row.ItemCategoryID,
				CategoryName:   row.CategoryName,
				AccountID:      row.AccountID,
				CreatedAt:      row.CreatedAt,
				UpdatedAt:      row.UpdatedAt,
			},
			OnHandQuantity: formatDecimal(row.OnHandQuantity),
			OnHandUnitID:   row.OnHandUnitID,
		}
	}

	return &domain.ExportItemsResult{
		Items: items,
		Count: int64(len(items)),
	}, nil
}

func (r *itemRepoImpl) Update(ctx context.Context, params domain.UpdateItemParams) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.update")
	defer span.End()

	// Update SKU and/or description/notes via COALESCE (non-null fields only)
	sku := gosql.NullString{}
	if params.SKU != nil {
		sku = gosql.NullString{String: *params.SKU, Valid: true}
	}

	description := gosql.NullString{}
	if params.Description != nil {
		description = gosql.NullString{String: *params.Description, Valid: true}
	}

	notes := gosql.NullString{}
	if params.Notes != nil {
		notes = gosql.NullString{String: *params.Notes, Valid: true}
	}

	// Use COALESCE-based update for non-null values
	err := r.queries.UpdateItem(ctx, sqlc.UpdateItemParams{
		Sku:         sku,
		Description: description,
		Notes:       notes,
		ID:          params.ItemID,
		AccountID:   params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// If the caller explicitly set description to null, use the dedicated setter
	if params.UpdateDescription && params.Description == nil {
		err := r.queries.SetItemDescription(ctx, sqlc.SetItemDescriptionParams{
			Description: gosql.NullString{},
			ID:          params.ItemID,
			AccountID:   params.AccountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// If the caller explicitly set notes to null, use the dedicated setter
	if params.UpdateNotes && params.Notes == nil {
		err := r.queries.SetItemNotes(ctx, sqlc.SetItemNotesParams{
			Notes:     gosql.NullString{},
			ID:        params.ItemID,
			AccountID: params.AccountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func (r *itemRepoImpl) CheckSKUExists(ctx context.Context, accountID, sku, excludeID string) (bool, *apierror.APIError) {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.check_sku_exists")
	defer span.End()

	exists, err := r.queries.CheckItemSKUExists(ctx, sqlc.CheckItemSKUExistsParams{
		Sku:       sku,
		AccountID: accountID,
		ExcludeID: excludeID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exists, nil
}

func (r *itemRepoImpl) AddAttribute(ctx context.Context, params domain.AddItemAttributeParams) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.add_attribute")
	defer span.End()

	err := r.queries.AddItemAttribute(ctx, sqlc.AddItemAttributeParams{
		AttributeID: params.AttributeID,
		ItemID:      params.ItemID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *itemRepoImpl) RemoveAttribute(ctx context.Context, params domain.RemoveItemAttributeParams) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.remove_attribute")
	defer span.End()

	result, err := r.queries.RemoveItemAttribute(ctx, sqlc.RemoveItemAttributeParams{
		AttributeID: params.AttributeID,
		ItemID:      params.ItemID,
		AccountID:   params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Item attribute relationship not found."))
	}

	return nil
}

func (r *itemRepoImpl) ChangeCategory(ctx context.Context, params domain.ChangeItemCategoryParams) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.change_category")
	defer span.End()

	err := r.queries.ChangeItemCategory(ctx, sqlc.ChangeItemCategoryParams{
		CategoryID: params.CategoryID,
		ID:         params.ItemID,
		AccountID:  params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *itemRepoImpl) UpdateRateUnits(ctx context.Context, accountID, itemID, newUnitID string) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.update_rate_units")
	defer span.End()

	// Update unit_value denominator
	err := r.queries.UpdateItemRateUnitValue(ctx, sqlc.UpdateItemRateUnitValueParams{
		NewUnitID: newUnitID,
		ItemID:    itemID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Update unit_cost denominator
	err = r.queries.UpdateItemRateUnitCost(ctx, sqlc.UpdateItemRateUnitCostParams{
		NewUnitID: newUnitID,
		ItemID:    itemID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Update burn_rate numerator
	err = r.queries.UpdateItemRateBurnRate(ctx, sqlc.UpdateItemRateBurnRateParams{
		NewUnitID: newUnitID,
		ItemID:    itemID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *itemRepoImpl) UpdateMaterialOrderPointUnit(ctx context.Context, accountID, itemID, newUnitID string) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.update_material_order_point_unit")
	defer span.End()

	err := r.queries.UpdateMaterialOrderPointUnit(ctx, sqlc.UpdateMaterialOrderPointUnitParams{
		NewUnitID: newUnitID,
		ItemID:    itemID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *itemRepoImpl) UpdateConsumptionProductionQuantityUnits(ctx context.Context, accountID, itemID, newUnitID string) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.update_consumption_production_quantity_units")
	defer span.End()

	err := r.queries.UpdateItemConsumptionQuantityUnits(ctx, sqlc.UpdateItemConsumptionQuantityUnitsParams{
		NewUnitID: newUnitID,
		ItemID:    itemID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	err = r.queries.UpdateItemProductionQuantityUnits(ctx, sqlc.UpdateItemProductionQuantityUnitsParams{
		NewUnitID: newUnitID,
		ItemID:    itemID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *itemRepoImpl) GetCategoryBaseUnitID(ctx context.Context, categoryID string) (string, *apierror.APIError) {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.get_category_base_unit_id")
	defer span.End()

	baseUnitID, err := r.queries.GetCategoryBaseUnitID(ctx, categoryID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return baseUnitID, nil
}

func (r *itemRepoImpl) FindBySKU(ctx context.Context, accountID, sku string) (*string, *string, *apierror.APIError) {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.find_by_sku")
	defer span.End()

	row, err := r.queries.FindItemBySKU(ctx, sqlc.FindItemBySKUParams{
		Sku:       sku,
		AccountID: accountID,
	})
	if err != nil {
		if apiErr := db.MapSQLError(err); apierror.IsNotFound(apiErr) {
			return nil, nil, nil
		} else if apiErr != nil {
			return nil, nil, tracing.Trace(span, apiErr)
		}
	}
	return &row.ID, &row.UnitValueID, nil
}

func (r *itemRepoImpl) UpdateRateValue(ctx context.Context, rateID, value string) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.update_rate_value")
	defer span.End()

	valueStr := gosql.NullString{String: value, Valid: true}
	_, err := r.queries.UpdateRateByID(ctx, sqlc.UpdateRateByIDParams{
		ID:    rateID,
		Value: valueStr,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *itemRepoImpl) UpdateRate(ctx context.Context, rateID string, params domain.CreateRateParams) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.update_rate")
	defer span.End()

	_, err := r.queries.UpdateRateByID(ctx, sqlc.UpdateRateByIDParams{
		ID:                rateID,
		Value:             gosql.NullString{String: params.Value, Valid: true},
		NumeratorUnitID:   gosql.NullString{String: params.NumeratorUnitID, Valid: true},
		DenominatorUnitID: gosql.NullString{String: params.DenominatorUnitID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *itemRepoImpl) ClearItemDirtyFlag(ctx context.Context, accountID, itemID string) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.clear_dirty_flag")
	defer span.End()

	err := r.queries.ClearItemDirtyFlag(ctx, sqlc.ClearItemDirtyFlagParams{
		ItemID:    itemID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *itemRepoImpl) FetchItemsBySKU(ctx context.Context, accountID string, skus []string) ([]domain.ItemSKUInfo, *apierror.APIError) {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.fetch_items_by_sku")
	defer span.End()

	rows, err := r.queries.FetchItemsBySKU(ctx, sqlc.FetchItemsBySKUParams{
		AccountID: accountID,
		Skus:      skus,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := make([]domain.ItemSKUInfo, len(rows))
	for i, row := range rows {
		result[i] = domain.ItemSKUInfo{
			ItemID:     row.ItemID,
			SKU:        row.Sku,
			BaseUnitID: row.BaseUnitID,
		}
	}

	return result, nil
}

func formatDecimal(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%f", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
