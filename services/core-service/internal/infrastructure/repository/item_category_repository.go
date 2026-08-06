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

var itemCategoryRepoTracer = tracing.GetTracer("core-service.item_category_repository")

type itemCategoryRepoImpl struct {
	queries *sqlc.Queries
}

func NewItemCategoryRepo(queries *sqlc.Queries) domain.ItemCategoryRepo {
	return &itemCategoryRepoImpl{queries: queries}
}

func itemCategoryCreatedAt(ic *domain.ItemCategoryFull) time.Time { return ic.CreatedAt }
func itemCategoryID(ic *domain.ItemCategoryFull) string           { return ic.ID }

func buildItemCategorySearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func buildItemCategoryTypeFilter(t *string) gosql.NullString {
	if t == nil || *t == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *t, Valid: true}
}

func mapItemCategoryForwardRow(row sqlc.ListItemCategoriesForwardRow) *domain.ItemCategoryFull {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}
	return &domain.ItemCategoryFull{
		ID:                   row.ID,
		Name:                 row.Name,
		Notes:                notes,
		ItemCategoryTypeCode: row.ItemCategoryTypeCode,
		UnitGroupID:          row.UnitGroupID,
		AccountID:            accountID,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func mapItemCategoryBackwardRow(row sqlc.ListItemCategoriesBackwardRow) *domain.ItemCategoryFull {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}
	return &domain.ItemCategoryFull{
		ID:                   row.ID,
		Name:                 row.Name,
		Notes:                notes,
		ItemCategoryTypeCode: row.ItemCategoryTypeCode,
		UnitGroupID:          row.UnitGroupID,
		AccountID:            accountID,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func mapGetItemCategoryRow(row sqlc.GetItemCategoryRow) *domain.ItemCategoryFull {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}
	return &domain.ItemCategoryFull{
		ID:                   row.ID,
		Name:                 row.Name,
		Notes:                notes,
		ItemCategoryTypeCode: row.ItemCategoryTypeCode,
		UnitGroupID:          row.UnitGroupID,
		AccountID:            accountID,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func mapGetItemCategoriesByIDsRow(row sqlc.GetItemCategoriesByIDsRow) *domain.ItemCategoryFull {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}
	return &domain.ItemCategoryFull{
		ID:                   row.ID,
		Name:                 row.Name,
		Notes:                notes,
		ItemCategoryTypeCode: row.ItemCategoryTypeCode,
		UnitGroupID:          row.UnitGroupID,
		AccountID:            accountID,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func (r *itemCategoryRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.ItemCategoryFull, *apierror.APIError) {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.queries.GetItemCategoriesByIDs(ctx, sqlc.GetItemCategoriesByIDsParams{
		Ids:       ids,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]*domain.ItemCategoryFull, len(rows))
	for i, row := range rows {
		items[i] = mapGetItemCategoriesByIDsRow(row)
	}
	return items, nil
}

func (r *itemCategoryRepoImpl) Export(ctx context.Context, params domain.ExportItemCategoriesParams) ([]*domain.ItemCategoryFull, *apierror.APIError) {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.export")
	defer span.End()

	rows, err := r.queries.ExportItemCategories(ctx, sqlc.ExportItemCategoriesParams{
		AccountID:   gosql.NullString{String: params.AccountID, Valid: true},
		SearchQuery: buildItemCategorySearchParams(params.Query),
		Limit:       exportQueryLimit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	categories := make([]*domain.ItemCategoryFull, len(rows))
	ids := make([]string, len(rows))
	for i, row := range rows {
		category := &domain.ItemCategoryFull{
			ID:                   row.ID,
			Name:                 row.Name,
			ItemCategoryTypeCode: row.ItemCategoryTypeCode,
			UnitGroupID:          row.UnitGroupID,
			CreatedAt:            row.CreatedAt,
			UpdatedAt:            row.UpdatedAt,
		}
		if row.Notes.Valid {
			category.Notes = &row.Notes.String
		}
		if row.AccountID.Valid {
			category.AccountID = &row.AccountID.String
		}
		if row.UnitGroupName.Valid {
			category.UnitGroup = &domain.ItemCategoryUnitGroup{Name: row.UnitGroupName.String}
		}
		categories[i] = category
		ids[i] = row.ID
	}

	// The sheet lists each category's property names, so they load in one extra query.
	if len(ids) > 0 {
		propertyRows, err := r.queries.ListItemCategoryPropertiesForCategories(ctx, ids)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		propertiesByCategory := make(map[string][]*domain.ItemCategoryProperty, len(ids))
		for _, pr := range propertyRows {
			propertiesByCategory[pr.ItemCategoryID] = append(propertiesByCategory[pr.ItemCategoryID], &domain.ItemCategoryProperty{
				ID:        pr.PropertyID,
				Name:      pr.PropertyName,
				CreatedAt: pr.PropertyCreatedAt,
				UpdatedAt: pr.PropertyUpdatedAt,
			})
		}
		for _, category := range categories {
			category.Properties = propertiesByCategory[category.ID]
		}
	}

	return categories, nil
}

func (r *itemCategoryRepoImpl) List(ctx context.Context, params domain.ListItemCategoriesParams) (*domain.ListItemCategoriesResult, *apierror.APIError) {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.list")
	defer span.End()

	searchQuery := buildItemCategorySearchParams(params.Query)
	typeFilter := buildItemCategoryTypeFilter(params.Type)
	accountID := gosql.NullString{String: params.AccountID, Valid: true}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListItemCategoriesBackward(ctx, sqlc.ListItemCategoriesBackwardParams{
				AccountID:            accountID,
				ItemCategoryTypeCode: typeFilter,
				SearchQuery:          searchQuery,
				CursorCreatedAt:      cur.OccurredAt,
				CursorID:             cur.ID,
				Limit:                params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			items := make([]*domain.ItemCategoryFull, len(rows))
			for i, row := range rows {
				items[i] = mapItemCategoryBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, itemCategoryCreatedAt, itemCategoryID)
			return &domain.ListItemCategoriesResult{ItemCategories: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListItemCategoriesForward(ctx, sqlc.ListItemCategoriesForwardParams{
			AccountID:            accountID,
			ItemCategoryTypeCode: typeFilter,
			SearchQuery:          searchQuery,
			CursorCreatedAt:      gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:             gosql.NullString{String: cur.ID, Valid: true},
			Limit:                params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items := make([]*domain.ItemCategoryFull, len(rows))
		for i, row := range rows {
			items[i] = mapItemCategoryForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, itemCategoryCreatedAt, itemCategoryID)
		return &domain.ListItemCategoriesResult{ItemCategories: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListItemCategoriesForward(ctx, sqlc.ListItemCategoriesForwardParams{
		AccountID:            accountID,
		ItemCategoryTypeCode: typeFilter,
		SearchQuery:          searchQuery,
		Limit:                params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]*domain.ItemCategoryFull, len(rows))
	for i, row := range rows {
		items[i] = mapItemCategoryForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, itemCategoryCreatedAt, itemCategoryID)
	return &domain.ListItemCategoriesResult{ItemCategories: result, PageInfo: pageInfo}, nil
}

func (r *itemCategoryRepoImpl) Get(ctx context.Context, params domain.GetItemCategoryParams) (*domain.ItemCategoryFull, *apierror.APIError) {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.get")
	defer span.End()

	row, err := r.queries.GetItemCategory(ctx, sqlc.GetItemCategoryParams{
		ID:        params.ItemCategoryID,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetItemCategoryRow(row), nil
}

func (r *itemCategoryRepoImpl) Create(ctx context.Context, id string, params domain.CreateItemCategoryParams) (*domain.ItemCategoryFull, *apierror.APIError) {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.create")
	defer span.End()

	err := r.queries.InsertItemCategory(ctx, sqlc.InsertItemCategoryParams{
		ID:                   id,
		Name:                 params.Name,
		Notes:                toNullString(params.Notes),
		ItemCategoryTypeCode: params.ItemCategoryTypeCode,
		UnitGroupID:          params.UnitGroupID,
		AccountID:            gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLErrorWithDuplicateKeys(err, db.DuplicateKeyMapping{
		"item_category_account_id_name_key": func() *apierror.APIError {
			return apierror.NewConflictErrorWithParam("An item category with this name already exists.", "name")
		},
	}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetItemCategoryParams{AccountID: params.AccountID, ItemCategoryID: id})
}

func (r *itemCategoryRepoImpl) Update(ctx context.Context, params domain.UpdateItemCategoryParams) (*domain.ItemCategoryFull, *apierror.APIError) {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.update")
	defer span.End()

	result, err := r.queries.UpdateItemCategory(ctx, sqlc.UpdateItemCategoryParams{
		ID:        params.ItemCategoryID,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
		Name:      toNullString(params.Name),
		Notes:     toNullString(params.Notes),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Item category not found."))
	}

	return r.Get(ctx, domain.GetItemCategoryParams{AccountID: params.AccountID, ItemCategoryID: params.ItemCategoryID})
}

func (r *itemCategoryRepoImpl) UpdateWithUnitGroup(ctx context.Context, params domain.UpdateItemCategoryWithUnitGroupParams) (*domain.ItemCategoryFull, *apierror.APIError) {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.update_with_unit_group")
	defer span.End()

	result, err := r.queries.UpdateItemCategoryWithUnitGroup(ctx, sqlc.UpdateItemCategoryWithUnitGroupParams{
		ID:          params.ItemCategoryID,
		AccountID:   gosql.NullString{String: params.AccountID, Valid: true},
		Name:        toNullString(params.Name),
		Notes:       toNullString(params.Notes),
		UnitGroupID: params.UnitGroupID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Item category not found."))
	}

	return r.Get(ctx, domain.GetItemCategoryParams{AccountID: params.AccountID, ItemCategoryID: params.ItemCategoryID})
}

func (r *itemCategoryRepoImpl) Delete(ctx context.Context, params domain.DeleteItemCategoryParams) *apierror.APIError {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.delete")
	defer span.End()

	result, err := r.queries.DeleteItemCategory(ctx, sqlc.DeleteItemCategoryParams{
		ID:        params.ItemCategoryID,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Item category not found."))
	}

	return nil
}

func (r *itemCategoryRepoImpl) IsInAccount(ctx context.Context, accountID, itemCategoryID string) (bool, *apierror.APIError) {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.is_in_account")
	defer span.End()

	count, err := r.queries.CountItemCategoryInAccount(ctx, sqlc.CountItemCategoryInAccountParams{
		ID:        itemCategoryID,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *itemCategoryRepoImpl) AddProperty(ctx context.Context, params domain.AddItemCategoryPropertyParams) *apierror.APIError {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.add_property")
	defer span.End()

	err := r.queries.InsertItemCategoryProperty(ctx, sqlc.InsertItemCategoryPropertyParams{
		ItemCategoryID: params.ItemCategoryID,
		PropertyID:     params.PropertyID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *itemCategoryRepoImpl) UpsertProperty(ctx context.Context, itemCategoryID, propertyID string) *apierror.APIError {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.upsert_property")
	defer span.End()

	err := r.queries.UpsertItemCategoryProperty(ctx, sqlc.UpsertItemCategoryPropertyParams{
		ItemCategoryID: itemCategoryID,
		PropertyID:     propertyID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *itemCategoryRepoImpl) RemoveProperty(ctx context.Context, params domain.RemoveItemCategoryPropertyParams) *apierror.APIError {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.remove_property")
	defer span.End()

	err := r.queries.DeleteItemCategoryProperty(ctx, sqlc.DeleteItemCategoryPropertyParams{
		ItemCategoryID: params.ItemCategoryID,
		PropertyID:     params.PropertyID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *itemCategoryRepoImpl) ChangeUnitGroup(ctx context.Context, params domain.ChangeItemCategoryUnitGroupParams) *apierror.APIError {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.change_unit_group")
	defer span.End()

	result, err := r.queries.UpdateItemCategoryUnitGroup(ctx, sqlc.UpdateItemCategoryUnitGroupParams{
		ID:          params.ItemCategoryID,
		AccountID:   gosql.NullString{String: params.AccountID, Valid: true},
		UnitGroupID: params.UnitGroupID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Item category not found."))
	}

	return nil
}

func (r *itemCategoryRepoImpl) GetProperties(ctx context.Context, itemCategoryID string) ([]*domain.ItemCategoryProperty, *apierror.APIError) {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.get_properties")
	defer span.End()

	rows, err := r.queries.ListItemCategoryProperties(ctx, itemCategoryID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	properties := make([]*domain.ItemCategoryProperty, len(rows))
	for i, row := range rows {
		properties[i] = &domain.ItemCategoryProperty{
			ID:        row.ID,
			Name:      row.Name,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}
	return properties, nil
}

func (r *itemCategoryRepoImpl) GetUnitGroup(ctx context.Context, unitGroupID string, includes []string) (*domain.ItemCategoryUnitGroup, *apierror.APIError) {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.get_unit_group")
	defer span.End()

	row, err := r.queries.GetUnitGroupForCategory(ctx, unitGroupID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	ug := &domain.ItemCategoryUnitGroup{
		ID:         row.ID,
		Name:       row.Name,
		BaseUnitID: row.BaseUnitID,
		Type:       row.UnitTypeCode,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}

	if slices.Contains(includes, "unit_group.base_unit") && ug.BaseUnitID != "" {
		unitRows, err := r.queries.GetUnitsByIDs(ctx, []string{ug.BaseUnitID})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if len(unitRows) > 0 {
			lu := mapGetUnitsByIDsRowToLightUnit(unitRows[0])
			ug.BaseUnit = &lu
		}
	}

	if slices.Contains(includes, "unit_group.associated_units") {
		ugUnitRows, err := r.queries.ListUnitGroupUnitsByUnitGroupIDs(ctx, []string{ug.ID})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		units := make([]*domain.UnitGroupUnit, len(ugUnitRows))
		for i, ugRow := range ugUnitRows {
			units[i] = mapUnitGroupUnitsByUnitGroupIDsRow(ugRow)
		}
		ug.AssociatedUnits = units
	}

	return ug, nil
}

func mapFindItemCategoriesByNamesRow(row sqlc.FindItemCategoriesByNamesRow) *domain.ItemCategoryFull {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}
	return &domain.ItemCategoryFull{
		ID:                   row.ID,
		Name:                 row.Name,
		Notes:                notes,
		ItemCategoryTypeCode: row.ItemCategoryTypeCode,
		UnitGroupID:          row.UnitGroupID,
		AccountID:            accountID,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func (r *itemCategoryRepoImpl) FindByNames(ctx context.Context, accountID string, names []string) ([]*domain.ItemCategoryFull, *apierror.APIError) {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.find_by_names")
	defer span.End()

	if len(names) == 0 {
		return nil, nil
	}

	rows, err := r.queries.FindItemCategoriesByNames(ctx, sqlc.FindItemCategoriesByNamesParams{
		Names:     names,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]*domain.ItemCategoryFull, len(rows))
	for i, row := range rows {
		items[i] = mapFindItemCategoriesByNamesRow(row)
	}
	return items, nil
}

func (r *itemCategoryRepoImpl) PropertyExistsByNameInCategory(ctx context.Context, accountID, itemCategoryID, name string, excludePropertyID *string) (bool, *apierror.APIError) {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.property_exists_by_name_in_category")
	defer span.End()

	count, err := r.queries.CountPropertiesInCategoryByName(ctx, sqlc.CountPropertiesInCategoryByNameParams{
		ItemCategoryID:    itemCategoryID,
		Name:              name,
		AccountID:         accountID,
		ExcludePropertyID: toNullString(excludePropertyID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *itemCategoryRepoImpl) IsPropertyInAccount(ctx context.Context, accountID, propertyID string) (bool, *apierror.APIError) {
	ctx, span := itemCategoryRepoTracer.Start(ctx, "repository.item_category.is_property_in_account")
	defer span.End()

	count, err := r.queries.CountPropertyInAccount(ctx, sqlc.CountPropertyInAccountParams{
		ID:        propertyID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}
