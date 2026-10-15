package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
)

var departmentRepoTracer = tracing.GetTracer("core-service.department_repository")

type departmentRepoImpl struct {
	queries *sqlc.Queries
}

func NewDepartmentRepo(queries *sqlc.Queries) domain.DepartmentRepo {
	return &departmentRepoImpl{queries: queries}
}

func departmentCreatedAt(d *domain.Department) time.Time { return d.CreatedAt }
func departmentID(d *domain.Department) string           { return d.ID }

func deptToNullString(s *string) gosql.NullString {
	if s == nil {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *s, Valid: true}
}

func deptBuildSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func mapDepartmentForwardRow(row sqlc.ListDepartmentsForwardRow) *domain.Department {
	dept := &domain.Department{
		ID:        row.ID,
		Name:      row.Name,
		AccountID: row.AccountID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.Notes.Valid {
		dept.Notes = &row.Notes.String
	}
	if row.LocationID.Valid {
		dept.LocationID = &row.LocationID.String
	}
	if row.LocationName.Valid {
		dept.LocationName = &row.LocationName.String
	}
	if row.LocationTypeCode.Valid {
		dept.LocationTypeCode = &row.LocationTypeCode.String
	}
	dept.LaborRate = unitPriceFromColumns(row.LaborRateAmount, row.LaborRateCurrency, row.LaborRatePerUnitID,
		row.LegacyLaborRateValue, row.LegacyLaborRatePerUnitID)
	dept.LaborRateRowID = nullStringPtr(row.LaborRateID)
	return dept
}

func mapDepartmentBackwardRow(row sqlc.ListDepartmentsBackwardRow) *domain.Department {
	dept := &domain.Department{
		ID:        row.ID,
		Name:      row.Name,
		AccountID: row.AccountID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.Notes.Valid {
		dept.Notes = &row.Notes.String
	}
	if row.LocationID.Valid {
		dept.LocationID = &row.LocationID.String
	}
	if row.LocationName.Valid {
		dept.LocationName = &row.LocationName.String
	}
	if row.LocationTypeCode.Valid {
		dept.LocationTypeCode = &row.LocationTypeCode.String
	}
	dept.LaborRate = unitPriceFromColumns(row.LaborRateAmount, row.LaborRateCurrency, row.LaborRatePerUnitID,
		row.LegacyLaborRateValue, row.LegacyLaborRatePerUnitID)
	dept.LaborRateRowID = nullStringPtr(row.LaborRateID)
	return dept
}

func mapGetDepartmentRow(row sqlc.GetDepartmentRow) *domain.Department {
	dept := &domain.Department{
		ID:        row.ID,
		Name:      row.Name,
		AccountID: row.AccountID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.Notes.Valid {
		dept.Notes = &row.Notes.String
	}
	if row.LocationID.Valid {
		dept.LocationID = &row.LocationID.String
	}
	if row.LocationName.Valid {
		dept.LocationName = &row.LocationName.String
	}
	if row.LocationTypeCode.Valid {
		dept.LocationTypeCode = &row.LocationTypeCode.String
	}
	dept.LaborRate = unitPriceFromColumns(row.LaborRateAmount, row.LaborRateCurrency, row.LaborRatePerUnitID,
		row.LegacyLaborRateValue, row.LegacyLaborRatePerUnitID)
	dept.LaborRateRowID = nullStringPtr(row.LaborRateID)
	return dept
}

// attachMembers loads the departments' scanning stations and machines, newest first, in one query apiece. A positive
// limit keeps that many of each per department.
func (r *departmentRepoImpl) attachMembers(ctx context.Context, accountID string, departments []*domain.Department, limit int) *apierror.APIError {
	if len(departments) == 0 {
		return nil
	}
	ids := make([]string, len(departments))
	for i, d := range departments {
		ids[i] = d.ID
	}

	stationRows, err := r.queries.ListScanningStationsByDepartmentIDs(ctx, sqlc.ListScanningStationsByDepartmentIDsParams{
		DepartmentIds: ids,
		AccountID:     accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	stationsByDept := make(map[string][]domain.DepartmentScanningStation, len(ids))
	for _, s := range stationRows {
		if limit > 0 && len(stationsByDept[s.DepartmentID]) >= limit {
			continue
		}
		stationsByDept[s.DepartmentID] = append(stationsByDept[s.DepartmentID], domain.DepartmentScanningStation{
			ID:                  s.ID,
			Name:                s.Name,
			Type:                s.ScanningStationTypeCode,
			OperatorRequirement: string(boolToOperatorRequirement(s.MaterialCheckRequired)),
			CreatedAt:           s.CreatedAt,
			UpdatedAt:           s.UpdatedAt,
		})
	}

	machineRows, err := r.queries.ListMachinesByDepartmentIDs(ctx, sqlc.ListMachinesByDepartmentIDsParams{
		DepartmentIds: ids,
		AccountID:     accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	machinesByDept := make(map[string][]domain.DepartmentMachine, len(ids))
	for _, m := range machineRows {
		if limit > 0 && len(machinesByDept[m.DepartmentID]) >= limit {
			continue
		}
		machinesByDept[m.DepartmentID] = append(machinesByDept[m.DepartmentID], domain.DepartmentMachine{
			ID:           m.ID,
			Name:         m.Name,
			SerialNumber: m.SerialNumber,
			CreatedAt:    m.CreatedAt,
			UpdatedAt:    m.UpdatedAt,
		})
	}

	for _, dept := range departments {
		dept.ScanningStations = stationsByDept[dept.ID]
		dept.Machines = machinesByDept[dept.ID]
	}
	return nil
}

func (r *departmentRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.Department, *apierror.APIError) {
	ctx, span := departmentRepoTracer.Start(ctx, "repository.department.get_by_ids")
	defer span.End()

	rows, err := r.queries.GetDepartmentsFullByIDs(ctx, sqlc.GetDepartmentsFullByIDsParams{
		Ids:       ids,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	departments := make([]*domain.Department, len(rows))
	for i, row := range rows {
		dept := &domain.Department{
			ID:        row.ID,
			Name:      row.Name,
			AccountID: row.AccountID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
		if row.Notes.Valid {
			dept.Notes = &row.Notes.String
		}
		if row.LocationID.Valid {
			dept.LocationID = &row.LocationID.String
		}
		dept.LaborRate = unitPriceFromColumns(row.LaborRateAmount, row.LaborRateCurrency, row.LaborRatePerUnitID,
			row.LegacyLaborRateValue, row.LegacyLaborRatePerUnitID)
		dept.LaborRateRowID = nullStringPtr(row.LaborRateID)
		departments[i] = dept
	}

	if apiErr := r.attachMembers(ctx, accountID, departments, domain.EmbeddedDepartmentMembers); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return departments, nil
}

func (r *departmentRepoImpl) Export(ctx context.Context, params domain.ExportDepartmentsParams) ([]*domain.Department, *apierror.APIError) {
	ctx, span := departmentRepoTracer.Start(ctx, "repository.department.export")
	defer span.End()

	rows, err := r.queries.ExportDepartments(ctx, sqlc.ExportDepartmentsParams{
		AccountID:   params.AccountID,
		SearchQuery: deptBuildSearchParams(params.Query),
		Limit:       exportQueryLimit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	departments := make([]*domain.Department, len(rows))
	ids := make([]string, len(rows))
	for i, row := range rows {
		dept := &domain.Department{
			ID:        row.ID,
			Name:      row.Name,
			AccountID: row.AccountID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
		if row.Notes.Valid {
			dept.Notes = &row.Notes.String
		}
		if row.LocationID.Valid {
			dept.LocationID = &row.LocationID.String
		}
		if row.LocationName.Valid {
			dept.LocationName = &row.LocationName.String
		}
		departments[i] = dept
		ids[i] = row.ID
	}

	// The sheet lists each department's stations and machines.
	if apiErr := r.attachMembers(ctx, params.AccountID, departments, 0); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return departments, nil
}

func (r *departmentRepoImpl) List(ctx context.Context, params domain.ListDepartmentsParams) (*domain.ListDepartmentsResult, *apierror.APIError) {
	ctx, span := departmentRepoTracer.Start(ctx, "repository.department.list")
	defer span.End()

	searchQuery := deptBuildSearchParams(params.Query)
	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListDepartmentsBackward(ctx, sqlc.ListDepartmentsBackwardParams{
				AccountID:       params.AccountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			depts := make([]*domain.Department, len(rows))
			for i, row := range rows {
				depts[i] = mapDepartmentBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(depts, params.Limit, cursorDir, departmentCreatedAt, departmentID)

			return &domain.ListDepartmentsResult{Departments: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListDepartmentsForward(ctx, sqlc.ListDepartmentsForwardParams{
			AccountID:       params.AccountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		depts := make([]*domain.Department, len(rows))
		for i, row := range rows {
			depts[i] = mapDepartmentForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(depts, params.Limit, cursorDir, departmentCreatedAt, departmentID)

		return &domain.ListDepartmentsResult{Departments: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListDepartmentsForward(ctx, sqlc.ListDepartmentsForwardParams{
		AccountID:   params.AccountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	depts := make([]*domain.Department, len(rows))
	for i, row := range rows {
		depts[i] = mapDepartmentForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(depts, params.Limit, cursorDir, departmentCreatedAt, departmentID)

	return &domain.ListDepartmentsResult{Departments: result, PageInfo: pageInfo}, nil
}

func (r *departmentRepoImpl) Get(ctx context.Context, params domain.GetDepartmentParams) (*domain.Department, *apierror.APIError) {
	ctx, span := departmentRepoTracer.Start(ctx, "repository.department.get")
	defer span.End()

	row, err := r.queries.GetDepartment(ctx, sqlc.GetDepartmentParams{
		ID:        params.DepartmentID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	dept := mapGetDepartmentRow(row)
	if apiErr := r.attachMembers(ctx, params.AccountID, []*domain.Department{dept}, 0); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return dept, nil
}

func (r *departmentRepoImpl) Create(ctx context.Context, id string, params domain.CreateDepartmentParams) (*domain.Department, *apierror.APIError) {
	ctx, span := departmentRepoTracer.Start(ctx, "repository.department.create")
	defer span.End()

	insert := sqlc.InsertDepartmentParams{
		ID:          id,
		Name:        params.Name,
		Notes:       deptToNullString(params.Notes),
		LocationID:  deptToNullString(params.LocationID),
		LaborRateID: deptToNullString(params.LaborRateID),
		AccountID:   params.AccountID,
	}
	insert.LaborRateAmount, insert.LaborRateCurrency, insert.LaborRatePerUnitID = unitPriceParams(params.LaborRate)
	err := r.queries.InsertDepartment(ctx, insert)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetDepartmentParams{AccountID: params.AccountID, DepartmentID: id})
}

func (r *departmentRepoImpl) Update(ctx context.Context, params domain.UpdateDepartmentParams) (*domain.Department, *apierror.APIError) {
	ctx, span := departmentRepoTracer.Start(ctx, "repository.department.update")
	defer span.End()

	update := sqlc.UpdateDepartmentParams{
		ID:          params.DepartmentID,
		AccountID:   params.AccountID,
		Name:        deptToNullString(params.Name),
		Notes:       field.StringToNullString(params.Notes),
		LocationID:  deptToNullString(params.LocationID),
		LaborRateID: deptToNullString(params.LaborRateID),
	}
	update.LaborRateAmount, update.LaborRateCurrency, update.LaborRatePerUnitID = unitPriceParams(params.LaborRate)
	result, err := r.queries.UpdateDepartment(ctx, update)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Department not found."))
	}

	return r.Get(ctx, domain.GetDepartmentParams{AccountID: params.AccountID, DepartmentID: params.DepartmentID})
}

func (r *departmentRepoImpl) Delete(ctx context.Context, params domain.DeleteDepartmentParams) *apierror.APIError {
	ctx, span := departmentRepoTracer.Start(ctx, "repository.department.delete")
	defer span.End()

	result, err := r.queries.DeleteDepartment(ctx, sqlc.DeleteDepartmentParams{
		ID:        params.DepartmentID,
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
		return tracing.Trace(span, apierror.NewNotFoundError("Department not found."))
	}

	return nil
}

func (r *departmentRepoImpl) ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := departmentRepoTracer.Start(ctx, "repository.department.exists_by_name")
	defer span.End()

	count, err := r.queries.CountDepartmentsByName(ctx, sqlc.CountDepartmentsByNameParams{
		Name:      name,
		AccountID: accountID,
		ExcludeID: deptToNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *departmentRepoImpl) FindByNames(ctx context.Context, accountID string, names []string) ([]*domain.Department, *apierror.APIError) {
	ctx, span := departmentRepoTracer.Start(ctx, "repository.department.find_by_names")
	defer span.End()

	if len(names) == 0 {
		return nil, nil
	}

	rows, err := r.queries.FindDepartmentsByNames(ctx, sqlc.FindDepartmentsByNamesParams{
		Names:     names,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]*domain.Department, len(rows))
	for i, row := range rows {
		dept := &domain.Department{
			ID:        row.ID,
			Name:      row.Name,
			AccountID: row.AccountID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
		if row.Notes.Valid {
			dept.Notes = &row.Notes.String
		}
		if row.LocationID.Valid {
			dept.LocationID = &row.LocationID.String
		}
		out[i] = dept
	}
	return out, nil
}

func (r *departmentRepoImpl) SetMachinesDepartmentID(ctx context.Context, departmentID, accountID string, machineIDs []string) *apierror.APIError {
	ctx, span := departmentRepoTracer.Start(ctx, "repository.department.set_machines")
	defer span.End()

	if len(machineIDs) == 0 {
		return nil
	}

	err := r.queries.SetMachinesDepartmentID(ctx, sqlc.SetMachinesDepartmentIDParams{
		DepartmentID: departmentID,
		MachineIds:   machineIDs,
		AccountID:    accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *departmentRepoImpl) SetScanningStationsDepartmentID(ctx context.Context, departmentID, accountID string, scanningStationIDs []string) *apierror.APIError {
	ctx, span := departmentRepoTracer.Start(ctx, "repository.department.set_scanning_stations")
	defer span.End()

	if len(scanningStationIDs) == 0 {
		return nil
	}

	err := r.queries.SetScanningStationsDepartmentID(ctx, sqlc.SetScanningStationsDepartmentIDParams{
		DepartmentID:       departmentID,
		ScanningStationIds: scanningStationIDs,
		AccountID:          accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// InsertLaborRate writes the rate row a department's labor rate points at.
func (r *departmentRepoImpl) InsertLaborRate(ctx context.Context, rateID string, params domain.CreateRateParams) *apierror.APIError {
	ctx, span := departmentRepoTracer.Start(ctx, "repository.department.insert_labor_rate")
	defer span.End()

	err := r.queries.InsertRateForDepartment(ctx, sqlc.InsertRateForDepartmentParams{
		ID:                rateID,
		Value:             params.Value,
		NumeratorUnitID:   params.NumeratorUnitID,
		DenominatorUnitID: params.DenominatorUnitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// UpdateLaborRate rewrites an existing labor rate row in place, which is how a department's rate changes without re-linking.
func (r *departmentRepoImpl) UpdateLaborRate(ctx context.Context, rateID string, params domain.CreateRateParams) *apierror.APIError {
	ctx, span := departmentRepoTracer.Start(ctx, "repository.department.update_labor_rate")
	defer span.End()

	_, err := r.queries.UpdateRateByID(ctx, sqlc.UpdateRateByIDParams{
		ID:                rateID,
		Value:             gosql.NullString{String: params.Value, Valid: true},
		NumeratorUnitID:   gosql.NullString{String: params.NumeratorUnitID, Valid: true},
		DenominatorUnitID: gosql.NullString{String: params.DenominatorUnitID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
