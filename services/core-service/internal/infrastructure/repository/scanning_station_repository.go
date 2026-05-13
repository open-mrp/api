package repository

import (
	"context"
	gosql "database/sql"
	"slices"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var scanningStationRepoTracer = tracing.GetTracer("core-service.scanning_station_repository")

type scanningStationRepoImpl struct {
	queries *sqlc.Queries
}

func NewScanningStationRepo(queries *sqlc.Queries) domain.ScanningStationRepo {
	return &scanningStationRepoImpl{queries: queries}
}

func ssCreatedAt(ss *domain.ScanningStation) time.Time { return ss.CreatedAt }
func ssID(ss *domain.ScanningStation) string           { return ss.ID }

func ssBuildSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func boolToOperatorRequirement(v bool) constants.OperatorRequirement {
	if v {
		return constants.OperatorRequirementMaterialCheck
	}
	return constants.OperatorRequirementNone
}

func operatorRequirementToBool(r constants.OperatorRequirement) bool {
	return r == constants.OperatorRequirementMaterialCheck
}

func mapScanningStationForwardRow(row sqlc.ListScanningStationsForwardRow) *domain.ScanningStation {
	ss := &domain.ScanningStation{
		ID:                  row.ID,
		Name:                row.Name,
		Type:                constants.ScanningStationType(row.ScanningStationTypeCode),
		OperatorRequirement: boolToOperatorRequirement(row.MaterialCheckRequired),
		DepartmentID:        row.DepartmentID,
		AccountID:           row.AccountID,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
	if row.Notes.Valid {
		ss.Notes = &row.Notes.String
	}
	if row.LabelSizeCode.Valid {
		ss.LabelSizeCode = &row.LabelSizeCode.String
	}
	if row.LabelTypeCode.Valid {
		ss.LabelTypeCode = &row.LabelTypeCode.String
	}
	if row.DepartmentName.Valid {
		ss.DepartmentName = row.DepartmentName.String
	}
	if row.DepartmentCreatedAt.Valid {
		ss.DepartmentCreatedAt = &row.DepartmentCreatedAt.Time
	}
	if row.DepartmentUpdatedAt.Valid {
		ss.DepartmentUpdatedAt = &row.DepartmentUpdatedAt.Time
	}
	return ss
}

func mapScanningStationBackwardRow(row sqlc.ListScanningStationsBackwardRow) *domain.ScanningStation {
	ss := &domain.ScanningStation{
		ID:                  row.ID,
		Name:                row.Name,
		Type:                constants.ScanningStationType(row.ScanningStationTypeCode),
		OperatorRequirement: boolToOperatorRequirement(row.MaterialCheckRequired),
		DepartmentID:        row.DepartmentID,
		AccountID:           row.AccountID,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
	if row.Notes.Valid {
		ss.Notes = &row.Notes.String
	}
	if row.LabelSizeCode.Valid {
		ss.LabelSizeCode = &row.LabelSizeCode.String
	}
	if row.LabelTypeCode.Valid {
		ss.LabelTypeCode = &row.LabelTypeCode.String
	}
	if row.DepartmentName.Valid {
		ss.DepartmentName = row.DepartmentName.String
	}
	if row.DepartmentCreatedAt.Valid {
		ss.DepartmentCreatedAt = &row.DepartmentCreatedAt.Time
	}
	if row.DepartmentUpdatedAt.Valid {
		ss.DepartmentUpdatedAt = &row.DepartmentUpdatedAt.Time
	}
	return ss
}

func (r *scanningStationRepoImpl) List(ctx context.Context, params domain.ListScanningStationsParams) (*domain.ListScanningStationsResult, *apierror.APIError) {
	ctx, span := scanningStationRepoTracer.Start(ctx, "repository.scanning_station.list")
	defer span.End()

	searchQuery := ssBuildSearchParams(params.Query)
	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListScanningStationsBackward(ctx, sqlc.ListScanningStationsBackwardParams{
				AccountID:       params.AccountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			stations := make([]*domain.ScanningStation, len(rows))
			for i, row := range rows {
				stations[i] = mapScanningStationBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(stations, params.Limit, cursorDir, ssCreatedAt, ssID)

			if slices.Contains(params.Includes, "production_steps") {
				for _, ss := range result {
					if apiErr := r.attachSubResources(ctx, ss); apiErr != nil {
						return nil, tracing.Trace(span, apiErr)
					}
				}
			}

			return &domain.ListScanningStationsResult{ScanningStations: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListScanningStationsForward(ctx, sqlc.ListScanningStationsForwardParams{
			AccountID:       params.AccountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		stations := make([]*domain.ScanningStation, len(rows))
		for i, row := range rows {
			stations[i] = mapScanningStationForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(stations, params.Limit, cursorDir, ssCreatedAt, ssID)

		if slices.Contains(params.Includes, "production_steps") {
			for _, ss := range result {
				if apiErr := r.attachSubResources(ctx, ss); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
			}
		}

		return &domain.ListScanningStationsResult{ScanningStations: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListScanningStationsForward(ctx, sqlc.ListScanningStationsForwardParams{
		AccountID:   params.AccountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	stations := make([]*domain.ScanningStation, len(rows))
	for i, row := range rows {
		stations[i] = mapScanningStationForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(stations, params.Limit, cursorDir, ssCreatedAt, ssID)

	if slices.Contains(params.Includes, "production_steps") {
		for _, ss := range result {
			if apiErr := r.attachSubResources(ctx, ss); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}

	return &domain.ListScanningStationsResult{ScanningStations: result, PageInfo: pageInfo}, nil
}

func (r *scanningStationRepoImpl) Get(ctx context.Context, params domain.GetScanningStationParams) (*domain.ScanningStation, *apierror.APIError) {
	ctx, span := scanningStationRepoTracer.Start(ctx, "repository.scanning_station.get")
	defer span.End()

	row, err := r.queries.GetScanningStation(ctx, sqlc.GetScanningStationParams{
		ID:        params.ScanningStationID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	ss := mapGetScanningStationRow(row)
	if slices.Contains(params.Includes, "production_steps") {
		if apiErr := r.attachSubResources(ctx, ss); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return ss, nil
}

func mapGetScanningStationRow(row sqlc.GetScanningStationRow) *domain.ScanningStation {
	ss := &domain.ScanningStation{
		ID:                  row.ID,
		Name:                row.Name,
		Type:                constants.ScanningStationType(row.ScanningStationTypeCode),
		OperatorRequirement: boolToOperatorRequirement(row.MaterialCheckRequired),
		DepartmentID:        row.DepartmentID,
		AccountID:           row.AccountID,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
	if row.Notes.Valid {
		ss.Notes = &row.Notes.String
	}
	if row.LabelSizeCode.Valid {
		ss.LabelSizeCode = &row.LabelSizeCode.String
	}
	if row.LabelTypeCode.Valid {
		ss.LabelTypeCode = &row.LabelTypeCode.String
	}
	if row.DepartmentName.Valid {
		ss.DepartmentName = row.DepartmentName.String
	}
	if row.DepartmentCreatedAt.Valid {
		ss.DepartmentCreatedAt = &row.DepartmentCreatedAt.Time
	}
	if row.DepartmentUpdatedAt.Valid {
		ss.DepartmentUpdatedAt = &row.DepartmentUpdatedAt.Time
	}
	return ss
}

func (r *scanningStationRepoImpl) attachSubResources(ctx context.Context, ss *domain.ScanningStation) *apierror.APIError {
	steps, err := r.queries.ListProductionStepsByScanningStationID(ctx, sqlc.ListProductionStepsByScanningStationIDParams{
		ScanningStationID: gosql.NullString{String: ss.ID, Valid: true},
		AccountID:         ss.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	ss.ProductionSteps = make([]domain.ProductionStepRef, len(steps))
	for i, s := range steps {
		ss.ProductionSteps[i] = domain.ProductionStepRef{
			ID:             s.ID,
			Name:           s.Name,
			LevelingFactor: s.LevelingFactor,
			Allowances:     s.Allowances,
			CreatedAt:      s.CreatedAt,
			UpdatedAt:      s.UpdatedAt,
		}
	}
	return nil
}

func (r *scanningStationRepoImpl) Create(ctx context.Context, id string, params domain.CreateScanningStationParams) (*domain.ScanningStation, *apierror.APIError) {
	ctx, span := scanningStationRepoTracer.Start(ctx, "repository.scanning_station.create")
	defer span.End()

	err := r.queries.InsertScanningStation(ctx, sqlc.InsertScanningStationParams{
		ID:                      id,
		Name:                    params.Name,
		Notes:                   toNullString(params.Notes),
		ScanningStationTypeCode: string(params.Type),
		MaterialCheckRequired:   operatorRequirementToBool(params.OperatorRequirement),
		DepartmentID:            params.DepartmentID,
		AccountID:               params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetScanningStationParams{AccountID: params.AccountID, ScanningStationID: id, Includes: params.Includes})
}

func (r *scanningStationRepoImpl) Update(ctx context.Context, params domain.UpdateScanningStationParams) (*domain.ScanningStation, *apierror.APIError) {
	ctx, span := scanningStationRepoTracer.Start(ctx, "repository.scanning_station.update")
	defer span.End()

	result, err := r.queries.UpdateScanningStation(ctx, sqlc.UpdateScanningStationParams{
		ID:            params.ScanningStationID,
		AccountID:     params.AccountID,
		Name:          toNullString(params.Name),
		Notes:         toNullString(params.Notes),
		LabelSizeCode: toNullString(params.LabelSizeCode),
		LabelTypeCode: toNullString(params.LabelTypeCode),
		MaterialCheckRequired: toNullBool(func() *bool {
			if params.OperatorRequirement == nil {
				return nil
			}
			v := operatorRequirementToBool(*params.OperatorRequirement)
			return &v
		}()),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Scanning station not found."))
	}

	return r.Get(ctx, domain.GetScanningStationParams{AccountID: params.AccountID, ScanningStationID: params.ScanningStationID, Includes: params.Includes})
}

func (r *scanningStationRepoImpl) Delete(ctx context.Context, params domain.DeleteScanningStationParams) *apierror.APIError {
	ctx, span := scanningStationRepoTracer.Start(ctx, "repository.scanning_station.delete")
	defer span.End()

	result, err := r.queries.DeleteScanningStation(ctx, sqlc.DeleteScanningStationParams{
		ID:        params.ScanningStationID,
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
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Scanning station not found."))
	}

	return nil
}

func (r *scanningStationRepoImpl) ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := scanningStationRepoTracer.Start(ctx, "repository.scanning_station.exists_by_name")
	defer span.End()

	count, err := r.queries.CountScanningStationsByName(ctx, sqlc.CountScanningStationsByNameParams{
		Name:      name,
		AccountID: accountID,
		ExcludeID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return count > 0, nil
}

func (r *scanningStationRepoImpl) FindIDByName(ctx context.Context, accountID, name string) (*string, *apierror.APIError) {
	ctx, span := scanningStationRepoTracer.Start(ctx, "repository.scanning_station.find_id_by_name")
	defer span.End()

	id, err := r.queries.FindScanningStationIDByName(ctx, sqlc.FindScanningStationIDByNameParams{
		Name:      name,
		AccountID: accountID,
	})
	if err != nil {
		if err == gosql.ErrNoRows {
			return nil, nil
		}
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	return &id, nil
}

func (r *scanningStationRepoImpl) ConnectProductionStepsByName(ctx context.Context, accountID, scanningStationID, name string) *apierror.APIError {
	ctx, span := scanningStationRepoTracer.Start(ctx, "repository.scanning_station.connect_production_steps_by_name")
	defer span.End()

	_, err := r.queries.ConnectProductionStepsByName(ctx, sqlc.ConnectProductionStepsByNameParams{
		ScanningStationID: gosql.NullString{String: scanningStationID, Valid: true},
		AccountID:         accountID,
		Name:              name,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *scanningStationRepoImpl) IsInAccount(ctx context.Context, accountID, id string) (bool, *apierror.APIError) {
	ctx, span := scanningStationRepoTracer.Start(ctx, "repository.scanning_station.is_in_account")
	defer span.End()

	count, err := r.queries.IsScanningStationInAccount(ctx, sqlc.IsScanningStationInAccountParams{
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return count > 0, nil
}

func (r *scanningStationRepoImpl) FindType(ctx context.Context, accountID, id string) (string, *apierror.APIError) {
	ctx, span := scanningStationRepoTracer.Start(ctx, "repository.scanning_station.find_type")
	defer span.End()

	stationType, err := r.queries.GetScanningStationType(ctx, sqlc.GetScanningStationTypeParams{
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return stationType, nil
}
