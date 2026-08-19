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
func (s *productSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

// runs the accept-phase structural checks: no duplicate SKU within the request. No DB.
func validateBulkUpsertProductRows(rows []domain.UpsertProductParams) *apierror.APIError {
	skuSpace := make(map[string]struct{}, len(rows))
	var rowErrs apierror.RowErrors
	for i, p := range rows {
		if _, dup := skuSpace[p.SKU]; dup {
			rowErrs.AddValidation(i, fmt.Sprintf("products[%d].sku", i), fmt.Sprintf("duplicate SKU %q in request", p.SKU))
		}
		skuSpace[p.SKU] = struct{}{}
	}
	return rowErrs.Summary("products")
}

// resolves each row's category and optional product line, checking category type and base
// unit. Category is required on every row, so this needs no SKU lookup to spot the creates.
func resolveBulkUpsertProductRows(ctx context.Context, repos domain.RepoFactory, accountID string, rows []domain.UpsertProductParams) ([]domain.ResolvedUpsertProductRow, *apierror.APIError) {
	catIdentifiers := make([]*domain.ObjectIdentifier, len(rows))
	plIdentifiers := make([]*domain.ObjectIdentifier, len(rows))
	for i := range rows {
		catIdentifiers[i] = &rows[i].Category
		plIdentifiers[i] = rows[i].ProductLine // nil when the row omits it
	}

	catRepo := repos.NewItemCategoryRepo()
	catIDByIdentifier, apiErr := resolveObjectIdentifiers(
		ctx, accountID, "products", "category", "category", "categories", catIdentifiers,
		catRepo.GetByIDs, catRepo.FindByNames,
		func(c *domain.ItemCategoryFull) string { return c.ID },
		func(c *domain.ItemCategoryFull) string { return c.Name },
	)
	if apiErr != nil {
		return nil, apiErr
	}

	plRepo := repos.NewProductLineRepo()
	plIDByIdentifier, apiErr := resolveObjectIdentifiers(
		ctx, accountID, "products", "product_line", "product line", "product lines", plIdentifiers,
		plRepo.GetByIDs, plRepo.FindByNames,
		func(pl *domain.ProductLineFull) string { return pl.ID },
		func(pl *domain.ProductLineFull) string { return pl.Name },
	)
	if apiErr != nil {
		return nil, apiErr
	}

	catRows := make([]bulkCategoryRow, len(rows))
	for i, p := range rows {
		catRows[i] = bulkCategoryRow{Index: i, CategoryID: catIDByIdentifier[p.Category]}
	}
	if apiErr := validateBulkCreateCategoriesInTx(ctx, repos, "products", "category", string(constants.ItemTypeCodeProduct), catRows); apiErr != nil {
		return nil, apiErr
	}

	resolved := make([]domain.ResolvedUpsertProductRow, len(rows))
	for i, p := range rows {
		var productLineID *string
		if p.ProductLine != nil {
			id := plIDByIdentifier[*p.ProductLine]
			productLineID = &id
		}
		resolved[i] = domain.ResolvedUpsertProductRow{
			SKU:             p.SKU,
			ProductTypeCode: p.ProductTypeCode,
			Description:     p.Description,
			Notes:           p.Notes,
			CategoryID:      catIDByIdentifier[p.Category],
			ProductLineID:   productLineID,
			IsPortalReady:   p.IsPortalReady,
			UnitPrice:       p.UnitPrice,
			UnitCost:        p.UnitCost,
			Properties:      p.Properties,
		}
	}
	return resolved, nil
}

// the engine's Write hook: it matches each row against existing products by SKU and
// upserts it in its own savepoint, so a bad row rolls back only itself.
func (s *productSvcImpl) writeBulkUpsertProducts(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.ResolvedUpsertProductRow) (BulkWriteResult, *apierror.APIError) {
	// createProductInTx/updateProductInTx read their repos off the receiver, so bind a
	// service to the job's transaction — otherwise the row writes land outside it.
	txSvc := &productSvcImpl{repos: txRepos, mediatorFactory: s.mediatorFactory, jobSvcFactory: s.jobSvcFactory, txManager: s.txManager}

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

	existing, apiErr := txRepos.NewProductRepo().FindBySKUs(txCtx, accountID, skus)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	bySKU := make(map[string]*domain.ProductSKUMatch, len(existing))
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
				if _, apiErr := txSvc.updateProductInTx(spCtx, domain.UpdateProductParams{
					AccountID:     accountID,
					ProductID:     old.ProductID,
					Description:   clearableFromPtr(row.Description),
					Notes:         clearableFromPtr(row.Notes),
					IsPortalReady: row.IsPortalReady,
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
				upsertedID = old.ProductID
				return nil
			}

			productType := row.ProductTypeCode
			if productType == "" {
				productType = "sale"
			}
			isPortalReady := false
			if row.IsPortalReady != nil {
				isPortalReady = *row.IsPortalReady
			}
			// Precedes the attribute link, which rejects an attribute whose property the category does not carry yet.
			if apiErr := linkRowPropertiesToCategoryInTx(spCtx, txRepos, row.CategoryID, row.Properties, propIDByName); apiErr != nil {
				return apiErr
			}
			created, apiErr := txSvc.createProductInTx(spCtx, domain.CreateProductParams{
				AccountID:       accountID,
				SKU:             row.SKU,
				ProductTypeCode: productType,
				Description:     row.Description,
				Notes:           row.Notes,
				CategoryID:      row.CategoryID,
				ProductLineID:   row.ProductLineID,
				IsPortalReady:   isPortalReady,
				UnitPrice:       row.UnitPrice,
				UnitCost:        row.UnitCost,
				AttributeIDs:    attrIDs,
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

// wires products into the async bulk engine
func (s *productSvcImpl) bulkUpsertSpec() bulkOperationSpec[domain.UpsertProductParams, domain.ResolvedUpsertProductRow] {
	return bulkOperationSpec[domain.UpsertProductParams, domain.ResolvedUpsertProductRow]{
		JobType:          constants.JobTypeBulkUpsert,
		ResourceType:     constants.ObjectTypeProduct,
		RoutingKey:       messaging.BulkUpsertProducts.RoutingKey(),
		PermissionDomain: types.PermissionDomainItems,
		Actions:          []types.Action{types.ActionCreate, types.ActionUpdate},
		EntityName:       "products",
		Validate:         validateBulkUpsertProductRows,
		Resolve:          resolveBulkUpsertProductRows,
		Write:            s.writeBulkUpsertProducts,
	}
}

// accepts a bulk upsert: validates and resolves synchronously, records the resolved rows
// on a job, and returns that job to poll. ExecuteBulkUpsertProducts writes.
func (s *productSvcImpl) BulkUpsertProducts(ctx context.Context, params domain.BulkUpsertProductsParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), params.Products)
}

// performs the writes for an enqueued bulk upsert. Delivery is at-least-once; the inbox
// de-dup and the engine's terminal-job guard make it effectively-once.
func (s *productSvcImpl) ExecuteBulkUpsertProducts(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), event)
}

// --- Export ---

// wires products into the export engine
func (s *productSvcImpl) exportSpec() exportSpec[*domain.ProductFull, domain.ExportProductsParams] {
	return exportSpec[*domain.ProductFull, domain.ExportProductsParams]{
		CheckPermission: checkProductReadPermission,
		ExternalAccess:  externalCounterparty,
		Name:            "Products",
		Slug:            "products",
		ResourceType:    constants.ObjectTypeProduct,

		ColumnsFor: func(products []*domain.ProductFull) []excel.ColumnSpec {
			base := itemBaseColumns(
				excel.ColumnSpec{Header: "Product Line", Key: "product_line", Width: 20},
				excel.ColumnSpec{Header: "Type", Key: "type", Width: 12},
				excel.ColumnSpec{Header: "Portal Visibility", Key: "portal_visibility", Width: 16},
			)
			return append(base, itemPropertyColumns(productItems(products))...)
		},

		// A customer user sees only what their own account may buy.
		NarrowFilters: func(identity *types.Identity, filters domain.ExportProductsParams) domain.ExportProductsParams {
			if !identity.IsCustomerUser() {
				return filters
			}
			if actorAccountID := identity.ActorAccountID(); actorAccountID != nil {
				filters.CustomerIDs = []string{*actorAccountID}
			}
			isPortalReady := true
			filters.IsPortalReady = &isPortalReady
			return filters
		},

		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportProductsParams) ([]*domain.ProductFull, *apierror.APIError) {
			filters.AccountID = accountID
			return repos.NewProductRepo().Export(ctx, filters)
		},

		Project: func(product *domain.ProductFull) excel.Row {
			productLine := ""
			if product.ProductLine != nil {
				productLine = product.ProductLine.Name
			}
			row := excel.Row{
				"product_line":      productLine,
				"type":              product.ProductTypeCode,
				"portal_visibility": yesNo(product.IsPortalReady),
			}
			addItemBaseCells(row, product.ID, product.Item)
			return row
		},
	}
}

// accepts an export: records the filters on a job and returns it to poll.
func (s *productSvcImpl) ExportProducts(ctx context.Context, params domain.ExportProductsParams) (*domain.Job, *apierror.APIError) {
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), params)
}

// renders the file an accepted export recorded
func (s *productSvcImpl) BuildExportProducts(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}

// lifts the items out of a product list, for the shared property-column helpers
func productItems(products []*domain.ProductFull) []*domain.Item {
	items := make([]*domain.Item, 0, len(products))
	for _, product := range products {
		items = append(items, product.Item)
	}
	return items
}
