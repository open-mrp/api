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
func (s *productLineSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

// no duplicate name within the request
func validateBulkUpsertProductLineRows(rows []domain.UpsertProductLineParams) *apierror.APIError {
	nameInputSpace := make(map[string]struct{}, len(rows))
	var rowErrs apierror.RowErrors
	for i, pl := range rows {
		lower := strings.ToLower(pl.Name)
		if _, dup := nameInputSpace[lower]; dup {
			rowErrs.AddValidation(i, fmt.Sprintf("product_lines[%d].name", i), fmt.Sprintf("duplicate name %q in request", pl.Name))
		}
		nameInputSpace[lower] = struct{}{}
	}
	return rowErrs.Summary("product lines")
}

// resolves each row's unit group reference by id or name
func resolveBulkUpsertProductLineRows(ctx context.Context, repos domain.RepoFactory, accountID string, rows []domain.UpsertProductLineParams) ([]domain.ResolvedUpsertProductLineRow, *apierror.APIError) {
	ugIdentifiers := make([]*domain.ObjectIdentifier, len(rows))
	for i := range rows {
		ugIdentifiers[i] = &rows[i].UnitGroup
	}
	ugRepo := repos.NewUnitGroupRepo()
	ugIDByIdentifier, apiErr := resolveObjectIdentifiers(
		ctx, accountID, "product_lines", "unit_group", "unit group", "unit groups", ugIdentifiers,
		ugRepo.GetByIDs, ugRepo.FindByNames,
		func(g *domain.UnitGroupFull) string { return g.ID },
		func(g *domain.UnitGroupFull) string { return g.Name },
	)
	if apiErr != nil {
		return nil, apiErr
	}

	resolved := make([]domain.ResolvedUpsertProductLineRow, len(rows))
	for i, pl := range rows {
		resolved[i] = domain.ResolvedUpsertProductLineRow{
			Name:             pl.Name,
			UnitGroupID:      ugIDByIdentifier[pl.UnitGroup],
			CommissionPolicy: pl.CommissionPolicy,
			FreightPolicy:    pl.FreightPolicy,
		}
	}
	return resolved, nil
}

// matches each row against existing product lines by name and upserts it in its own savepoint
func writeBulkUpsertProductLines(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.ResolvedUpsertProductLineRow) (BulkWriteResult, *apierror.APIError) {
	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = strings.ToLower(row.Name)
	}

	existing, apiErr := txRepos.NewProductLineRepo().FindByNames(txCtx, accountID, names)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	// When a system and an account line share a name, the account-owned line is the upsert target.
	byName := make(map[string]*domain.ProductLineFull, len(existing))
	for _, pl := range existing {
		key := strings.ToLower(pl.Name)
		if current := byName[key]; current == nil || current.AccountID == nil {
			byName[key] = pl
		}
	}

	results := make([]domain.RowResult, 0, len(rows))
	var rowErrors []apierror.RowError
	for i := range rows {
		row := rows[i]

		var upsertedID string
		var isCreate bool
		rowErr := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
			old := byName[names[i]]
			id, apiErr := upsertProductLineInTx(spCtx, txRepos, accountID, row, old)
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

// creates or updates one product line inside an existing transaction. System and default product lines cannot be modified.
func upsertProductLineInTx(txCtx context.Context, txRepos domain.RepoFactory, accountID string, row domain.ResolvedUpsertProductLineRow, old *domain.ProductLineFull) (string, *apierror.APIError) {
	ctx, span := productLineSvcTracer.Start(txCtx, "service.product_line.upsert_in_tx")
	defer span.End()

	txRepo := txRepos.NewProductLineRepo()

	if old == nil {
		newID, apiErr := id.GenID(id.ProductLineIDPrefix, nil)
		if apiErr != nil {
			return "", apiErr
		}

		created, apiErr := txRepo.Create(ctx, newID, domain.CreateProductLineParams{
			AccountID:        accountID,
			Name:             row.Name,
			UnitGroupID:      row.UnitGroupID,
			CommissionPolicy: row.CommissionPolicy,
			FreightPolicy:    row.FreightPolicy,
		})
		if apiErr != nil {
			return "", apiErr
		}

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeProductLine,
			ResourceID:   created.ID,
			Changes:      changes,
		}); apiErr != nil {
			return "", apiErr
		}

		return created.ID, nil
	}

	if old.AccountID == nil || domain.IsDefaultProductLine(old.ID) {
		return "", apierror.NewValidationErrorWithParam(
			fmt.Sprintf("System product line %q cannot be modified.", row.Name), "name")
	}

	updated, apiErr := txRepo.Update(ctx, domain.UpdateProductLineParams{
		AccountID:        accountID,
		ProductLineID:    old.ID,
		Name:             &row.Name,
		UnitGroupID:      &row.UnitGroupID,
		CommissionPolicy: &row.CommissionPolicy,
		FreightPolicy:    &row.FreightPolicy,
	})
	if apiErr != nil {
		return "", apiErr
	}

	changes := audit.ComputeChanges(old, updated)
	if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeProductLine,
		ResourceID:   updated.ID,
		Changes:      changes,
	}); apiErr != nil {
		return "", apiErr
	}

	return updated.ID, nil
}

// wires product lines into the async bulk engine.
func (s *productLineSvcImpl) bulkUpsertSpec() bulkOperationSpec[domain.UpsertProductLineParams, domain.ResolvedUpsertProductLineRow] {
	return bulkOperationSpec[domain.UpsertProductLineParams, domain.ResolvedUpsertProductLineRow]{
		JobType:          constants.JobTypeBulkUpsert,
		ResourceType:     constants.ObjectTypeProductLine,
		RoutingKey:       messaging.BulkUpsertProductLines.RoutingKey(),
		PermissionDomain: types.PermissionDomainProductLines,
		Actions:          []types.Action{types.ActionCreate, types.ActionUpdate},
		EntityName:       "product lines",
		Validate:         validateBulkUpsertProductLineRows,
		Resolve:          resolveBulkUpsertProductLineRows,
		Write:            writeBulkUpsertProductLines,
	}
}

// it validates and resolves synchronously, records the resolved rows on a job, and returns that job to poll.
func (s *productLineSvcImpl) BulkUpsertProductLines(ctx context.Context, params domain.BulkUpsertProductLinesParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), params.ProductLines)
}

// performs the writes for an enqueued bulk upsert.
func (s *productLineSvcImpl) ExecuteBulkUpsertProductLines(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), event)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// lists the columns shared by the product line export and its import template.
// Unlike the other resources the export carries no ID column, so the two match.
var productLineTemplateColumns = []excel.ColumnSpec{
	{Header: "Name", Key: "name", Width: 30},
	{Header: "Unit Group", Key: "unit_group", Width: 25},
	{Header: "Commission Exempt", Key: "commission_exempt", Width: 18},
	{Header: "Freight Exempt", Key: "freight_exempt", Width: 18},
}

// hands the export engine the plumbing it runs on.

// wires product lines into the export engine.
func (s *productLineSvcImpl) exportSpec() exportSpec[*domain.ProductLineFull, domain.ExportProductLinesParams] {
	return exportSpec[*domain.ProductLineFull, domain.ExportProductLinesParams]{
		PermissionDomain: types.PermissionDomainProductLines,
		Name:             "Product Lines",
		Slug:             "product_lines",
		ResourceType:     constants.ObjectTypeProductLine,
		Columns:          productLineTemplateColumns,

		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportProductLinesParams) ([]*domain.ProductLineFull, *apierror.APIError) {
			filters.AccountID = accountID
			return repos.NewProductLineRepo().Export(ctx, filters)
		},

		Project: func(line *domain.ProductLineFull) excel.Row {
			unitGroup := ""
			if line.UnitGroup != nil {
				unitGroup = line.UnitGroup.Name
			}
			return excel.Row{
				"name":              line.Name,
				"unit_group":        unitGroup,
				"commission_exempt": yesNo(line.CommissionPolicy.ToBool()),
				"freight_exempt":    yesNo(line.FreightPolicy.ToBool()),
			}
		},
	}
}

// accepts an export: records what to build on a job and returns it to poll.
func (s *productLineSvcImpl) ExportProductLines(ctx context.Context, params domain.ExportProductLinesParams) (*domain.Job, *apierror.APIError) {
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), params)
}

// renders the file an accepted export recorded
func (s *productLineSvcImpl) BuildExportProductLines(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}
