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

var machineRepoTracer = tracing.GetTracer("core-service.machine_repository")

type machineRepoImpl struct {
	queries *sqlc.Queries
}

func NewMachineRepo(queries *sqlc.Queries) domain.MachineRepo {
	return &machineRepoImpl{queries: queries}
}

func machineCreatedAt(m *domain.Machine) time.Time { return m.CreatedAt }
func machineID(m *domain.Machine) string           { return m.ID }

func machToNullString(s *string) gosql.NullString {
	if s == nil {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *s, Valid: true}
}

func machBuildSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *query + "%", Valid: true}
}

func mapMachineForwardRow(row sqlc.ListMachinesForwardRow) *domain.Machine {
	m := &domain.Machine{
		ID:                  row.ID,
		Name:                row.Name,
		SerialNumber:        row.SerialNumber,
		DepartmentID:        &row.DepartmentID,
		DepartmentName:      &row.DepartmentName,
		DepartmentCreatedAt: &row.DepartmentCreatedAt,
		DepartmentUpdatedAt: &row.DepartmentUpdatedAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
	if row.Notes.Valid {
		m.Notes = &row.Notes.String
	}
	if row.ProductionStepID.Valid {
		m.ProductionStepID = &row.ProductionStepID.String
	}
	return m
}

func mapMachineBackwardRow(row sqlc.ListMachinesBackwardRow) *domain.Machine {
	m := &domain.Machine{
		ID:                  row.ID,
		Name:                row.Name,
		SerialNumber:        row.SerialNumber,
		DepartmentID:        &row.DepartmentID,
		DepartmentName:      &row.DepartmentName,
		DepartmentCreatedAt: &row.DepartmentCreatedAt,
		DepartmentUpdatedAt: &row.DepartmentUpdatedAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
	if row.Notes.Valid {
		m.Notes = &row.Notes.String
	}
	if row.ProductionStepID.Valid {
		m.ProductionStepID = &row.ProductionStepID.String
	}
	return m
}

func mapGetMachineRow(row sqlc.GetMachineRow) *domain.Machine {
	m := &domain.Machine{
		ID:                  row.ID,
		Name:                row.Name,
		SerialNumber:        row.SerialNumber,
		DepartmentID:        &row.DepartmentID,
		DepartmentName:      &row.DepartmentName,
		DepartmentCreatedAt: &row.DepartmentCreatedAt,
		DepartmentUpdatedAt: &row.DepartmentUpdatedAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
	if row.Notes.Valid {
		m.Notes = &row.Notes.String
	}
	if row.ProductionStepID.Valid {
		m.ProductionStepID = &row.ProductionStepID.String
	}
	return m
}

func mapGetMachinesByIDsRow(row sqlc.GetMachinesByIDsRow) *domain.Machine {
	m := &domain.Machine{
		ID:                  row.ID,
		Name:                row.Name,
		SerialNumber:        row.SerialNumber,
		DepartmentID:        &row.DepartmentID,
		DepartmentName:      &row.DepartmentName,
		DepartmentCreatedAt: &row.DepartmentCreatedAt,
		DepartmentUpdatedAt: &row.DepartmentUpdatedAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
	if row.Notes.Valid {
		m.Notes = &row.Notes.String
	}
	if row.ProductionStepID.Valid {
		m.ProductionStepID = &row.ProductionStepID.String
	}
	return m
}

func (r *machineRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.Machine, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.get_by_ids")
	defer span.End()

	rows, err := r.queries.GetMachinesByIDs(ctx, sqlc.GetMachinesByIDsParams{
		Ids:       ids,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	machines := make([]*domain.Machine, len(rows))
	for i, row := range rows {
		machines[i] = mapGetMachinesByIDsRow(row)
	}
	return machines, nil
}

func (r *machineRepoImpl) List(ctx context.Context, params domain.ListMachinesParams) (*domain.ListMachinesResult, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.list")
	defer span.End()

	searchQuery := machBuildSearchParams(params.Query)
	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListMachinesBackward(ctx, sqlc.ListMachinesBackwardParams{
				AccountID:       params.AccountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			machines := make([]*domain.Machine, len(rows))
			for i, row := range rows {
				machines[i] = mapMachineBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(machines, params.Limit, cursorDir, machineCreatedAt, machineID)
			return &domain.ListMachinesResult{Machines: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListMachinesForward(ctx, sqlc.ListMachinesForwardParams{
			AccountID:       params.AccountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		machines := make([]*domain.Machine, len(rows))
		for i, row := range rows {
			machines[i] = mapMachineForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(machines, params.Limit, cursorDir, machineCreatedAt, machineID)
		return &domain.ListMachinesResult{Machines: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListMachinesForward(ctx, sqlc.ListMachinesForwardParams{
		AccountID:   params.AccountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	machines := make([]*domain.Machine, len(rows))
	for i, row := range rows {
		machines[i] = mapMachineForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(machines, params.Limit, cursorDir, machineCreatedAt, machineID)
	return &domain.ListMachinesResult{Machines: result, PageInfo: pageInfo}, nil
}

func (r *machineRepoImpl) Get(ctx context.Context, params domain.GetMachineParams) (*domain.Machine, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.get")
	defer span.End()

	row, err := r.queries.GetMachine(ctx, sqlc.GetMachineParams{
		ID:        params.MachineID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetMachineRow(row), nil
}

func (r *machineRepoImpl) Create(ctx context.Context, id string, params domain.CreateMachineParams) (*domain.Machine, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.create")
	defer span.End()

	err := r.queries.InsertMachine(ctx, sqlc.InsertMachineParams{
		ID:           id,
		Name:         params.Name,
		SerialNumber: params.SerialNumber,
		Notes:        machToNullString(params.Notes),
		DepartmentID: params.DepartmentID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetMachineParams{AccountID: params.AccountID, MachineID: id})
}

func (r *machineRepoImpl) Update(ctx context.Context, params domain.UpdateMachineParams) (*domain.Machine, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.update")
	defer span.End()

	result, err := r.queries.UpdateMachine(ctx, sqlc.UpdateMachineParams{
		ID:           params.MachineID,
		AccountID:    params.AccountID,
		Name:         machToNullString(params.Name),
		SerialNumber: machToNullString(params.SerialNumber),
		Notes:        machToNullString(params.Notes),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Machine not found."))
	}

	return r.Get(ctx, domain.GetMachineParams{AccountID: params.AccountID, MachineID: params.MachineID})
}

func (r *machineRepoImpl) Delete(ctx context.Context, params domain.DeleteMachineParams) *apierror.APIError {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.delete")
	defer span.End()

	result, err := r.queries.DeleteMachine(ctx, sqlc.DeleteMachineParams{
		ID:        params.MachineID,
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
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Machine not found."))
	}

	return nil
}

func (r *machineRepoImpl) ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.exists_by_name")
	defer span.End()

	count, err := r.queries.CountMachinesByName(ctx, sqlc.CountMachinesByNameParams{
		Name:      name,
		AccountID: accountID,
		ExcludeID: machToNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}
