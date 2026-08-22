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

// hands the async bulk engine the plumbing it runs on
func (s *itemCategorySvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

// runs the accept-phase structural checks: no duplicate name within the request
// (case-insensitive, matching how existing categories are found). No DB.
func validateBulkUpsertItemCategoryRows(rows []domain.UpsertItemCategoryParams) *apierror.APIError {
	nameInputSpace := make(map[string]struct{}, len(rows))
	var rowErrs apierror.RowErrors
	for i, ic := range rows {
		lower := strings.ToLower(ic.Name)
		if _, dup := nameInputSpace[lower]; dup {
			rowErrs.AddValidation(i, fmt.Sprintf("item_categories[%d].name", i), fmt.Sprintf("duplicate name %q in request", ic.Name))
		}
		nameInputSpace[lower] = struct{}{}
	}
	return rowErrs.Summary("item categories")
}

// resolves each row's unit group reference by id or name. Property names are left as
// written: they are found-or-created by the write, so there is nothing to fail on here.
func resolveBulkUpsertItemCategoryRows(ctx context.Context, repos domain.RepoFactory, accountID string, rows []domain.UpsertItemCategoryParams) ([]domain.ResolvedUpsertItemCategoryRow, *apierror.APIError) {
	ugIdentifiers := make([]*domain.ObjectIdentifier, len(rows))
	for i := range rows {
		ugIdentifiers[i] = &rows[i].UnitGroup
	}
	ugRepo := repos.NewUnitGroupRepo()
	ugIDByIdentifier, apiErr := resolveObjectIdentifiers(
		ctx, accountID, "item_categories", "unit_group", "unit group", "unit groups", ugIdentifiers,
		ugRepo.GetByIDs, ugRepo.FindByNames,
		func(g *domain.UnitGroupFull) string { return g.ID },
		func(g *domain.UnitGroupFull) string { return g.Name },
	)
	if apiErr != nil {
		return nil, apiErr
	}

	resolved := make([]domain.ResolvedUpsertItemCategoryRow, len(rows))
	for i, ic := range rows {
		resolved[i] = domain.ResolvedUpsertItemCategoryRow{
			Name:                 ic.Name,
			Notes:                ic.Notes,
			ItemCategoryTypeCode: ic.ItemCategoryTypeCode,
			UnitGroupID:          ugIDByIdentifier[ic.UnitGroup],
			PropertyNames:        ic.PropertyNames,
		}
	}
	return resolved, nil
}

// the engine's Write hook: it matches each row against existing categories by name and
// upserts it in its own savepoint, so a bad row rolls back only itself.
func writeBulkUpsertItemCategories(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.ResolvedUpsertItemCategoryRow) (BulkWriteResult, *apierror.APIError) {
	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = strings.ToLower(row.Name)
	}

	icRepo := txRepos.NewItemCategoryRepo()
	existing, apiErr := icRepo.FindByNames(txCtx, accountID, names)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}

	// A system category and an account-owned one can share a name; prefer the account-owned
	// row so the upsert targets it. A lone system match is rejected per-row by the upsert.
	byName := make(map[string]*domain.ItemCategoryFull, len(existing))
	for _, ic := range existing {
		key := strings.ToLower(ic.Name)
		if current := byName[key]; current == nil || current.AccountID == nil {
			byName[key] = ic
		}
	}

	// One query for every unit group the batch touches — the incoming groups and the
	// current group of each matched row — so the same-unit-type check needs no per-row read.
	ugIDSet := make(map[string]struct{}, len(rows))
	for i, row := range rows {
		ugIDSet[row.UnitGroupID] = struct{}{}
		if old := byName[names[i]]; old != nil {
			ugIDSet[old.UnitGroupID] = struct{}{}
		}
	}
	ugIDs := make([]string, 0, len(ugIDSet))
	for ugID := range ugIDSet {
		ugIDs = append(ugIDs, ugID)
	}
	unitGroupTypes, apiErr := txRepos.NewUnitGroupRepo().GetTypesByIDs(txCtx, accountID, ugIDs)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}

	propertyByName, apiErr := findOrCreateRowProperties(txCtx, txRepos, accountID, rows)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}

	results := make([]domain.RowResult, 0, len(rows))
	var rowErrors []apierror.RowError
	for i := range rows {
		row := rows[i]

		var upsertedID string
		var isCreate bool
		rowErr := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
			old := byName[names[i]]
			id, apiErr := upsertItemCategoryInTx(spCtx, txRepos, accountID, row, old, unitGroupTypes)
			if apiErr != nil {
				return apiErr
			}
			for _, propName := range row.PropertyNames {
				prop, ok := propertyByName[strings.ToLower(propName)]
				if !ok {
					continue
				}
				if apiErr := icRepo.UpsertProperty(spCtx, id, prop.ID); apiErr != nil {
					return apiErr
				}
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

// finds-or-creates every property name the batch mentions, keyed by lowercased name.
// Casing comes from the first row that supplies each name.
func findOrCreateRowProperties(txCtx context.Context, txRepos domain.RepoFactory, accountID string, rows []domain.ResolvedUpsertItemCategoryRow) (map[string]*domain.Property, *apierror.APIError) {
	nameCasing := make(map[string]string)
	for _, row := range rows {
		for _, name := range row.PropertyNames {
			lower := strings.ToLower(name)
			if _, ok := nameCasing[lower]; !ok {
				nameCasing[lower] = name
			}
		}
	}
	byName, _, apiErr := findOrCreatePropertiesInTx(txCtx, txRepos, accountID, nameCasing)
	return byName, apiErr
}

// creates or updates one item category inside an existing transaction. A system category
// cannot be modified, and a unit group move is confined to the same unit type.
func upsertItemCategoryInTx(txCtx context.Context, txRepos domain.RepoFactory, accountID string, row domain.ResolvedUpsertItemCategoryRow, old *domain.ItemCategoryFull, unitGroupTypes map[string]string) (string, *apierror.APIError) {
	ctx, span := itemCategorySvcTracer.Start(txCtx, "service.item_category.upsert_in_tx")
	defer span.End()

	txRepo := txRepos.NewItemCategoryRepo()

	if old == nil {
		newID, apiErr := id.GenID(id.ItemCategoryIDPrefix, nil)
		if apiErr != nil {
			return "", apiErr
		}

		created, apiErr := txRepo.Create(ctx, newID, domain.CreateItemCategoryParams{
			AccountID:            accountID,
			Name:                 row.Name,
			Notes:                row.Notes,
			ItemCategoryTypeCode: row.ItemCategoryTypeCode,
			UnitGroupID:          row.UnitGroupID,
		})
		if apiErr != nil {
			return "", apiErr
		}

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeItemCategory,
			ResourceID:   created.ID,
			Changes:      changes,
		}); apiErr != nil {
			return "", apiErr
		}

		return created.ID, nil
	}

	if old.AccountID == nil {
		return "", apierror.NewValidationErrorWithParam(
			fmt.Sprintf("System item category %q cannot be modified.", row.Name), "name")
	}

	// The incoming group is resolved so it is in the type map; a miss on the stored one is
	// an invariant violation, not bad input.
	if old.UnitGroupID != row.UnitGroupID {
		currentType, currentExists := unitGroupTypes[old.UnitGroupID]
		if !currentExists {
			return "", apierror.NewInvariantViolationError(
				fmt.Sprintf("Current unit group %q for item category %q not found.", old.UnitGroupID, row.Name))
		}
		if newType := unitGroupTypes[row.UnitGroupID]; currentType != newType {
			return "", apierror.NewValidationErrorWithParam(
				fmt.Sprintf("The new unit group must have the same unit type as the current unit group (%s).", currentType), "unit_group")
		}
	}

	updated, apiErr := txRepo.UpdateWithUnitGroup(ctx, domain.UpdateItemCategoryWithUnitGroupParams{
		AccountID:      accountID,
		ItemCategoryID: old.ID,
		Name:           &row.Name,
		Notes:          row.Notes,
		UnitGroupID:    row.UnitGroupID,
	})
	if apiErr != nil {
		return "", apiErr
	}

	changes := audit.ComputeChanges(old, updated)
	if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeItemCategory,
		ResourceID:   updated.ID,
		Changes:      changes,
	}); apiErr != nil {
		return "", apiErr
	}

	return updated.ID, nil
}

// wires item categories into the async bulk engine
func (s *itemCategorySvcImpl) bulkUpsertSpec() bulkOperationSpec[domain.UpsertItemCategoryParams, domain.ResolvedUpsertItemCategoryRow] {
	return bulkOperationSpec[domain.UpsertItemCategoryParams, domain.ResolvedUpsertItemCategoryRow]{
		JobType:          constants.JobTypeBulkUpsert,
		ResourceType:     constants.ObjectTypeItemCategory,
		RoutingKey:       messaging.BulkUpsertItemCategories.RoutingKey(),
		PermissionDomain: types.PermissionDomainCategories,
		Actions:          []types.Action{types.ActionCreate, types.ActionUpdate},
		EntityName:       "item categories",
		Validate:         validateBulkUpsertItemCategoryRows,
		Resolve:          resolveBulkUpsertItemCategoryRows,
		Write:            writeBulkUpsertItemCategories,
	}
}

// accepts a bulk upsert: validates and resolves synchronously, records the resolved rows
// on a job, and returns that job to poll. ExecuteBulkUpsertItemCategories writes.
func (s *itemCategorySvcImpl) BulkUpsertItemCategories(ctx context.Context, params domain.BulkUpsertItemCategoriesParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), params.ItemCategories)
}

// performs the writes for an enqueued bulk upsert. Delivery is at-least-once; the inbox
// de-dup and the engine's terminal-job guard make it effectively-once.
func (s *itemCategorySvcImpl) ExecuteBulkUpsertItemCategories(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), event)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// hands the export engine the plumbing it runs on.

// wires item categories into the export engine.
func (s *itemCategorySvcImpl) exportSpec() exportSpec[*domain.ItemCategoryFull, domain.ExportItemCategoriesParams] {
	return exportSpec[*domain.ItemCategoryFull, domain.ExportItemCategoriesParams]{
		PermissionDomain: types.PermissionDomainCategories,
		Name:             "Categories",
		Slug:             "item_categories",
		ResourceType:     constants.ObjectTypeItemCategory,
		Columns: []excel.ColumnSpec{
			{Header: "ID", Key: "id", Width: 28},
			{Header: "Name", Key: "name", Width: 28},
			{Header: "Type", Key: "type", Width: 22},
			{Header: "Unit Group", Key: "unit_group", Width: 24},
			{
				Header: "Properties", Key: "properties", Width: 40,
				Note: `OPTIONAL. Enter property names separated by semicolons (;). Names are created if they do not already exist. e.g. "Color; Size; Weight"`,
			},
			{Header: "Notes", Key: "notes", Width: 40},
		},

		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportItemCategoriesParams) ([]*domain.ItemCategoryFull, *apierror.APIError) {
			filters.AccountID = accountID
			return repos.NewItemCategoryRepo().Export(ctx, filters)
		},

		Project: func(category *domain.ItemCategoryFull) excel.Row {
			properties := make([]string, len(category.Properties))
			for i, p := range category.Properties {
				properties[i] = p.Name
			}
			unitGroup := ""
			if category.UnitGroup != nil {
				unitGroup = category.UnitGroup.Name
			}
			return excel.Row{
				"id":         category.ID,
				"name":       category.Name,
				"type":       category.ItemCategoryTypeCode,
				"unit_group": unitGroup,
				"properties": excel.JoinNames(properties),
				"notes":      excel.Str(category.Notes),
			}
		},
	}
}

// accepts an export: records what to build on a job and returns it to poll.
func (s *itemCategorySvcImpl) ExportItemCategories(ctx context.Context, params domain.ExportItemCategoriesParams) (*domain.Job, *apierror.APIError) {
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), params)
}

// renders the file an accepted export recorded
func (s *itemCategorySvcImpl) BuildExportItemCategories(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}
