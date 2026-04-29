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

type propertyRepoImpl struct {
	queries *sqlc.Queries
}

var propertyRepoTracer = tracing.GetTracer("core-service.property_repository")

func propertyCreatedAt(p *domain.Property) time.Time { return p.CreatedAt }
func propertyID(p *domain.Property) string           { return p.ID }

func NewPropertyRepo(queries *sqlc.Queries) domain.PropertyRepo {
	return &propertyRepoImpl{queries: queries}
}

func mapForwardPropertyRow(row sqlc.ListPropertiesForwardRow) *domain.Property {
	return &domain.Property{
		ID:        row.ID,
		Name:      row.Name,
		AccountID: row.AccountID,
		IsPublic:  row.IsPublic,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func mapBackwardPropertyRow(row sqlc.ListPropertiesBackwardRow) *domain.Property {
	return &domain.Property{
		ID:        row.ID,
		Name:      row.Name,
		AccountID: row.AccountID,
		IsPublic:  row.IsPublic,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func mapGetPropertyRow(row sqlc.GetPropertyRow) *domain.Property {
	return &domain.Property{
		ID:        row.ID,
		Name:      row.Name,
		AccountID: row.AccountID,
		IsPublic:  row.IsPublic,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func buildPropertySearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func (r *propertyRepoImpl) List(ctx context.Context, params domain.ListPropertiesParams) (*domain.ListPropertiesResult, *apierror.APIError) {
	ctx, span := propertyRepoTracer.Start(ctx, "repository.property.list")
	defer span.End()

	searchQuery := buildPropertySearchParams(params.Query)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListPropertiesBackward(ctx, sqlc.ListPropertiesBackwardParams{
				AccountID:       params.AccountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
				CursorID:        gosql.NullString{String: cur.ID, Valid: true},
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			properties := make([]*domain.Property, len(rows))
			for i, row := range rows {
				properties[i] = mapBackwardPropertyRow(row)
			}
			result, pageInfo := pagination.BuildPageString(properties, params.Limit, cursorDir, propertyCreatedAt, propertyID)
			return &domain.ListPropertiesResult{Properties: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListPropertiesForward(ctx, sqlc.ListPropertiesForwardParams{
			AccountID:       params.AccountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		properties := make([]*domain.Property, len(rows))
		for i, row := range rows {
			properties[i] = mapForwardPropertyRow(row)
		}
		result, pageInfo := pagination.BuildPageString(properties, params.Limit, cursorDir, propertyCreatedAt, propertyID)
		return &domain.ListPropertiesResult{Properties: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListPropertiesForward(ctx, sqlc.ListPropertiesForwardParams{
		AccountID:   params.AccountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	properties := make([]*domain.Property, len(rows))
	for i, row := range rows {
		properties[i] = mapForwardPropertyRow(row)
	}
	result, pageInfo := pagination.BuildPageString(properties, params.Limit, cursorDir, propertyCreatedAt, propertyID)
	return &domain.ListPropertiesResult{Properties: result, PageInfo: pageInfo}, nil
}

func (r *propertyRepoImpl) Get(ctx context.Context, params domain.GetPropertyParams) (*domain.Property, *apierror.APIError) {
	ctx, span := propertyRepoTracer.Start(ctx, "repository.property.get")
	defer span.End()

	row, err := r.queries.GetProperty(ctx, sqlc.GetPropertyParams{
		ID:        params.PropertyID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetPropertyRow(row), nil
}

func (r *propertyRepoImpl) Create(ctx context.Context, id string, params domain.CreatePropertyParams) (*domain.Property, *apierror.APIError) {
	ctx, span := propertyRepoTracer.Start(ctx, "repository.property.create")
	defer span.End()

	err := r.queries.InsertProperty(ctx, sqlc.InsertPropertyParams{
		ID:        id,
		Name:      params.Name,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetPropertyParams{PropertyID: id, AccountID: params.AccountID})
}

func (r *propertyRepoImpl) Update(ctx context.Context, params domain.UpdatePropertyParams) (*domain.Property, *apierror.APIError) {
	ctx, span := propertyRepoTracer.Start(ctx, "repository.property.update")
	defer span.End()

	result, err := r.queries.UpdateProperty(ctx, sqlc.UpdatePropertyParams{
		Name:      toNullString(params.Name),
		ID:        params.PropertyID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Property not found."))
	}

	return r.Get(ctx, domain.GetPropertyParams{PropertyID: params.PropertyID, AccountID: params.AccountID})
}

func (r *propertyRepoImpl) Delete(ctx context.Context, params domain.DeletePropertyParams) *apierror.APIError {
	ctx, span := propertyRepoTracer.Start(ctx, "repository.property.delete")
	defer span.End()

	result, err := r.queries.DeleteProperty(ctx, sqlc.DeletePropertyParams{
		ID:        params.PropertyID,
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
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Property not found."))
	}

	return nil
}

func (r *propertyRepoImpl) ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := propertyRepoTracer.Start(ctx, "repository.property.exists_by_name")
	defer span.End()

	count, err := r.queries.CountPropertiesByName(ctx, sqlc.CountPropertiesByNameParams{
		Name:      name,
		AccountID: accountID,
		ExcludeID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return count > 0, nil
}

func (r *propertyRepoImpl) IsInAccount(ctx context.Context, accountID, propertyID string) (bool, *apierror.APIError) {
	ctx, span := propertyRepoTracer.Start(ctx, "repository.property.is_in_account")
	defer span.End()

	count, err := r.queries.CheckPropertyInAccount(ctx, sqlc.CheckPropertyInAccountParams{
		ID:        propertyID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return count > 0, nil
}

func (r *propertyRepoImpl) DeleteAttributesByPropertyID(ctx context.Context, propertyID, accountID string) *apierror.APIError {
	ctx, span := propertyRepoTracer.Start(ctx, "repository.property.delete_attributes_by_property_id")
	defer span.End()

	err := r.queries.DeletePropertyAttributes(ctx, sqlc.DeletePropertyAttributesParams{
		PropertyID: propertyID,
		AccountID:  accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
