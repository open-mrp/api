package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/excel"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

// asyncBulkDeps hands the async bulk engine the plumbing this service already holds.
func (s *productionStepSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

// BulkCreateProductionSteps creates multiple production steps in a single operation.
// Steps that already exist (by name) are updated; new ones are created. Individual row
// failures are captured in the results rather than failing the whole operation.
func (s *productionStepSvcImpl) BulkCreateProductionSteps(ctx context.Context, params domain.BulkCreateProductionStepsParams) ([]domain.BulkCreateProductionStepResult, *apierror.APIError) {
	ctx, span := productionStepSvcTracer.Start(ctx, "service.production_step.bulk_create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSteps, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[[]domain.BulkCreateProductionStepResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return *cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Resolve units needed for rates: dollar (numerator for labor/overhead), hr (denominator for labor/overhead).
		unitRepo := s.repos.NewUnitRepo()
		abbreviationsNeeded := []string{"$", "hr"}
		// Collect all labor time unit abbreviations from input.
		for _, step := range params.Steps {
			ltu := "hr"
			if step.LaborTimeUnit != nil && *step.LaborTimeUnit != "" {
				ltu = *step.LaborTimeUnit
			}
			abbreviationsNeeded = append(abbreviationsNeeded, ltu)
		}
		units, apiErr := unitRepo.FindByAbbreviations(ctx, accountID, abbreviationsNeeded)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		unitByAbbr := make(map[string]string) // abbreviation -> unit ID
		for _, u := range units {
			unitByAbbr[strings.ToLower(u.Abbreviation)] = u.ID
		}

		dollarUnitID := unitByAbbr["$"]
		hrUnitID := unitByAbbr["hr"]

		if dollarUnitID == "" || hrUnitID == "" {
			return nil, tracing.Trace(span, apierror.NewInternalError(nil, "Could not resolve required units (dollar, hr)."))
		}

		// Collect all SKUs across all rows to batch-resolve items.
		allSKUs := make(map[string]struct{})
		for _, step := range params.Steps {
			for _, c := range step.Consumptions {
				allSKUs[c.SKU] = struct{}{}
			}
			for _, p := range step.Productions {
				allSKUs[p.SKU] = struct{}{}
			}
		}
		skuList := make([]string, 0, len(allSKUs))
		for sku := range allSKUs {
			skuList = append(skuList, sku)
		}

		itemBySKU := make(map[string]domain.ItemSKUInfo) // sku -> ItemSKUInfo
		if len(skuList) > 0 {
			items, apiErr := s.repos.NewItemRepo().FetchItemsBySKU(ctx, accountID, skuList)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			for _, item := range items {
				itemBySKU[item.SKU] = item
			}
		}

		results := make([]domain.BulkCreateProductionStepResult, len(params.Steps))

		for i, input := range params.Steps {
			results[i] = s.bulkCreateSingleStep(ctx, accountID, input, itemBySKU, unitByAbbr, dollarUnitID, hrUnitID)
		}

		cacheErr := s.withTx(ctx, func(txCtx context.Context, txSvc *productionStepSvcImpl) *apierror.APIError {
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, results)
		})
		if cacheErr != nil {
			// Caching failure is not fatal; return results anyway.
			_ = cacheErr
		}

		return results, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// validateBulkUpsertStepRows runs the accept-phase structural checks: reject duplicate
// names within the request (case-insensitive, matching how existing steps are
// resolved), require a production item, and reject duplicate consumption items within a
// step. It touches no database.
func validateBulkUpsertStepRows(rows []domain.UpsertProductionStepParams) *apierror.APIError {
	nameInputSpace := make(map[string]struct{}, len(rows))
	var rowErrs apierror.RowErrors
	for i, row := range rows {
		lowerName := strings.ToLower(row.Name)
		if _, dup := nameInputSpace[lowerName]; dup {
			rowErrs.AddValidation(i, fmt.Sprintf("production_steps[%d].name", i), fmt.Sprintf("duplicate name %q in request", row.Name))
		}
		nameInputSpace[lowerName] = struct{}{}

		if row.Production.Item == (domain.ItemIdentifier{}) {
			rowErrs.AddValidation(i, fmt.Sprintf("production_steps[%d].production", i), "a production is required")
		}

		consumptionInputSpace := make(map[domain.ItemIdentifier]struct{}, len(row.Consumptions))
		for j, c := range row.Consumptions {
			key := domain.ItemIdentifier{ID: c.Item.ID, SKU: strings.ToLower(c.Item.SKU)}
			if _, dup := consumptionInputSpace[key]; dup {
				rowErrs.AddValidation(i, fmt.Sprintf("production_steps[%d].consumptions[%d].item", i, j), fmt.Sprintf("duplicate consumption item %q in the row", itemIdentifierLabel(c.Item)))
			}
			consumptionInputSpace[key] = struct{}{}
		}
	}
	return rowErrs.Summary("production steps")
}

// resolveBulkUpsertStepRows resolves every fuzzy reference in the rows — units, items,
// departments, and scanning stations — against the account, batching the lookups per
// reference kind, and fails fast on the first unresolved reference with a row-indexed
// param. Cost-rate units are dimension-checked here too: the unit resolver carries each
// unit's dimension code, so no extra lookup is needed. It returns rows with every
// reference replaced by its resolved ID — the payload the job stores and the worker
// writes from.
func resolveBulkUpsertStepRows(ctx context.Context, repos domain.RepoFactory, accountID string, rows []domain.UpsertProductionStepParams) ([]domain.ResolvedUpsertStepRow, *apierror.APIError) {
	// Collect every identifier across all rows, then batch-load each kind. The resolvers ignore
	// empty identifiers, so unset optional references cost nothing.
	var unitIdentifiers []domain.UnitIdentifier
	var itemIdentifiers []domain.ItemIdentifier
	var deptIdentifiers, stationIdentifiers []domain.ObjectIdentifier
	for _, row := range rows {
		for _, r := range []domain.UpsertRateParams{row.LaborRate, row.LaborTime, row.OverheadRate} {
			unitIdentifiers = append(unitIdentifiers, r.NumeratorUnit, r.DenominatorUnit)
		}
		unitIdentifiers = append(unitIdentifiers, row.Production.QuantityUnit)
		itemIdentifiers = append(itemIdentifiers, row.Production.Item)
		for _, c := range row.Consumptions {
			unitIdentifiers = append(unitIdentifiers, c.QuantityUnit)
			if c.WasteQuantityUnit != nil {
				unitIdentifiers = append(unitIdentifiers, *c.WasteQuantityUnit)
			}
			itemIdentifiers = append(itemIdentifiers, c.Item)
		}
		if row.Department != nil {
			deptIdentifiers = append(deptIdentifiers, *row.Department)
		}
		if row.ScanningStation != nil {
			stationIdentifiers = append(stationIdentifiers, *row.ScanningStation)
		}
	}

	units, apiErr := newUnitIdentifierResolver(ctx, repos, accountID, unitIdentifiers)
	if apiErr != nil {
		return nil, apiErr
	}
	items, apiErr := newItemIdentifierResolver(ctx, repos, accountID, itemIdentifiers)
	if apiErr != nil {
		return nil, apiErr
	}
	deptRepo := repos.NewDepartmentRepo()
	depts, apiErr := newObjectIdentifierResolver(ctx, accountID, "department", deptIdentifiers,
		deptRepo.GetByIDs, deptRepo.FindByNames,
		func(d *domain.Department) string { return d.ID },
		func(d *domain.Department) string { return d.Name },
	)
	if apiErr != nil {
		return nil, apiErr
	}
	stationRepo := repos.NewScanningStationRepo()
	stations, apiErr := newObjectIdentifierResolver(ctx, accountID, "scanning station", stationIdentifiers,
		stationRepo.GetByIDs, stationRepo.FindByNames,
		func(s *domain.ScanningStation) string { return s.ID },
		func(s *domain.ScanningStation) string { return s.Name },
	)
	if apiErr != nil {
		return nil, apiErr
	}

	resolved := make([]domain.ResolvedUpsertStepRow, len(rows))
	for i, row := range rows {
		param := func(field string) string {
			return fmt.Sprintf("production_steps[%d].%s", i, field)
		}

		out := domain.ResolvedUpsertStepRow{
			Name:           row.Name,
			Notes:          row.Notes,
			LevelingFactor: row.LevelingFactor,
			Allowances:     row.Allowances,
		}

		if row.Department != nil && *row.Department != (domain.ObjectIdentifier{}) {
			deptID, apiErr := depts.resolveOrError(*row.Department, param("department"))
			if apiErr != nil {
				return nil, apiErr
			}
			out.DepartmentID = &deptID
		}
		if row.ScanningStation != nil && *row.ScanningStation != (domain.ObjectIdentifier{}) {
			stationID, apiErr := stations.resolveOrError(*row.ScanningStation, param("scanning_station"))
			if apiErr != nil {
				return nil, apiErr
			}
			out.ScanningStationID = &stationID
		}

		// resolveRate resolves a rate's two units. labor_rate and overhead_rate are cost
		// rates whose numerator must be a currency unit and denominator must not be;
		// labor_time is a time-per-unit rate and is dimension-exempt.
		resolveRate := func(field string, r domain.UpsertRateParams, costRate bool) (domain.ResolvedUpsertRate, *apierror.APIError) {
			numParam := param(field + ".numerator_unit")
			denParam := param(field + ".denominator_unit")
			numerator, apiErr := units.resolveOrError(r.NumeratorUnit, numParam)
			if apiErr != nil {
				return domain.ResolvedUpsertRate{}, apiErr
			}
			denominator, apiErr := units.resolveOrError(r.DenominatorUnit, denParam)
			if apiErr != nil {
				return domain.ResolvedUpsertRate{}, apiErr
			}
			if costRate {
				if numerator.DimensionCode != unitDimensionCodeCurrency {
					return domain.ResolvedUpsertRate{}, apierror.NewValidationErrorWithParam(
						fmt.Sprintf("%s.numerator_unit must be a currency unit.", field), numParam)
				}
				if denominator.DimensionCode == unitDimensionCodeCurrency {
					return domain.ResolvedUpsertRate{}, apierror.NewValidationErrorWithParam(
						fmt.Sprintf("%s.denominator_unit must not be a currency unit.", field), denParam)
				}
			}
			return domain.ResolvedUpsertRate{Value: r.Value, NumeratorUnitID: numerator.ID, DenominatorUnitID: denominator.ID}, nil
		}

		if out.LaborRate, apiErr = resolveRate("labor_rate", row.LaborRate, true); apiErr != nil {
			return nil, apiErr
		}
		if out.LaborTime, apiErr = resolveRate("labor_time", row.LaborTime, false); apiErr != nil {
			return nil, apiErr
		}
		if out.OverheadRate, apiErr = resolveRate("overhead_rate", row.OverheadRate, true); apiErr != nil {
			return nil, apiErr
		}

		productionItemID, apiErr := items.resolveOrError(row.Production.Item, param("production.item"))
		if apiErr != nil {
			return nil, apiErr
		}
		productionUnit, apiErr := units.resolveOrError(row.Production.QuantityUnit, param("production.quantity_unit"))
		if apiErr != nil {
			return nil, apiErr
		}
		out.Production = domain.ResolvedUpsertProduction{
			ItemID:         productionItemID,
			QuantityValue:  row.Production.QuantityValue,
			QuantityUnitID: productionUnit.ID,
		}

		out.Consumptions = make([]domain.ResolvedUpsertConsumption, len(row.Consumptions))
		for j, c := range row.Consumptions {
			cParam := func(field string) string {
				return fmt.Sprintf("production_steps[%d].consumptions[%d].%s", i, j, field)
			}
			itemID, apiErr := items.resolveOrError(c.Item, cParam("item"))
			if apiErr != nil {
				return nil, apiErr
			}
			quantityUnit, apiErr := units.resolveOrError(c.QuantityUnit, cParam("quantity_unit"))
			if apiErr != nil {
				return nil, apiErr
			}
			rc := domain.ResolvedUpsertConsumption{
				ItemID:             itemID,
				QuantityValue:      c.QuantityValue,
				QuantityUnitID:     quantityUnit.ID,
				WasteQuantityValue: c.WasteQuantityValue,
				Instructions:       c.Instructions,
			}
			if c.WasteQuantityUnit != nil {
				wasteUnit, apiErr := units.resolveOrError(*c.WasteQuantityUnit, cParam("waste_quantity_unit"))
				if apiErr != nil {
					return nil, apiErr
				}
				wasteUnitID := wasteUnit.ID
				rc.WasteQuantityUnitID = &wasteUnitID
			}
			out.Consumptions[j] = rc
		}

		resolved[i] = out
	}

	return resolved, nil
}

// bulkUpsertSpec wires production steps into the async bulk-operation engine. The
// engine owns the job lifecycle, idempotency, transaction, and outbox; these hooks
// carry the production-step-specific validate / resolve / write / relink. As an upsert,
// it requires both create and update, and its acknowledgment is a bare job handle — the
// created/updated split is not known until the writes run.
func (s *productionStepSvcImpl) bulkUpsertSpec() bulkOperationSpec[domain.UpsertProductionStepParams, domain.ResolvedUpsertStepRow] {
	return bulkOperationSpec[domain.UpsertProductionStepParams, domain.ResolvedUpsertStepRow]{
		JobType:          constants.JobTypeBulkUpsert,
		RoutingKey:       messaging.BulkUpsertProductionSteps.RoutingKey(),
		PermissionDomain: types.PermissionDomainProductionSteps,
		Actions:          []types.Action{types.ActionCreate, types.ActionUpdate},
		EntityName:       "production steps",
		Validate:         validateBulkUpsertStepRows,
		Resolve:          resolveBulkUpsertStepRows,
		// No AcceptResults: an upsert cannot know the created/updated split until Write
		// runs, so its results fill in then rather than at accept.
		Write: writeBulkUpsertProductionSteps,
		// Re-derive flow DAG edges from item flows for every written step, after the
		// write commits and non-fatally — mirroring single create.
		AfterCommit: func(ctx context.Context, repos domain.RepoFactory, accountID string, writtenIDs []string) *apierror.APIError {
			meds := s.mediatorFactory.Build(repos)

			// Each step relinks independently, so one failure does not stop the others.
			// The engine logs what is returned, so the first cause is reported naming
			// every step that did not relink — enough to re-drive them by hand.
			var unlinked []string
			var cause *apierror.APIError
			for _, stepID := range writtenIDs {
				if apiErr := meds.ProductionFlow.LinkFlow(ctx, stepID, accountID); apiErr != nil {
					unlinked = append(unlinked, stepID)
					if cause == nil {
						cause = apiErr
					}
				}
			}
			if cause == nil {
				return nil
			}
			return apierror.NewInternalError(cause, fmt.Sprintf(
				"Failed to relink production flows for %d of %d written steps: %s",
				len(unlinked), len(writtenIDs), strings.Join(unlinked, ", "),
			))
		},
	}
}

// BulkUpsertProductionSteps accepts a bulk upsert: it validates and resolves
// synchronously, records the resolved rows on a job, and returns the job to poll. The
// steps are created or updated asynchronously by ExecuteBulkUpsertProductionSteps.
func (s *productionStepSvcImpl) BulkUpsertProductionSteps(ctx context.Context, params domain.BulkUpsertProductionStepsParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), params.ProductionSteps)
}

// ExecuteBulkUpsertProductionSteps performs the writes for an enqueued bulk upsert.
// Called by the bulk upsert consumer; exactly-once is provided by the message inbox.
func (s *productionStepSvcImpl) ExecuteBulkUpsertProductionSteps(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), event)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// lists the columns shared by the production step export and its import template,
// with the guidance a reader needs to edit and re-upload the file
var productionStepColumns = []excel.ColumnSpec{
	{Header: "Name", Key: "name", Width: 22, Note: "Required. Step name. A row with a Name starts a step; a matching existing step (by name) is updated. Leave blank on following rows to add more consumptions to the step above."},
	{Header: "Department", Key: "department", Width: 16, Note: "Optional. Must match an existing department. Can't be changed after the step is created."},
	{Header: "Scanning Station", Key: "station", Width: 18, Note: "Optional. Scanning station this step reports at (must exist)."},
	{Header: "Labor Rate", Key: "labor_rate", Width: 14, Note: "Required. Labor rate value (a number)."},
	{Header: "LR Currency Unit", Key: "labor_rate_currency_unit", Width: 14, Note: "Optional. Currency unit for the labor rate. Defaults to $."},
	{Header: "LR Time Unit", Key: "labor_rate_time_unit", Width: 12, Note: "Optional. Time unit for the labor rate (e.g. hr). Defaults to hr."},
	{Header: "Labor Time", Key: "labor_time", Width: 10, Note: "Required. Labor time value (a number) per one produced unit."},
	{Header: "LT Unit", Key: "labor_time_unit", Width: 8, Note: "Optional. Time unit for labor time (hr, min, sec, day). Defaults to hr."},
	{Header: "LT Per Unit", Key: "labor_time_per_unit", Width: 10, Note: "Optional. Unit the labor time is measured per. Defaults to the Produced Unit."},
	{Header: "Overhead Rate", Key: "overhead_rate", Width: 12, Note: "Required. Overhead rate value (a number)."},
	{Header: "OR Currency Unit", Key: "overhead_rate_currency_unit", Width: 14, Note: "Optional. Currency unit for the overhead rate. Defaults to $."},
	{Header: "OR Time Unit", Key: "overhead_rate_time_unit", Width: 12, Note: "Optional. Time unit for the overhead rate (e.g. hr). Defaults to hr."},
	{Header: "Allowances", Key: "allowances", Width: 11, Note: "Optional. Allowance correction factor applied to labor time. Defaults to 0."},
	{Header: "Leveling Factor", Key: "leveling_factor", Width: 14, Note: "Optional. Leveling correction factor applied to labor time. Defaults to 0."},
	{Header: "Produced Item", Key: "produced_item", Width: 18, Note: "Required. Exactly one produced item per step. SKU (or item ID) of the item this step produces. Quantify it in a unit from its own unit group."},
	{Header: "Produced Quantity", Key: "produced_quantity", Width: 16, Note: "Required. Quantity produced (a number)."},
	{Header: "Produced Unit", Key: "produced_unit", Width: 14, Note: "Required. Unit for the produced quantity — must belong to the produced item's unit group."},
	{Header: "Consumed Item", Key: "consumed_item", Width: 18, Note: "Optional. SKU (or item ID) of a consumed material. Quantify it in a unit from its own unit group. Add more consumptions on rows below with a blank Name."},
	{Header: "Consumed Quantity", Key: "consumed_quantity", Width: 16, Note: "Required when a Consumed Item is set. Quantity consumed (a number)."},
	{Header: "Consumed Unit", Key: "consumed_unit", Width: 14, Note: "Required when a Consumed Item is set. Unit for the consumed quantity — must belong to the consumed item's unit group."},
	{Header: "Consumed Waste Quantity", Key: "consumed_waste_quantity", Width: 22, Note: "Optional. Expected scrap/waste quantity for this consumption."},
	{Header: "Consumed Waste Unit", Key: "consumed_waste_unit", Width: 18, Note: "Optional. Unit for the waste quantity. Defaults to the Consumed Unit."},
	{Header: "Instructions", Key: "instructions", Width: 24, Note: "Optional. Instructions for how this material is consumed."},
	{Header: "Notes", Key: "notes", Width: 30, Note: "Optional. Free-form notes about the step."},
}

// hands the export engine the plumbing it runs on.

// wires production steps into the export engine. A step's consumptions are listed
// one per row, so the step's own columns sit on the first of them.
func (s *productionStepSvcImpl) exportSpec() exportSpec[*domain.ProductionStepExport, domain.ExportProductionStepsParams] {
	return exportSpec[*domain.ProductionStepExport, domain.ExportProductionStepsParams]{
		PermissionDomain: types.PermissionDomainProductionSteps,
		Name:             "Production Steps",
		Slug:             "production_steps",
		Columns:          productionStepColumns,

		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportProductionStepsParams) ([]*domain.ProductionStepExport, *apierror.APIError) {
			filters.AccountID = accountID
			return repos.NewProductionStepQueryRepo().Export(ctx, filters)
		},

		Expand: func(step *domain.ProductionStepExport) []excel.Row {
			parent := excel.Row{
				"name":                        step.Name,
				"department":                  excel.Str(step.DepartmentName),
				"station":                     excel.Str(step.ScanningStationName),
				"labor_rate":                  decimalCellPtr(step.LaborRate),
				"labor_rate_currency_unit":    excel.Str(step.LaborRateCurrencyUnit),
				"labor_rate_time_unit":        excel.Str(step.LaborRateTimeUnit),
				"labor_time":                  decimalCellPtr(step.LaborTime),
				"labor_time_unit":             excel.Str(step.LaborTimeUnit),
				"labor_time_per_unit":         decimalCellPtr(step.LaborTimePerUnit),
				"overhead_rate":               decimalCellPtr(step.OverheadRate),
				"overhead_rate_currency_unit": excel.Str(step.OverheadRateCurrencyUnit),
				"overhead_rate_time_unit":     excel.Str(step.OverheadRateTimeUnit),
				// 0 is the default; blank so a re-import keeps it rather than resetting.
				"allowances":        blankIfZero(step.Allowances),
				"leveling_factor":   blankIfZero(step.LevelingFactor),
				"produced_item":     step.ProducedItemSKU,
				"produced_quantity": decimalCell(step.ProducedQuantity),
				"produced_unit":     step.ProducedUnit,
				"notes":             excel.Str(step.Notes),
			}

			children := make([]excel.Row, len(step.Consumptions))
			for i, c := range step.Consumptions {
				children[i] = excel.Row{
					"consumed_item":           c.ItemSKU,
					"consumed_quantity":       decimalCell(c.Quantity),
					"consumed_unit":           c.Unit,
					"consumed_waste_quantity": blankIfZero(excel.Str(c.WasteQuantity)),
					"consumed_waste_unit":     excel.Str(c.WasteUnit),
					"instructions":            excel.Str(c.Instructions),
				}
			}

			return excel.Group(parent, children)
		},
	}
}

// blanks a stored zero so a re-import keeps the field's default rather than
// writing an explicit 0 back
func blankIfZero(stored string) string {
	if value, err := strconv.ParseFloat(stored, 64); err == nil && value == 0 {
		return ""
	}
	return decimalCell(stored)
}

// accepts an export: records what to build on a job and returns it to poll.
func (s *productionStepSvcImpl) ExportProductionSteps(ctx context.Context, params domain.ExportProductionStepsParams) (*domain.Job, *apierror.APIError) {
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), params)
}

// renders the file an accepted export recorded
func (s *productionStepSvcImpl) BuildExportProductionSteps(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}

// writeBulkUpsertProductionSteps is the engine's Write hook: in one transaction it
// matches the resolved rows against existing steps by name, creates or updates each,
// and returns the two id lists. Matched by name (case-insensitive) within the account.
// On update, rate rows are never mutated — fresh rate rows are inserted and the step
// re-pointed — and the production and consumptions are replaced wholesale. The
// department is create-only: a matched row stating a different department is rejected.
func writeBulkUpsertProductionSteps(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.ResolvedUpsertStepRow) (BulkWriteResult, *apierror.APIError) {
	txRepo := txRepos.NewProductionStepRepo()
	consumptionRepo := txRepos.NewConsumptionRepo()

	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = strings.ToLower(row.Name)
	}

	existing, apiErr := txRepo.FindByNames(txCtx, accountID, names)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	results := make([]domain.RowResult, 0, len(rows))
	var rowErrors []apierror.RowError
	byName := make(map[string]*domain.ProductionStepBulkRow, len(existing))
	for _, step := range existing {
		byName[strings.ToLower(step.Name)] = step
	}

	// insertRates inserts three fresh rate rows and returns their IDs. Rate rows are
	// never mutated in place — on update the step is re-pointed.
	insertRates := func(row domain.ResolvedUpsertStepRow) (laborRateID, laborTimeID, overheadRateID string, apiErr *apierror.APIError) {
		rateFor := func(r domain.ResolvedUpsertRate) (string, *apierror.APIError) {
			rateID, apiErr := id.GenID(id.RateIDPrefix, nil)
			if apiErr != nil {
				return "", apiErr
			}
			return rateID, txRepo.InsertRate(txCtx, rateID, domain.CreateRateParams(r))
		}
		if laborRateID, apiErr = rateFor(row.LaborRate); apiErr != nil {
			return
		}
		if laborTimeID, apiErr = rateFor(row.LaborTime); apiErr != nil {
			return
		}
		overheadRateID, apiErr = rateFor(row.OverheadRate)
		return
	}

	// insertOutputsAndInputs writes the row's production and consumptions.
	insertOutputsAndInputs := func(stepID string, row domain.ResolvedUpsertStepRow) *apierror.APIError {
		productionID, apiErr := id.GenID(id.ProductionIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := txRepo.InsertQuantity(txCtx, quantityID, row.Production.QuantityValue, row.Production.QuantityUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txRepo.InsertProduction(txCtx, productionID, row.Production.ItemID, quantityID, stepID); apiErr != nil {
			return apiErr
		}
		for _, c := range row.Consumptions {
			consumptionID, apiErr := id.GenID(id.ConsumptionIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			cQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			wasteQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			wasteValue := "0"
			if c.WasteQuantityValue != nil {
				wasteValue = *c.WasteQuantityValue
			}
			// Waste defaults to the consumption's own quantity unit when its unit is
			// omitted.
			wasteUnitID := c.QuantityUnitID
			if c.WasteQuantityUnitID != nil {
				wasteUnitID = *c.WasteQuantityUnitID
			}
			_, apiErr = consumptionRepo.Create(txCtx, consumptionID, cQuantityID, wasteQuantityID, domain.CreateConsumptionParams{
				AccountID:           accountID,
				ProductionStepID:    stepID,
				ItemID:              c.ItemID,
				QuantityValue:       c.QuantityValue,
				QuantityUnitID:      c.QuantityUnitID,
				WasteQuantityValue:  wasteValue,
				WasteQuantityUnitID: wasteUnitID,
				Instructions:        c.Instructions,
			})
			if apiErr != nil {
				return apiErr
			}
		}
		return nil
	}

	for i, row := range rows {
		var createdID, updatedID string
		// Each step upserts inside its own savepoint: a row rejected (create-only
		// department change) or failing to write rolls back only itself; the batch continues.
		rowErr := sp.Run(txCtx, func(_ context.Context) *apierror.APIError {
			old := byName[names[i]]
			stationID := row.ScanningStationID

			// The department is create-only: a matched row stating a department that
			// differs from the step's current one (including a step with none) is rejected
			// rather than silently ignored.
			if old != nil && row.DepartmentID != nil &&
				!(old.DepartmentID != nil && *old.DepartmentID == *row.DepartmentID) {
				return apierror.NewValidationErrorWithParam(
					fmt.Sprintf("Production step %q's department cannot be changed by bulk upsert.", old.Name),
					fmt.Sprintf("production_steps[%d].department", i),
				)
			}

			if old == nil {
				stepID, apiErr := id.GenID(id.ProductionStepIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				laborRateID, laborTimeID, overheadRateID, apiErr := insertRates(row)
				if apiErr != nil {
					return apiErr
				}

				levelingFactor := "0"
				if row.LevelingFactor != nil {
					levelingFactor = *row.LevelingFactor
				}
				allowances := "0"
				if row.Allowances != nil {
					allowances = *row.Allowances
				}

				if apiErr := txRepo.InsertStep(txCtx, stepID, row.Name, row.Notes, levelingFactor, allowances, laborRateID, laborTimeID, overheadRateID, stationID, row.DepartmentID, accountID); apiErr != nil {
					return apiErr
				}
				if apiErr := insertOutputsAndInputs(stepID, row); apiErr != nil {
					return apiErr
				}

				created, apiErr := txRepo.Get(txCtx, accountID, stepID)
				if apiErr != nil {
					return apiErr
				}
				if apiErr := audit.NewPublisher().Publish(txCtx, txRepos.NewOutboxRepo(), audit.EventData{
					ServiceName:  domain.ServiceName,
					Action:       constants.AuditActionCreate,
					ResourceType: constants.ObjectTypeProductionStep,
					ResourceID:   created.ID,
					Changes:      audit.ComputeChanges(nil, created),
				}); apiErr != nil {
					return apiErr
				}

				createdID = stepID
				return nil
			}

			// Update: name adopts the request's casing; notes, leveling factor, allowances,
			// and scanning station are preserved when omitted; fresh rate rows are inserted
			// and the step re-pointed; productions and consumptions are replaced wholesale.
			// The department is create-only, validated above, and left unchanged.
			oldFull, apiErr := txRepo.Get(txCtx, accountID, old.ID)
			if apiErr != nil {
				return apiErr
			}

			laborRateID, laborTimeID, overheadRateID, apiErr := insertRates(row)
			if apiErr != nil {
				return apiErr
			}

			levelingFactor := old.LevelingFactor
			if row.LevelingFactor != nil {
				levelingFactor = *row.LevelingFactor
			}
			allowances := old.Allowances
			if row.Allowances != nil {
				allowances = *row.Allowances
			}
			notes := old.Notes
			if row.Notes != nil {
				notes = row.Notes
			}
			if stationID == nil {
				stationID = old.ScanningStationID
			}

			if apiErr := txRepo.UpdateForBulkUpsert(txCtx, domain.UpdateProductionStepForBulkUpsertParams{
				AccountID:         accountID,
				ProductionStepID:  old.ID,
				Name:              row.Name,
				Notes:             notes,
				LevelingFactor:    levelingFactor,
				Allowances:        allowances,
				ScanningStationID: stationID,
				LaborRateID:       laborRateID,
				LaborTimeID:       laborTimeID,
				OverheadRateID:    overheadRateID,
			}); apiErr != nil {
				return apiErr
			}

			if apiErr := txRepo.DeleteProductionsByStepID(txCtx, old.ID); apiErr != nil {
				return apiErr
			}
			if apiErr := txRepo.DeleteConsumptionsByStepID(txCtx, old.ID); apiErr != nil {
				return apiErr
			}
			if apiErr := insertOutputsAndInputs(old.ID, row); apiErr != nil {
				return apiErr
			}

			updated, apiErr := txRepo.Get(txCtx, accountID, old.ID)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := audit.NewPublisher().Publish(txCtx, txRepos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeProductionStep,
				ResourceID:   updated.ID,
				Changes:      audit.ComputeChanges(oldFull, updated),
			}); apiErr != nil {
				return apiErr
			}

			updatedID = old.ID
			return nil
		})
		if rowErr != nil {
			rowErrors = append(rowErrors, apierror.NewRowError(i, rowErr))
			continue
		}

		if createdID != "" {
			results = append(results, newRowResult(i, createdID, true))
		} else {
			results = append(results, newRowResult(i, updatedID, false))
		}
	}

	return BulkWriteResult{
		Results:    results,
		Errors:     rowErrors,
		WrittenIDs: resultIDs(results),
	}, nil
}

// bulkCreateSingleStep processes a single row in the bulk create operation.
// Errors are captured in the result rather than propagated.
func (s *productionStepSvcImpl) bulkCreateSingleStep(
	ctx context.Context,
	accountID string,
	input domain.BulkCreateProductionStepInput,
	itemBySKU map[string]domain.ItemSKUInfo,
	unitByAbbr map[string]string,
	dollarUnitID, hrUnitID string,
) domain.BulkCreateProductionStepResult {
	skipResult := func(name, reason string) domain.BulkCreateProductionStepResult {
		return domain.BulkCreateProductionStepResult{
			Name:    name,
			Success: false,
			Error:   &reason,
			Action:  "skipped",
		}
	}

	// Validation: name required.
	if strings.TrimSpace(input.Name) == "" {
		return skipResult(input.Name, "Name is required")
	}

	// Validation: at least one production required.
	if len(input.Productions) == 0 {
		return skipResult(input.Name, "No productions found in row")
	}

	// Resolve scanning station by name if provided.
	var scanningStationID *string
	if input.Station != nil && *input.Station != "" {
		stationID, apiErr := s.repos.NewScanningStationRepo().FindIDByName(ctx, accountID, *input.Station)
		if apiErr != nil {
			return skipResult(input.Name, "Failed to resolve station")
		}
		if stationID == nil {
			return skipResult(input.Name, "Invalid station name")
		}
		scanningStationID = stationID
	}

	// Validate labor time unit.
	laborTimeUnit := "hr"
	if input.LaborTimeUnit != nil && *input.LaborTimeUnit != "" {
		laborTimeUnit = *input.LaborTimeUnit
	}
	if !validLaborTimeUnits[strings.ToLower(laborTimeUnit)] {
		return skipResult(input.Name, "Invalid labor time unit")
	}

	laborTimeUnitID, ok := unitByAbbr[strings.ToLower(laborTimeUnit)]
	if !ok {
		return skipResult(input.Name, "Invalid labor time unit")
	}

	// Resolve production items.
	for _, p := range input.Productions {
		if _, ok := itemBySKU[p.SKU]; !ok {
			return skipResult(input.Name, fmt.Sprintf("Missing item for production SKU: %s", p.SKU))
		}
	}

	// Resolve consumption items.
	for _, c := range input.Consumptions {
		if _, ok := itemBySKU[c.SKU]; !ok {
			return skipResult(input.Name, fmt.Sprintf("Missing item for consumption SKU: %s", c.SKU))
		}
	}

	// Determine leveling factor and allowances (defaults to "0").
	levelingFactor := "0"
	if input.LevelingFactor != nil {
		levelingFactor = *input.LevelingFactor
	}
	allowances := "0"
	if input.Allowances != nil {
		allowances = *input.Allowances
	}

	// Check if step already exists by name.
	stepRepo := s.repos.NewProductionStepRepo()
	existingStepID, apiErr := stepRepo.FindIDByName(ctx, accountID, input.Name)
	if apiErr != nil {
		return skipResult(input.Name, "Failed to check existing step")
	}

	if existingStepID != nil {
		// UPDATE existing step.
		return s.bulkUpdateExistingStep(ctx, accountID, *existingStepID, input, itemBySKU, unitByAbbr, dollarUnitID, hrUnitID, laborTimeUnitID, levelingFactor, allowances, scanningStationID)
	}

	// CREATE new step.
	return s.bulkCreateNewStep(ctx, accountID, input, itemBySKU, dollarUnitID, hrUnitID, laborTimeUnitID, levelingFactor, allowances, scanningStationID)
}

// bulkUpdateExistingStep updates an existing production step with new data (delete old consumptions/productions, recreate).
func (s *productionStepSvcImpl) bulkUpdateExistingStep(
	ctx context.Context,
	accountID, stepID string,
	input domain.BulkCreateProductionStepInput,
	itemBySKU map[string]domain.ItemSKUInfo,
	_ map[string]string,
	_, _, _ string,
	levelingFactor, allowances string,
	scanningStationID *string,
) domain.BulkCreateProductionStepResult {
	// Fetch old state before mutation for audit diff.
	old, fetchErr := s.repos.NewProductionStepRepo().Get(ctx, accountID, stepID)
	if fetchErr != nil {
		msg := "Update failed"
		return domain.BulkCreateProductionStepResult{
			Name:             input.Name,
			Success:          false,
			Error:            &msg,
			ProductionStepID: &stepID,
			Action:           "skipped",
		}
	}

	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *productionStepSvcImpl) *apierror.APIError {
		txStepRepo := txSvc.repos.NewProductionStepRepo()

		// Delete existing consumptions and productions.
		if apiErr := txStepRepo.DeleteConsumptionsByStepID(txCtx, stepID); apiErr != nil {
			return apiErr
		}
		if apiErr := txStepRepo.DeleteProductionsByStepID(txCtx, stepID); apiErr != nil {
			return apiErr
		}

		// Update the step record.
		if apiErr := txStepRepo.UpdateStepFull(txCtx, stepID, accountID, levelingFactor, allowances, scanningStationID); apiErr != nil {
			return apiErr
		}

		// Recreate productions.
		for _, p := range input.Productions {
			item := itemBySKU[p.SKU]
			productionID, apiErr := id.GenID(id.ProductionIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := txStepRepo.InsertQuantity(txCtx, quantityID, p.Measure, item.BaseUnitID); apiErr != nil {
				return apiErr
			}
			if apiErr := txStepRepo.InsertProduction(txCtx, productionID, item.ItemID, quantityID, stepID); apiErr != nil {
				return apiErr
			}
		}

		// Recreate consumptions.
		consumptionRepo := txSvc.repos.NewConsumptionRepo()
		for _, c := range input.Consumptions {
			item := itemBySKU[c.SKU]
			consumptionID, apiErr := id.GenID(id.ConsumptionIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			cQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			wasteQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			_, apiErr = consumptionRepo.Create(txCtx, consumptionID, cQuantityID, wasteQuantityID, domain.CreateConsumptionParams{
				AccountID:           accountID,
				ProductionStepID:    stepID,
				ItemID:              item.ItemID,
				QuantityValue:       c.Measure,
				QuantityUnitID:      item.BaseUnitID,
				WasteQuantityValue:  "0",
				WasteQuantityUnitID: item.BaseUnitID,
				Instructions:        c.Instructions,
			})
			if apiErr != nil {
				return apiErr
			}
		}

		// Re-fetch updated state for audit diff.
		updated, apiErr := txStepRepo.Get(txCtx, accountID, stepID)
		if apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(old, updated)
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeProductionStep,
			ResourceID:   stepID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
	if apiErr != nil {
		msg := "Update failed"
		return domain.BulkCreateProductionStepResult{
			Name:             input.Name,
			Success:          false,
			Error:            &msg,
			ProductionStepID: &stepID,
			Action:           "skipped",
		}
	}

	// Link production flow (non-fatal).
	if linkErr := s.mediators().ProductionFlow.LinkFlow(ctx, stepID, accountID); linkErr != nil {
		_ = linkErr
	}

	return domain.BulkCreateProductionStepResult{
		Name:             input.Name,
		Success:          true,
		ProductionStepID: &stepID,
		Action:           "updated",
	}
}

// bulkCreateNewStep creates a brand new production step with rates, production, and consumptions.
func (s *productionStepSvcImpl) bulkCreateNewStep(
	ctx context.Context,
	accountID string,
	input domain.BulkCreateProductionStepInput,
	itemBySKU map[string]domain.ItemSKUInfo,
	dollarUnitID, hrUnitID, laborTimeUnitID string,
	levelingFactor, allowances string,
	scanningStationID *string,
) domain.BulkCreateProductionStepResult {
	// Get the base unit of the first production item for labor time denominator.
	firstProdItem := itemBySKU[input.Productions[0].SKU]
	laborTimeDenUnitID := firstProdItem.BaseUnitID

	// Generate IDs.
	stepID, apiErr := id.GenID(id.ProductionStepIDPrefix, nil)
	if apiErr != nil {
		msg := "Create failed"
		return domain.BulkCreateProductionStepResult{Name: input.Name, Success: false, Error: &msg, Action: "skipped"}
	}
	laborRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		msg := "Create failed"
		return domain.BulkCreateProductionStepResult{Name: input.Name, Success: false, Error: &msg, Action: "skipped"}
	}
	laborTimeID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		msg := "Create failed"
		return domain.BulkCreateProductionStepResult{Name: input.Name, Success: false, Error: &msg, Action: "skipped"}
	}
	overheadRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		msg := "Create failed"
		return domain.BulkCreateProductionStepResult{Name: input.Name, Success: false, Error: &msg, Action: "skipped"}
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionStepSvcImpl) *apierror.APIError {
		txStepRepo := txSvc.repos.NewProductionStepRepo()

		// Insert rates: labor rate ($/hr), overhead rate ($/hr), labor time (laborTimeUnit/productionUnit).
		if apiErr := txStepRepo.InsertRate(txCtx, laborRateID, domain.CreateRateParams{
			Value:             input.LaborRate,
			NumeratorUnitID:   dollarUnitID,
			DenominatorUnitID: hrUnitID,
		}); apiErr != nil {
			return apiErr
		}
		if apiErr := txStepRepo.InsertRate(txCtx, laborTimeID, domain.CreateRateParams{
			Value:             input.LaborTime,
			NumeratorUnitID:   laborTimeUnitID,
			DenominatorUnitID: laborTimeDenUnitID,
		}); apiErr != nil {
			return apiErr
		}
		if apiErr := txStepRepo.InsertRate(txCtx, overheadRateID, domain.CreateRateParams{
			Value:             input.OverheadRate,
			NumeratorUnitID:   dollarUnitID,
			DenominatorUnitID: hrUnitID,
		}); apiErr != nil {
			return apiErr
		}

		// Insert the production step.
		if apiErr := txStepRepo.InsertStep(txCtx, stepID, input.Name, nil, levelingFactor, allowances, laborRateID, laborTimeID, overheadRateID, scanningStationID, nil, accountID); apiErr != nil {
			return apiErr
		}

		// Insert productions.
		for _, p := range input.Productions {
			item := itemBySKU[p.SKU]
			productionID, apiErr := id.GenID(id.ProductionIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := txStepRepo.InsertQuantity(txCtx, quantityID, p.Measure, item.BaseUnitID); apiErr != nil {
				return apiErr
			}
			if apiErr := txStepRepo.InsertProduction(txCtx, productionID, item.ItemID, quantityID, stepID); apiErr != nil {
				return apiErr
			}
		}

		// Insert consumptions.
		consumptionRepo := txSvc.repos.NewConsumptionRepo()
		for _, c := range input.Consumptions {
			item := itemBySKU[c.SKU]
			consumptionID, apiErr := id.GenID(id.ConsumptionIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			cQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			wasteQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			_, apiErr = consumptionRepo.Create(txCtx, consumptionID, cQuantityID, wasteQuantityID, domain.CreateConsumptionParams{
				AccountID:           accountID,
				ProductionStepID:    stepID,
				ItemID:              item.ItemID,
				QuantityValue:       c.Measure,
				QuantityUnitID:      item.BaseUnitID,
				WasteQuantityValue:  "0",
				WasteQuantityUnitID: item.BaseUnitID,
				Instructions:        c.Instructions,
			})
			if apiErr != nil {
				return apiErr
			}
		}

		// Re-fetch the created step for audit diff.
		created, apiErr := txStepRepo.Get(txCtx, accountID, stepID)
		if apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeProductionStep,
			ResourceID:   stepID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
	if apiErr != nil {
		msg := "Create failed"
		return domain.BulkCreateProductionStepResult{Name: input.Name, Success: false, Error: &msg, Action: "skipped"}
	}

	// Link production flow (non-fatal).
	if linkErr := s.mediators().ProductionFlow.LinkFlow(ctx, stepID, accountID); linkErr != nil {
		_ = linkErr
	}

	return domain.BulkCreateProductionStepResult{
		Name:             input.Name,
		Success:          true,
		ProductionStepID: &stepID,
		Action:           "created",
	}
}
