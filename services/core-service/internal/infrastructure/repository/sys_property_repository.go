package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var sysPropertyRepoTracer = tracing.GetTracer("core-service.sys_property_repository")

type sysPropertyRepoImpl struct {
	queries *sqlc.Queries
}

func NewSysPropertyRepo(queries *sqlc.Queries) domain.SysPropertyRepo {
	return &sysPropertyRepoImpl{queries: queries}
}

func spCreatedAt(sp *domain.SysProperty) time.Time { return sp.CreatedAt }
func spID(sp *domain.SysProperty) string           { return sp.ID }

func mapSPForwardRow(row sqlc.ListSysPropertiesForwardRow) *domain.SysProperty {
	return &domain.SysProperty{
		ID: row.ID, TypeID: row.TypeID,
		TypeCode: constants.SysPropertyTypeCode(row.TypeCode), TypeName: row.TypeName,
		Value: row.Value, AccountID: row.AccountID,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func mapSPBackwardRow(row sqlc.ListSysPropertiesBackwardRow) *domain.SysProperty {
	return &domain.SysProperty{
		ID: row.ID, TypeID: row.TypeID,
		TypeCode: constants.SysPropertyTypeCode(row.TypeCode), TypeName: row.TypeName,
		Value: row.Value, AccountID: row.AccountID,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func mapSPGetRow(row sqlc.GetSysPropertyRow) *domain.SysProperty {
	return &domain.SysProperty{
		ID: row.ID, TypeID: row.TypeID,
		TypeCode: constants.SysPropertyTypeCode(row.TypeCode), TypeName: row.TypeName,
		Value: row.Value, AccountID: row.AccountID,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func mapSPGetByTypeCodeRow(row sqlc.GetSysPropertyByTypeCodeRow) *domain.SysProperty {
	return &domain.SysProperty{
		ID: row.ID, TypeID: row.TypeID,
		TypeCode: constants.SysPropertyTypeCode(row.TypeCode), TypeName: row.TypeName,
		Value: row.Value, AccountID: row.AccountID,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func (r *sysPropertyRepoImpl) List(ctx context.Context, params domain.ListSysPropertiesParams) (*domain.ListSysPropertiesResult, *apierror.APIError) {
	ctx, span := sysPropertyRepoTracer.Start(ctx, "repository.sys_property.list")
	defer span.End()

	var searchQuery gosql.NullString
	if params.Query != nil && *params.Query != "" {
		searchQuery = gosql.NullString{String: "%" + db.EscapeLike(*params.Query) + "%", Valid: true}
	}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListSysPropertiesBackward(ctx, sqlc.ListSysPropertiesBackwardParams{
				AccountID:       params.AccountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			items := make([]*domain.SysProperty, len(rows))
			for i, row := range rows {
				items[i] = mapSPBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, spCreatedAt, spID)
			return &domain.ListSysPropertiesResult{SysProperties: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListSysPropertiesForward(ctx, sqlc.ListSysPropertiesForwardParams{
			AccountID:       params.AccountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items := make([]*domain.SysProperty, len(rows))
		for i, row := range rows {
			items[i] = mapSPForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, spCreatedAt, spID)
		return &domain.ListSysPropertiesResult{SysProperties: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListSysPropertiesForward(ctx, sqlc.ListSysPropertiesForwardParams{
		AccountID:   params.AccountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	items := make([]*domain.SysProperty, len(rows))
	for i, row := range rows {
		items[i] = mapSPForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, spCreatedAt, spID)
	return &domain.ListSysPropertiesResult{SysProperties: result, PageInfo: pageInfo}, nil
}

func (r *sysPropertyRepoImpl) Get(ctx context.Context, accountID, id string) (*domain.SysProperty, *apierror.APIError) {
	ctx, span := sysPropertyRepoTracer.Start(ctx, "repository.sys_property.get")
	defer span.End()

	row, err := r.queries.GetSysProperty(ctx, sqlc.GetSysPropertyParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return mapSPGetRow(row), nil
}

func (r *sysPropertyRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.SysProperty, *apierror.APIError) {
	ctx, span := sysPropertyRepoTracer.Start(ctx, "repository.sys_property.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetSysPropertiesByIDs(ctx, sqlc.GetSysPropertiesByIDsParams{
		Ids:       ids,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.SysProperty, len(rows))
	for i, row := range rows {
		out[i] = &domain.SysProperty{
			ID:        row.ID,
			TypeID:    row.TypeID,
			TypeCode:  constants.SysPropertyTypeCode(row.TypeCode),
			TypeName:  row.TypeName,
			Value:     row.Value,
			AccountID: row.AccountID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}
	return out, nil
}

func (r *sysPropertyRepoImpl) GetByTypeCode(ctx context.Context, accountID string, typeCode constants.SysPropertyTypeCode) (*domain.SysProperty, *apierror.APIError) {
	ctx, span := sysPropertyRepoTracer.Start(ctx, "repository.sys_property.get_by_type_code")
	defer span.End()

	row, err := r.queries.GetSysPropertyByTypeCode(ctx, sqlc.GetSysPropertyByTypeCodeParams{
		TypeCode: string(typeCode), AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return mapSPGetByTypeCodeRow(row), nil
}

func (r *sysPropertyRepoImpl) Create(ctx context.Context, id, accountID string, typeCode constants.SysPropertyTypeCode, value int32) (*domain.SysProperty, *apierror.APIError) {
	ctx, span := sysPropertyRepoTracer.Start(ctx, "repository.sys_property.create")
	defer span.End()

	err := r.queries.InsertSysProperty(ctx, sqlc.InsertSysPropertyParams{
		ID: id, TypeCode: string(typeCode), Value: value, AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return r.Get(ctx, accountID, id)
}

func (r *sysPropertyRepoImpl) UpdateValue(ctx context.Context, accountID, id string, value int32) (*domain.SysProperty, *apierror.APIError) {
	ctx, span := sysPropertyRepoTracer.Start(ctx, "repository.sys_property.update_value")
	defer span.End()

	result, err := r.queries.UpdateSysPropertyValue(ctx, sqlc.UpdateSysPropertyValueParams{
		ID: id, AccountID: accountID, Value: value,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("System property not found."))
	}
	return r.Get(ctx, accountID, id)
}

func (r *sysPropertyRepoImpl) IncrementValue(ctx context.Context, accountID, id string) (*domain.SysProperty, *apierror.APIError) {
	ctx, span := sysPropertyRepoTracer.Start(ctx, "repository.sys_property.increment_value")
	defer span.End()

	result, err := r.queries.IncrementSysPropertyValue(ctx, sqlc.IncrementSysPropertyValueParams{
		ID: id, AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("System property not found."))
	}
	return r.Get(ctx, accountID, id)
}

func (r *sysPropertyRepoImpl) IsDuplicate(ctx context.Context, accountID string, typeCode constants.SysPropertyTypeCode, value string) (bool, *apierror.APIError) {
	ctx, span := sysPropertyRepoTracer.Start(ctx, "repository.sys_property.is_duplicate")
	defer span.End()

	switch typeCode {
	case constants.SysPropertyTypeCodeTransactionNumber:
		count, err := r.queries.CheckDuplicateTransactionNumber(ctx, sqlc.CheckDuplicateTransactionNumberParams{Value: value, AccountID: accountID})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return false, tracing.Trace(span, apiErr)
		}
		return count > 0, nil

	case constants.SysPropertyTypeCodeSettlementNumber:
		count, err := r.queries.CheckDuplicateSettlementNumber(ctx, sqlc.CheckDuplicateSettlementNumberParams{Value: value, AccountID: accountID})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return false, tracing.Trace(span, apiErr)
		}
		return count > 0, nil

	case constants.SysPropertyTypeCodeSalesOrderNumber:
		count, err := r.queries.CheckDuplicateSalesOrderNumber(ctx, sqlc.CheckDuplicateSalesOrderNumberParams{Value: value, AccountID: accountID})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return false, tracing.Trace(span, apiErr)
		}
		return count > 0, nil

	case constants.SysPropertyTypeCodePurchaseOrderNumber:
		count, err := r.queries.CheckDuplicatePurchaseOrderNumber(ctx, sqlc.CheckDuplicatePurchaseOrderNumberParams{Value: value, AccountID: accountID})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return false, tracing.Trace(span, apiErr)
		}
		return count > 0, nil

	case constants.SysPropertyTypeCodeSupplierNumber:
		count, err := r.queries.CheckDuplicateSupplierNumber(ctx, sqlc.CheckDuplicateSupplierNumberParams{Value: value, AccountID: accountID})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return false, tracing.Trace(span, apiErr)
		}
		return count > 0, nil

	case constants.SysPropertyTypeCodeCustomerNumber:
		count, err := r.queries.CheckDuplicateCustomerNumber(ctx, sqlc.CheckDuplicateCustomerNumberParams{Value: value, AccountID: accountID})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return false, tracing.Trace(span, apiErr)
		}
		return count > 0, nil

	case constants.SysPropertyTypeCodeProductionRunNumber:
		count, err := r.queries.CheckDuplicateProductionRunNumber(ctx, sqlc.CheckDuplicateProductionRunNumberParams{Value: value, AccountID: accountID})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return false, tracing.Trace(span, apiErr)
		}
		return count > 0, nil

	case constants.SysPropertyTypeCodeSsccCount:
		return false, nil

	default:
		return false, tracing.Trace(span, apierror.NewValidationError(fmt.Sprintf("Unknown system property type code: %s", typeCode)))
	}
}
