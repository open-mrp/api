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
func (s *materialSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

// runs the accept-phase structural checks: no duplicate SKU within the request. No DB.
func validateBulkUpsertMaterialRows(rows []domain.UpsertMaterialParams) *apierror.APIError {
	skuSpace := make(map[string]struct{}, len(rows))
	var rowErrs apierror.RowErrors
	for i, m := range rows {
		if _, dup := skuSpace[m.SKU]; dup {
			rowErrs.AddValidation(i, fmt.Sprintf("materials[%d].sku", i), fmt.Sprintf("duplicate SKU %q in request", m.SKU))
		}
		skuSpace[m.SKU] = struct{}{}
	}
	return rowErrs.Summary("materials")
}

// resolves each row's category by id or name and checks its type and base unit. Category
// is create-only but required on every row, so resolving all of them needs no SKU lookup.
func resolveBulkUpsertMaterialRows(ctx context.Context, repos domain.RepoFactory, accountID string, rows []domain.UpsertMaterialParams) ([]domain.ResolvedUpsertMaterialRow, *apierror.APIError) {
	catIdentifiers := make([]*domain.ObjectIdentifier, len(rows))
	for i := range rows {
		catIdentifiers[i] = &rows[i].Category
	}
	catRepo := repos.NewItemCategoryRepo()
	catIDByIdentifier, apiErr := resolveObjectIdentifiers(
		ctx, accountID, "materials", "category", "category", "categories", catIdentifiers,
		catRepo.GetByIDs, catRepo.FindByNames,
		func(c *domain.ItemCategoryFull) string { return c.ID },
		func(c *domain.ItemCategoryFull) string { return c.Name },
	)
	if apiErr != nil {
		return nil, apiErr
	}

	catRows := make([]bulkCategoryRow, len(rows))
	for i, m := range rows {
		catRows[i] = bulkCategoryRow{Index: i, CategoryID: catIDByIdentifier[m.Category]}
	}
	if apiErr := validateBulkCreateCategoriesInTx(ctx, repos, "materials", "category", string(constants.ItemTypeCodeMaterial), catRows); apiErr != nil {
		return nil, apiErr
	}

	resolved := make([]domain.ResolvedUpsertMaterialRow, len(rows))
	for i, m := range rows {
		resolved[i] = domain.ResolvedUpsertMaterialRow{
			SKU:         m.SKU,
			Description: m.Description,
			Notes:       m.Notes,
			CategoryID:  catIDByIdentifier[m.Category],
			OrderPoint:  m.OrderPoint,
			LeadTime:    m.LeadTime,
			UnitPrice:   m.UnitPrice,
			UnitCost:    m.UnitCost,
			Properties:  m.Properties,
		}
	}
	return resolved, nil
}

// the engine's Write hook: it matches each row against existing materials by SKU and
// upserts it in its own savepoint, so a bad row rolls back only itself.
func (s *materialSvcImpl) writeBulkUpsertMaterials(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.ResolvedUpsertMaterialRow) (BulkWriteResult, *apierror.APIError) {
	// createMaterialInTx/updateMaterialInTx read their repos off the receiver, so bind a
	// service to the job's transaction — otherwise the row writes land outside it.
	txSvc := &materialSvcImpl{repos: txRepos, mediatorFactory: s.mediatorFactory, jobSvcFactory: s.jobSvcFactory, txManager: s.txManager}

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

	existing, apiErr := txRepos.NewMaterialRepo().FindBySKUs(txCtx, accountID, skus)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	bySKU := make(map[string]*domain.MaterialSKUMatch, len(existing))
	for _, mm := range existing {
		bySKU[mm.SKU] = mm
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
				if _, apiErr := txSvc.updateMaterialInTx(spCtx, domain.UpdateMaterialParams{
					AccountID:         accountID,
					MaterialID:        old.MaterialID,
					Description:       row.Description,
					UpdateDescription: row.Description != nil,
					Notes:             row.Notes,
					UpdateNotes:       row.Notes != nil,
					OrderPoint:        row.OrderPoint,
					LeadTime:          row.LeadTime,
				}); apiErr != nil {
					return apiErr
				}
				if apiErr := applyItemRatesInTx(spCtx, txRepos, old.UnitValueRateID, old.UnitCostRateID, row.UnitPrice, row.UnitCost); apiErr != nil {
					return apiErr
				}
				if apiErr := attachItemAttributesInTx(spCtx, txRepos, accountID, old.ItemID, attrIDs); apiErr != nil {
					return apiErr
				}
				if apiErr := linkRowPropertiesToCategoryInTx(spCtx, txRepos, old.CategoryID, row.Properties, propIDByName); apiErr != nil {
					return apiErr
				}
				upsertedID = old.MaterialID
				return nil
			}

			created, apiErr := txSvc.createMaterialInTx(spCtx, domain.CreateMaterialParams{
				AccountID:    accountID,
				SKU:          row.SKU,
				Description:  row.Description,
				Notes:        row.Notes,
				CategoryID:   row.CategoryID,
				OrderPoint:   row.OrderPoint,
				LeadTime:     row.LeadTime,
				UnitPrice:    row.UnitPrice,
				UnitCost:     row.UnitCost,
				AttributeIDs: attrIDs,
			})
			if apiErr != nil {
				return apiErr
			}
			if apiErr := linkRowPropertiesToCategoryInTx(spCtx, txRepos, row.CategoryID, row.Properties, propIDByName); apiErr != nil {
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

// wires materials into the async bulk engine
func (s *materialSvcImpl) bulkUpsertSpec() bulkOperationSpec[domain.UpsertMaterialParams, domain.ResolvedUpsertMaterialRow] {
	return bulkOperationSpec[domain.UpsertMaterialParams, domain.ResolvedUpsertMaterialRow]{
		JobType:          constants.JobTypeBulkUpsert,
		RoutingKey:       messaging.BulkUpsertMaterials.RoutingKey(),
		PermissionDomain: types.PermissionDomainMaterials,
		Actions:          []types.Action{types.ActionCreate, types.ActionUpdate},
		EntityName:       "materials",
		Validate:         validateBulkUpsertMaterialRows,
		Resolve:          resolveBulkUpsertMaterialRows,
		Write:            s.writeBulkUpsertMaterials,
	}
}

// accepts a bulk upsert: validates and resolves synchronously, records the resolved rows
// on a job, and returns that job to poll. ExecuteBulkUpsertMaterials writes.
func (s *materialSvcImpl) BulkUpsertMaterials(ctx context.Context, params domain.BulkUpsertMaterialsParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), params.Materials)
}

// performs the writes for an enqueued bulk upsert. Delivery is at-least-once; the inbox
// de-dup and the engine's terminal-job guard make it effectively-once.
func (s *materialSvcImpl) ExecuteBulkUpsertMaterials(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), event)
}

// --- Export ---

// wires materials into the export engine
func (s *materialSvcImpl) exportSpec() exportSpec[*domain.Material, domain.ExportMaterialsParams] {
	return exportSpec[*domain.Material, domain.ExportMaterialsParams]{
		CheckPermission: checkMaterialReadPermission,
		ExternalAccess:  externalDirect,
		Name:            "Materials",
		Slug:            "materials",

		ColumnsFor: func(materials []*domain.Material) []excel.ColumnSpec {
			base := itemBaseColumns(
				excel.ColumnSpec{Header: "Order Point", Key: "order_point", Width: 14},
				excel.ColumnSpec{Header: "Lead Time", Key: "lead_time", Width: 14},
			)
			return append(base, itemPropertyColumns(materialItems(materials))...)
		},

		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportMaterialsParams) ([]*domain.Material, *apierror.APIError) {
			filters.AccountID = accountID
			return repos.NewMaterialRepo().Export(ctx, filters)
		},

		Project: func(material *domain.Material) excel.Row {
			row := excel.Row{
				"order_point": quantityValue(material.OrderPoint),
				"lead_time":   quantityValue(material.LeadTime),
			}
			addItemBaseCells(row, material.ID, material.Item)
			return row
		},
	}
}

// accepts an export: records the filters on a job and returns it to poll.
func (s *materialSvcImpl) ExportMaterials(ctx context.Context, params domain.ExportMaterialsParams) (*domain.Job, *apierror.APIError) {
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), params)
}

// renders the file an accepted export recorded
func (s *materialSvcImpl) BuildExportMaterials(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}

// lifts the items out of a material list, for the shared property-column helpers
func materialItems(materials []*domain.Material) []*domain.Item {
	items := make([]*domain.Item, 0, len(materials))
	for _, material := range materials {
		items = append(items, material.Item)
	}
	return items
}
