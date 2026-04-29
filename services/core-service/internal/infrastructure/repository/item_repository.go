package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
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

func mapItemForwardRow(row sqlc.ListItemsForwardRow) *domain.Item {
	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}

	return &domain.Item{
		ID:             row.ID,
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
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
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
		},
	}
}

func mapItemBackwardRow(row sqlc.ListItemsBackwardRow) *domain.Item {
	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}

	return &domain.Item{
		ID:             row.ID,
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
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
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
		},
	}
}

func mapGetItemRow(row sqlc.GetItemRow) *domain.Item {
	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}

	return &domain.Item{
		ID:             row.ID,
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
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
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
		},
	}
}

func buildItemSearchParams(query *string) (gosql.NullString, gosql.NullString) {
	if query == nil || *query == "" {
		return gosql.NullString{}, gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}, gosql.NullString{String: *query, Valid: true}
}

func (r *itemRepoImpl) List(ctx context.Context, params domain.ListItemsParams) (*domain.ListItemsResult, *apierror.APIError) {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.list")
	defer span.End()

	searchQuery, searchExact := buildItemSearchParams(params.Query)
	includeTypeFilter := len(params.Types) > 0
	includeCategoryFilter := len(params.CategoryIDs) > 0
	includeAttributeFilter := len(params.AttributeIDs) > 0

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

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListItemsBackward(ctx, sqlc.ListItemsBackwardParams{
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
				IsExactMatch:             params.IsExactMatch,
				OnlyInitialSubassemblies: params.OnlyInitialSubassemblies,
				CursorCreatedAt:          cur.OccurredAt,
				CursorID:                 cur.ID,
				Limit:                    params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			items := make([]*domain.Item, len(rows))
			for i, row := range rows {
				items[i] = mapItemBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, itemCreatedAt, itemID)
			return &domain.ListItemsResult{Items: result, PageInfo: pageInfo}, nil
		}

		// Forward
		rows, err := r.queries.ListItemsForward(ctx, sqlc.ListItemsForwardParams{
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
			IsExactMatch:             params.IsExactMatch,
			OnlyInitialSubassemblies: params.OnlyInitialSubassemblies,
			CursorCreatedAt:          gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:                 gosql.NullString{String: cur.ID, Valid: true},
			Limit:                    params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items := make([]*domain.Item, len(rows))
		for i, row := range rows {
			items[i] = mapItemForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, itemCreatedAt, itemID)
		return &domain.ListItemsResult{Items: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListItemsForward(ctx, sqlc.ListItemsForwardParams{
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
		IsExactMatch:             params.IsExactMatch,
		OnlyInitialSubassemblies: params.OnlyInitialSubassemblies,
		Limit:                    params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	items := make([]*domain.Item, len(rows))
	for i, row := range rows {
		items[i] = mapItemForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, itemCreatedAt, itemID)
	return &domain.ListItemsResult{Items: result, PageInfo: pageInfo}, nil
}

func (r *itemRepoImpl) loadItemAttributes(ctx context.Context, item *domain.Item) *apierror.APIError {
	rows, err := r.queries.GetItemAttributes(ctx, item.ID)
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
			PropertyID: row.PropertyID,
		}
	}
	item.Attributes = attrs
	return nil
}

func (r *itemRepoImpl) Get(ctx context.Context, params domain.GetItemParams) (*domain.Item, *apierror.APIError) {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.get")
	defer span.End()

	row, err := r.queries.GetItem(ctx, sqlc.GetItemParams{
		ID:        params.ItemID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	item := mapGetItemRow(row)

	if apiErr := r.loadItemAttributes(ctx, item); apiErr != nil {
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

func formatDecimal(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%f", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
