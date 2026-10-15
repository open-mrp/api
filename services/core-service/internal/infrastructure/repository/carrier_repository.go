package repository

import (
	"context"
	gosql "database/sql"
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

var carrierRepoTracer = tracing.GetTracer("core-service.carrier_repository")

type carrierRepoImpl struct {
	queries *sqlc.Queries
}

func NewCarrierRepo(queries *sqlc.Queries) domain.CarrierRepo {
	return &carrierRepoImpl{queries: queries}
}

func carrierCreatedAt(c *domain.Carrier) time.Time { return c.CreatedAt }
func carrierID(c *domain.Carrier) string           { return c.ID }

func mapCarrierRow(row sqlc.GetCarrierRow) *domain.Carrier {
	return &domain.Carrier{
		ID:                     row.ID,
		Name:                   row.Name,
		Type:                   constants.CarrierTypeFromStored(nullStringPtr(row.Code)),
		SCAC:                   nullStringPtr(row.Scac),
		ShippoCarrierAccountID: nullStringPtr(row.ShippoCarrierAccountID),
		AccountNumber:          nullStringPtr(row.AccountNumber),
		IsPortalEnabled:        row.IsPortalEnabled,
		IsActive:               row.IsActive,
		DefaultServiceLevelID:  nullStringPtr(row.DefaultCarrierOptionID),
		AccountID:              nullStringPtr(row.AccountID),
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}

func buildCarrierSearchParam(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func (r *carrierRepoImpl) List(ctx context.Context, params domain.ListCarriersParams) (*domain.ListCarriersResult, *apierror.APIError) {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.list")
	defer span.End()

	searchQuery := buildCarrierSearchParam(params.Query)
	accountID := gosql.NullString{String: params.AccountID, Valid: true}
	isActive := toNullBool(params.IsActive)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListCarriersBackward(ctx, sqlc.ListCarriersBackwardParams{
				AccountID:       accountID,
				SearchQuery:     searchQuery,
				IsActive:        isActive,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			carriers := make([]*domain.Carrier, len(rows))
			for i, row := range rows {
				carriers[i] = mapCarrierRow(sqlc.GetCarrierRow(row))
			}
			result, pageInfo := pagination.BuildPageString(carriers, params.Limit, cursorDir, carrierCreatedAt, carrierID)
			return &domain.ListCarriersResult{Carriers: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListCarriersForward(ctx, sqlc.ListCarriersForwardParams{
			AccountID:       accountID,
			SearchQuery:     searchQuery,
			IsActive:        isActive,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		carriers := make([]*domain.Carrier, len(rows))
		for i, row := range rows {
			carriers[i] = mapCarrierRow(sqlc.GetCarrierRow(row))
		}
		result, pageInfo := pagination.BuildPageString(carriers, params.Limit, cursorDir, carrierCreatedAt, carrierID)
		return &domain.ListCarriersResult{Carriers: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListCarriersForward(ctx, sqlc.ListCarriersForwardParams{
		AccountID:   accountID,
		SearchQuery: searchQuery,
		IsActive:    isActive,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	carriers := make([]*domain.Carrier, len(rows))
	for i, row := range rows {
		carriers[i] = mapCarrierRow(sqlc.GetCarrierRow(row))
	}
	result, pageInfo := pagination.BuildPageString(carriers, params.Limit, cursorDir, carrierCreatedAt, carrierID)
	return &domain.ListCarriersResult{Carriers: result, PageInfo: pageInfo}, nil
}

func (r *carrierRepoImpl) Get(ctx context.Context, params domain.GetCarrierParams) (*domain.Carrier, *apierror.APIError) {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.get")
	defer span.End()

	row, err := r.queries.GetCarrier(ctx, sqlc.GetCarrierParams{
		ID:        params.CarrierID,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapCarrierRow(row), nil
}

func (r *carrierRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.Carrier, *apierror.APIError) {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetCarriersByIDs(ctx, sqlc.GetCarriersByIDsParams{
		Ids:       ids,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.Carrier, len(rows))
	for i, row := range rows {
		out[i] = mapCarrierRow(sqlc.GetCarrierRow(row))
	}
	return out, nil
}

func (r *carrierRepoImpl) Create(ctx context.Context, id string, params domain.CreateCarrierParams) (*domain.Carrier, *apierror.APIError) {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.create")
	defer span.End()

	err := r.queries.InsertCarrier(ctx, sqlc.InsertCarrierParams{
		ID:                     id,
		Name:                   params.Name,
		Code:                   gosql.NullString{String: string(params.Type), Valid: true},
		Scac:                   toNullString(params.SCAC),
		ShippoCarrierAccountID: toNullString(params.ShippoCarrierAccountID),
		AccountNumber:          toNullString(params.AccountNumber),
		IsPortalEnabled:        params.IsPortalEnabled,
		IsActive:               params.IsActive,
		AccountID:              gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetCarrierParams{AccountID: params.AccountID, CarrierID: id, Includes: params.Includes})
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
		IsActive:        toNullBool(params.IsActive),
		// Assigned rather than merged so they can be cleared; the service backfills the existing values when the caller omits them.
		Scac:                   field.StringToNullString(params.SCAC),
		DefaultCarrierOptionID: field.StringToNullString(params.DefaultServiceLevelID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Carrier not found."))
	}

	return r.Get(ctx, domain.GetCarrierParams{AccountID: params.AccountID, CarrierID: params.CarrierID, Includes: params.Includes})
}

func (r *carrierRepoImpl) SetDefaultServiceLevel(ctx context.Context, accountID, carrierID string, serviceLevelID *string) *apierror.APIError {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.set_default_service_level")
	defer span.End()

	err := r.queries.SetCarrierDefaultCarrierOption(ctx, sqlc.SetCarrierDefaultCarrierOptionParams{
		DefaultCarrierOptionID: toNullString(serviceLevelID),
		ID:                     carrierID,
		AccountID:              gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *carrierRepoImpl) Delete(ctx context.Context, accountID, carrierID string) *apierror.APIError {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.delete")
	defer span.End()

	result, err := r.queries.DeleteCarrier(ctx, sqlc.DeleteCarrierParams{
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
		return tracing.Trace(span, apierror.NewNotFoundError("Carrier not found."))
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

func (r *carrierRepoImpl) ListOptionIDsForCarriers(ctx context.Context, accountID string, carrierIDs []string) (map[string][]string, *apierror.APIError) {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.list_option_ids_for_carriers")
	defer span.End()

	if len(carrierIDs) == 0 {
		return nil, nil
	}
	rows, err := r.queries.ListCarrierOptionIDsForCarriers(ctx, sqlc.ListCarrierOptionIDsForCarriersParams{
		CarrierIds: carrierIDs,
		AccountID:  gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make(map[string][]string, len(carrierIDs))
	for _, row := range rows {
		out[row.CarrierID] = append(out[row.CarrierID], row.ID)
	}
	return out, nil
}

func (r *carrierRepoImpl) GetOptionsByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.ServiceLevel, *apierror.APIError) {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.get_options_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetCarrierOptionsByIDs(ctx, sqlc.GetCarrierOptionsByIDsParams{
		Ids:       ids,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.ServiceLevel, len(rows))
	for i, row := range rows {
		out[i] = mapServiceLevelByIDRow(row)
	}
	return out, nil
}

func mapServiceLevelByIDRow(row sqlc.GetCarrierOptionsByIDsRow) *domain.ServiceLevel {
	return mapServiceLevelRow(sqlc.GetCarrierOptionRow(row))
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

// ListOptionsByCarrierIDs is the batched form of ListOptionsByCarrierID, keyed by carrier id.
func (r *carrierRepoImpl) ListOptionsByCarrierIDs(ctx context.Context, accountID string, carrierIDs []string) (map[string][]*domain.ServiceLevel, *apierror.APIError) {
	ctx, span := carrierRepoTracer.Start(ctx, "repository.carrier.list_options_by_carrier_ids")
	defer span.End()

	byCarrier := make(map[string][]*domain.ServiceLevel, len(carrierIDs))
	if len(carrierIDs) == 0 {
		return byCarrier, nil
	}
	rows, err := r.queries.ListCarrierOptionsByCarrierIDs(ctx, sqlc.ListCarrierOptionsByCarrierIDsParams{
		CarrierIds: carrierIDs,
		AccountID:  gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, row := range rows {
		level := mapServiceLevelListByCarrierRow(sqlc.ListCarrierOptionsByCarrierIDRow(row))
		byCarrier[row.CarrierID] = append(byCarrier[row.CarrierID], level)
	}
	return byCarrier, nil
}

func mapServiceLevelListByCarrierRow(row sqlc.ListCarrierOptionsByCarrierIDRow) *domain.ServiceLevel {
	return mapServiceLevelRow(sqlc.GetCarrierOptionRow(row))
}
