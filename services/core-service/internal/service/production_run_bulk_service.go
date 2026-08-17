package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/excel"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/messaging"
	"github.com/shopspring/decimal"
)

// asyncBulkDeps hands the async bulk engine the plumbing this service already holds.
func (s *productionRunSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

// items, units, and scanning stations keyed by the fuzzy identifier the row carried.
type bulkRunIdentifiers struct {
	itemIDByIdentifier    map[domain.ItemIdentifier]string
	unitIDByIdentifier    map[domain.UnitIdentifier]string
	stationIDByIdentifier map[domain.ObjectIdentifier]string
}

// resolveBulkRunIdentifiers validates every reference in the runs' batches and resolves them
// to IDs the async executor can write with. Items are referenced by a fuzzy identifier (id or
// SKU); units by a fuzzy identifier (id, name, or abbreviation); scanning stations by a fuzzy
// identifier (id or name); production step IDs are validated against the account. Each
// reference kind is looked up in one or two batched queries and the row-by-row pass
// fails fast on the first unresolved reference with a row-indexed param. Validating
// synchronously keeps bad references out of the async execution path, where they would
// fail invisibly.
func resolveBulkRunIdentifiers(ctx context.Context, repos domain.RepoFactory, accountID string, runs []domain.BulkCreateProductionRunParams) (*bulkRunIdentifiers, *apierror.APIError) {
	// Collect every identifier across all batches, then batch-load each kind. The resolvers
	// ignore empty identifiers, so unset optional references cost nothing.
	var itemIdentifiers []domain.ItemIdentifier
	var unitIdentifiers []domain.UnitIdentifier
	var stationIdentifiers []domain.ObjectIdentifier
	for _, run := range runs {
		for _, b := range run.Batches {
			itemIdentifiers = append(itemIdentifiers, b.Item)
			unitIdentifiers = append(unitIdentifiers, b.QuantityUnit)
			if b.SecondsUnit != nil {
				unitIdentifiers = append(unitIdentifiers, *b.SecondsUnit)
			}
			if b.WasteUnit != nil {
				unitIdentifiers = append(unitIdentifiers, *b.WasteUnit)
			}
			if b.ScanningStation != nil {
				stationIdentifiers = append(stationIdentifiers, *b.ScanningStation)
			}
		}
	}

	items, apiErr := newItemIdentifierResolver(ctx, repos, accountID, itemIdentifiers)
	if apiErr != nil {
		return nil, apiErr
	}
	units, apiErr := newUnitIdentifierResolver(ctx, repos, accountID, unitIdentifiers)
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

	identifiers := &bulkRunIdentifiers{
		itemIDByIdentifier:    map[domain.ItemIdentifier]string{},
		unitIDByIdentifier:    map[domain.UnitIdentifier]string{},
		stationIDByIdentifier: map[domain.ObjectIdentifier]string{},
	}

	// Fail fast on the first unresolved reference, row by row. Production step IDs are
	// validated against the account (checked once per distinct ID).
	stepQueryRepo := repos.NewProductionStepQueryRepo()
	validStepIDs := make(map[string]struct{})
	for i, run := range runs {
		for j, b := range run.Batches {
			param := func(field string) string {
				return fmt.Sprintf("production_runs[%d].batches[%d].%s", i, j, field)
			}

			itemID, apiErr := items.resolveOrError(b.Item, param("item"))
			if apiErr != nil {
				return nil, apiErr
			}
			identifiers.itemIDByIdentifier[b.Item] = itemID

			quantityUnit, apiErr := units.resolveOrError(b.QuantityUnit, param("quantity_unit"))
			if apiErr != nil {
				return nil, apiErr
			}
			identifiers.unitIDByIdentifier[b.QuantityUnit] = quantityUnit.ID

			if b.SecondsUnit != nil {
				u, apiErr := units.resolveOrError(*b.SecondsUnit, param("seconds_unit"))
				if apiErr != nil {
					return nil, apiErr
				}
				identifiers.unitIDByIdentifier[*b.SecondsUnit] = u.ID
			}
			if b.WasteUnit != nil {
				u, apiErr := units.resolveOrError(*b.WasteUnit, param("waste_unit"))
				if apiErr != nil {
					return nil, apiErr
				}
				identifiers.unitIDByIdentifier[*b.WasteUnit] = u.ID
			}
			if b.ScanningStation != nil && *b.ScanningStation != (domain.ObjectIdentifier{}) {
				stationID, apiErr := stations.resolveOrError(*b.ScanningStation, param("scanning_station"))
				if apiErr != nil {
					return nil, apiErr
				}
				identifiers.stationIDByIdentifier[*b.ScanningStation] = stationID
			}
			if b.ProductionStepID != nil && *b.ProductionStepID != "" {
				if _, ok := validStepIDs[*b.ProductionStepID]; !ok {
					inAccount, apiErr := stepQueryRepo.IsInAccount(ctx, accountID, *b.ProductionStepID)
					if apiErr != nil {
						return nil, apiErr
					}
					if !inAccount {
						return nil, apierror.NewValidationErrorWithParam(
							fmt.Sprintf("Production step %q was not found.", *b.ProductionStepID),
							param("production_step_id"),
						)
					}
					validStepIDs[*b.ProductionStepID] = struct{}{}
				}
			}
		}
	}

	return identifiers, nil
}

// bulkCreateSpec wires production runs into the async bulk-operation engine. As a
// create, it requires only create permission, and its acknowledgment carries the
// pre-generated run and batch ids so the client can poll each — the engine's job
// handle plus those ids.
func (s *productionRunSvcImpl) bulkCreateSpec() bulkOperationSpec[domain.BulkCreateProductionRunParams, domain.BulkCreateProductionRunEventRun] {
	return bulkOperationSpec[domain.BulkCreateProductionRunParams, domain.BulkCreateProductionRunEventRun]{
		JobType:          constants.JobTypeBulkCreate,
		ResourceType:     constants.ObjectTypeProductionRun,
		RoutingKey:       messaging.BulkCreateProductionRuns.RoutingKey(),
		PermissionDomain: types.PermissionDomainProductionRuns,
		Actions:          []types.Action{types.ActionCreate},
		EntityName:       "production runs",
		Validate:         validateBulkCreateRunRows,
		Resolve:          resolveBulkCreateRunRows,
		// A create knows its run and batch ids at accept (they are pre-generated in
		// Resolve), so it records them on the job now — the 202 returns the job already
		// carrying them, and the execute-phase Write records the same ids.
		AcceptResults: func(resolved []domain.BulkCreateProductionRunEventRun) []domain.RowResult {
			results := make([]domain.RowResult, len(resolved))
			for i, r := range resolved {
				batchIDs := make([]string, len(r.Batches))
				for j, b := range r.Batches {
					batchIDs[j] = b.BatchID
				}
				results[i] = runRowResult(i, r.ProductionRunID, batchIDs)
			}
			return results
		},
		Write: writeBulkCreateProductionRuns,
	}
}

// BulkCreateProductionRuns accepts a bulk create: it validates and resolves
// synchronously, pre-generates the run and batch ids, records the resolved rows and
// those ids on a job, and returns the job to poll — already carrying the ids in its
// results. The runs are created asynchronously by ExecuteBulkCreateProductionRuns.
func (s *productionRunSvcImpl) BulkCreateProductionRuns(ctx context.Context, params domain.BulkCreateProductionRunsParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkCreateSpec(), params.ProductionRuns)
}

// validateBulkCreateRunRows runs the accept-phase structural checks: every run needs at
// least one batch and every quantity/seconds/waste value must be a decimal. It touches
// no database.
func validateBulkCreateRunRows(rows []domain.BulkCreateProductionRunParams) *apierror.APIError {
	var rowErrs apierror.RowErrors
	for i, run := range rows {
		if len(run.Batches) == 0 {
			rowErrs.AddValidation(i, fmt.Sprintf("production_runs[%d].batches", i), "at least one batch is required")
		}
		for j, b := range run.Batches {
			if _, err := decimal.NewFromString(b.QuantityValue); err != nil {
				rowErrs.AddValidation(i, fmt.Sprintf("production_runs[%d].batches[%d].quantity_value", i, j), fmt.Sprintf("quantity value %q is not a decimal", b.QuantityValue))
			}
			if b.SecondsValue != nil && b.SecondsUnit != nil {
				if _, err := decimal.NewFromString(*b.SecondsValue); err != nil {
					rowErrs.AddValidation(i, fmt.Sprintf("production_runs[%d].batches[%d].seconds_value", i, j), fmt.Sprintf("seconds value %q is not a decimal", *b.SecondsValue))
				}
			}
			if b.WasteValue != nil && b.WasteUnit != nil {
				if _, err := decimal.NewFromString(*b.WasteValue); err != nil {
					rowErrs.AddValidation(i, fmt.Sprintf("production_runs[%d].batches[%d].waste_value", i, j), fmt.Sprintf("waste value %q is not a decimal", *b.WasteValue))
				}
			}
		}
	}
	return rowErrs.Summary("production runs")
}

// resolveBulkCreateRunRows resolves responsible users and every fuzzy reference, then
// pre-generates the run and batch ids. The ids go into the resolved rows so the same
// values are stored on the job, returned in the 202 acknowledgment, and written by the
// worker — a redelivery converges on them.
func resolveBulkCreateRunRows(ctx context.Context, repos domain.RepoFactory, accountID string, rows []domain.BulkCreateProductionRunParams) ([]domain.BulkCreateProductionRunEventRun, *apierror.APIError) {
	// Resolve responsible users up front (account_user or user ids → account_user ids),
	// failing fast on the first unknown one.
	accountUserRepo := repos.NewAccountUserRepo()
	resolvedUserByInput := make(map[string]string)
	for i, run := range rows {
		if _, ok := resolvedUserByInput[run.ResponsibleUserID]; ok {
			continue
		}
		resolvedID, apiErr := accountUserRepo.ResolveAccountUserID(ctx, accountID, run.ResponsibleUserID)
		if apiErr != nil {
			return nil, apierror.NewValidationErrorWithParam(
				"The responsible user was not found in this account.",
				fmt.Sprintf("production_runs[%d].responsible_user_id", i),
			)
		}
		resolvedUserByInput[run.ResponsibleUserID] = resolvedID
	}

	identifiers, apiErr := resolveBulkRunIdentifiers(ctx, repos, accountID, rows)
	if apiErr != nil {
		return nil, apiErr
	}

	resolved := make([]domain.BulkCreateProductionRunEventRun, len(rows))
	for i, run := range rows {
		productionRunID, apiErr := id.GenID(id.ProductionRunIDPrefix, nil)
		if apiErr != nil {
			return nil, apiErr
		}
		eventRun := domain.BulkCreateProductionRunEventRun{
			ProductionRunID:   productionRunID,
			ResponsibleUserID: resolvedUserByInput[run.ResponsibleUserID],
			Batches:           make([]domain.BulkCreateProductionRunEventBatch, len(run.Batches)),
		}
		for j, b := range run.Batches {
			batchID, apiErr := id.GenID(id.BatchIDPrefix, nil)
			if apiErr != nil {
				return nil, apiErr
			}
			eventBatch := domain.BulkCreateProductionRunEventBatch{
				BatchID:        batchID,
				ItemID:         identifiers.itemIDByIdentifier[b.Item],
				QuantityValue:  b.QuantityValue,
				QuantityUnitID: identifiers.unitIDByIdentifier[b.QuantityUnit],
				SecondsValue:   b.SecondsValue,
				WasteValue:     b.WasteValue,
			}
			if b.SecondsValue != nil && b.SecondsUnit != nil {
				secondsUnitID := identifiers.unitIDByIdentifier[*b.SecondsUnit]
				eventBatch.SecondsUnitID = &secondsUnitID
			}
			if b.WasteValue != nil && b.WasteUnit != nil {
				wasteUnitID := identifiers.unitIDByIdentifier[*b.WasteUnit]
				eventBatch.WasteUnitID = &wasteUnitID
			}
			if b.ProductionStepID != nil && *b.ProductionStepID != "" {
				eventBatch.ProductionStepID = b.ProductionStepID
			}
			if b.ScanningStation != nil && *b.ScanningStation != (domain.ObjectIdentifier{}) {
				stationID := identifiers.stationIDByIdentifier[*b.ScanningStation]
				eventBatch.ScanningStationID = &stationID
			}
			eventRun.Batches[j] = eventBatch
		}
		resolved[i] = eventRun
	}
	return resolved, nil
}

// ExecuteBulkCreateProductionRuns performs the writes for an enqueued bulk create.
// Called by the bulk create consumer; exactly-once is provided by the message inbox.
func (s *productionRunSvcImpl) ExecuteBulkCreateProductionRuns(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkCreateSpec(), event)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// hands the export engine the plumbing it runs on.

// wires production runs into the export engine. This sheet is run and batch
// history; the import template is creation, so the two share no column list.
func (s *productionRunSvcImpl) exportSpec() exportSpec[*domain.ProductionRunExport, domain.ExportProductionRunsParams] {
	return exportSpec[*domain.ProductionRunExport, domain.ExportProductionRunsParams]{
		PermissionDomain: types.PermissionDomainProductionRuns,
		Name:             "Production Runs",
		Slug:             "production_runs",
		ResourceType:     constants.ObjectTypeProductionRun,
		Columns: []excel.ColumnSpec{
			{Header: "ID", Key: "id", Width: 24},
			{Header: "Number", Key: "number", Width: 18},
			{Header: "Responsible User", Key: "responsible_user", Width: 26},
			{Header: "Started At", Key: "started_at", Width: 16},
			{Header: "Completed At", Key: "completed_at", Width: 16},
			{Header: "Order ID", Key: "order_id", Width: 24},
			{Header: "Batch Item", Key: "batch_item", Width: 24},
			{Header: "Batch Quantity", Key: "batch_quantity", Width: 16},
			{Header: "Batch Unit", Key: "batch_unit", Width: 12},
			{Header: "Batch Department", Key: "batch_department", Width: 20},
			{Header: "Batch Machines", Key: "batch_machines", Width: 28},
			{Header: "Batch Scanned At", Key: "batch_scanned_at", Width: 18},
		},

		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportProductionRunsParams) ([]*domain.ProductionRunExport, *apierror.APIError) {
			filters.AccountID = accountID
			return repos.NewProductionRunRepo().Export(ctx, filters)
		},

		Expand: func(run *domain.ProductionRunExport) []excel.Row {
			parent := excel.Row{
				"id":               run.ID,
				"number":           run.Number,
				"responsible_user": run.ResponsibleUserName,
				"started_at":       excel.Date(run.StartedAt),
				"completed_at":     excel.Date(run.CompletedAt),
				"order_id":         excel.Str(run.OrderID),
			}

			children := make([]excel.Row, len(run.Batches))
			for i, batch := range run.Batches {
				children[i] = excel.Row{
					"batch_item":       batch.ItemSKU,
					"batch_quantity":   decimalCell(batch.QuantityValue),
					"batch_unit":       batch.QuantityUnit,
					"batch_department": excel.Str(batch.DepartmentName),
					"batch_machines":   excel.JoinNames(batch.MachineNames),
					"batch_scanned_at": excel.Date(batch.ScannedAt),
				}
			}

			return excel.Group(parent, children)
		},
	}
}

// accepts an export: records what to build on a job and returns it to poll.
func (s *productionRunSvcImpl) ExportProductionRuns(ctx context.Context, params domain.ExportProductionRunsParams) (*domain.Job, *apierror.APIError) {
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), params)
}

// renders the file an accepted export recorded
func (s *productionRunSvcImpl) BuildExportProductionRuns(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}

func derefStringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// builds one production-run result row: always a create, carrying the run's batch ids as
// the row's sub-resources
func runRowResult(index int, runID string, batchIDs []string) domain.RowResult {
	r := newRowResult(index, runID, true)
	r.SubResources = domain.NewSubResourceRefs(constants.ObjectTypeBatch, batchIDs)
	return r
}

// writeBulkCreateProductionRuns is the engine's Write hook: in one transaction it
// creates each run with the next sequential number and its pre-generated id, then its
// batches. On redelivery the pre-generated ids converge on the same rows.
func writeBulkCreateProductionRuns(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, runs []domain.BulkCreateProductionRunEventRun) (BulkWriteResult, *apierror.APIError) {
	prRepo := txRepos.NewProductionRunRepo()
	batchRepo := txRepos.NewBatchRepo()

	parseQuantity := func(value, unitID string) (domain.CreateQuantityParams, *apierror.APIError) {
		measure, err := decimal.NewFromString(value)
		if err != nil {
			return domain.CreateQuantityParams{}, apierror.NewValidationError(fmt.Sprintf("Quantity value %q is not a decimal.", value))
		}
		return domain.CreateQuantityParams{Measure: measure, UnitID: unitID}, nil
	}

	// Allocate once: the underlying read holds an account-wide FOR UPDATE lock for the whole batch.
	numbers, apiErr := prRepo.GetNextNumbers(txCtx, accountID, len(runs))
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	used := 0

	results := make([]domain.RowResult, 0, len(runs))
	var rowErrors []apierror.RowError

	for i, run := range runs {
		var batchIDs []string
		number := numbers[used]
		// Each run (and its batches) is created inside its own savepoint: a run that fails
		// rolls back only itself and its batches, and the batch continues.
		rowErr := sp.Run(txCtx, func(_ context.Context) *apierror.APIError {
			created, apiErr := prRepo.Create(txCtx, run.ProductionRunID, domain.CreateProductionRunParams{
				AccountID:         accountID,
				ResponsibleUserID: run.ResponsibleUserID,
			}, number)
			if apiErr != nil {
				return apiErr
			}

			if apiErr := audit.NewPublisher().Publish(txCtx, txRepos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeProductionRun,
				ResourceID:   created.ID,
				Changes:      audit.ComputeChanges(nil, created),
			}); apiErr != nil {
				return apiErr
			}

			batchIDs = make([]string, 0, len(run.Batches))
			for _, b := range run.Batches {
				quantity, apiErr := parseQuantity(b.QuantityValue, b.QuantityUnitID)
				if apiErr != nil {
					return apiErr
				}
				var seconds, waste *domain.CreateQuantityParams
				if b.SecondsValue != nil && b.SecondsUnitID != nil {
					q, apiErr := parseQuantity(*b.SecondsValue, *b.SecondsUnitID)
					if apiErr != nil {
						return apiErr
					}
					seconds = &q
				}
				if b.WasteValue != nil && b.WasteUnitID != nil {
					q, apiErr := parseQuantity(*b.WasteValue, *b.WasteUnitID)
					if apiErr != nil {
						return apiErr
					}
					waste = &q
				}

				if _, apiErr := batchRepo.Create(txCtx, b.BatchID, domain.CreateBatchParams{
					AccountID:         accountID,
					ItemID:            b.ItemID,
					Quantity:          quantity,
					Seconds:           seconds,
					Waste:             waste,
					ProductionStepID:  derefStringOrEmpty(b.ProductionStepID),
					ScanningStationID: derefStringOrEmpty(b.ScanningStationID),
				}); apiErr != nil {
					return apiErr
				}

				// Connect the batch to the production run, mirroring AddBatchesToProductionRun.
				if apiErr := prRepo.SetBatchProductionRunID(txCtx, accountID, b.BatchID, run.ProductionRunID); apiErr != nil {
					return apiErr
				}

				batchIDs = append(batchIDs, b.BatchID)
			}
			return nil
		})
		if rowErr != nil {
			rowErrors = append(rowErrors, apierror.NewRowError(i, rowErr))
			continue
		}
		// A rolled-back row wrote no number, so only a committed one consumes it.
		used++

		results = append(results, runRowResult(i, run.ProductionRunID, batchIDs))
	}

	return BulkWriteResult{Results: results, Errors: rowErrors, WrittenIDs: resultIDs(results)}, nil
}
