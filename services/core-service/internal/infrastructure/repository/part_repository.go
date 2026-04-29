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

var partRepoTracer = tracing.GetTracer("core-service.part_repository")

type partRepoImpl struct {
	queries *sqlc.Queries
}

func NewPartRepo(queries *sqlc.Queries) domain.PartRepo {
	return &partRepoImpl{queries: queries}
}

func partItemCreatedAt(p *domain.Part) time.Time { return p.Item.CreatedAt }
func partItemID(p *domain.Part) string           { return p.ItemID }

func mapGetPartRow(row sqlc.GetPartRow) *domain.Part {
	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}

	return &domain.Part{
		ID:        row.PartID,
		ItemID:    row.ID,
		CreatedAt: row.PartCreatedAt,
		UpdatedAt: row.PartUpdatedAt,
		Item: &domain.Item{
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
		},
	}
}

func mapPartForwardRow(row sqlc.ListPartsForwardRow) *domain.Part {
	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}

	return &domain.Part{
		ID:        row.PartID,
		ItemID:    row.ID,
		CreatedAt: row.PartCreatedAt,
		UpdatedAt: row.PartUpdatedAt,
		Item: &domain.Item{
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
		},
	}
}

func mapPartBackwardRow(row sqlc.ListPartsBackwardRow) *domain.Part {
	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}

	return &domain.Part{
		ID:        row.PartID,
		ItemID:    row.ID,
		CreatedAt: row.PartCreatedAt,
		UpdatedAt: row.PartUpdatedAt,
		Item: &domain.Item{
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
		},
	}
}

func (r *partRepoImpl) loadPartAttributes(ctx context.Context, part *domain.Part) *apierror.APIError {
	rows, err := r.queries.GetPartAttributes(ctx, part.ItemID)
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
		}
	}
	part.Item.Attributes = attrs
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
	})
}

func (r *partRepoImpl) Get(ctx context.Context, params domain.GetPartParams) (*domain.Part, *apierror.APIError) {
	ctx, span := partRepoTracer.Start(ctx, "repository.part.get")
	defer span.End()

	row, err := r.queries.GetPart(ctx, sqlc.GetPartParams{
		PartID:    params.PartID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	part := mapGetPartRow(row)

	if apiErr := r.loadPartAttributes(ctx, part); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return part, nil
}

func (r *partRepoImpl) List(ctx context.Context, params domain.ListPartsParams) (*domain.ListPartsResult, *apierror.APIError) {
	ctx, span := partRepoTracer.Start(ctx, "repository.part.list")
	defer span.End()

	searchQuery := gosql.NullString{}
	if params.Query != nil && *params.Query != "" {
		searchQuery = gosql.NullString{String: "%" + *params.Query + "%", Valid: true}
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
			rows, err := r.queries.ListPartsBackward(ctx, sqlc.ListPartsBackwardParams{
				AccountID:              params.AccountID,
				IncludeCategoryFilter:  includeCategoryFilter,
				CategoryIds:            categoryIDs,
				IncludeAttributeFilter: includeAttributeFilter,
				AttributeIds:           attributeIDs,
				StartDate:              startDate,
				EndDate:                endDate,
				SearchQuery:            searchQuery,
				CursorCreatedAt:        cur.OccurredAt,
				CursorID:               cur.ID,
				Limit:                  params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			parts := make([]*domain.Part, len(rows))
			for i, row := range rows {
				parts[i] = mapPartBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(parts, params.Limit, cursorDir, partItemCreatedAt, partItemID)
			for _, p := range result {
				if apiErr := r.loadPartAttributes(ctx, p); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
			}
			return &domain.ListPartsResult{Parts: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListPartsForward(ctx, sqlc.ListPartsForwardParams{
			AccountID:              params.AccountID,
			IncludeCategoryFilter:  includeCategoryFilter,
			CategoryIds:            categoryIDs,
			IncludeAttributeFilter: includeAttributeFilter,
			AttributeIds:           attributeIDs,
			StartDate:              startDate,
			EndDate:                endDate,
			SearchQuery:            searchQuery,
			CursorCreatedAt:        gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:               gosql.NullString{String: cur.ID, Valid: true},
			Limit:                  params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		parts := make([]*domain.Part, len(rows))
		for i, row := range rows {
			parts[i] = mapPartForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(parts, params.Limit, cursorDir, partItemCreatedAt, partItemID)
		for _, p := range result {
			if apiErr := r.loadPartAttributes(ctx, p); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
		return &domain.ListPartsResult{Parts: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListPartsForward(ctx, sqlc.ListPartsForwardParams{
		AccountID:              params.AccountID,
		IncludeCategoryFilter:  includeCategoryFilter,
		CategoryIds:            categoryIDs,
		IncludeAttributeFilter: includeAttributeFilter,
		AttributeIds:           attributeIDs,
		StartDate:              startDate,
		EndDate:                endDate,
		SearchQuery:            searchQuery,
		Limit:                  params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	parts := make([]*domain.Part, len(rows))
	for i, row := range rows {
		parts[i] = mapPartForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(parts, params.Limit, cursorDir, partItemCreatedAt, partItemID)
	for _, p := range result {
		if apiErr := r.loadPartAttributes(ctx, p); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
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

func (r *partRepoImpl) TouchUpdatedAt(ctx context.Context, partID string) *apierror.APIError {
	ctx, span := partRepoTracer.Start(ctx, "repository.part.touch_updated_at")
	defer span.End()

	err := r.queries.TouchPartUpdatedAt(ctx, partID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
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
