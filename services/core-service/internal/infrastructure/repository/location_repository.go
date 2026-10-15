package repository

import (
	"context"
	gosql "database/sql"
	"slices"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
)

var locationRepoTracer = tracing.GetTracer("core-service.location_repository")

type locationRepoImpl struct {
	queries *sqlc.Queries
}

func NewLocationRepo(queries *sqlc.Queries) domain.LocationRepo {
	return &locationRepoImpl{queries: queries}
}

func locationCreatedAt(sl *domain.Location) time.Time { return sl.CreatedAt }
func locationID(sl *domain.Location) string           { return sl.ID }

func locationTypeCreatedAt(slt *domain.LocationType) time.Time { return slt.CreatedAt }
func locationTypeID(slt *domain.LocationType) string           { return slt.ID }

func mapLocationRow(row sqlc.GetLocationRow) *domain.Location {
	sl := &domain.Location{
		ID:        row.ID,
		Name:      row.Name,
		TypeCode:  row.TypeCode,
		Status:    constants.ArchiveStatusFromIsActive(row.IsActive),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.ParentID.Valid {
		sl.ParentID = &row.ParentID.String
	}
	if row.ParentName.Valid {
		sl.ParentName = &row.ParentName.String
	}
	if row.ParentTypeCode.Valid {
		sl.ParentTypeCode = &row.ParentTypeCode.String
	}
	if row.AddressID.Valid {
		sl.AddressID = &row.AddressID.String
	}
	return sl
}

func (r *locationRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.Location, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.get_by_ids")
	defer span.End()

	rows, err := r.queries.GetLocationsByIDs(ctx, sqlc.GetLocationsByIDsParams{
		Ids:       ids,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	locations := make([]*domain.Location, len(rows))
	for i, row := range rows {
		locations[i] = mapLocationRow(sqlc.GetLocationRow(row))
	}
	if apiErr := r.loadChildren(ctx, accountID, locations, domain.EmbeddedLocationChildren); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return locations, nil
}

// loadChildren sets each location's children, newest first, keeping at most limit per location (0 keeps all).
func (r *locationRepoImpl) loadChildren(ctx context.Context, accountID string, locations []*domain.Location, limit int) *apierror.APIError {
	if len(locations) == 0 {
		return nil
	}
	parentIDs := make([]gosql.NullString, len(locations))
	for i, loc := range locations {
		parentIDs[i] = gosql.NullString{String: loc.ID, Valid: true}
	}
	childRows, err := r.queries.ListLocationChildrenByParentIDs(ctx, sqlc.ListLocationChildrenByParentIDsParams{
		ParentIds: parentIDs,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	childrenByParent := make(map[string][]domain.LocationChild, len(locations))
	for _, cr := range childRows {
		if !cr.ParentID.Valid {
			continue
		}
		if limit > 0 && len(childrenByParent[cr.ParentID.String]) >= limit {
			continue
		}
		childrenByParent[cr.ParentID.String] = append(childrenByParent[cr.ParentID.String], domain.LocationChild{
			ID:       cr.ID,
			Name:     cr.Name,
			TypeCode: cr.TypeCode,
		})
	}
	for _, loc := range locations {
		loc.Children = childrenByParent[loc.ID]
	}
	return nil
}

func buildLocationSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func (r *locationRepoImpl) Export(ctx context.Context, params domain.ExportLocationsParams) ([]*domain.Location, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.export")
	defer span.End()

	rows, err := r.queries.ExportLocations(ctx, sqlc.ExportLocationsParams{
		AccountID:   params.AccountID,
		SearchQuery: buildLocationSearchParams(params.Query),
		Limit:       exportQueryLimit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	locations := make([]*domain.Location, len(rows))
	for i, row := range rows {
		loc := &domain.Location{
			ID:        row.ID,
			Name:      row.Name,
			TypeCode:  row.TypeCode,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
		if row.ParentID.Valid {
			loc.ParentID = &row.ParentID.String
		}
		if row.ParentName.Valid {
			loc.ParentName = &row.ParentName.String
		}
		locations[i] = loc
	}

	// The sheet lists each location's children, so they load in one extra query.
	if apiErr := r.loadChildren(ctx, params.AccountID, locations, 0); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return locations, nil
}

func (r *locationRepoImpl) List(ctx context.Context, params domain.ListLocationsParams) (*domain.ListLocationsResult, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.list")
	defer span.End()

	cursor, apiErr := decodeListCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	query, args := buildLocationListQuery(locationListFilter{
		accountID: params.AccountID,
		search:    buildLocationSearchParams(params.Query).String,
		isActive:  params.IsActive,
		parentIDs: params.ParentIDs,
		cursor:    cursor,
		limit:     params.Limit + 1,
	})
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()
	scanned, err := scanLocationListRows(rows)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	locations := make([]*domain.Location, len(scanned))
	for i, row := range scanned {
		locations[i] = mapLocationRow(row)
	}
	result, pageInfo := pagination.BuildPageString(locations, params.Limit, cursorDirection(cursor), locationCreatedAt, locationID)
	if slices.Contains(params.Includes, "children") {
		if apiErr := r.loadChildren(ctx, params.AccountID, result, domain.EmbeddedLocationChildren); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	return &domain.ListLocationsResult{Locations: result, PageInfo: pageInfo}, nil
}

func (r *locationRepoImpl) Get(ctx context.Context, params domain.GetLocationParams) (*domain.Location, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.get")
	defer span.End()

	row, err := r.queries.GetLocation(ctx, sqlc.GetLocationParams{
		ID:        params.LocationID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	sl := mapLocationRow(row)

	if slices.Contains(params.Includes, "children") {
		if apiErr := r.loadChildren(ctx, params.AccountID, []*domain.Location{sl}, domain.EmbeddedLocationChildren); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return sl, nil
}

func (r *locationRepoImpl) Create(ctx context.Context, id string, params domain.CreateLocationParams) (*domain.Location, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.create")
	defer span.End()

	status := constants.ArchiveStatusActive
	if params.Status != nil {
		status = *params.Status
	}
	if err := r.queries.InsertLocation(ctx, sqlc.InsertLocationParams{
		ID:                      id,
		AccountID:               params.AccountID,
		Name:                    params.Name,
		StorageLocationTypeCode: params.TypeCode,
		ParentID:                toNullString(params.ParentID),
		AddressID:               toNullString(params.AddressID),
		IsActive:                status.IsActive(),
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Connect existing locations as children by updating their parent_id.
	for _, childID := range params.ChildIDs {
		if err := r.queries.ConnectLocationChildren(ctx, sqlc.ConnectLocationChildrenParams{
			ParentID:  gosql.NullString{String: id, Valid: true},
			ChildID:   childID,
			AccountID: params.AccountID,
		}); err != nil {
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}

	return r.Get(ctx, domain.GetLocationParams{
		AccountID:  params.AccountID,
		LocationID: id,
		Includes:   params.Includes,
	})
}

// Update writes the fields params sets. ParentID and AddressID are written as given, so the caller backfills an
// unset one with the current value.
func (r *locationRepoImpl) Update(ctx context.Context, params domain.UpdateLocationParams) (*domain.Location, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.update")
	defer span.End()

	var isActive gosql.NullBool
	if params.Status != nil {
		isActive = gosql.NullBool{Bool: params.Status.IsActive(), Valid: true}
	}
	result, err := r.queries.UpdateLocation(ctx, sqlc.UpdateLocationParams{
		ID:                      params.LocationID,
		AccountID:               params.AccountID,
		Name:                    toNullString(params.Name),
		StorageLocationTypeCode: toNullString(params.TypeCode),
		ParentID:                field.StringToNullString(params.ParentID),
		AddressID:               field.StringToNullString(params.AddressID),
		IsActive:                isActive,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Location not found."))
	}

	if params.ChildIDs.WasProvided() {
		if err := r.queries.DisconnectLocationChildren(ctx, sqlc.DisconnectLocationChildrenParams{
			ParentID:  gosql.NullString{String: params.LocationID, Valid: true},
			AccountID: params.AccountID,
		}); err != nil {
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
		childIDs, _ := params.ChildIDs.Value()
		for _, childID := range childIDs {
			if err := r.queries.ConnectLocationChildren(ctx, sqlc.ConnectLocationChildrenParams{
				ParentID:  gosql.NullString{String: params.LocationID, Valid: true},
				ChildID:   childID,
				AccountID: params.AccountID,
			}); err != nil {
				if apiErr := db.MapSQLError(err); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
			}
		}
	}

	return r.Get(ctx, domain.GetLocationParams{
		AccountID:  params.AccountID,
		LocationID: params.LocationID,
		Includes:   params.Includes,
	})
}

func (r *locationRepoImpl) Delete(ctx context.Context, params domain.DeleteLocationParams) *apierror.APIError {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.delete")
	defer span.End()

	if err := r.queries.DeleteLocation(ctx, sqlc.DeleteLocationParams{
		ID:        params.LocationID,
		AccountID: params.AccountID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func (r *locationRepoImpl) FindByNames(ctx context.Context, accountID string, names []string) ([]*domain.Location, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.find_by_names")
	defer span.End()

	rows, err := r.queries.FindLocationsByNames(ctx, sqlc.FindLocationsByNamesParams{
		Names:     names,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	results := make([]*domain.Location, len(rows))
	for i, row := range rows {
		results[i] = mapLocationRow(sqlc.GetLocationRow(row))
	}
	return results, nil
}

func (r *locationRepoImpl) LinkParent(ctx context.Context, accountID, childID, parentID string) *apierror.APIError {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.link_parent")
	defer span.End()

	if err := r.queries.ConnectLocationChildren(ctx, sqlc.ConnectLocationChildrenParams{
		ParentID:  gosql.NullString{String: parentID, Valid: true},
		ChildID:   childID,
		AccountID: accountID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
}

func (r *locationRepoImpl) IsInAccount(ctx context.Context, accountID, locationID string) (bool, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.is_in_account")
	defer span.End()

	exists, err := r.queries.CheckLocationInAccount(ctx, sqlc.CheckLocationInAccountParams{
		ID:        locationID,
		AccountID: accountID,
	})
	if err != nil {
		return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check location in account."))
	}
	return exists, nil
}

func (r *locationRepoImpl) DefaultSiteID(ctx context.Context, accountID string) (*string, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.default_site_id")
	defer span.End()

	id, err := r.queries.GetAccountDefaultSiteID(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	if !id.Valid {
		return nil, nil
	}
	return &id.String, nil
}

func (r *locationRepoImpl) SetDefaultSiteIfUnset(ctx context.Context, accountID, siteID string) (bool, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.set_default_site_if_unset")
	defer span.End()

	result, err := r.queries.SetAccountDefaultSiteIfUnset(ctx, sqlc.SetAccountDefaultSiteIfUnsetParams{
		SiteID:    gosql.NullString{String: siteID, Valid: true},
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	return n > 0, nil
}

func (r *locationRepoImpl) ListTreeNodes(ctx context.Context, accountID string) ([]domain.LocationTreeNode, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.list_tree_nodes")
	defer span.End()

	rows, err := r.queries.ListLocationTreeNodes(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	nodes := make([]domain.LocationTreeNode, len(rows))
	for i, row := range rows {
		nodes[i] = domain.LocationTreeNode{ID: row.ID, TypeCode: row.TypeCode}
		if row.ParentID.Valid {
			nodes[i].ParentID = &row.ParentID.String
		}
	}
	return nodes, nil
}

func (r *locationRepoImpl) ListTypes(ctx context.Context, params domain.ListLocationTypesParams) (*domain.ListLocationTypesResult, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.list_types")
	defer span.End()

	searchQuery := buildLocationSearchParams(params.Query)
	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListLocationTypesBackward(ctx, sqlc.ListLocationTypesBackwardParams{
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			types := make([]*domain.LocationType, len(rows))
			for i, row := range rows {
				types[i] = &domain.LocationType{
					ID: row.ID, Code: row.Code, Name: row.Name,
					CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
				}
			}
			result, pageInfo := pagination.BuildPageString(types, params.Limit, cursorDir, locationTypeCreatedAt, locationTypeID)
			return &domain.ListLocationTypesResult{LocationTypes: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListLocationTypesForward(ctx, sqlc.ListLocationTypesForwardParams{
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		types := make([]*domain.LocationType, len(rows))
		for i, row := range rows {
			types[i] = &domain.LocationType{
				ID: row.ID, Code: row.Code, Name: row.Name,
				CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
			}
		}
		result, pageInfo := pagination.BuildPageString(types, params.Limit, cursorDir, locationTypeCreatedAt, locationTypeID)
		return &domain.ListLocationTypesResult{LocationTypes: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListLocationTypesForward(ctx, sqlc.ListLocationTypesForwardParams{
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	types := make([]*domain.LocationType, len(rows))
	for i, row := range rows {
		types[i] = &domain.LocationType{
			ID: row.ID, Code: row.Code, Name: row.Name,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		}
	}
	result, pageInfo := pagination.BuildPageString(types, params.Limit, cursorDir, locationTypeCreatedAt, locationTypeID)
	return &domain.ListLocationTypesResult{LocationTypes: result, PageInfo: pageInfo}, nil
}

func (r *locationRepoImpl) GetType(ctx context.Context, idOrCode string) (*domain.LocationType, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.get_type")
	defer span.End()

	row, err := r.queries.GetLocationType(ctx, sqlc.GetLocationTypeParams{
		ID:   idOrCode,
		Code: idOrCode,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.LocationType{
		ID:        row.ID,
		Code:      row.Code,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}
