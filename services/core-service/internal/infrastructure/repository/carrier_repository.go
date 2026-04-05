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

var carrierRepoTracer = tracing.GetTracer("core-service.carrier_repository")

type carrierRepoImpl struct {
	queries *sqlc.Queries
}

func NewCarrierRepo(queries *sqlc.Queries) domain.CarrierRepo {
	return &carrierRepoImpl{queries: queries}
}

func carrierCreatedAt(c *domain.Carrier) time.Time { return c.CreatedAt }
func carrierID(c *domain.Carrier) string           { return c.ID }

func mapCarrierForwardRow(row sqlc.ListCarriersForwardRow) *domain.Carrier {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var code *string
	if row.Code.Valid {
		code = &row.Code.String
	}
	var shippoID *string
	if row.ShippoCarrierAccountID.Valid {
		shippoID = &row.ShippoCarrierAccountID.String
	}
	var accountNumber *string
	if row.AccountNumber.Valid {
		accountNumber = &row.AccountNumber.String
	}
	var deletedAt *time.Time
	if row.DeletedAt.Valid {
		deletedAt = &row.DeletedAt.Time
	}
	return &domain.Carrier{
		ID:                     row.ID,
		Name:                   row.Name,
		Code:                   code,
		ShippoCarrierAccountID: shippoID,
		AccountNumber:          accountNumber,
		IsPortalEnabled:        row.IsPortalEnabled,
		AccountID:              accountID,
		DeletedAt:              deletedAt,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}

func mapCarrierBackwardRow(row sqlc.ListCarriersBackwardRow) *domain.Carrier {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var code *string
	if row.Code.Valid {
		code = &row.Code.String
	}
	var shippoID *string
	if row.ShippoCarrierAccountID.Valid {
		shippoID = &row.ShippoCarrierAccountID.String
	}
	var accountNumber *string
	if row.AccountNumber.Valid {
		accountNumber = &row.AccountNumber.String
	}
	var deletedAt *time.Time
	if row.DeletedAt.Valid {
		deletedAt = &row.DeletedAt.Time
	}
	return &domain.Carrier{
		ID:                     row.ID,
		Name:                   row.Name,
		Code:                   code,
		ShippoCarrierAccountID: shippoID,
		AccountNumber:          accountNumber,
		IsPortalEnabled:        row.IsPortalEnabled,
		AccountID:              accountID,
		DeletedAt:              deletedAt,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}

func mapGetCarrierRow(row sqlc.GetCarrierRow) *domain.Carrier {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var code *string
	if row.Code.Valid {
		code = &row.Code.String
	}
	var shippoID *string
	if row.ShippoCarrierAccountID.Valid {
		shippoID = &row.ShippoCarrierAccountID.String
	}
	var accountNumber *string
	if row.AccountNumber.Valid {
		accountNumber = &row.AccountNumber.String
	}
	var deletedAt *time.Time
	if row.DeletedAt.Valid {
		deletedAt = &row.DeletedAt.Time
	}
	return &domain.Carrier{
		ID:                     row.ID,
		Name:                   row.Name,
		Code:                   code,
		ShippoCarrierAccountID: shippoID,
		AccountNumber:          accountNumber,
		IsPortalEnabled:        row.IsPortalEnabled,
		AccountID:              accountID,
		DeletedAt:              deletedAt,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}

func buildCarrierSearchParam(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + *query + "%", Valid: true}
}

func (r *carrierRepoImpl) List(ctx context.Context, params domain.ListCarriersParams) (*domain.ListCarriersResult, *apierror.APIError) {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.list")
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
			rows, err := r.queries.ListCarriersBackward(ctx, sqlc.ListCarriersBackwardParams{
				AccountID:       accountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			carriers := make([]*domain.Carrier, len(rows))
			for i, row := range rows {
				carriers[i] = mapCarrierBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(carriers, params.Limit, cursorDir, carrierCreatedAt, carrierID)
			return &domain.ListCarriersResult{Carriers: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListCarriersForward(ctx, sqlc.ListCarriersForwardParams{
			AccountID:       accountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		carriers := make([]*domain.Carrier, len(rows))
		for i, row := range rows {
			carriers[i] = mapCarrierForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(carriers, params.Limit, cursorDir, carrierCreatedAt, carrierID)
		return &domain.ListCarriersResult{Carriers: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListCarriersForward(ctx, sqlc.ListCarriersForwardParams{
		AccountID:   accountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	carriers := make([]*domain.Carrier, len(rows))
	for i, row := range rows {
		carriers[i] = mapCarrierForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(carriers, params.Limit, cursorDir, carrierCreatedAt, carrierID)
	return &domain.ListCarriersResult{Carriers: result, PageInfo: pageInfo}, nil
}

func (r *carrierRepoImpl) Get(ctx context.Context, accountID, carrierID string) (*domain.Carrier, *apierror.APIError) {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.get")
	defer span.End()

	row, err := r.queries.GetCarrier(ctx, sqlc.GetCarrierParams{
		ID:        carrierID,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetCarrierRow(row), nil
}

func (r *carrierRepoImpl) Create(ctx context.Context, id string, params domain.CreateCarrierParams) (*domain.Carrier, *apierror.APIError) {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.create")
	defer span.End()

	err := r.queries.InsertCarrier(ctx, sqlc.InsertCarrierParams{
		ID:                     id,
		Name:                   params.Name,
		Code:                   toNullString(params.Code),
		ShippoCarrierAccountID: toNullString(params.ShippoCarrierAccountID),
		AccountNumber:          toNullString(params.AccountNumber),
		IsPortalEnabled:        params.IsPortalEnabled,
		AccountID:              gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, params.AccountID, id)
}

func (r *carrierRepoImpl) Update(ctx context.Context, params domain.UpdateCarrierParams) (*domain.Carrier, *apierror.APIError) {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.update")
	defer span.End()

	var isPortalEnabled gosql.NullBool
	if params.IsPortalEnabled != nil {
		isPortalEnabled = gosql.NullBool{Bool: *params.IsPortalEnabled, Valid: true}
	}

	result, err := r.queries.UpdateCarrier(ctx, sqlc.UpdateCarrierParams{
		ID:              params.CarrierID,
		AccountID:       gosql.NullString{String: params.AccountID, Valid: true},
		Name:            toNullString(params.Name),
		IsPortalEnabled: isPortalEnabled,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Carrier not found."))
	}

	return r.Get(ctx, params.AccountID, params.CarrierID)
}

func (r *carrierRepoImpl) SoftDelete(ctx context.Context, accountID, carrierID string) *apierror.APIError {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.soft_delete")
	defer span.End()

	result, err := r.queries.SoftDeleteCarrier(ctx, sqlc.SoftDeleteCarrierParams{
		ID:        carrierID,
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
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Carrier not found."))
	}

	return nil
}

func (r *carrierRepoImpl) DeleteOptionsByCarrierID(ctx context.Context, accountID, carrierID string) *apierror.APIError {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.delete_options_by_carrier_id")
	defer span.End()

	err := r.queries.DeleteCarrierOptionsByCarrierID(ctx, sqlc.DeleteCarrierOptionsByCarrierIDParams{
		CarrierID: carrierID,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *carrierRepoImpl) ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.exists_by_name")
	defer span.End()

	count, err := r.queries.CountCarriersByName(ctx, sqlc.CountCarriersByNameParams{
		Name:      name,
		AccountID: gosql.NullString{String: accountID, Valid: true},
		ExcludeID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *carrierRepoImpl) ListOptionsByCarrierID(ctx context.Context, accountID, carrierID string) ([]*domain.ServiceLevel, *apierror.APIError) {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.list_options_by_carrier_id")
	defer span.End()

	rows, err := r.queries.ListCarrierOptionsByCarrierID(ctx, sqlc.ListCarrierOptionsByCarrierIDParams{
		CarrierID: carrierID,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	levels := make([]*domain.ServiceLevel, len(rows))
	for i, row := range rows {
		levels[i] = mapServiceLevelListByCarrierRow(row)
	}
	return levels, nil
}

func mapServiceLevelListByCarrierRow(row sqlc.ListCarrierOptionsByCarrierIDRow) *domain.ServiceLevel {
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
