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

var materialRepoTracer = tracing.GetTracer("core-service.material_repository")

type materialRepoImpl struct {
	queries *sqlc.Queries
}

func NewMaterialRepo(queries *sqlc.Queries) domain.MaterialRepo {
	return &materialRepoImpl{queries: queries}
}

func materialCreatedAt(m *domain.Material) time.Time { return m.CreatedAt }
func materialID(m *domain.Material) string           { return m.ID }

func mapMaterialForwardRow(row sqlc.ListMaterialsForwardRow) *domain.Material {
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

func mapMaterialBackwardRow(row sqlc.ListMaterialsBackwardRow) *domain.Material {
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

func (r *materialRepoImpl) List(ctx context.Context, params domain.ListMaterialsParams) (*domain.ListMaterialsResult, *apierror.APIError) {
	ctx, span := materialRepoTracer.Start(ctx, "repository.material.list")
	defer span.End()

	searchQuery := db.NullStringLikePtr(params.Query)
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
			rows, err := r.queries.ListMaterialsBackward(ctx, sqlc.ListMaterialsBackwardParams{
				AccountID:              params.AccountID,
				SearchQuery:            searchQuery,
				IncludeCategoryFilter:  includeCategoryFilter,
				CategoryIds:            categoryIDs,
				IncludeAttributeFilter: includeAttributeFilter,
				AttributeIds:           attributeIDs,
				StartDate:              startDate,
				EndDate:                endDate,
				CursorCreatedAt:        cur.OccurredAt,
				CursorID:               cur.ID,
				Limit:                  params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			materials := make([]*domain.Material, len(rows))
			for i, row := range rows {
				materials[i] = mapMaterialBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(materials, params.Limit, cursorDir, materialCreatedAt, materialID)
			return &domain.ListMaterialsResult{Materials: result, PageInfo: pageInfo}, nil
		}

		// Forward
		rows, err := r.queries.ListMaterialsForward(ctx, sqlc.ListMaterialsForwardParams{
			AccountID:              params.AccountID,
			SearchQuery:            searchQuery,
			IncludeCategoryFilter:  includeCategoryFilter,
			CategoryIds:            categoryIDs,
			IncludeAttributeFilter: includeAttributeFilter,
			AttributeIds:           attributeIDs,
			StartDate:              startDate,
			EndDate:                endDate,
			CursorCreatedAt:        gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:               gosql.NullString{String: cur.ID, Valid: true},
			Limit:                  params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		materials := make([]*domain.Material, len(rows))
		for i, row := range rows {
			materials[i] = mapMaterialForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(materials, params.Limit, cursorDir, materialCreatedAt, materialID)
		return &domain.ListMaterialsResult{Materials: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListMaterialsForward(ctx, sqlc.ListMaterialsForwardParams{
		AccountID:              params.AccountID,
		SearchQuery:            searchQuery,
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
		materials[i] = mapMaterialForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(materials, params.Limit, cursorDir, materialCreatedAt, materialID)
	return &domain.ListMaterialsResult{Materials: result, PageInfo: pageInfo}, nil
}

func (r *materialRepoImpl) loadItemAttributes(ctx context.Context, item *domain.Item) *apierror.APIError {
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

	if material.Item != nil {
		if apiErr := r.loadItemAttributes(ctx, material.Item); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

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

	// Look up the material by item ID to get the material's own ID.
	existing, apiErr := r.GetByItemID(ctx, params.AccountID, params.ItemID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	_, err := r.queries.UpdateMaterial(ctx, existing.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *materialRepoImpl) DeleteByItemID(ctx context.Context, accountID, itemID string) *apierror.APIError {
	ctx, span := materialRepoTracer.Start(ctx, "repository.material.delete_by_item_id")
	defer span.End()

	result, err := r.queries.DeleteMaterialByItemID(ctx, sqlc.DeleteMaterialByItemIDParams{
		ID:        itemID,
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

func (r *materialRepoImpl) UpdateItem(ctx context.Context, params domain.UpdateMaterialParams) *apierror.APIError {
	ctx, span := materialRepoTracer.Start(ctx, "repository.material.update_item")
	defer span.End()

	_, err := r.queries.MaterialUpdateItem(ctx, sqlc.MaterialUpdateItemParams{
		Sku:               toNullString(params.SKU),
		UpdateDescription: params.UpdateDescription,
		Description:       toNullString(params.Description),
		UpdateNotes:       params.UpdateNotes,
		Notes:             toNullString(params.Notes),
		ID:                params.ItemID,
		AccountID:         params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
