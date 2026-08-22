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
	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/messaging"
)

// asyncBulkDeps hands the async bulk engine the plumbing it runs on.
func (s *scanningStationSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

// runs the accept-phase structural checks
func validateBulkUpsertScanningStationRows(rows []domain.UpsertScanningStationParams) *apierror.APIError {
	nameInputSpace := make(map[string]struct{}, len(rows))
	var rowErrs apierror.RowErrors

	for i, ss := range rows {
		lower := strings.ToLower(ss.Name)
		if _, dup := nameInputSpace[lower]; dup {
			rowErrs.AddValidation(i, fmt.Sprintf("scanning_stations[%d].name", i), fmt.Sprintf("duplicate name %q in request", ss.Name))
		}
		nameInputSpace[lower] = struct{}{}

		if !ss.Type.IsValid() {
			rowErrs.AddValidation(i, fmt.Sprintf("scanning_stations[%d].type", i), fmt.Sprintf("invalid scanning station type %q", ss.Type))
		}
		if !ss.OperatorRequirement.IsValid() {
			rowErrs.AddValidation(i, fmt.Sprintf("scanning_stations[%d].operator_requirement", i), fmt.Sprintf("invalid operator requirement %q", ss.OperatorRequirement))
		}
	}

	return rowErrs.Summary("scanning stations")
}

// resolves each row's department reference
func resolveBulkUpsertScanningStationRows(ctx context.Context, repos domain.RepoFactory, accountID string, rows []domain.UpsertScanningStationParams) ([]domain.ResolvedUpsertScanningStationRow, *apierror.APIError) {
	deptIdentifiers := make([]domain.ObjectIdentifier, len(rows))
	for i, ss := range rows {
		deptIdentifiers[i] = ss.Department
	}
	deptIDByIdentifier, apiErr := resolveDepartmentIdentifiersInTx(ctx, repos, accountID, "scanning_stations", deptIdentifiers)
	if apiErr != nil {
		return nil, apiErr
	}

	resolved := make([]domain.ResolvedUpsertScanningStationRow, len(rows))
	for i, ss := range rows {
		resolved[i] = domain.ResolvedUpsertScanningStationRow{
			Name:                ss.Name,
			Notes:               ss.Notes,
			Type:                ss.Type,
			LabelSizeCode:       ss.LabelSizeCode,
			LabelTypeCode:       ss.LabelTypeCode,
			OperatorRequirement: ss.OperatorRequirement,
			DepartmentID:        deptIDByIdentifier[ss.Department],
		}
	}
	return resolved, nil
}

// the engine's Write hook
func writeBulkUpsertScanningStations(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.ResolvedUpsertScanningStationRow) (BulkWriteResult, *apierror.APIError) {
	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = strings.ToLower(row.Name)
	}

	txRepo := txRepos.NewScanningStationRepo()
	existing, apiErr := txRepo.FindByNames(txCtx, accountID, names)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	byName := make(map[string]*domain.ScanningStation, len(existing))
	for _, ss := range existing {
		byName[strings.ToLower(ss.Name)] = ss
	}

	results := make([]domain.RowResult, 0, len(rows))
	var rowErrors []apierror.RowError
	for i := range rows {
		row := rows[i]

		var upsertedID string
		var isCreate bool
		rowErr := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
			old := byName[names[i]]
			id, apiErr := upsertScanningStationInTx(spCtx, txRepos, accountID, row, old)
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

// creates or updates one scanning station inside an existing transaction
func upsertScanningStationInTx(txCtx context.Context, txRepos domain.RepoFactory, accountID string, row domain.ResolvedUpsertScanningStationRow, old *domain.ScanningStation) (string, *apierror.APIError) {
	ctx, span := scanningStationSvcTracer.Start(txCtx, "service.scanning_station.upsert_in_tx")
	defer span.End()

	txRepo := txRepos.NewScanningStationRepo()

	if old == nil {
		scanningStationID, apiErr := id.GenID(id.ScanningStationIDPrefix, nil)
		if apiErr != nil {
			return "", apiErr
		}
		created, apiErr := txRepo.Create(ctx, scanningStationID, domain.CreateScanningStationParams{
			AccountID:           accountID,
			Name:                row.Name,
			Notes:               row.Notes,
			Type:                row.Type,
			LabelSizeCode:       row.LabelSizeCode.ValuePtr(),
			LabelTypeCode:       row.LabelTypeCode.ValuePtr(),
			OperatorRequirement: row.OperatorRequirement,
			DepartmentID:        row.DepartmentID,
		})
		if apiErr != nil {
			return "", apiErr
		}

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeScanningStation,
			ResourceID:   created.ID,
			Changes:      changes,
		}); apiErr != nil {
			return "", apiErr
		}

		return created.ID, nil
	}

	if old.DepartmentID != row.DepartmentID {
		return "", apierror.NewValidationErrorWithParam(
			fmt.Sprintf("Scanning station %q belongs to department %q and cannot be moved by bulk upsert.", old.Name, old.DepartmentName), "department")
	}
	if old.Type != row.Type {
		return "", apierror.NewValidationErrorWithParam(
			fmt.Sprintf("Scanning station %q has type %q, which cannot be changed by bulk upsert.", old.Name, old.Type), "type")
	}

	notes := field.Unset[string]()
	if row.Notes != nil {
		notes = field.Set(*row.Notes)
	}
	updated, apiErr := txRepo.Update(ctx, domain.UpdateScanningStationParams{
		AccountID:           accountID,
		ScanningStationID:   old.ID,
		Name:                &row.Name,
		Notes:               notes.BackfillUnsetPtr(old.Notes),
		LabelSizeCode:       row.LabelSizeCode.BackfillUnsetPtr(old.LabelSizeCode),
		LabelTypeCode:       row.LabelTypeCode.BackfillUnsetPtr(old.LabelTypeCode),
		OperatorRequirement: &row.OperatorRequirement,
	})
	if apiErr != nil {
		return "", apiErr
	}

	changes := audit.ComputeChanges(old, updated)
	if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeScanningStation,
		ResourceID:   updated.ID,
		Changes:      changes,
	}); apiErr != nil {
		return "", apiErr
	}

	return updated.ID, nil
}

// wires scanning stations into the async bulk engine.
func (s *scanningStationSvcImpl) bulkUpsertSpec() bulkOperationSpec[domain.UpsertScanningStationParams, domain.ResolvedUpsertScanningStationRow] {
	return bulkOperationSpec[domain.UpsertScanningStationParams, domain.ResolvedUpsertScanningStationRow]{
		JobType:          constants.JobTypeBulkUpsert,
		ResourceType:     constants.ObjectTypeScanningStation,
		RoutingKey:       messaging.BulkUpsertScanningStations.RoutingKey(),
		PermissionDomain: types.PermissionDomainScanningStations,
		Actions:          []types.Action{types.ActionCreate, types.ActionUpdate},
		EntityName:       "scanning stations",
		Validate:         validateBulkUpsertScanningStationRows,
		Resolve:          resolveBulkUpsertScanningStationRows,
		Write:            writeBulkUpsertScanningStations,
	}
}

// accepts a bulk upsert: validates and resolves synchronously,
// records the resolved rows on a job, and returns that job to poll.
func (s *scanningStationSvcImpl) BulkUpsertScanningStations(ctx context.Context, params domain.BulkUpsertScanningStationsParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), params.ScanningStations)
}

// performs the writes for an enqueued bulk upsert.
func (s *scanningStationSvcImpl) ExecuteBulkUpsertScanningStations(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), event)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// lists the columns shared by the scanning station export and its import
// template; the export prefixes ID, which is not importable
var scanningStationTemplateColumns = []excel.ColumnSpec{
	{Header: "Name", Key: "name", Width: 28},
	{Header: "Type", Key: "type", Width: 18},
	{Header: "Department", Key: "department", Width: 24},
	{Header: "Operator Requirement", Key: "operator_requirement", Width: 22},
	{Header: "Batch Label Type", Key: "batch_label_type", Width: 20},
	{Header: "Batch Label Tag Size", Key: "batch_label_tag_size", Width: 20},
	{Header: "Notes", Key: "notes", Width: 40},
}

// hands the export engine the plumbing it runs on.

// wires scanning stations into the export engine.
func (s *scanningStationSvcImpl) exportSpec() exportSpec[*domain.ScanningStation, domain.ExportScanningStationsParams] {
	return exportSpec[*domain.ScanningStation, domain.ExportScanningStationsParams]{
		PermissionDomain: types.PermissionDomainScanningStations,
		Name:             "Scanning Stations",
		Slug:             "scanning_stations",
		ResourceType:     constants.ObjectTypeScanningStation,
		Columns:          append([]excel.ColumnSpec{{Header: "ID", Key: "id", Width: 24}}, scanningStationTemplateColumns...),

		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportScanningStationsParams) ([]*domain.ScanningStation, *apierror.APIError) {
			filters.AccountID = accountID
			return repos.NewScanningStationRepo().Export(ctx, filters)
		},

		Project: func(station *domain.ScanningStation) excel.Row {
			return excel.Row{
				"id":                   station.ID,
				"name":                 station.Name,
				"type":                 string(station.Type),
				"department":           scanningStationDepartmentName(station),
				"operator_requirement": string(station.OperatorRequirement),
				"batch_label_type":     excel.Str(station.LabelTypeCode),
				"batch_label_tag_size": excel.Str(station.LabelSizeCode),
				"notes":                excel.Str(station.Notes),
			}
		},
	}
}

// prefers the department's name, falling back to its id so a station whose
// department was removed still exports
func scanningStationDepartmentName(station *domain.ScanningStation) string {
	if station.DepartmentName != "" {
		return station.DepartmentName
	}
	return station.DepartmentID
}

// accepts an export: records what to build on a job and returns it to poll.
func (s *scanningStationSvcImpl) ExportScanningStations(ctx context.Context, params domain.ExportScanningStationsParams) (*domain.Job, *apierror.APIError) {
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), params)
}

// renders the file an accepted export recorded
func (s *scanningStationSvcImpl) BuildExportScanningStations(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}
