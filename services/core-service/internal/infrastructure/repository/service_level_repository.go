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

var serviceLevelRepoTracer = tracing.GetTracer("core-service.service_level_repository")

type serviceLevelRepoImpl struct {
	queries *sqlc.Queries
}

func NewServiceLevelRepo(queries *sqlc.Queries) domain.ServiceLevelRepo {
	return &serviceLevelRepoImpl{queries: queries}
}

func serviceLevelCreatedAt(c *domain.ServiceLevel) time.Time { return c.CreatedAt }
func serviceLevelID(c *domain.ServiceLevel) string           { return c.ID }

func mapServiceLevelForwardRow(row sqlc.ListCarrierOptionsForwardRow) *domain.ServiceLevel {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var serviceLevelToken *string
	if row.ServiceLevelToken.Valid {
		serviceLevelToken = &row.ServiceLevelToken.String
	}
	return &domain.ServiceLevel{
		ID:                row.ID,
		Name:              row.Name,
		Code:              row.Code,
		ServiceLevelToken: serviceLevelToken,
		IsPortalEnabled:   row.IsPortalEnabled,
		IsDefault:         row.IsDefault,
		CarrierID:         row.CarrierID,
		AccountID:         accountID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func mapServiceLevelBackwardRow(row sqlc.ListCarrierOptionsBackwardRow) *domain.ServiceLevel {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var serviceLevelToken *string
	if row.ServiceLevelToken.Valid {
		serviceLevelToken = &row.ServiceLevelToken.String
	}
	return &domain.ServiceLevel{
		ID:                row.ID,
		Name:              row.Name,
		Code:              row.Code,
		ServiceLevelToken: serviceLevelToken,
		IsPortalEnabled:   row.IsPortalEnabled,
		IsDefault:         row.IsDefault,
		CarrierID:         row.CarrierID,
		AccountID:         accountID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func mapGetServiceLevelRow(row sqlc.GetCarrierOptionRow) *domain.ServiceLevel {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var serviceLevelToken *string
	if row.ServiceLevelToken.Valid {
		serviceLevelToken = &row.ServiceLevelToken.String
	}
	return &domain.ServiceLevel{
		ID:                row.ID,
		Name:              row.Name,
		Code:              row.Code,
		ServiceLevelToken: serviceLevelToken,
		IsPortalEnabled:   row.IsPortalEnabled,
		IsDefault:         row.IsDefault,
		CarrierID:         row.CarrierID,
		AccountID:         accountID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func (r *serviceLevelRepoImpl) List(ctx context.Context, params domain.ListServiceLevelsParams) (*domain.ListServiceLevelsResult, *apierror.APIError) {
	ctx, span := serviceLevelRepoTracer.Start(ctx, "repository.service_level.list")
	defer span.End()

	searchQuery := buildCarrierSearchParam(params.Query)
	accountID := gosql.NullString{String: params.AccountID, Valid: true}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListCarrierOptionsBackward(ctx, sqlc.ListCarrierOptionsBackwardParams{
				CarrierID:       params.CarrierID,
				AccountID:       accountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			serviceLevels := make([]*domain.ServiceLevel, len(rows))
			for i, row := range rows {
				serviceLevels[i] = mapServiceLevelBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(serviceLevels, params.Limit, cursorDir, serviceLevelCreatedAt, serviceLevelID)
			return &domain.ListServiceLevelsResult{ServiceLevels: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListCarrierOptionsForward(ctx, sqlc.ListCarrierOptionsForwardParams{
			CarrierID:       params.CarrierID,
			AccountID:       accountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		serviceLevels := make([]*domain.ServiceLevel, len(rows))
		for i, row := range rows {
			serviceLevels[i] = mapServiceLevelForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(serviceLevels, params.Limit, cursorDir, serviceLevelCreatedAt, serviceLevelID)
		return &domain.ListServiceLevelsResult{ServiceLevels: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListCarrierOptionsForward(ctx, sqlc.ListCarrierOptionsForwardParams{
		CarrierID:   params.CarrierID,
		AccountID:   accountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	serviceLevels := make([]*domain.ServiceLevel, len(rows))
	for i, row := range rows {
		serviceLevels[i] = mapServiceLevelForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(serviceLevels, params.Limit, cursorDir, serviceLevelCreatedAt, serviceLevelID)
	return &domain.ListServiceLevelsResult{ServiceLevels: result, PageInfo: pageInfo}, nil
}

func (r *serviceLevelRepoImpl) Get(ctx context.Context, accountID, serviceLevelID string) (*domain.ServiceLevel, *apierror.APIError) {
	ctx, span := serviceLevelRepoTracer.Start(ctx, "repository.service_level.get")
	defer span.End()

	row, err := r.queries.GetCarrierOption(ctx, sqlc.GetCarrierOptionParams{
		ID:        serviceLevelID,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetServiceLevelRow(row), nil
}

func (r *serviceLevelRepoImpl) Create(ctx context.Context, id string, params domain.CreateServiceLevelParams) (*domain.ServiceLevel, *apierror.APIError) {
	ctx, span := serviceLevelRepoTracer.Start(ctx, "repository.service_level.create")
	defer span.End()

	err := r.queries.InsertCarrierOption(ctx, sqlc.InsertCarrierOptionParams{
		ID:                id,
		Name:              params.Name,
		Code:              params.Code,
		ServiceLevelToken: toNullString(params.ServiceLevelToken),
		IsPortalEnabled:   params.IsPortalEnabled,
		IsDefault:         params.IsDefault,
		CarrierID:         params.CarrierID,
		AccountID:         gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, params.AccountID, id)
}

func (r *serviceLevelRepoImpl) Update(ctx context.Context, params domain.UpdateServiceLevelParams) (*domain.ServiceLevel, *apierror.APIError) {
	ctx, span := serviceLevelRepoTracer.Start(ctx, "repository.service_level.update")
	defer span.End()

	var isPortalEnabled gosql.NullBool
	if params.IsPortalEnabled != nil {
		isPortalEnabled = gosql.NullBool{Bool: *params.IsPortalEnabled, Valid: true}
	}
	var isDefault gosql.NullBool
	if params.IsDefault != nil {
		isDefault = gosql.NullBool{Bool: *params.IsDefault, Valid: true}
	}

	result, err := r.queries.UpdateCarrierOption(ctx, sqlc.UpdateCarrierOptionParams{
		ID:              params.ServiceLevelID,
		AccountID:       gosql.NullString{String: params.AccountID, Valid: true},
		Name:            toNullString(params.Name),
		Code:            toNullString(params.Code),
		IsPortalEnabled: isPortalEnabled,
		IsDefault:       isDefault,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Service level not found."))
	}

	return r.Get(ctx, params.AccountID, params.ServiceLevelID)
}

func (r *serviceLevelRepoImpl) Delete(ctx context.Context, accountID, serviceLevelID string) *apierror.APIError {
	ctx, span := serviceLevelRepoTracer.Start(ctx, "repository.service_level.delete")
	defer span.End()

	result, err := r.queries.DeleteCarrierOption(ctx, sqlc.DeleteCarrierOptionParams{
		ID:        serviceLevelID,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Service level not found."))
	}

	return nil
}

func (r *serviceLevelRepoImpl) IsInCarrier(ctx context.Context, serviceLevelID, carrierID string) (bool, *apierror.APIError) {
	ctx, span := serviceLevelRepoTracer.Start(ctx, "repository.service_level.is_in_carrier")
	defer span.End()

	exists, err := r.queries.CheckCarrierOptionInCarrier(ctx, sqlc.CheckCarrierOptionInCarrierParams{
		ID:        serviceLevelID,
		CarrierID: carrierID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return exists, nil
}

func (r *serviceLevelRepoImpl) ClearDefaultsForCarrier(ctx context.Context, accountID, carrierID string) *apierror.APIError {
	ctx, span := serviceLevelRepoTracer.Start(ctx, "repository.service_level.clear_defaults_for_carrier")
	defer span.End()

	err := r.queries.ClearDefaultServiceLevelsForCarrier(ctx, sqlc.ClearDefaultServiceLevelsForCarrierParams{
		CarrierID: carrierID,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *serviceLevelRepoImpl) ExistsByCodeInCarrier(ctx context.Context, carrierID, code string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := serviceLevelRepoTracer.Start(ctx, "repository.service_level.exists_by_code_in_carrier")
	defer span.End()

	count, err := r.queries.CountCarrierOptionsByCodeInCarrier(ctx, sqlc.CountCarrierOptionsByCodeInCarrierParams{
		Code:      code,
		CarrierID: carrierID,
		ExcludeID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}
