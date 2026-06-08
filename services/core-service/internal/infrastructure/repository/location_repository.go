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
	"github.com/augno/api/shared/field"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
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

func mapLocationRow(
	id string,
	name string,
	typeCode string,
	parentID gosql.NullString,
	parentName gosql.NullString,
	parentTypeCode gosql.NullString,
	createdAt, updatedAt time.Time,
) *domain.Location {
	sl := &domain.Location{
		ID:        id,
		Name:      name,
		TypeCode:  typeCode,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	if parentID.Valid {
		sl.ParentID = &parentID.String
	}
	if parentName.Valid {
		sl.ParentName = &parentName.String
	}
	if parentTypeCode.Valid {
		sl.ParentTypeCode = &parentTypeCode.String
	}
	return sl
}

func mapForwardLocationRow(row sqlc.ListLocationsForwardRow) *domain.Location {
	return mapLocationRow(
		row.ID, row.Name, row.TypeCode,
		row.ParentID, row.ParentName, row.ParentTypeCode,
		row.CreatedAt, row.UpdatedAt,
	)
}

func mapBackwardLocationRow(row sqlc.ListLocationsBackwardRow) *domain.Location {
	return mapLocationRow(
		row.ID, row.Name, row.TypeCode,
		row.ParentID, row.ParentName, row.ParentTypeCode,
		row.CreatedAt, row.UpdatedAt,
	)
}

func mapGetLocationRow(row sqlc.GetLocationRow) *domain.Location {
	return mapLocationRow(
		row.ID, row.Name, row.TypeCode,
		row.ParentID, row.ParentName, row.ParentTypeCode,
		row.CreatedAt, row.UpdatedAt,
	)
}

func mapGetLocationsByIDsRow(row sqlc.GetLocationsByIDsRow) *domain.Location {
	return mapLocationRow(
		row.ID, row.Name, row.TypeCode,
		row.ParentID, row.ParentName, row.ParentTypeCode,
		row.CreatedAt, row.UpdatedAt,
	)
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
	parentIDs := make([]gosql.NullString, 0, len(rows))
	for i, row := range rows {
		locations[i] = mapGetLocationsByIDsRow(row)
		parentIDs = append(parentIDs, gosql.NullString{String: row.ID, Valid: true})
	}

	if len(parentIDs) > 0 {
		childRows, err := r.queries.ListLocationChildrenByParentIDs(ctx, sqlc.ListLocationChildrenByParentIDsParams{
			ParentIds: parentIDs,
			AccountID: accountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		childrenByParent := make(map[string][]domain.LocationChild)
		for _, cr := range childRows {
			if cr.ParentID.Valid {
				childrenByParent[cr.ParentID.String] = append(childrenByParent[cr.ParentID.String], domain.LocationChild{
					ID:       cr.ID,
					Name:     cr.Name,
					TypeCode: cr.TypeCode,
				})
			}
		}
		for _, loc := range locations {
			if children, ok := childrenByParent[loc.ID]; ok {
				loc.Children = children
			}
		}
	}

	return locations, nil
}

func buildLocationSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func (r *locationRepoImpl) fetchChildren(ctx context.Context, accountID, parentID string) ([]domain.LocationChild, *apierror.APIError) {
	rows, err := r.queries.ListLocationChildren(ctx, sqlc.ListLocationChildrenParams{
		ParentID:  gosql.NullString{String: parentID, Valid: true},
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	children := make([]domain.LocationChild, len(rows))
	for i, row := range rows {
		children[i] = domain.LocationChild{
			ID:       row.ID,
			Name:     row.Name,
			TypeCode: row.TypeCode,
		}
	}
	return children, nil
}

func (r *locationRepoImpl) populateChildren(ctx context.Context, accountID string, locations []*domain.Location, incs []string) *apierror.APIError {
	if !slices.Contains(incs, "children") {
		return nil
	}
	for _, sl := range locations {
		children, apiErr := r.fetchChildren(ctx, accountID, sl.ID)
		if apiErr != nil {
			return apiErr
		}
		sl.Children = children
	}
	return nil
}

func (r *locationRepoImpl) List(ctx context.Context, params domain.ListLocationsParams) (*domain.ListLocationsResult, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.list")
	defer span.End()

	searchQuery := buildLocationSearchParams(params.Query)
	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListLocationsBackward(ctx, sqlc.ListLocationsBackwardParams{
				AccountID:       params.AccountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			locations := make([]*domain.Location, len(rows))
			for i, row := range rows {
				locations[i] = mapBackwardLocationRow(row)
			}
			result, pageInfo := pagination.BuildPageString(locations, params.Limit, cursorDir, locationCreatedAt, locationID)
			if apiErr := r.populateChildren(ctx, params.AccountID, result, params.Includes); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			return &domain.ListLocationsResult{Locations: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListLocationsForward(ctx, sqlc.ListLocationsForwardParams{
			AccountID:       params.AccountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		locations := make([]*domain.Location, len(rows))
		for i, row := range rows {
			locations[i] = mapForwardLocationRow(row)
		}
		result, pageInfo := pagination.BuildPageString(locations, params.Limit, cursorDir, locationCreatedAt, locationID)
		if apiErr := r.populateChildren(ctx, params.AccountID, result, params.Includes); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return &domain.ListLocationsResult{Locations: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListLocationsForward(ctx, sqlc.ListLocationsForwardParams{
		AccountID:   params.AccountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	locations := make([]*domain.Location, len(rows))
	for i, row := range rows {
		locations[i] = mapForwardLocationRow(row)
	}
	result, pageInfo := pagination.BuildPageString(locations, params.Limit, cursorDir, locationCreatedAt, locationID)
	if apiErr := r.populateChildren(ctx, params.AccountID, result, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
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

	sl := mapGetLocationRow(row)

	if slices.Contains(params.Includes, "children") {
		children, apiErr := r.fetchChildren(ctx, params.AccountID, sl.ID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		sl.Children = children
	}

	return sl, nil
}

func (r *locationRepoImpl) Create(ctx context.Context, id string, params domain.CreateLocationParams) (*domain.Location, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.create")
	defer span.End()

	if err := r.queries.InsertLocation(ctx, sqlc.InsertLocationParams{
		ID:                      id,
		AccountID:               params.AccountID,
		Name:                    params.Name,
		StorageLocationTypeCode: params.TypeCode,
		ParentID:                toNullString(params.ParentID),
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

func (r *locationRepoImpl) Update(ctx context.Context, params domain.UpdateLocationParams) (*domain.Location, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.update")
	defer span.End()

	result, err := r.queries.UpdateLocation(ctx, sqlc.UpdateLocationParams{
		ID:                      params.LocationID,
		AccountID:               params.AccountID,
		Name:                    toNullString(params.Name),
		StorageLocationTypeCode: toNullString(params.TypeCode),
		ParentID:                field.StringToNullString(params.ParentID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Location not found."))
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

func (r *locationRepoImpl) CountChildren(ctx context.Context, accountID, parentID string) (int64, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.count_children")
	defer span.End()

	count, err := r.queries.CountLocationChildren(ctx, sqlc.CountLocationChildrenParams{
		ParentID:  gosql.NullString{String: parentID, Valid: true},
		AccountID: accountID,
	})
	if err != nil {
		return 0, tracing.Trace(span, apierror.NewInternalError(err, "Failed to count location children."))
	}
	return count, nil
}

func (r *locationRepoImpl) ListTypes(ctx context.Context, params domain.ListLocationTypesParams) (*domain.ListLocationTypesResult, *apierror.APIError) {
	ctx, span := locationRepoTracer.Start(ctx, "repository.location.list_types")
	defer span.End()

	searchQuery := buildLocationSearchParams(params.Query)
	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
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
