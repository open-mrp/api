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

type attributeRepoImpl struct {
	queries *sqlc.Queries
}

var attributeRepoTracer = tracing.GetTracer("core-service.attribute_repository")

func attributeOrder(a *domain.Attribute) time.Time { return time.Unix(int64(a.SortOrder), 0) }
func attributeID(a *domain.Attribute) string       { return a.ID }

func NewAttributeRepo(queries *sqlc.Queries) domain.AttributeRepo {
	return &attributeRepoImpl{queries: queries}
}

func mapForwardAttributeRow(row sqlc.ListAttributesForwardRow) *domain.Attribute {
	return &domain.Attribute{
		ID:         row.ID,
		Value:      row.Text,
		PropertyID: row.PropertyID,
		AccountID:  row.AccountID,
		ColorCode:  row.ColorCode,
		SortOrder:  row.Order,
		IsPublic:   row.IsPublic,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func mapBackwardAttributeRow(row sqlc.ListAttributesBackwardRow) *domain.Attribute {
	return &domain.Attribute{
		ID:         row.ID,
		Value:      row.Text,
		PropertyID: row.PropertyID,
		AccountID:  row.AccountID,
		ColorCode:  row.ColorCode,
		SortOrder:  row.Order,
		IsPublic:   row.IsPublic,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func mapGetAttributeRow(row sqlc.GetAttributeRow) *domain.Attribute {
	return &domain.Attribute{
		ID:         row.ID,
		Value:      row.Text,
		PropertyID: row.PropertyID,
		AccountID:  row.AccountID,
		ColorCode:  row.ColorCode,
		SortOrder:  row.Order,
		IsPublic:   row.IsPublic,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func buildAttributeSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func toNullInt32(v *int32) gosql.NullInt32 {
	if v == nil {
		return gosql.NullInt32{}
	}
	return gosql.NullInt32{Int32: *v, Valid: true}
}

func mapListByPropertyIDsRow(row sqlc.ListAttributesByPropertyIDsRow) *domain.Attribute {
	return &domain.Attribute{
		ID:         row.ID,
		Value:      row.Text,
		PropertyID: row.PropertyID,
		AccountID:  row.AccountID,
		ColorCode:  row.ColorCode,
		SortOrder:  row.Order,
		IsPublic:   row.IsPublic,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func (r *attributeRepoImpl) ListByPropertyIDs(ctx context.Context, accountID string, propertyIDs []string) ([]*domain.Attribute, *apierror.APIError) {
	ctx, span := attributeRepoTracer.Start(ctx, "repository.attribute.list_by_property_ids")
	defer span.End()

	if len(propertyIDs) == 0 {
		return nil, nil
	}

	rows, err := r.queries.ListAttributesByPropertyIDs(ctx, sqlc.ListAttributesByPropertyIDsParams{
		PropertyIds: propertyIDs,
		AccountID:   accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	attributes := make([]*domain.Attribute, len(rows))
	for i, row := range rows {
		attributes[i] = mapListByPropertyIDsRow(row)
	}

	return attributes, nil
}

func (r *attributeRepoImpl) List(ctx context.Context, params domain.ListAttributesParams) (*domain.ListAttributesResult, *apierror.APIError) {
	ctx, span := attributeRepoTracer.Start(ctx, "repository.attribute.list")
	defer span.End()

	searchQuery := buildAttributeSearchParams(params.Query)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction
		cursorOrder := int32(cur.OccurredAt.Unix()) // #nosec G115 - order values are small integers

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListAttributesBackward(ctx, sqlc.ListAttributesBackwardParams{
				PropertyID:  params.PropertyID,
				AccountID:   params.AccountID,
				SearchQuery: searchQuery,
				CursorOrder: gosql.NullInt32{Int32: cursorOrder, Valid: true},
				CursorID:    gosql.NullString{String: cur.ID, Valid: true},
				Limit:       params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			attributes := make([]*domain.Attribute, len(rows))
			for i, row := range rows {
				attributes[i] = mapBackwardAttributeRow(row)
			}
			result, pageInfo := pagination.BuildPageString(attributes, params.Limit, cursorDir, attributeOrder, attributeID)
			return &domain.ListAttributesResult{Attributes: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListAttributesForward(ctx, sqlc.ListAttributesForwardParams{
			PropertyID:  params.PropertyID,
			AccountID:   params.AccountID,
			SearchQuery: searchQuery,
			CursorOrder: gosql.NullInt32{Int32: cursorOrder, Valid: true},
			CursorID:    gosql.NullString{String: cur.ID, Valid: true},
			Limit:       params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		attributes := make([]*domain.Attribute, len(rows))
		for i, row := range rows {
			attributes[i] = mapForwardAttributeRow(row)
		}
		result, pageInfo := pagination.BuildPageString(attributes, params.Limit, cursorDir, attributeOrder, attributeID)
		return &domain.ListAttributesResult{Attributes: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListAttributesForward(ctx, sqlc.ListAttributesForwardParams{
		PropertyID:  params.PropertyID,
		AccountID:   params.AccountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	attributes := make([]*domain.Attribute, len(rows))
	for i, row := range rows {
		attributes[i] = mapForwardAttributeRow(row)
	}
	result, pageInfo := pagination.BuildPageString(attributes, params.Limit, cursorDir, attributeOrder, attributeID)
	return &domain.ListAttributesResult{Attributes: result, PageInfo: pageInfo}, nil
}

func (r *attributeRepoImpl) Get(ctx context.Context, params domain.GetAttributeParams) (*domain.Attribute, *apierror.APIError) {
	ctx, span := attributeRepoTracer.Start(ctx, "repository.attribute.get")
	defer span.End()

	row, err := r.queries.GetAttribute(ctx, sqlc.GetAttributeParams{
		ID:         params.AttributeID,
		PropertyID: params.PropertyID,
		AccountID:  params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetAttributeRow(row), nil
}

func (r *attributeRepoImpl) Create(ctx context.Context, id string, params domain.CreateAttributeParams) (*domain.Attribute, *apierror.APIError) {
	ctx, span := attributeRepoTracer.Start(ctx, "repository.attribute.create")
	defer span.End()

	err := r.queries.InsertAttribute(ctx, sqlc.InsertAttributeParams{
		ID:         id,
		Text:       params.Value,
		PropertyID: params.PropertyID,
		AccountID:  params.AccountID,
		ColorCode:  params.ColorCode,
		Order:      params.SortOrder,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetAttributeParams{AttributeID: id, PropertyID: params.PropertyID, AccountID: params.AccountID})
}

func (r *attributeRepoImpl) Update(ctx context.Context, params domain.UpdateAttributeParams) (*domain.Attribute, *apierror.APIError) {
	ctx, span := attributeRepoTracer.Start(ctx, "repository.attribute.update")
	defer span.End()

	result, err := r.queries.UpdateAttribute(ctx, sqlc.UpdateAttributeParams{
		Text:       toNullString(params.Value),
		ColorCode:  toNullString(params.ColorCode),
		Order:      toNullInt32(params.SortOrder),
		ID:         params.AttributeID,
		PropertyID: params.PropertyID,
		AccountID:  params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Attribute not found."))
	}

	return r.Get(ctx, domain.GetAttributeParams{AttributeID: params.AttributeID, PropertyID: params.PropertyID, AccountID: params.AccountID})
}

func (r *attributeRepoImpl) Delete(ctx context.Context, params domain.DeleteAttributeParams) *apierror.APIError {
	ctx, span := attributeRepoTracer.Start(ctx, "repository.attribute.delete")
	defer span.End()

	result, err := r.queries.DeleteAttribute(ctx, sqlc.DeleteAttributeParams{
		ID:         params.AttributeID,
		PropertyID: params.PropertyID,
		AccountID:  params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Attribute not found."))
	}

	return nil
}

func (r *attributeRepoImpl) ExistsByValueInAccount(ctx context.Context, accountID, value string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := attributeRepoTracer.Start(ctx, "repository.attribute.exists_by_value_in_account")
	defer span.End()

	count, err := r.queries.CountAttributesByTextInAccount(ctx, sqlc.CountAttributesByTextInAccountParams{
		Text:      value,
		AccountID: accountID,
		ExcludeID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return count > 0, nil
}

func (r *attributeRepoImpl) CountByProperty(ctx context.Context, propertyID, accountID string) (int64, *apierror.APIError) {
	ctx, span := attributeRepoTracer.Start(ctx, "repository.attribute.count_by_property")
	defer span.End()

	count, err := r.queries.CountAttributesByProperty(ctx, sqlc.CountAttributesByPropertyParams{
		PropertyID: propertyID,
		AccountID:  accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return count, nil
}

func (r *attributeRepoImpl) ShiftOrdersUp(ctx context.Context, propertyID, accountID string, fromOrder int32) *apierror.APIError {
	ctx, span := attributeRepoTracer.Start(ctx, "repository.attribute.shift_orders_up")
	defer span.End()

	err := r.queries.ShiftAttributeOrdersUp(ctx, sqlc.ShiftAttributeOrdersUpParams{
		PropertyID: propertyID,
		AccountID:  accountID,
		FromOrder:  fromOrder,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *attributeRepoImpl) ShiftOrdersDown(ctx context.Context, propertyID, accountID string, afterOrder int32) *apierror.APIError {
	ctx, span := attributeRepoTracer.Start(ctx, "repository.attribute.shift_orders_down")
	defer span.End()

	err := r.queries.ShiftAttributeOrdersDown(ctx, sqlc.ShiftAttributeOrdersDownParams{
		PropertyID: propertyID,
		AccountID:  accountID,
		AfterOrder: afterOrder,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *attributeRepoImpl) ShiftOrdersUpBounded(ctx context.Context, propertyID, accountID string, fromOrder, toOrder int32) *apierror.APIError {
	ctx, span := attributeRepoTracer.Start(ctx, "repository.attribute.shift_orders_up_bounded")
	defer span.End()

	err := r.queries.ShiftAttributeOrdersUpBounded(ctx, sqlc.ShiftAttributeOrdersUpBoundedParams{
		PropertyID: propertyID,
		AccountID:  accountID,
		FromOrder:  fromOrder,
		ToOrder:    toOrder,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *attributeRepoImpl) ShiftOrdersDownBounded(ctx context.Context, propertyID, accountID string, afterOrder, upToOrder int32) *apierror.APIError {
	ctx, span := attributeRepoTracer.Start(ctx, "repository.attribute.shift_orders_down_bounded")
	defer span.End()

	err := r.queries.ShiftAttributeOrdersDownBounded(ctx, sqlc.ShiftAttributeOrdersDownBoundedParams{
		PropertyID: propertyID,
		AccountID:  accountID,
		AfterOrder: afterOrder,
		UpToOrder:  upToOrder,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
