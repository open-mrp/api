package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/excel"
	"github.com/augno/api/shared/messaging"
)

// hands the async bulk engine the plumbing it runs on
func (s *partSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

// runs the accept-phase structural checks: no duplicate SKU within the request. No DB.
func validateBulkUpsertPartRows(rows []domain.UpsertPartParams) *apierror.APIError {
	skuSpace := make(map[string]struct{}, len(rows))
	var rowErrs apierror.RowErrors
	for i, p := range rows {
		if _, dup := skuSpace[p.SKU]; dup {
			rowErrs.AddValidation(i, fmt.Sprintf("parts[%d].sku", i), fmt.Sprintf("duplicate SKU %q in request", p.SKU))
		}
		skuSpace[p.SKU] = struct{}{}
	}
	return rowErrs.Summary("parts")
}

// resolves each row's category by id or name and checks its type and base unit. Category
// is create-only but required on every row, so resolving all of them needs no SKU lookup.
func resolveBulkUpsertPartRows(ctx context.Context, repos domain.RepoFactory, accountID string, rows []domain.UpsertPartParams) ([]domain.ResolvedUpsertPartRow, *apierror.APIError) {
	catIdentifiers := make([]*domain.ObjectIdentifier, len(rows))
	for i := range rows {
		catIdentifiers[i] = &rows[i].Category
	}
	catRepo := repos.NewItemCategoryRepo()
	catIDByIdentifier, apiErr := resolveObjectIdentifiers(
		ctx, accountID, "parts", "category", "category", "categories", catIdentifiers,
		catRepo.GetByIDs, catRepo.FindByNames,
		func(c *domain.ItemCategoryFull) string { return c.ID },
		func(c *domain.ItemCategoryFull) string { return c.Name },
	)
	if apiErr != nil {
		return nil, apiErr
	}

	catRows := make([]bulkCategoryRow, len(rows))
	for i, p := range rows {
		catRows[i] = bulkCategoryRow{Index: i, CategoryID: catIDByIdentifier[p.Category]}
	}
	if apiErr := validateBulkCreateCategoriesInTx(ctx, repos, "parts", "category", string(constants.ItemTypeCodePart), catRows); apiErr != nil {
		return nil, apiErr
	}

	resolved := make([]domain.ResolvedUpsertPartRow, len(rows))
	for i, p := range rows {
		resolved[i] = domain.ResolvedUpsertPartRow{
			SKU:         p.SKU,
			Description: p.Description,
			Notes:       p.Notes,
			CategoryID:  catIDByIdentifier[p.Category],
			UnitPrice:   p.UnitPrice,
			UnitCost:    p.UnitCost,
			Properties:  p.Properties,
		}
	}
	return resolved, nil
}

// the engine's Write hook: it matches each row against existing parts by SKU and upserts
// it in its own savepoint, so a bad row rolls back only itself.
func (s *partSvcImpl) writeBulkUpsertParts(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.ResolvedUpsertPartRow) (BulkWriteResult, *apierror.APIError) {
	// createPartInTx/updatePartInTx read their repos off the receiver, so bind a service to
	// the job's transaction — otherwise the row writes land outside it.
	txSvc := &partSvcImpl{repos: txRepos, mediatorFactory: s.mediatorFactory, jobSvcFactory: s.jobSvcFactory, txManager: s.txManager}

	skus := make([]string, len(rows))
	var allProps []domain.UpsertItemPropertyParams
	for i, row := range rows {
		skus[i] = row.SKU
		allProps = append(allProps, row.Properties...)
	}

	attrResolver, propIDByName, apiErr := resolvePropertyAttributesInTx(txCtx, txRepos, accountID, allProps)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}

	existing, apiErr := txRepos.NewPartRepo().FindBySKUs(txCtx, accountID, skus)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	bySKU := make(map[string]*domain.PartSKUMatch, len(existing))
	for _, m := range existing {
		bySKU[m.SKU] = m
	}

	results := make([]domain.RowResult, 0, len(rows))
	var rowErrors []apierror.RowError
	for i := range rows {
		row := rows[i]

		var upsertedID string
		var isCreate bool
		rowErr := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
			attrIDs := attributeIDsForProperties(row.Properties, attrResolver)

			if old := bySKU[row.SKU]; old != nil {
				if _, apiErr := txSvc.updatePartInTx(spCtx, domain.UpdatePartParams{
					AccountID:   accountID,
					PartID:      old.PartID,
					Description: clearableFromPtr(row.Description),
					Notes:       clearableFromPtr(row.Notes),
				}); apiErr != nil {
					return apiErr
				}
				if apiErr := applyItemRatesInTx(spCtx, txRepos, old.UnitValueRateID, old.UnitCostRateID, row.UnitPrice, row.UnitCost); apiErr != nil {
					return apiErr
				}
				// Precedes the attribute link, which rejects an attribute whose property the category does not carry yet.
				if apiErr := linkRowPropertiesToCategoryInTx(spCtx, txRepos, old.CategoryID, row.Properties, propIDByName); apiErr != nil {
					return apiErr
				}
				if apiErr := attachItemAttributesInTx(spCtx, txRepos, accountID, old.CategoryID, old.ItemID, attrIDs); apiErr != nil {
					return apiErr
				}
				upsertedID = old.PartID
				return nil
			}

			// Precedes the attribute link, which rejects an attribute whose property the category does not carry yet.
			if apiErr := linkRowPropertiesToCategoryInTx(spCtx, txRepos, row.CategoryID, row.Properties, propIDByName); apiErr != nil {
				return apiErr
			}
			created, apiErr := txSvc.createPartInTx(spCtx, domain.CreatePartParams{
				AccountID:    accountID,
				SKU:          row.SKU,
				Description:  row.Description,
				Notes:        row.Notes,
				CategoryID:   row.CategoryID,
				UnitPrice:    row.UnitPrice,
				UnitCost:     row.UnitCost,
				AttributeIDs: attrIDs,
			})
			if apiErr != nil {
				return apiErr
			}
			upsertedID = created.ID
			isCreate = true
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

// wires parts into the async bulk engine
func (s *partSvcImpl) bulkUpsertSpec() bulkOperationSpec[domain.UpsertPartParams, domain.ResolvedUpsertPartRow] {
	return bulkOperationSpec[domain.UpsertPartParams, domain.ResolvedUpsertPartRow]{
		JobType:          constants.JobTypeBulkUpsert,
		ResourceType:     constants.ObjectTypePart,
		RoutingKey:       messaging.BulkUpsertParts.RoutingKey(),
		PermissionDomain: types.PermissionDomainParts,
		Actions:          []types.Action{types.ActionCreate, types.ActionUpdate},
		EntityName:       "parts",
		Validate:         validateBulkUpsertPartRows,
		Resolve:          resolveBulkUpsertPartRows,
		Write:            s.writeBulkUpsertParts,
	}
}

// accepts a bulk upsert: validates and resolves synchronously, records the resolved rows
// on a job, and returns that job to poll. ExecuteBulkUpsertParts writes.
func (s *partSvcImpl) BulkUpsertParts(ctx context.Context, params domain.BulkUpsertPartsParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), params.Parts)
}

// performs the writes for an enqueued bulk upsert. Delivery is at-least-once; the inbox
// de-dup and the engine's terminal-job guard make it effectively-once.
func (s *partSvcImpl) ExecuteBulkUpsertParts(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), event)
}

// --- Export ---

// wires parts into the export engine
func (s *partSvcImpl) exportSpec() exportSpec[*domain.Part, domain.ExportPartsParams] {
	return exportSpec[*domain.Part, domain.ExportPartsParams]{
		CheckPermission: checkPartReadPermission,
		ExternalAccess:  externalDirect,
		Name:            "Parts",
		Slug:            "parts",
		ResourceType:    constants.ObjectTypePart,

		ColumnsFor: func(parts []*domain.Part) []excel.ColumnSpec {
			return append(itemBaseColumns(), itemPropertyColumns(partItems(parts))...)
		},

		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportPartsParams) ([]*domain.Part, *apierror.APIError) {
			filters.AccountID = accountID
			return repos.NewPartRepo().Export(ctx, filters)
		},

		Project: func(part *domain.Part) excel.Row {
			row := excel.Row{}
			addItemBaseCells(row, part.ID, part.Item)
			return row
		},
	}
}

// accepts an export: records the filters on a job and returns it to poll.
func (s *partSvcImpl) ExportParts(ctx context.Context, params domain.ExportPartsParams) (*domain.Job, *apierror.APIError) {
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), params)
}

// renders the file an accepted export recorded
func (s *partSvcImpl) BuildExportParts(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}

// lifts the items out of a part list, for the shared property-column helpers
func partItems(parts []*domain.Part) []*domain.Item {
	items := make([]*domain.Item, 0, len(parts))
	for _, part := range parts {
		items = append(items, part.Item)
	}
	return items
}
