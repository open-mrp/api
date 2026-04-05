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

var productTypeRepoTracer = tracing.GetTracer("core-service.product_type_repository")

var productTypeDuplicateKeyMapping = db.DuplicateKeyMapping{
	"product_type_name_key": func() *apierror.APIError {
		return apierror.NewConflictErrorWithParam("A product type with this name already exists.", "name")
	},
	"product_type_code_key": func() *apierror.APIError {
		return apierror.NewConflictErrorWithParam("A product type with this code already exists.", "code")
	},
}

type productTypeRepoImpl struct {
	queries *sqlc.Queries
}

func NewProductTypeRepo(queries *sqlc.Queries) domain.ProductTypeRepo {
	return &productTypeRepoImpl{queries: queries}
}

func productTypeCreatedAt(pt *domain.ProductType) time.Time { return pt.CreatedAt }
func productTypeID(pt *domain.ProductType) string           { return pt.ID }

func mapProductType(row sqlc.ProductType) *domain.ProductType {
	return &domain.ProductType{
		ID:        row.ID,
		Name:      row.Name,
		Code:      row.Code,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func buildProductTypeSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + *query + "%", Valid: true}
}

func (r *productTypeRepoImpl) List(ctx context.Context, params domain.ListProductTypesParams) (*domain.ListProductTypesResult, *apierror.APIError) {
	ctx, span := productTypeRepoTracer.Start(ctx, "repository.product_type.list")
	defer span.End()

	searchQuery := buildProductTypeSearchParams(params.Query)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListProductTypesBackward(ctx, sqlc.ListProductTypesBackwardParams{
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			productTypes := make([]*domain.ProductType, len(rows))
			for i, row := range rows {
				productTypes[i] = mapProductType(row)
			}
			result, pageInfo := pagination.BuildPageString(productTypes, params.Limit, cursorDir, productTypeCreatedAt, productTypeID)
			return &domain.ListProductTypesResult{ProductTypes: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListProductTypesForward(ctx, sqlc.ListProductTypesForwardParams{
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		productTypes := make([]*domain.ProductType, len(rows))
		for i, row := range rows {
			productTypes[i] = mapProductType(row)
		}
		result, pageInfo := pagination.BuildPageString(productTypes, params.Limit, cursorDir, productTypeCreatedAt, productTypeID)
		return &domain.ListProductTypesResult{ProductTypes: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListProductTypesForward(ctx, sqlc.ListProductTypesForwardParams{
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	productTypes := make([]*domain.ProductType, len(rows))
	for i, row := range rows {
		productTypes[i] = mapProductType(row)
	}
	result, pageInfo := pagination.BuildPageString(productTypes, params.Limit, cursorDir, productTypeCreatedAt, productTypeID)
	return &domain.ListProductTypesResult{ProductTypes: result, PageInfo: pageInfo}, nil
}

func (r *productTypeRepoImpl) Get(ctx context.Context, identifier string) (*domain.ProductType, *apierror.APIError) {
	ctx, span := productTypeRepoTracer.Start(ctx, "repository.product_type.get")
	defer span.End()

	row, err := r.queries.GetProductType(ctx, sqlc.GetProductTypeParams{
		ID:   identifier,
		Code: identifier,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapProductType(row), nil
}

func (r *productTypeRepoImpl) Create(ctx context.Context, id string, params domain.CreateProductTypeParams) (*domain.ProductType, *apierror.APIError) {
	ctx, span := productTypeRepoTracer.Start(ctx, "repository.product_type.create")
	defer span.End()

	err := r.queries.InsertProductType(ctx, sqlc.InsertProductTypeParams{
		ID:   id,
		Name: params.Name,
		Code: params.Code,
	})
	if apiErr := db.MapSQLErrorWithDuplicateKeys(err, productTypeDuplicateKeyMapping); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, id)
}

func (r *productTypeRepoImpl) Update(ctx context.Context, params domain.UpdateProductTypeParams) (*domain.ProductType, *apierror.APIError) {
	ctx, span := productTypeRepoTracer.Start(ctx, "repository.product_type.update")
	defer span.End()

	result, err := r.queries.UpdateProductType(ctx, sqlc.UpdateProductTypeParams{
		ID:   params.ProductTypeID,
		Name: toNullString(params.Name),
		Code: toNullString(params.Code),
	})
	if apiErr := db.MapSQLErrorWithDuplicateKeys(err, productTypeDuplicateKeyMapping); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Product type not found."))
	}

	return r.Get(ctx, params.ProductTypeID)
}

func (r *productTypeRepoImpl) Delete(ctx context.Context, id string) *apierror.APIError {
	ctx, span := productTypeRepoTracer.Start(ctx, "repository.product_type.delete")
	defer span.End()

	result, err := r.queries.DeleteProductType(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Product type not found."))
	}

	return nil
}

func (r *productTypeRepoImpl) ExistsByName(ctx context.Context, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := productTypeRepoTracer.Start(ctx, "repository.product_type.exists_by_name")
	defer span.End()

	count, err := r.queries.CountProductTypesByName(ctx, sqlc.CountProductTypesByNameParams{
		Name:      name,
		ExcludeID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *productTypeRepoImpl) ExistsByCode(ctx context.Context, code string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := productTypeRepoTracer.Start(ctx, "repository.product_type.exists_by_code")
	defer span.End()

	count, err := r.queries.CountProductTypesByCode(ctx, sqlc.CountProductTypesByCodeParams{
		Code:      code,
		ExcludeID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *productTypeRepoImpl) ExistsByID(ctx context.Context, id string) (bool, *apierror.APIError) {
	ctx, span := productTypeRepoTracer.Start(ctx, "repository.product_type.exists_by_id")
	defer span.End()

	count, err := r.queries.ProductTypeExistsByID(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}
