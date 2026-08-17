package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/excel"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/messaging"
)

// asyncBulkDeps hands the async bulk engine the plumbing it runs on.
func (s *departmentSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

// checks duplicate name
func validateBulkUpsertDepartmentRows(rows []domain.UpsertDepartmentParams) *apierror.APIError {
	nameInputSpace := make(map[string]struct{}, len(rows))
	var rowErrs apierror.RowErrors
	for i, d := range rows {
		lower := strings.ToLower(d.Name)
		if _, dup := nameInputSpace[lower]; dup {
			rowErrs.AddValidation(i, fmt.Sprintf("departments[%d].name", i), fmt.Sprintf("duplicate name %q in request", d.Name))
		}
		nameInputSpace[lower] = struct{}{}
	}
	return rowErrs.Summary("departments")
}

// resolves each row's location reference by id or name
func resolveBulkUpsertDepartmentRows(ctx context.Context, repos domain.RepoFactory, accountID string, rows []domain.UpsertDepartmentParams) ([]domain.ResolvedUpsertDepartmentRow, *apierror.APIError) {
	identifiers := make([]*domain.ObjectIdentifier, len(rows))
	for i, row := range rows {
		if row.Location != nil && *row.Location != (domain.ObjectIdentifier{}) {
			identifiers[i] = row.Location
		}
	}

	repo := repos.NewLocationRepo()
	locationIDByIdentifier, apiErr := resolveObjectIdentifiers(
		ctx, accountID, "departments", "location", "location", "locations", identifiers,
		repo.GetByIDs, repo.FindByNames,
		func(l *domain.Location) string { return l.ID },
		func(l *domain.Location) string { return l.Name },
	)
	if apiErr != nil {
		return nil, apiErr
	}

	resolved := make([]domain.ResolvedUpsertDepartmentRow, len(rows))
	for i, row := range rows {
		var locationID *string
		if identifiers[i] != nil {
			resolvedID := locationIDByIdentifier[*identifiers[i]]
			locationID = &resolvedID
		}
		resolved[i] = domain.ResolvedUpsertDepartmentRow{
			Name:       row.Name,
			Notes:      row.Notes,
			LocationID: locationID,
		}
	}
	return resolved, nil
}

// the engine's Write hook: matches each row against existing departments by name and upserts it in its own savepoint, so a bad row rolls back only itself.
func writeBulkUpsertDepartments(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.ResolvedUpsertDepartmentRow) (BulkWriteResult, *apierror.APIError) {
	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = strings.ToLower(row.Name)
	}

	txRepo := txRepos.NewDepartmentRepo()
	existing, apiErr := txRepo.FindByNames(txCtx, accountID, names)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	departmentsByName := make(map[string]*domain.Department, len(existing))
	for _, d := range existing {
		departmentsByName[strings.ToLower(d.Name)] = d
	}

	results := make([]domain.RowResult, 0, len(rows))
	var rowErrors []apierror.RowError
	for i := range rows {
		row := rows[i]

		var upsertedID string
		var isCreate bool
		rowErr := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
			old := departmentsByName[names[i]]
			id, apiErr := upsertDepartmentInTx(spCtx, txRepos, accountID, row, old)
			if apiErr != nil {
				return apiErr
			}
			upsertedID = id
			isCreate = old == nil
			return nil
		})
		if rowErr != nil {
			rowErrors = append(rowErrors, apierror.NewRowError(i, rowErr))
			continue
		}

		results = append(results, newRowResult(i, upsertedID, isCreate))
	}

	return BulkWriteResult{Results: results, Errors: rowErrors, WrittenIDs: resultIDs(results)}, nil
}

// creates or updates one department inside an existing transaction.
func upsertDepartmentInTx(txCtx context.Context, txRepos domain.RepoFactory, accountID string, row domain.ResolvedUpsertDepartmentRow, old *domain.Department) (string, *apierror.APIError) {
	ctx, span := departmentSvcTracer.Start(txCtx, "service.department.upsert_in_tx")
	defer span.End()

	txRepo := txRepos.NewDepartmentRepo()

	if old == nil {
		departmentID, apiErr := id.GenID(id.DepartmentIDPrefix, nil)
		if apiErr != nil {
			return "", apiErr
		}

		created, apiErr := txRepo.Create(ctx, departmentID, domain.CreateDepartmentParams{
			AccountID:  accountID,
			Name:       row.Name,
			Notes:      row.Notes,
			LocationID: row.LocationID,
		})
		if apiErr != nil {
			return "", apiErr
		}

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeDepartment,
			ResourceID:   created.ID,
			Changes:      changes,
		}); apiErr != nil {
			return "", apiErr
		}

		return created.ID, nil
	}

	name := row.Name
	// The SQL assigns notes directly, so an omitted note has to carry the old value forward.
	notes := row.Notes
	if notes == nil {
		notes = old.Notes
	}
	updated, apiErr := txRepo.Update(ctx, domain.UpdateDepartmentParams{
		AccountID:    accountID,
		DepartmentID: old.ID,
		Name:         &name,
		Notes:        notes,
		LocationID:   row.LocationID,
	})
	if apiErr != nil {
		return "", apiErr
	}

	changes := audit.ComputeChanges(old, updated)
	if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeDepartment,
		ResourceID:   updated.ID,
		Changes:      changes,
	}); apiErr != nil {
		return "", apiErr
	}

	return updated.ID, nil
}

// wires departments into the async bulk engine.
func (s *departmentSvcImpl) bulkUpsertSpec() bulkOperationSpec[domain.UpsertDepartmentParams, domain.ResolvedUpsertDepartmentRow] {
	return bulkOperationSpec[domain.UpsertDepartmentParams, domain.ResolvedUpsertDepartmentRow]{
		JobType:          constants.JobTypeBulkUpsert,
		ResourceType:     constants.ObjectTypeDepartment,
		RoutingKey:       messaging.BulkUpsertDepartments.RoutingKey(),
		PermissionDomain: types.PermissionDomainDepartments,
		Actions:          []types.Action{types.ActionCreate, types.ActionUpdate},
		EntityName:       "departments",
		Validate:         validateBulkUpsertDepartmentRows,
		Resolve:          resolveBulkUpsertDepartmentRows,
		Write:            writeBulkUpsertDepartments,
	}
}

// accepts a bulk upsert: validates and resolves synchronously, records the resolved rows on a job, returns that job to poll.
func (s *departmentSvcImpl) BulkUpsertDepartments(ctx context.Context, params domain.BulkUpsertDepartmentsParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), params.Departments)
}

// performs the writes for an enqueued bulk upsert.
func (s *departmentSvcImpl) ExecuteBulkUpsertDepartments(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), event)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// hands the export engine the plumbing it runs on.

// wires departments into the export engine.
func (s *departmentSvcImpl) exportSpec() exportSpec[*domain.Department, domain.ExportDepartmentsParams] {
	return exportSpec[*domain.Department, domain.ExportDepartmentsParams]{
		PermissionDomain: types.PermissionDomainDepartments,
		Name:             "Departments",
		Slug:             "departments",
		ResourceType:     constants.ObjectTypeDepartment,
		Columns: []excel.ColumnSpec{
			{Header: "ID", Key: "id", Width: 24},
			{Header: "Name", Key: "name", Width: 28},
			{Header: "Location", Key: "location", Width: 24},
			// Read-only: neither child list is importable.
			{Header: "Scanning Stations", Key: "scanning_stations", Width: 36},
			{Header: "Machines", Key: "machines", Width: 36},
			{Header: "Notes", Key: "notes", Width: 40},
		},

		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportDepartmentsParams) ([]*domain.Department, *apierror.APIError) {
			filters.AccountID = accountID
			return repos.NewDepartmentRepo().Export(ctx, filters)
		},

		Project: func(dept *domain.Department) excel.Row {
			stations := make([]string, len(dept.ScanningStations))
			for i, s := range dept.ScanningStations {
				stations[i] = s.Name
			}
			machines := make([]string, len(dept.Machines))
			for i, m := range dept.Machines {
				machines[i] = m.Name
			}
			return excel.Row{
				"id":                dept.ID,
				"name":              dept.Name,
				"location":          excel.Str(dept.LocationName),
				"scanning_stations": excel.JoinNames(stations),
				"machines":          excel.JoinNames(machines),
				"notes":             excel.Str(dept.Notes),
			}
		},
	}
}

// accepts an export: records the filters on a job and returns it to poll.
func (s *departmentSvcImpl) ExportDepartments(ctx context.Context, params domain.ExportDepartmentsParams) (*domain.Job, *apierror.APIError) {
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), params)
}

// renders the file an accepted export recorded
func (s *departmentSvcImpl) BuildExportDepartments(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}
