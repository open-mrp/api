package repository

import (
	"context"
	gosql "database/sql"
	"slices"
	"strings"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/patch"
	"github.com/augno/api/shared/tracing"
)

var productRepoTracer = tracing.GetTracer("core-service.product_repository")

type productRepoImpl struct {
	queries *sqlc.Queries
}

func NewProductRepo(queries *sqlc.Queries) domain.ProductRepo {
	return &productRepoImpl{queries: queries}
}

func productFullCreatedAt(p *domain.ProductFull) time.Time { return p.CreatedAt }
func productFullID(p *domain.ProductFull) string           { return p.ID }

func mapProductFullFindRow(row sqlc.FindProductsBySKUsRow) *domain.ProductFull {
	var description *string
	if row.ItemDescription.Valid {
		description = &row.ItemDescription.String
	}
	var notes *string
	if row.ItemNotes.Valid {
		notes = &row.ItemNotes.String
	}
	var productLineID *string
	if row.ProductLineID.Valid {
		productLineID = &row.ProductLineID.String
	}

	product := &domain.ProductFull{
		ID:              row.ID,
		ProductTypeCode: row.ProductTypeCode,
		IsPortalReady:   row.IsPortalReady,
		ProductLineID:   productLineID,
		ItemID:          row.ItemID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
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
				ID:                               row.UnitValueRateID,
				Value:                            row.UnitValueRateValue,
				NumeratorUnitID:                  row.UnitValueNumeratorUnitID,
				NumeratorUnitName:                row.UnitValueNumeratorUnitName,
				NumeratorUnitAbbreviation:        row.UnitValueNumeratorUnitAbbreviation,
				NumeratorUnitType:                row.UnitValueNumeratorUnitType,
				NumeratorUnitRatioNumerator:      row.UnitValueNumeratorUnitRatioNumerator,
				NumeratorUnitRatioDenominator:    row.UnitValueNumeratorUnitRatioDenominator,
				NumeratorUnitOffsetNumerator:     row.UnitValueNumeratorUnitOffsetNumerator,
				NumeratorUnitOffsetDenominator:   row.UnitValueNumeratorUnitOffsetDenominator,
				NumeratorUnitCreatedAt:           row.UnitValueNumeratorUnitCreatedAt,
				NumeratorUnitUpdatedAt:           row.UnitValueNumeratorUnitUpdatedAt,
				DenominatorUnitID:                row.UnitValueDenominatorUnitID,
				DenominatorUnitName:              row.UnitValueDenominatorUnitName,
				DenominatorUnitAbbreviation:      row.UnitValueDenominatorUnitAbbreviation,
				DenominatorUnitType:              row.UnitValueDenominatorUnitType,
				DenominatorUnitRatioNumerator:    row.UnitValueDenominatorUnitRatioNumerator,
				DenominatorUnitRatioDenominator:  row.UnitValueDenominatorUnitRatioDenominator,
				DenominatorUnitOffsetNumerator:   row.UnitValueDenominatorUnitOffsetNumerator,
				DenominatorUnitOffsetDenominator: row.UnitValueDenominatorUnitOffsetDenominator,
				DenominatorUnitCreatedAt:         row.UnitValueDenominatorUnitCreatedAt,
				DenominatorUnitUpdatedAt:         row.UnitValueDenominatorUnitUpdatedAt,
				CreatedAt:                        row.UnitValueCreatedAt,
				UpdatedAt:                        row.UnitValueUpdatedAt,
			},
			UnitCost: &domain.Rate{
				ID:                               row.UnitCostRateID,
				Value:                            row.UnitCostRateValue,
				NumeratorUnitID:                  row.UnitCostNumeratorUnitID,
				NumeratorUnitName:                row.UnitCostNumeratorUnitName,
				NumeratorUnitAbbreviation:        row.UnitCostNumeratorUnitAbbreviation,
				NumeratorUnitType:                row.UnitCostNumeratorUnitType,
				NumeratorUnitRatioNumerator:      row.UnitCostNumeratorUnitRatioNumerator,
				NumeratorUnitRatioDenominator:    row.UnitCostNumeratorUnitRatioDenominator,
				NumeratorUnitOffsetNumerator:     row.UnitCostNumeratorUnitOffsetNumerator,
				NumeratorUnitOffsetDenominator:   row.UnitCostNumeratorUnitOffsetDenominator,
				NumeratorUnitCreatedAt:           row.UnitCostNumeratorUnitCreatedAt,
				NumeratorUnitUpdatedAt:           row.UnitCostNumeratorUnitUpdatedAt,
				DenominatorUnitID:                row.UnitCostDenominatorUnitID,
				DenominatorUnitName:              row.UnitCostDenominatorUnitName,
				DenominatorUnitAbbreviation:      row.UnitCostDenominatorUnitAbbreviation,
				DenominatorUnitType:              row.UnitCostDenominatorUnitType,
				DenominatorUnitRatioNumerator:    row.UnitCostDenominatorUnitRatioNumerator,
				DenominatorUnitRatioDenominator:  row.UnitCostDenominatorUnitRatioDenominator,
				DenominatorUnitOffsetNumerator:   row.UnitCostDenominatorUnitOffsetNumerator,
				DenominatorUnitOffsetDenominator: row.UnitCostDenominatorUnitOffsetDenominator,
				DenominatorUnitCreatedAt:         row.UnitCostDenominatorUnitCreatedAt,
				DenominatorUnitUpdatedAt:         row.UnitCostDenominatorUnitUpdatedAt,
				CreatedAt:                        row.UnitCostCreatedAt,
				UpdatedAt:                        row.UnitCostUpdatedAt,
			},
			BurnRate: &domain.Rate{
				ID:                               row.BurnRateIDJoined,
				Value:                            row.BurnRateValue,
				NumeratorUnitID:                  row.BurnRateNumeratorUnitID,
				NumeratorUnitName:                row.BurnRateNumeratorUnitName,
				NumeratorUnitAbbreviation:        row.BurnRateNumeratorUnitAbbreviation,
				NumeratorUnitType:                row.BurnRateNumeratorUnitType,
				NumeratorUnitRatioNumerator:      row.BurnRateNumeratorUnitRatioNumerator,
				NumeratorUnitRatioDenominator:    row.BurnRateNumeratorUnitRatioDenominator,
				NumeratorUnitOffsetNumerator:     row.BurnRateNumeratorUnitOffsetNumerator,
				NumeratorUnitOffsetDenominator:   row.BurnRateNumeratorUnitOffsetDenominator,
				NumeratorUnitCreatedAt:           row.BurnRateNumeratorUnitCreatedAt,
				NumeratorUnitUpdatedAt:           row.BurnRateNumeratorUnitUpdatedAt,
				DenominatorUnitID:                row.BurnRateDenominatorUnitID,
				DenominatorUnitName:              row.BurnRateDenominatorUnitName,
				DenominatorUnitAbbreviation:      row.BurnRateDenominatorUnitAbbreviation,
				DenominatorUnitType:              row.BurnRateDenominatorUnitType,
				DenominatorUnitRatioNumerator:    row.BurnRateDenominatorUnitRatioNumerator,
				DenominatorUnitRatioDenominator:  row.BurnRateDenominatorUnitRatioDenominator,
				DenominatorUnitOffsetNumerator:   row.BurnRateDenominatorUnitOffsetNumerator,
				DenominatorUnitOffsetDenominator: row.BurnRateDenominatorUnitOffsetDenominator,
				DenominatorUnitCreatedAt:         row.BurnRateDenominatorUnitCreatedAt,
				DenominatorUnitUpdatedAt:         row.BurnRateDenominatorUnitUpdatedAt,
				CreatedAt:                        row.BurnRateCreatedAt,
				UpdatedAt:                        row.BurnRateUpdatedAt,
			},
			Category: &domain.ItemCategory{
				ID:                   row.ItemCategoryID,
				Name:                 row.CategoryName,
				ItemCategoryTypeCode: row.ItemCategoryTypeCode,
				UnitGroupID:          row.CategoryUnitGroupID,
				CreatedAt:            row.CategoryCreatedAt,
				UpdatedAt:            row.CategoryUpdatedAt,
				UnitGroupName:        row.CategoryUnitGroupName,
				UnitGroupTypeCode:    row.CategoryUnitGroupType,
				UnitGroupCreatedAt:   row.CategoryUnitGroupCreatedAt,
				UnitGroupUpdatedAt:   row.CategoryUnitGroupUpdatedAt,
			},
		},
		ProductType: &domain.ProductType{
			ID:        row.ProductTypeID,
			Name:      row.ProductTypeName,
			Code:      row.ProductTypeCodeJoined,
			CreatedAt: row.ProductTypeCreatedAt,
			UpdatedAt: row.ProductTypeUpdatedAt,
		},
	}

	if row.ProductLineIDJoined.Valid {
		var plDescription *string
		if row.ProductLineDescription.Valid {
			plDescription = &row.ProductLineDescription.String
		}
		var plNotes *string
		if row.ProductLineNotes.Valid {
			plNotes = &row.ProductLineNotes.String
		}
		var plAccountID *string
		if row.ProductLineAccountID.Valid {
			plAccountID = &row.ProductLineAccountID.String
		}
		product.ProductLine = &domain.ProductLineFull{
			ID:               row.ProductLineIDJoined.String,
			Name:             row.ProductLineName.String,
			Description:      plDescription,
			Notes:            plNotes,
			CommissionPolicy: constants.CommissionPolicyFromBool(row.ProductLineIsCommissionExempt.Bool),
			FreightPolicy:    constants.FreightPolicyFromBool(row.ProductLineIsFreightExempt.Bool),
			UnitGroupID:      row.ProductLineUnitGroupID.String,
			AccountID:        plAccountID,
			CreatedAt:        row.ProductLineCreatedAt.Time,
			UpdatedAt:        row.ProductLineUpdatedAt.Time,
		}
	}

	return product
}

func (r *productRepoImpl) SearchBySKU(ctx context.Context, accountID, query string) ([]domain.ProductInfo, *apierror.APIError) {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.search_by_sku")
	defer span.End()

	rows, err := r.queries.SearchProductsBySKU(ctx, sqlc.SearchProductsBySKUParams{
		AccountID: accountID,
		Sku:       query,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	products := make([]domain.ProductInfo, len(rows))
	for i, row := range rows {
		products[i] = domain.ProductInfo{
			ProductID:   row.ProductID,
			ItemID:      row.ItemID,
			SKU:         row.Sku,
			Description: row.Description.String,
			UnitPrice:   row.UnitPrice,
		}
	}
	return products, nil
}

func (r *productRepoImpl) ListByAccount(ctx context.Context, accountID string) ([]domain.ProductInfo, *apierror.APIError) {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.list_by_account")
	defer span.End()

	rows, err := r.queries.ListProductsByAccount(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	products := make([]domain.ProductInfo, len(rows))
	for i, row := range rows {
		products[i] = domain.ProductInfo{
			ProductID:   row.ProductID,
			ItemID:      row.ItemID,
			SKU:         row.Sku,
			Description: row.Description.String,
			UnitPrice:   row.UnitPrice,
		}
	}
	return products, nil
}

func (r *productRepoImpl) GetSystemProduct(ctx context.Context, accountID, productTypeCode string) (*domain.SystemProductInfo, *apierror.APIError) {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.get_system_product")
	defer span.End()

	row, err := r.queries.GetAccountSystemProduct(ctx, sqlc.GetAccountSystemProductParams{
		ProductTypeCode: productTypeCode,
		AccountID:       accountID,
	})
	if err != nil {
		if apiErr := db.MapSQLError(err); apierror.IsNotFound(apiErr) {
			return nil, nil
		} else if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	return &domain.SystemProductInfo{
		ProductID:      row.ProductID,
		ProductSKU:     row.ProductSku,
		QuantityUnitID: row.QuantityUnitID,
	}, nil
}

func mapProductBaseItem(itemID, sku string, description, notes gosql.NullString, itemTypeCode, itemCategoryID, categoryName, itemCategoryTypeCode, categoryUnitGroupID, unitValueID, unitCostID, burnRateID, accountID string, isDirty bool, itemCreatedAt, itemUpdatedAt, categoryCreatedAt, categoryUpdatedAt time.Time) *domain.Item {
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

func mapProductBaseProductType(ptID, ptName, ptCode string, ptCreatedAt, ptUpdatedAt time.Time) *domain.ProductType {
	return &domain.ProductType{
		ID:        ptID,
		Name:      ptName,
		Code:      ptCode,
		CreatedAt: ptCreatedAt,
		UpdatedAt: ptUpdatedAt,
	}
}

func mapProductForwardBaseRow(row sqlc.ListProductsFullForwardBaseRow) *domain.ProductFull {
	var productLineID *string
	if row.ProductLineID.Valid {
		productLineID = &row.ProductLineID.String
	}
	return &domain.ProductFull{
		ID:              row.ID,
		ProductTypeCode: row.ProductTypeCode,
		IsPortalReady:   row.IsPortalReady,
		ProductLineID:   productLineID,
		ItemID:          row.ItemID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		Item:            mapProductBaseItem(row.ItemID, row.Sku, row.ItemDescription, row.ItemNotes, row.ItemTypeCode, row.ItemCategoryID, row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID, row.UnitValueID, row.UnitCostID, row.BurnRateID, row.AccountID, row.IsDirty, row.ItemCreatedAt, row.ItemUpdatedAt, row.CategoryCreatedAt, row.CategoryUpdatedAt),
		ProductType:     mapProductBaseProductType(row.ProductTypeID, row.ProductTypeName, row.ProductTypeCodeJoined, row.ProductTypeCreatedAt, row.ProductTypeUpdatedAt),
	}
}

func mapProductBackwardBaseRow(row sqlc.ListProductsFullBackwardBaseRow) *domain.ProductFull {
	var productLineID *string
	if row.ProductLineID.Valid {
		productLineID = &row.ProductLineID.String
	}
	return &domain.ProductFull{
		ID:              row.ID,
		ProductTypeCode: row.ProductTypeCode,
		IsPortalReady:   row.IsPortalReady,
		ProductLineID:   productLineID,
		ItemID:          row.ItemID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		Item:            mapProductBaseItem(row.ItemID, row.Sku, row.ItemDescription, row.ItemNotes, row.ItemTypeCode, row.ItemCategoryID, row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID, row.UnitValueID, row.UnitCostID, row.BurnRateID, row.AccountID, row.IsDirty, row.ItemCreatedAt, row.ItemUpdatedAt, row.CategoryCreatedAt, row.CategoryUpdatedAt),
		ProductType:     mapProductBaseProductType(row.ProductTypeID, row.ProductTypeName, row.ProductTypeCodeJoined, row.ProductTypeCreatedAt, row.ProductTypeUpdatedAt),
	}
}

func mapProductGetBaseRow(row sqlc.GetProductByIDBaseRow) *domain.ProductFull {
	var productLineID *string
	if row.ProductLineID.Valid {
		productLineID = &row.ProductLineID.String
	}
	return &domain.ProductFull{
		ID:              row.ID,
		ProductTypeCode: row.ProductTypeCode,
		IsPortalReady:   row.IsPortalReady,
		ProductLineID:   productLineID,
		ItemID:          row.ItemID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		Item:            mapProductBaseItem(row.ItemID, row.Sku, row.ItemDescription, row.ItemNotes, row.ItemTypeCode, row.ItemCategoryID, row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID, row.UnitValueID, row.UnitCostID, row.BurnRateID, row.AccountID, row.IsDirty, row.ItemCreatedAt, row.ItemUpdatedAt, row.CategoryCreatedAt, row.CategoryUpdatedAt),
		ProductType:     mapProductBaseProductType(row.ProductTypeID, row.ProductTypeName, row.ProductTypeCodeJoined, row.ProductTypeCreatedAt, row.ProductTypeUpdatedAt),
	}
}

func stitchProductLines(ctx context.Context, queries *sqlc.Queries, products []*domain.ProductFull, incs []string) *apierror.APIError {
	if !slices.Contains(incs, "product_line") {
		return nil
	}

	seen := make(map[string]struct{})
	var plIDs []string
	for _, p := range products {
		if p.ProductLineID == nil {
			continue
		}
		if _, ok := seen[*p.ProductLineID]; !ok {
			seen[*p.ProductLineID] = struct{}{}
			plIDs = append(plIDs, *p.ProductLineID)
		}
	}
	if len(plIDs) == 0 {
		return nil
	}

	rows, err := queries.GetProductLinesByIDs(ctx, plIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	plMap := make(map[string]sqlc.GetProductLinesByIDsRow, len(rows))
	for _, row := range rows {
		plMap[row.ID] = row
	}

	for _, p := range products {
		if p.ProductLineID == nil {
			continue
		}
		row, ok := plMap[*p.ProductLineID]
		if !ok {
			continue
		}
		var plDescription *string
		if row.Description.Valid {
			plDescription = &row.Description.String
		}
		var plNotes *string
		if row.Notes.Valid {
			plNotes = &row.Notes.String
		}
		var plAccountID *string
		if row.AccountID.Valid {
			plAccountID = &row.AccountID.String
		}
		p.ProductLine = &domain.ProductLineFull{
			ID:               row.ID,
			Name:             row.Name,
			Description:      plDescription,
			Notes:            plNotes,
			CommissionPolicy: constants.CommissionPolicyFromBool(row.IsCommissionExempt),
			FreightPolicy:    constants.FreightPolicyFromBool(row.IsFreightExempt),
			UnitGroupID:      row.UnitGroupID,
			AccountID:        plAccountID,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		}
	}
	return nil
}

func applyProductStitches(ctx context.Context, queries *sqlc.Queries, products []*domain.ProductFull, incs []string) *apierror.APIError {
	items := itemsFromProductFulls(products)
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
	if apiErr := stitchProductLines(ctx, queries, products, incs); apiErr != nil {
		return apiErr
	}
	return nil
}

func (r *productRepoImpl) List(ctx context.Context, params domain.ListProductsFullParams) (*domain.ListProductsFullResult, *apierror.APIError) {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.list")
	defer span.End()

	catSearch := db.NewCatalogSearch(params.Query)
	searchQuery := catSearch.Contains
	searchRankEnabled := catSearch.Contains.Valid
	productSearchRank := func(p *domain.ProductFull) int32 {
		return db.CatalogSearchRank(p.Item.SKU, catSearch)
	}
	includeProductLineFilter := len(params.ProductLineIDs) > 0
	includeCategoryFilter := len(params.CategoryIDs) > 0
	includeAttributeFilter := len(params.AttributeIDs) > 0
	includeCustomerFilter := len(params.CustomerIDs) > 0

	productLineIDs := toNullStringSlice(params.ProductLineIDs)

	categoryIDs := params.CategoryIDs
	if categoryIDs == nil {
		categoryIDs = []string{}
	}
	attributeIDs := params.AttributeIDs
	if attributeIDs == nil {
		attributeIDs = []string{}
	}
	customerIDs := params.CustomerIDs
	if customerIDs == nil {
		customerIDs = []string{}
	}

	var isPortalReady gosql.NullBool
	if params.IsPortalReady != nil {
		isPortalReady = gosql.NullBool{Bool: *params.IsPortalReady, Valid: true}
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
			rows, err := r.queries.ListProductsFullBackwardBase(ctx, sqlc.ListProductsFullBackwardBaseParams{
				AccountID:                params.AccountID,
				SearchQuery:              searchQuery,
				SearchExact:              catSearch.Exact,
				SearchPrefix:             catSearch.Prefix,
				IncludeProductLineFilter: includeProductLineFilter,
				ProductLineIds:           productLineIDs,
				IncludeCategoryFilter:    includeCategoryFilter,
				CategoryIds:              categoryIDs,
				IncludeAttributeFilter:   includeAttributeFilter,
				AttributeIds:             attributeIDs,
				IncludeCustomerFilter:    includeCustomerFilter,
				CustomerIds:              customerIDs,
				IsPortalReady:            isPortalReady,
				StartDate:                startDate,
				EndDate:                  endDate,
				CursorMatchTier:          db.NullTierInt64Param(cur.MatchTier),
				CursorCreatedAt:          cur.OccurredAt,
				CursorID:                 cur.ID,
				Limit:                    params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			items := make([]*domain.ProductFull, len(rows))
			for i, row := range rows {
				items[i] = mapProductBackwardBaseRow(row)
			}
			result, pageInfo := pagination.BuildPageStringWithSearchRank(items, params.Limit, cursorDir, searchRankEnabled, productFullCreatedAt, productFullID, productSearchRank)
			if apiErr := applyProductStitches(ctx, r.queries, result, params.Includes); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			return &domain.ListProductsFullResult{Products: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListProductsFullForwardBase(ctx, sqlc.ListProductsFullForwardBaseParams{
			AccountID:                params.AccountID,
			SearchQuery:              searchQuery,
			SearchExact:              catSearch.Exact,
			SearchPrefix:             catSearch.Prefix,
			IncludeProductLineFilter: includeProductLineFilter,
			ProductLineIds:           productLineIDs,
			IncludeCategoryFilter:    includeCategoryFilter,
			CategoryIds:              categoryIDs,
			IncludeAttributeFilter:   includeAttributeFilter,
			AttributeIds:             attributeIDs,
			IncludeCustomerFilter:    includeCustomerFilter,
			CustomerIds:              customerIDs,
			IsPortalReady:            isPortalReady,
			StartDate:                startDate,
			EndDate:                  endDate,
			CursorCreatedAt:          gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorMatchTier:          db.NullTierInt64Param(cur.MatchTier),
			CursorID:                 gosql.NullString{String: cur.ID, Valid: true},
			Limit:                    params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items := make([]*domain.ProductFull, len(rows))
		for i, row := range rows {
			items[i] = mapProductForwardBaseRow(row)
		}
		result, pageInfo := pagination.BuildPageStringWithSearchRank(items, params.Limit, cursorDir, searchRankEnabled, productFullCreatedAt, productFullID, productSearchRank)
		if apiErr := applyProductStitches(ctx, r.queries, result, params.Includes); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return &domain.ListProductsFullResult{Products: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListProductsFullForwardBase(ctx, sqlc.ListProductsFullForwardBaseParams{
		AccountID:                params.AccountID,
		SearchQuery:              searchQuery,
		SearchExact:              catSearch.Exact,
		SearchPrefix:             catSearch.Prefix,
		IncludeProductLineFilter: includeProductLineFilter,
		ProductLineIds:           productLineIDs,
		IncludeCategoryFilter:    includeCategoryFilter,
		CategoryIds:              categoryIDs,
		IncludeAttributeFilter:   includeAttributeFilter,
		AttributeIds:             attributeIDs,
		IncludeCustomerFilter:    includeCustomerFilter,
		CustomerIds:              customerIDs,
		IsPortalReady:            isPortalReady,
		StartDate:                startDate,
		EndDate:                  endDate,
		Limit:                    params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]*domain.ProductFull, len(rows))
	for i, row := range rows {
		items[i] = mapProductForwardBaseRow(row)
	}
	result, pageInfo := pagination.BuildPageStringWithSearchRank(items, params.Limit, cursorDir, searchRankEnabled, productFullCreatedAt, productFullID, productSearchRank)
	if apiErr := applyProductStitches(ctx, r.queries, result, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.ListProductsFullResult{Products: result, PageInfo: pageInfo}, nil
}

func (r *productRepoImpl) Get(ctx context.Context, params domain.GetProductFullParams) (*domain.ProductFull, *apierror.APIError) {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.get")
	defer span.End()

	row, err := r.queries.GetProductByIDBase(ctx, sqlc.GetProductByIDBaseParams{
		ID:        params.ProductID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	p := mapProductGetBaseRow(row)
	if apiErr := applyProductStitches(ctx, r.queries, []*domain.ProductFull{p}, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return p, nil
}

func (r *productRepoImpl) Create(ctx context.Context, productID, itemID string, params domain.CreateProductParams) (*domain.ProductFull, *apierror.APIError) {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.create")
	defer span.End()

	err := r.queries.InsertProduct(ctx, sqlc.InsertProductParams{
		ID:              productID,
		ItemID:          itemID,
		ProductTypeCode: params.ProductTypeCode,
		ProductLineID:   toNullString(params.ProductLineID),
		IsPortalReady:   params.IsPortalReady,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetProductFullParams{AccountID: params.AccountID, ProductID: productID, Includes: params.Includes})
}

func (r *productRepoImpl) Update(ctx context.Context, params domain.UpdateProductParams) (*domain.ProductFull, *apierror.APIError) {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.update")
	defer span.End()

	existing, apiErr := r.Get(ctx, domain.GetProductFullParams{
		AccountID: params.AccountID,
		ProductID: params.ProductID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	_, err := r.queries.ProductUpdateItem(ctx, sqlc.ProductUpdateItemParams{
		Sku:         toNullString(params.SKU),
		Description: patch.StringToNullString(params.Description),
		Notes:       patch.StringToNullString(params.Notes),
		ID:          existing.ItemID,
		AccountID:   params.AccountID,
	})
	if apiErr = db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	_, err = r.queries.UpdateProductFields(ctx, sqlc.UpdateProductFieldsParams{
		IsPortalReady: toNullBool(params.IsPortalReady),
		ID:            params.ProductID,
		AccountID:     params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetProductFullParams{AccountID: params.AccountID, ProductID: params.ProductID, Includes: params.Includes})
}

func (r *productRepoImpl) SoftDelete(ctx context.Context, params domain.DeleteProductParams) *apierror.APIError {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.soft_delete")
	defer span.End()

	result, err := r.queries.SoftDeleteProductByID(ctx, sqlc.SoftDeleteProductByIDParams{
		ID:        params.ProductID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Product not found."))
	}

	return nil
}

func (r *productRepoImpl) ChangeProductLine(ctx context.Context, params domain.ChangeProductProductLineParams) (*domain.ProductFull, *apierror.APIError) {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.change_product_line")
	defer span.End()

	result, err := r.queries.ChangeProductLineByID(ctx, sqlc.ChangeProductLineByIDParams{
		ProductLineID: gosql.NullString{String: params.ProductLineID, Valid: true},
		ID:            params.ProductID,
		AccountID:     params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Product not found."))
	}

	return r.Get(ctx, domain.GetProductFullParams{AccountID: params.AccountID, ProductID: params.ProductID, Includes: params.Includes})
}

func (r *productRepoImpl) ValidateProducts(ctx context.Context, params domain.ValidateProductsParams) (*domain.ValidateProductsResult, *apierror.APIError) {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.validate_products")
	defer span.End()

	skus := make([]string, 0, len(params.ProductsMap))
	for _, sku := range params.ProductsMap {
		skus = append(skus, sku)
	}

	rows, err := r.queries.FindProductsBySKUs(ctx, sqlc.FindProductsBySKUsParams{
		AccountID: params.AccountID,
		Skus:      skus,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Build a map from lowercase SKU to ProductFull for case-insensitive lookup.
	skuToProduct := make(map[string]*domain.ProductFull, len(rows))
	for _, row := range rows {
		product := mapProductFullFindRow(row)
		skuToProduct[strings.ToLower(product.Item.SKU)] = product
	}

	// Match results back to keys by SKU (case-insensitive).
	products := make(map[string]*domain.ProductFull, len(params.ProductsMap))
	for key, sku := range params.ProductsMap {
		if product, ok := skuToProduct[strings.ToLower(sku)]; ok {
			products[key] = product
		}
	}

	return &domain.ValidateProductsResult{Products: products}, nil
}

func (r *productRepoImpl) ExistsBySKU(ctx context.Context, accountID, sku string, excludeItemID *string) (bool, *apierror.APIError) {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.exists_by_sku")
	defer span.End()

	count, err := r.queries.CheckProductSKUExists(ctx, sqlc.CheckProductSKUExistsParams{
		AccountID:     accountID,
		Sku:           sku,
		ExcludeItemID: toNullString(excludeItemID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *productRepoImpl) InsertRate(ctx context.Context, id, value, numeratorUnitID, denominatorUnitID string) *apierror.APIError {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.insert_rate")
	defer span.End()

	err := r.queries.ProductInsertRate(ctx, sqlc.ProductInsertRateParams{
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

func mapProductExportRow(row sqlc.ExportProductsWithFiltersRow) *domain.ProductFull {
	var productLineID *string
	if row.ProductLineID.Valid {
		productLineID = &row.ProductLineID.String
	}
	return &domain.ProductFull{
		ID:              row.ID,
		ProductTypeCode: row.ProductTypeCode,
		IsPortalReady:   row.IsPortalReady,
		ProductLineID:   productLineID,
		ItemID:          row.ItemID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		Item:            mapProductBaseItem(row.ItemID, row.Sku, row.ItemDescription, row.ItemNotes, row.ItemTypeCode, row.ItemCategoryID, row.CategoryName, row.ItemCategoryTypeCode, row.CategoryUnitGroupID, row.UnitValueID, row.UnitCostID, row.BurnRateID, row.AccountID, row.IsDirty, row.ItemCreatedAt, row.ItemUpdatedAt, row.CategoryCreatedAt, row.CategoryUpdatedAt),
		ProductType:     mapProductBaseProductType(row.ProductTypeID, row.ProductTypeName, row.ProductTypeCodeJoined, row.ProductTypeCreatedAt, row.ProductTypeUpdatedAt),
	}
}

func (r *productRepoImpl) Export(ctx context.Context, params domain.ExportProductsParams) ([]*domain.ProductFull, *apierror.APIError) {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.export")
	defer span.End()

	catSearch := db.NewCatalogSearch(params.Query)
	searchQuery := catSearch.Contains
	includeProductLineFilter := len(params.ProductLineIDs) > 0
	includeCategoryFilter := len(params.CategoryIDs) > 0
	includeAttributeFilter := len(params.AttributeIDs) > 0
	includeCustomerFilter := len(params.CustomerIDs) > 0

	productLineIDs := toNullStringSlice(params.ProductLineIDs)
	categoryIDs := params.CategoryIDs
	if categoryIDs == nil {
		categoryIDs = []string{}
	}
	attributeIDs := params.AttributeIDs
	if attributeIDs == nil {
		attributeIDs = []string{}
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

	rows, err := r.queries.ExportProductsWithFilters(ctx, sqlc.ExportProductsWithFiltersParams{
		AccountID:                params.AccountID,
		SearchQuery:              searchQuery,
		SearchExact:              catSearch.Exact,
		SearchPrefix:             catSearch.Prefix,
		IncludeProductLineFilter: includeProductLineFilter,
		IncludeCustomerFilter:    includeCustomerFilter,
		ProductLineIds:           productLineIDs,
		CustomerIds:              customerIDs,
		IncludeCategoryFilter:    includeCategoryFilter,
		CategoryIds:              categoryIDs,
		IncludeAttributeFilter:   includeAttributeFilter,
		AttributeIds:             attributeIDs,
		StartDate:                startDate,
		EndDate:                  endDate,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	products := make([]*domain.ProductFull, len(rows))
	for i, row := range rows {
		products[i] = mapProductExportRow(row)
	}

	exportIncludes := []string{"product_line", "item", "item.category", "item.category.properties", "item.unit_value", "item.unit_cost", "item.attributes"}
	if apiErr := applyProductStitches(ctx, r.queries, products, exportIncludes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return products, nil
}

func (r *productRepoImpl) InsertItem(ctx context.Context, itemID string, params domain.CreateProductParams) *apierror.APIError {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.insert_item")
	defer span.End()

	err := r.queries.ProductInsertItem(ctx, sqlc.ProductInsertItemParams{
		ID:             itemID,
		Sku:            params.SKU,
		Description:    toNullString(params.Description),
		Notes:          toNullString(params.Notes),
		ItemCategoryID: params.CategoryID,
		UnitValueID:    params.UnitValueRateID,
		UnitCostID:     params.UnitCostRateID,
		BurnRateID:     params.BurnRateRateID,
		AccountID:      params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
