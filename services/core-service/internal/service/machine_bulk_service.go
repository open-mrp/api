package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/excel"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/messaging"
)

// asyncBulkDeps hands the async bulk engine the plumbing it runs on.
func (s *machineSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

// runs the accept-phase structural checks: no duplicate name
// or serial number within the request (case-insensitive, matching how existing machines are
// matched). No DB.
func validateBulkUpsertMachineRows(rows []domain.UpsertMachineParams) *apierror.APIError {
	nameInputSpace := make(map[string]struct{}, len(rows))
	serialInputSpace := make(map[string]struct{}, len(rows))
	var rowErrs apierror.RowErrors
	for i, m := range rows {
		lower := strings.ToLower(m.Name)
		if _, dup := nameInputSpace[lower]; dup {
			rowErrs.AddValidation(i, fmt.Sprintf("machines[%d].name", i), fmt.Sprintf("duplicate name %q in request", m.Name))
		}
		nameInputSpace[lower] = struct{}{}

		lowerSerial := strings.ToLower(m.SerialNumber)
		if _, dup := serialInputSpace[lowerSerial]; dup {
			rowErrs.AddValidation(i, fmt.Sprintf("machines[%d].serial_number", i), fmt.Sprintf("duplicate serial number %q in request", m.SerialNumber))
		}
		serialInputSpace[lowerSerial] = struct{}{}
	}
	return rowErrs.Summary("machines")
}

// resolves each row's department reference by id or name,
// collecting every unresolved one into a single row-indexed validation error.
func resolveBulkUpsertMachineRows(ctx context.Context, repos domain.RepoFactory, accountID string, rows []domain.UpsertMachineParams) ([]domain.ResolvedUpsertMachineRow, *apierror.APIError) {
	deptIdentifiers := make([]domain.ObjectIdentifier, len(rows))
	for i, row := range rows {
		deptIdentifiers[i] = row.Department
	}
	deptIDByIdentifier, apiErr := resolveDepartmentIdentifiersInTx(ctx, repos, accountID, "machines", deptIdentifiers)
	if apiErr != nil {
		return nil, apiErr
	}

	resolved := make([]domain.ResolvedUpsertMachineRow, len(rows))
	for i, row := range rows {
		resolved[i] = domain.ResolvedUpsertMachineRow{
			Name:         row.Name,
			SerialNumber: row.SerialNumber,
			Notes:        row.Notes,
			DepartmentID: deptIDByIdentifier[row.Department],
		}
	}
	return resolved, nil
}

// is the engine's Write hook: it reads the name and serial matches
// once, then per row resolves the dual-key match (matchMachineForUpsert) and upserts inside
// its own savepoint, so a bad row is recorded in errors and the rest still commit.
func writeBulkUpsertMachines(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.ResolvedUpsertMachineRow) (BulkWriteResult, *apierror.APIError) {
	names := make([]string, len(rows))
	serials := make([]string, len(rows))
	for i, row := range rows {
		names[i] = strings.ToLower(row.Name)
		serials[i] = strings.ToLower(row.SerialNumber)
	}

	txRepo := txRepos.NewMachineRepo()
	nameMatches, apiErr := txRepo.FindByNames(txCtx, accountID, names)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	byName := make(map[string]*domain.Machine, len(nameMatches))
	for _, m := range nameMatches {
		byName[strings.ToLower(m.Name)] = m
	}

	serialMatches, apiErr := txRepo.FindBySerialNumbers(txCtx, accountID, serials)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	bySerial := make(map[string]*domain.Machine, len(serialMatches))
	for _, m := range serialMatches {
		bySerial[strings.ToLower(m.SerialNumber)] = m
	}

	results := make([]domain.RowResult, 0, len(rows))
	var rowErrors []apierror.RowError
	for i := range rows {
		row := rows[i]

		var upsertedID string
		var isCreate bool
		rowErr := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
			old, apiErr := matchMachineForUpsert(row, byName[names[i]], bySerial[serials[i]])
			if apiErr != nil {
				return apiErr
			}
			id, apiErr := upsertMachineInTx(spCtx, txRepos, accountID, row, old)
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

// resolves a row's dual-key match into the machine it should update
// (nil for a create), or reports why the row cannot be upserted:
//   - name and serial match two different machines → ambiguous;
//   - one/both keys match a machine in a different department → a create-elsewhere collision,
//     since department is create-only.
func matchMachineForUpsert(row domain.ResolvedUpsertMachineRow, nameMatch, serialMatch *domain.Machine) (*domain.Machine, *apierror.APIError) {
	if nameMatch != nil && serialMatch != nil && nameMatch.ID != serialMatch.ID {
		return nil, apierror.NewConflictErrorWithParam(
			fmt.Sprintf("The name %q matches existing machine %q but the serial number %q matches a different existing machine %q.",
				row.Name, nameMatch.Name, row.SerialNumber, serialMatch.Name),
			"name, serial_number",
		)
	}

	old := nameMatch
	if old == nil {
		old = serialMatch
	}
	if old == nil || old.DepartmentID == nil || *old.DepartmentID == row.DepartmentID {
		return old, nil
	}

	oldDept := ""
	if old.DepartmentName != nil {
		oldDept = *old.DepartmentName
	}
	switch {
	case nameMatch != nil && serialMatch != nil:
		return nil, apierror.NewConflictErrorWithParam(
			fmt.Sprintf("Machine %q (matched by name and serial number) belongs to department %q and cannot be moved.", old.Name, oldDept),
			"department",
		)
	case nameMatch != nil:
		return nil, apierror.NewConflictErrorWithParam(fmt.Sprintf("A machine named %q already exists.", old.Name), "name")
	default:
		return nil, apierror.NewConflictErrorWithParam(
			fmt.Sprintf("Serial number %q is already used by machine %q in department %q.", row.SerialNumber, old.Name, oldDept),
			"serial_number",
		)
	}
}

// creates or updates one machine inside an existing transaction.
// An update adopts the request's name casing and serial, preserves omitted notes (COALESCE in SQL), and
// leaves the department unchanged — it is create-only.
func upsertMachineInTx(txCtx context.Context, txRepos domain.RepoFactory, accountID string, row domain.ResolvedUpsertMachineRow, old *domain.Machine) (string, *apierror.APIError) {
	ctx, span := machineSvcTracer.Start(txCtx, "service.machine.upsert_in_tx")
	defer span.End()

	txRepo := txRepos.NewMachineRepo()

	if old == nil {
		machineID, apiErr := id.GenID(id.MachineIDPrefix, nil)
		if apiErr != nil {
			return "", apiErr
		}
		created, apiErr := txRepo.Create(ctx, machineID, domain.CreateMachineParams{
			AccountID:    accountID,
			Name:         row.Name,
			SerialNumber: row.SerialNumber,
			Notes:        row.Notes,
			DepartmentID: row.DepartmentID,
		})
		if apiErr != nil {
			return "", apiErr
		}

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeMachine,
			ResourceID:   created.ID,
			Changes:      changes,
		}); apiErr != nil {
			return "", apiErr
		}

		return created.ID, nil
	}

	name := row.Name
	serial := row.SerialNumber
	updated, apiErr := txRepo.Update(ctx, domain.UpdateMachineParams{
		AccountID:    accountID,
		MachineID:    old.ID,
		Name:         &name,
		SerialNumber: &serial,
		Notes:        row.Notes,
	})
	if apiErr != nil {
		return "", apiErr
	}

	changes := audit.ComputeChanges(old, updated)
	if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeMachine,
		ResourceID:   updated.ID,
		Changes:      changes,
	}); apiErr != nil {
		return "", apiErr
	}

	return updated.ID, nil
}

// wires machines into the async bulk engine.
func (s *machineSvcImpl) bulkUpsertSpec() bulkOperationSpec[domain.UpsertMachineParams, domain.ResolvedUpsertMachineRow] {
	return bulkOperationSpec[domain.UpsertMachineParams, domain.ResolvedUpsertMachineRow]{
		JobType:          constants.JobTypeBulkUpsert,
		ResourceType:     constants.ObjectTypeMachine,
		RoutingKey:       messaging.BulkUpsertMachines.RoutingKey(),
		PermissionDomain: types.PermissionDomainMachines,
		Actions:          []types.Action{types.ActionCreate, types.ActionUpdate},
		EntityName:       "machines",
		Validate:         validateBulkUpsertMachineRows,
		Resolve:          resolveBulkUpsertMachineRows,
		Write:            writeBulkUpsertMachines,
	}
}

// accepts a bulk upsert: it validates and resolves synchronously, records
// the resolved rows on a job, and returns that job to poll. ExecuteBulkUpsertMachines writes.
func (s *machineSvcImpl) BulkUpsertMachines(ctx context.Context, params domain.BulkUpsertMachinesParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), params.Machines)
}

// performs the writes for an enqueued bulk upsert. Delivery is at-least-once;
// the inbox de-dup and the engine's terminal-job guard make it effectively-once.
func (s *machineSvcImpl) ExecuteBulkUpsertMachines(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), event)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// lists the columns shared by the machine export and its import template;
// the export prefixes ID, which is not importable
var machineTemplateColumns = []excel.ColumnSpec{
	{Header: "Name", Key: "name", Width: 28},
	{Header: "Serial Number", Key: "serial_number", Width: 24},
	{Header: "Department", Key: "department", Width: 24},
	{Header: "Notes", Key: "notes", Width: 40},
}

// hands the export engine the plumbing it runs on.

// wires machines into the export engine.
func (s *machineSvcImpl) exportSpec() exportSpec[*domain.Machine, domain.ExportMachinesParams] {
	return exportSpec[*domain.Machine, domain.ExportMachinesParams]{
		PermissionDomain: types.PermissionDomainMachines,
		Name:             "Machines",
		Slug:             "machines",
		ResourceType:     constants.ObjectTypeMachine,
		Columns:          append([]excel.ColumnSpec{{Header: "ID", Key: "id", Width: 24}}, machineTemplateColumns...),

		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportMachinesParams) ([]*domain.Machine, *apierror.APIError) {
			filters.AccountID = accountID
			return repos.NewMachineRepo().Export(ctx, filters)
		},

		Project: func(machine *domain.Machine) excel.Row {
			return excel.Row{
				"id":            machine.ID,
				"name":          machine.Name,
				"serial_number": machine.SerialNumber,
				"department":    excel.Str(machine.DepartmentName),
				"notes":         excel.Str(machine.Notes),
			}
		},
	}
}

// accepts an export: records the filters on a job and returns it to poll.
func (s *machineSvcImpl) ExportMachines(ctx context.Context, params domain.ExportMachinesParams) (*domain.Job, *apierror.APIError) {
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), params)
}

// renders the file an accepted export recorded
func (s *machineSvcImpl) BuildExportMachines(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}
