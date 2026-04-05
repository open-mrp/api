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

var ediRepoTracer = tracing.GetTracer("core-service.edi_repository")

type ediRepoImpl struct {
	queries *sqlc.Queries
}

func NewEDIRepo(queries *sqlc.Queries) domain.EDIRepo {
	return &ediRepoImpl{queries: queries}
}

// ---------------------------------------------------------------------------
// DC Location helpers
// ---------------------------------------------------------------------------

func dcLocationCreatedAt(d *domain.DCLocation) time.Time { return d.CreatedAt }
func dcLocationID(d *domain.DCLocation) string           { return d.ID }

func ediToNullString(s *string) gosql.NullString {
	if s == nil {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *s, Valid: true}
}

func ediBuildSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + *query + "%", Valid: true}
}

func mapDCLocationForwardRow(row sqlc.ListDCLocationsForwardRow) *domain.DCLocation {
	loc := &domain.DCLocation{
		ID:             row.ID,
		Location:       row.Location,
		AccountID:      row.AccountID,
		OwnerAccountID: row.OwnerAccountID,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
	if row.CustomerName.Valid {
		loc.CustomerName = row.CustomerName.String
	}
	return loc
}

func mapDCLocationBackwardRow(row sqlc.ListDCLocationsBackwardRow) *domain.DCLocation {
	loc := &domain.DCLocation{
		ID:             row.ID,
		Location:       row.Location,
		AccountID:      row.AccountID,
		OwnerAccountID: row.OwnerAccountID,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
	if row.CustomerName.Valid {
		loc.CustomerName = row.CustomerName.String
	}
	return loc
}

func mapGetDCLocationRow(row sqlc.GetDCLocationRow) *domain.DCLocation {
	loc := &domain.DCLocation{
		ID:             row.ID,
		Location:       row.Location,
		AccountID:      row.AccountID,
		OwnerAccountID: row.OwnerAccountID,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
	if row.CustomerName.Valid {
		loc.CustomerName = row.CustomerName.String
	}
	return loc
}

// ---------------------------------------------------------------------------
// DC Location repository methods
// ---------------------------------------------------------------------------

func (r *ediRepoImpl) ListDCLocations(ctx context.Context, params domain.ListDCLocationsParams) (*domain.ListDCLocationsResult, *apierror.APIError) {
	ctx, span := ediRepoTracer.Start(ctx, "repository.edi.list_dc_locations")
	defer span.End()

	searchQuery := ediBuildSearchParams(params.Query)
	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListDCLocationsBackward(ctx, sqlc.ListDCLocationsBackwardParams{
				OwnerAccountID:  params.OwnerAccountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			locs := make([]*domain.DCLocation, len(rows))
			for i, row := range rows {
				locs[i] = mapDCLocationBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(locs, params.Limit, cursorDir, dcLocationCreatedAt, dcLocationID)
			return &domain.ListDCLocationsResult{DCLocations: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListDCLocationsForward(ctx, sqlc.ListDCLocationsForwardParams{
			OwnerAccountID:  params.OwnerAccountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		locs := make([]*domain.DCLocation, len(rows))
		for i, row := range rows {
			locs[i] = mapDCLocationForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(locs, params.Limit, cursorDir, dcLocationCreatedAt, dcLocationID)
		return &domain.ListDCLocationsResult{DCLocations: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListDCLocationsForward(ctx, sqlc.ListDCLocationsForwardParams{
		OwnerAccountID: params.OwnerAccountID,
		SearchQuery:    searchQuery,
		Limit:          params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	locs := make([]*domain.DCLocation, len(rows))
	for i, row := range rows {
		locs[i] = mapDCLocationForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(locs, params.Limit, cursorDir, dcLocationCreatedAt, dcLocationID)
	return &domain.ListDCLocationsResult{DCLocations: result, PageInfo: pageInfo}, nil
}

func (r *ediRepoImpl) GetDCLocation(ctx context.Context, params domain.GetDCLocationParams) (*domain.DCLocation, *apierror.APIError) {
	ctx, span := ediRepoTracer.Start(ctx, "repository.edi.get_dc_location")
	defer span.End()

	row, err := r.queries.GetDCLocation(ctx, sqlc.GetDCLocationParams{
		ID:             params.DCLocationID,
		OwnerAccountID: params.OwnerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetDCLocationRow(row), nil
}

func (r *ediRepoImpl) CreateDCLocation(ctx context.Context, id string, params domain.CreateDCLocationParams) (*domain.DCLocation, *apierror.APIError) {
	ctx, span := ediRepoTracer.Start(ctx, "repository.edi.create_dc_location")
	defer span.End()

	err := r.queries.InsertDCLocation(ctx, sqlc.InsertDCLocationParams{
		ID:             id,
		Location:       params.Location,
		AccountID:      params.AccountID,
		OwnerAccountID: params.OwnerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.GetDCLocation(ctx, domain.GetDCLocationParams{OwnerAccountID: params.OwnerAccountID, DCLocationID: id})
}

func (r *ediRepoImpl) UpdateDCLocation(ctx context.Context, params domain.UpdateDCLocationParams) (*domain.DCLocation, *apierror.APIError) {
	ctx, span := ediRepoTracer.Start(ctx, "repository.edi.update_dc_location")
	defer span.End()

	result, err := r.queries.UpdateDCLocation(ctx, sqlc.UpdateDCLocationParams{
		ID:             params.DCLocationID,
		OwnerAccountID: params.OwnerAccountID,
		Location:       ediToNullString(params.Location),
		AccountID:      ediToNullString(params.AccountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("DC location not found."))
	}

	return r.GetDCLocation(ctx, domain.GetDCLocationParams{OwnerAccountID: params.OwnerAccountID, DCLocationID: params.DCLocationID})
}

func (r *ediRepoImpl) DeleteDCLocation(ctx context.Context, params domain.DeleteDCLocationParams) *apierror.APIError {
	ctx, span := ediRepoTracer.Start(ctx, "repository.edi.delete_dc_location")
	defer span.End()

	result, err := r.queries.DeleteDCLocation(ctx, sqlc.DeleteDCLocationParams{
		ID:             params.DCLocationID,
		OwnerAccountID: params.OwnerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("DC location not found."))
	}

	return nil
}

// ---------------------------------------------------------------------------
// EDI Run helpers
// ---------------------------------------------------------------------------

func ediRunCompletedAt(e *domain.EDIRun) time.Time { return e.CompletedAt }
func ediRunID(e *domain.EDIRun) string             { return e.ID }

func mapEDIRunForwardRow(row sqlc.ListEDIRunsForwardRow) *domain.EDIRun {
	return &domain.EDIRun{
		ID:           row.ID,
		CompletedAt:  row.CompletedAt,
		HasSucceeded: row.HasSucceeded,
		AccountID:    row.AccountID,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func mapEDIRunBackwardRow(row sqlc.ListEDIRunsBackwardRow) *domain.EDIRun {
	return &domain.EDIRun{
		ID:           row.ID,
		CompletedAt:  row.CompletedAt,
		HasSucceeded: row.HasSucceeded,
		AccountID:    row.AccountID,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func mapGetEDIRunRow(row sqlc.GetEDIRunRow) *domain.EDIRun {
	return &domain.EDIRun{
		ID:           row.ID,
		CompletedAt:  row.CompletedAt,
		HasSucceeded: row.HasSucceeded,
		AccountID:    row.AccountID,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func ediBoolToNullBool(b *bool) gosql.NullBool {
	if b == nil {
		return gosql.NullBool{}
	}
	return gosql.NullBool{Bool: *b, Valid: true}
}

// ---------------------------------------------------------------------------
// EDI Run repository methods
// ---------------------------------------------------------------------------

func optionalEDISearch(q *string) interface{} {
	if q == nil || *q == "" {
		return nil
	}
	return *q
}

func (r *ediRepoImpl) ListEDIRuns(ctx context.Context, params domain.ListEDIRunsParams) (*domain.ListEDIRunsResult, *apierror.APIError) {
	ctx, span := ediRepoTracer.Start(ctx, "repository.edi.list_edi_runs")
	defer span.End()

	hasSucceeded := ediBoolToNullBool(params.HasSucceeded)
	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListEDIRunsBackward(ctx, sqlc.ListEDIRunsBackwardParams{
				AccountID:         params.AccountID,
				HasSucceeded:      hasSucceeded,
				Search:            optionalEDISearch(params.Query),
				CursorCompletedAt: cur.OccurredAt,
				CursorID:          cur.ID,
				Limit:             params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			runs := make([]*domain.EDIRun, len(rows))
			for i, row := range rows {
				runs[i] = mapEDIRunBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(runs, params.Limit, cursorDir, ediRunCompletedAt, ediRunID)
			return &domain.ListEDIRunsResult{EDIRuns: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListEDIRunsForward(ctx, sqlc.ListEDIRunsForwardParams{
			AccountID:         params.AccountID,
			HasSucceeded:      hasSucceeded,
			Search:            optionalEDISearch(params.Query),
			CursorCompletedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:          gosql.NullString{String: cur.ID, Valid: true},
			Limit:             params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		runs := make([]*domain.EDIRun, len(rows))
		for i, row := range rows {
			runs[i] = mapEDIRunForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(runs, params.Limit, cursorDir, ediRunCompletedAt, ediRunID)
		return &domain.ListEDIRunsResult{EDIRuns: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListEDIRunsForward(ctx, sqlc.ListEDIRunsForwardParams{
		AccountID:         params.AccountID,
		HasSucceeded:      hasSucceeded,
		Search:            optionalEDISearch(params.Query),
		CursorCompletedAt: gosql.NullTime{},
		CursorID:          gosql.NullString{},
		Limit:             params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	runs := make([]*domain.EDIRun, len(rows))
	for i, row := range rows {
		runs[i] = mapEDIRunForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(runs, params.Limit, cursorDir, ediRunCompletedAt, ediRunID)
	return &domain.ListEDIRunsResult{EDIRuns: result, PageInfo: pageInfo}, nil
}

func (r *ediRepoImpl) GetEDIRun(ctx context.Context, accountID, ediRunID string) (*domain.EDIRun, *apierror.APIError) {
	ctx, span := ediRepoTracer.Start(ctx, "repository.edi.get_edi_run")
	defer span.End()

	row, err := r.queries.GetEDIRun(ctx, sqlc.GetEDIRunParams{
		ID:        ediRunID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetEDIRunRow(row), nil
}
