package repository

import (
	"context"
	gosql "database/sql"
	"strings"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
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

func buildProductSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + *query + "%", Valid: true}
}

func mapProductFullForwardRow(row sqlc.ListProductsFullForwardRow) *domain.ProductFull {
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

func mapProductFullBackwardRow(row sqlc.ListProductsFullBackwardRow) *domain.ProductFull {
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

func mapProductFullGetRow(row sqlc.GetProductByItemIDRow) *domain.ProductFull {
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

func (r *productRepoImpl) List(ctx context.Context, params domain.ListProductsFullParams) (*domain.ListProductsFullResult, *apierror.APIError) {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.list")
	defer span.End()

	searchQuery := buildProductSearchParams(params.Query)
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
			rows, err := r.queries.ListProductsFullBackward(ctx, sqlc.ListProductsFullBackwardParams{
				AccountID:                params.AccountID,
				SearchQuery:              searchQuery,
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
				CursorCreatedAt:          cur.OccurredAt,
				CursorID:                 cur.ID,
				Limit:                    params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			items := make([]*domain.ProductFull, len(rows))
			for i, row := range rows {
				items[i] = mapProductFullBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, productFullCreatedAt, productFullID)
			return &domain.ListProductsFullResult{Products: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListProductsFullForward(ctx, sqlc.ListProductsFullForwardParams{
			AccountID:                params.AccountID,
			SearchQuery:              searchQuery,
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
			CursorID:                 gosql.NullString{String: cur.ID, Valid: true},
			Limit:                    params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items := make([]*domain.ProductFull, len(rows))
		for i, row := range rows {
			items[i] = mapProductFullForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, productFullCreatedAt, productFullID)
		return &domain.ListProductsFullResult{Products: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListProductsFullForward(ctx, sqlc.ListProductsFullForwardParams{
		AccountID:                params.AccountID,
		SearchQuery:              searchQuery,
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
		items[i] = mapProductFullForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, productFullCreatedAt, productFullID)
	return &domain.ListProductsFullResult{Products: result, PageInfo: pageInfo}, nil
}

func (r *productRepoImpl) Get(ctx context.Context, params domain.GetProductFullParams) (*domain.ProductFull, *apierror.APIError) {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.get")
	defer span.End()

	row, err := r.queries.GetProductByItemID(ctx, sqlc.GetProductByItemIDParams{
		ItemID:    params.ItemID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapProductFullGetRow(row), nil
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

	return r.Get(ctx, domain.GetProductFullParams{AccountID: params.AccountID, ItemID: itemID})
}

func (r *productRepoImpl) Update(ctx context.Context, params domain.UpdateProductParams) (*domain.ProductFull, *apierror.APIError) {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.update")
	defer span.End()

	_, err := r.queries.ProductUpdateItem(ctx, sqlc.ProductUpdateItemParams{
		Sku:               toNullString(params.SKU),
		UpdateDescription: params.UpdateDescription,
		Description:       toNullString(params.Description),
		UpdateNotes:       params.UpdateNotes,
		Notes:             toNullString(params.Notes),
		ID:                params.ItemID,
		AccountID:         params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	_, err = r.queries.UpdateProductFields(ctx, sqlc.UpdateProductFieldsParams{
		IsPortalReady: toNullBool(params.IsPortalReady),
		ItemID:        params.ItemID,
		AccountID:     params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetProductFullParams{AccountID: params.AccountID, ItemID: params.ItemID})
}

func (r *productRepoImpl) SoftDelete(ctx context.Context, params domain.DeleteProductParams) *apierror.APIError {
	ctx, span := productRepoTracer.Start(ctx, "repository.product.soft_delete")
	defer span.End()

	result, err := r.queries.SoftDeleteProductByItemID(ctx, sqlc.SoftDeleteProductByItemIDParams{
		ID:        params.ItemID,
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

	result, err := r.queries.ChangeProductLineByItemID(ctx, sqlc.ChangeProductLineByItemIDParams{
		ProductLineID: gosql.NullString{String: params.ProductLineID, Valid: true},
		ItemID:        params.ItemID,
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

	return r.Get(ctx, domain.GetProductFullParams{AccountID: params.AccountID, ItemID: params.ItemID})
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
