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
	"github.com/augno/api/shared/tracing"
)

// asyncBulkDeps hands the async bulk engine the plumbing it runs on.
func (s *unitSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

// upsertUnitInTx upserts a unit of measure inside an existing transaction.
// The caller is responsible for identity checks, idempotency, and transaction scope.
func upsertUnitInTx(txCtx context.Context, txRepos domain.RepoFactory, accountID string, params domain.UpsertUnitTxParams) (string, *apierror.APIError) {
	ctx, span := unitSvcTracer.Start(txCtx, "service.unit.upsert_in_tx")
	defer span.End()
	txRepo := txRepos.NewUnitRepo()

	var upsertID string
	if params.OldUnit == nil { // create
		unitID, apiErr := id.GenID(id.UnitIDPrefix, nil)
		if apiErr != nil {
			return upsertID, tracing.Trace(span, apiErr)
		}

		unitCreate := domain.CreateUnitParams{
			AccountID:         accountID,
			Name:              params.Unit.Name,
			Abbreviation:      params.Unit.Abbreviation,
			UnitDimensionCode: params.Unit.UnitDimensionCode,
			RatioNumerator:    params.Unit.RatioNumerator,
			RatioDenominator:  params.Unit.RatioDenominator,
			OffsetNumerator:   params.Unit.OffsetNumerator,
			OffsetDenominator: params.Unit.OffsetDenominator,
			// Bulk upsert never creates base units: base units are designated through
			// unit groups, and updates reject any is_base_unit change as immutable.
			IsBaseUnit: false,
		}

		created, apiErr := txRepo.Create(ctx, unitID, unitCreate)
		if apiErr != nil {
			return upsertID, apiErr
		}
		upsertID = created.ID

		changes := audit.ComputeChanges(nil, created)

		if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeUnit,
			ResourceID:   created.ID,
			Changes:      changes,
		}); apiErr != nil {
			return upsertID, apiErr
		}

	} else { // update
		if params.OldUnit.AccountID == nil {
			return upsertID, tracing.Trace(span, apierror.NewValidationError("System units cannot be modified."))
		}

		if params.Unit.UnitDimensionCode != params.OldUnit.UnitDimensionCode {
			return upsertID, tracing.Trace(span, apierror.NewValidationError(fmt.Sprintf(
				"Unit %s (%s) already exists and the Unit Dimension Code is immutable", params.Unit.Name, params.Unit.Abbreviation,
			)))
		}

		// IsBaseUnit is immutable
		updateParams := domain.UpdateUnitParams{
			AccountID:         accountID,
			UnitID:            params.OldUnit.ID,
			Name:              &params.Unit.Name,
			Abbreviation:      &params.Unit.Abbreviation,
			RatioNumerator:    &params.Unit.RatioNumerator,
			RatioDenominator:  &params.Unit.RatioDenominator,
			OffsetNumerator:   &params.Unit.OffsetNumerator,
			OffsetDenominator: &params.Unit.OffsetDenominator,
		}

		updated, apiErr := txRepo.Update(ctx, updateParams)
		if apiErr != nil {
			return upsertID, apiErr
		}
		upsertID = updated.ID

		changes := audit.ComputeChanges(params.OldUnit, updated)

		if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeUnit,
			ResourceID:   updated.ID,
			Changes:      changes,
		}); apiErr != nil {
			return upsertID, apiErr
		}
	}
	return upsertID, nil
}

// validateBulkUpsertUnitRows runs the accept-phase structural checks: no duplicate name
// or abbreviation within the request (case-insensitive), and no zero denominator. It
// touches no database.
func validateBulkUpsertUnitRows(rows []domain.UpsertUnitParams) *apierror.APIError {
	nameInputSpace := make(map[string]struct{}, len(rows))
	abbrInputSpace := make(map[string]struct{}, len(rows))
	var rowErrs apierror.RowErrors
	for i, unit := range rows {
		lowerName := strings.ToLower(unit.Name)
		if _, exists := nameInputSpace[lowerName]; exists {
			rowErrs.AddValidation(i, fmt.Sprintf("units[%d].name", i), fmt.Sprintf("duplicate name %q in request", unit.Name))
		}
		nameInputSpace[lowerName] = struct{}{}

		lowerAbbr := strings.ToLower(unit.Abbreviation)
		if _, exists := abbrInputSpace[lowerAbbr]; exists {
			rowErrs.AddValidation(i, fmt.Sprintf("units[%d].abbreviation", i), fmt.Sprintf("duplicate abbreviation %q in request", unit.Abbreviation))
		}
		abbrInputSpace[lowerAbbr] = struct{}{}

		if isDenominatorZero(unit.RatioDenominator) {
			rowErrs.AddValidation(i, fmt.Sprintf("units[%d].ratio_denominator", i), "ratio denominator cannot be zero")
		}
		if isDenominatorZero(unit.OffsetDenominator) {
			rowErrs.AddValidation(i, fmt.Sprintf("units[%d].offset_denominator", i), "offset denominator cannot be zero")
		}
	}
	return rowErrs.Summary("units")
}

// bulkUpsertSpec wires units into the async bulk engine. A unit references no other
// entity, so Resolve is a passthrough and there is no AcceptResults; the created/updated
// split is decided against live rows in Write.
func (s *unitSvcImpl) bulkUpsertSpec() bulkOperationSpec[domain.UpsertUnitParams, domain.UpsertUnitParams] {
	return bulkOperationSpec[domain.UpsertUnitParams, domain.UpsertUnitParams]{
		JobType:          constants.JobTypeBulkUpsert,
		ResourceType:     constants.ObjectTypeUnit,
		RoutingKey:       messaging.BulkUpsertUnits.RoutingKey(),
		PermissionDomain: types.PermissionDomainUnits,
		Actions:          []types.Action{types.ActionCreate, types.ActionUpdate},
		EntityName:       "units",
		Validate:         validateBulkUpsertUnitRows,
		Resolve: func(_ context.Context, _ domain.RepoFactory, _ string, rows []domain.UpsertUnitParams) ([]domain.UpsertUnitParams, *apierror.APIError) {
			return rows, nil
		},
		Write: writeBulkUpsertUnits,
	}
}

// BulkUpsertUnits accepts a bulk upsert: it validates synchronously, records the rows on
// a job, and returns the raised job to poll. The units are created or updated
// asynchronously by ExecuteBulkUpsertUnits.
func (s *unitSvcImpl) BulkUpsertUnits(ctx context.Context, params domain.BulkUpsertUnitsParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), params.Units)
}

// ExecuteBulkUpsertUnits performs the writes for an enqueued bulk upsert. Called by the
// bulk upsert consumer; exactly-once is provided by the message inbox.
func (s *unitSvcImpl) ExecuteBulkUpsertUnits(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), event)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// lists the columns shared by the unit export and its import template; the
// export prefixes ID and appends Default, neither of which is importable
var unitTemplateColumns = []excel.ColumnSpec{
	{Header: "Name", Key: "name", Width: 22},
	{Header: "Abbreviation", Key: "abbreviation", Width: 14},
	{Header: "Type", Key: "type", Width: 14},
	{Header: "Ratio Numerator", Key: "ratio_numerator", Width: 18},
	{Header: "Ratio Denominator", Key: "ratio_denominator", Width: 18},
	{Header: "Offset Numerator", Key: "offset_numerator", Width: 18},
	{Header: "Offset Denominator", Key: "offset_denominator", Width: 18},
}

// hands the export engine the plumbing it runs on.

// wires units into the export engine.
func (s *unitSvcImpl) exportSpec() exportSpec[*domain.Unit, domain.ExportUnitsParams] {
	columns := append([]excel.ColumnSpec{{Header: "ID", Key: "id", Width: 6}}, unitTemplateColumns...)
	columns = append(columns, excel.ColumnSpec{Header: "Default", Key: "is_default", Width: 12})

	return exportSpec[*domain.Unit, domain.ExportUnitsParams]{
		PermissionDomain: types.PermissionDomainUnits,
		Name:             "Units",
		Slug:             "units",
		ResourceType:     constants.ObjectTypeUnit,
		Columns:          columns,

		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportUnitsParams) ([]*domain.Unit, *apierror.APIError) {
			filters.AccountID = accountID
			return repos.NewUnitRepo().Export(ctx, filters)
		},

		Project: func(unit *domain.Unit) excel.Row {
			return excel.Row{
				"id":                 unit.ID,
				"name":               unit.Name,
				"abbreviation":       unit.Abbreviation,
				"type":               unit.UnitDimensionCode,
				"ratio_numerator":    decimalCell(unit.RatioNumerator),
				"ratio_denominator":  decimalCell(unit.RatioDenominator),
				"offset_numerator":   decimalCell(unit.OffsetNumerator),
				"offset_denominator": decimalCell(unit.OffsetDenominator),
				// "Default" is the account-owned flag: a system unit has no owning account.
				"is_default": yesNo(unit.AccountID != nil),
			}
		},
	}
}

// accepts an export: records the filters on a job and returns it to poll.
func (s *unitSvcImpl) ExportUnits(ctx context.Context, params domain.ExportUnitsParams) (*domain.Job, *apierror.APIError) {
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), params)
}

// renders the file an accepted export recorded
func (s *unitSvcImpl) BuildExportUnits(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}

// writeBulkUpsertUnits is the engine's Write hook: in one transaction it matches each row
// against existing units by name or abbreviation (dual-key), rejects a row whose name and
// abbreviation point at two different units, enforces IsBaseUnit immutability, and upserts.
func writeBulkUpsertUnits(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.UpsertUnitParams) (BulkWriteResult, *apierror.APIError) {
	names := make([]string, len(rows))
	abbreviations := make([]string, len(rows))
	for i, unit := range rows {
		names[i] = strings.ToLower(unit.Name)
		abbreviations[i] = strings.ToLower(unit.Abbreviation)
	}

	existingUnits, apiErr := txRepos.NewUnitRepo().FindByAbbreviationsOrNames(txCtx, accountID, abbreviations, names)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}

	unitsByName := make(map[string]*domain.Unit, len(existingUnits))
	unitsByAbbr := make(map[string]*domain.Unit, len(existingUnits))
	for _, unit := range existingUnits {
		if unit.ID == "" {
			return BulkWriteResult{}, apierror.NewInvariantViolationError(fmt.Sprintf("Empty ID in unit: %+v", unit))
		}
		unitsByName[strings.ToLower(unit.Name)] = unit
		unitsByAbbr[strings.ToLower(unit.Abbreviation)] = unit
	}

	results := make([]domain.RowResult, 0, len(rows))
	var rowErrors []apierror.RowError
	for i := range rows {
		unit := rows[i]

		var upsertedID string
		var isCreate bool
		// Each row upserts inside its own savepoint: a row that conflicts, violates
		// immutability, or fails to write rolls back only itself, and the batch continues.
		rowErr := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
			// A row's name and abbreviation must not point at two different existing units.
			nameUnit := unitsByName[names[i]]
			abbrUnit := unitsByAbbr[abbreviations[i]]
			if nameUnit != nil && abbrUnit != nil && nameUnit.ID != abbrUnit.ID {
				return apierror.NewConflictErrorWithParam(
					fmt.Sprintf(
						"The name %q matches existing unit %q (%s) and the abbreviation %q matches a different existing unit %q (%s).",
						unit.Name, nameUnit.Name, nameUnit.Abbreviation,
						unit.Abbreviation, abbrUnit.Name, abbrUnit.Abbreviation,
					),
					"name, abbreviation",
				)
			}

			oldUnit := nameUnit
			if oldUnit == nil {
				oldUnit = abbrUnit
			}
			if oldUnit != nil && unit.IsBaseUnit != oldUnit.IsBaseUnit {
				return apierror.NewValidationError("IsBaseUnit is immutable and cannot be changed on an existing unit.")
			}

			id, apiErr := upsertUnitInTx(spCtx, txRepos, accountID, domain.UpsertUnitTxParams{Unit: &unit, OldUnit: oldUnit})
			if apiErr != nil {
				return apiErr
			}
			if id == "" {
				return apierror.NewInvariantViolationError(fmt.Sprintf("problem upserting unit, no ID exists. %+v", unit))
			}
			upsertedID = id
			isCreate = oldUnit == nil
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
