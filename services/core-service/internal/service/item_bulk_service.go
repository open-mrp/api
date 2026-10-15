package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/field"
)

// hands the async bulk engine the plumbing it runs on
func (s *itemSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

// validateBulkUpsertItemRows runs the accept-phase checks that need no database: a valid type, no SKU twice, and a
// material section only on materials.
func validateBulkUpsertItemRows(rows []domain.UpsertItemParams) *apierror.APIError {
	skus := make(map[string]struct{}, len(rows))
	var rowErrs apierror.RowErrors
	for i, row := range rows {
		if _, dup := skus[row.SKU]; dup {
			rowErrs.AddField(i, "sku", apierror.CodeInvalidFormat, fmt.Sprintf("duplicate SKU %q in request", row.SKU))
		}
		skus[row.SKU] = struct{}{}
		if !row.Type.IsValid() {
			rowErrs.AddField(i, "type", apierror.CodeInvalidFormat, "type must be one of material, part, product or service")
		}
		if row.Material != nil && row.Type != constants.ItemTypeCodeMaterial {
			rowErrs.AddField(i, "material", apierror.CodeInvalidFormat, "only materials have a material section")
		}
	}
	return rowErrs.Summary("items")
}

// resolveBulkUpsertItemRows resolves each row's category and checks it suits the row's type.
func resolveBulkUpsertItemRows(ctx context.Context, repos domain.RepoFactory, accountID string, rows []domain.UpsertItemParams) ([]domain.UpsertItemParams, *apierror.APIError) {
	identifiers := make([]*domain.ObjectIdentifier, len(rows))
	for i := range rows {
		identifiers[i] = &rows[i].Category
	}
	catRepo := repos.NewItemCategoryRepo()
	categoryIDs, apiErr := resolveObjectIdentifiers(
		ctx, accountID, "items", "category_id", "category", "categories", identifiers,
		catRepo.GetByIDs, catRepo.FindByNames,
		func(c *domain.ItemCategoryFull) string { return c.ID },
		func(c *domain.ItemCategoryFull) string { return c.Name },
	)
	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = categoryIDs[row.Category]
	}
	categories, apiErr := repos.NewItemRepo().GetCategoryBaseUnitIDs(ctx, uniqueIDs(ids))
	if apiErr != nil {
		return nil, apiErr
	}

	var rowErrs apierror.RowErrors
	resolved := make([]domain.UpsertItemParams, len(rows))
	for i, row := range rows {
		row.CategoryID = ids[i]
		category, ok := categories[row.CategoryID]
		switch {
		case !ok:
			rowErrs.AddField(i, "category_id", apierror.CodeInvalidFormat, fmt.Sprintf("category %q was not found", row.CategoryID))
		case !categoryTypeMatchesItem(string(row.Type), category.ItemCategoryTypeCode):
			rowErrs.AddField(i, "category_id", apierror.CodeInvalidFormat,
				"a material needs a material category; parts, products and services need a product category")
		}
		resolved[i] = row
	}
	if apiErr := rowErrs.Summary("items"); apiErr != nil {
		return nil, apiErr
	}
	return resolved, nil
}

// writeBulkUpsertItems matches each row to an existing item by SKU and creates or updates it in its own savepoint, so
// a bad row rolls back only itself.
func (s *itemSvcImpl) writeBulkUpsertItems(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.UpsertItemParams) (BulkWriteResult, *apierror.APIError) {
	var allProps []domain.UpsertItemPropertyParams
	skus := make([]string, len(rows))
	for i, row := range rows {
		skus[i] = row.SKU
		allProps = append(allProps, row.Attributes...)
	}
	attrResolver, propIDByName, apiErr := resolvePropertyAttributesInTx(txCtx, txRepos, accountID, allProps)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}

	existing, apiErr := txRepos.NewItemRepo().List(txCtx, domain.ListItemsParams{AccountID: accountID, SKUs: skus, Limit: int32(len(skus))})
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	bySKU := make(map[string]*domain.Item, len(existing.Items))
	for _, item := range existing.Items {
		bySKU[item.SKU] = item
	}

	results := make([]domain.RowResult, 0, len(rows))
	var rowErrors []apierror.RowError
	for i := range rows {
		row := rows[i]
		var itemID string
		var isCreate bool
		rowErr := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
			var attributeIDs []string
			if row.HasAttributes {
				if apiErr := linkRowPropertiesToCategoryInTx(spCtx, txRepos, row.CategoryID, row.Attributes, propIDByName); apiErr != nil {
					return apiErr
				}
				attributeIDs = attributeIDsForProperties(row.Attributes, attrResolver)
			}

			if old := bySKU[row.SKU]; old != nil {
				if constants.ItemTypeCode(old.ItemTypeCode) != row.Type {
					return apierror.NewFieldError("type", apierror.CodeInvalidFormat,
						fmt.Sprintf("SKU %q is a %s; an item's type can't change", row.SKU, old.ItemTypeCode))
				}
				update := domain.UpdateItemParams{
					AccountID:         accountID,
					ItemID:            old.ID,
					Description:       row.Description,
					UpdateDescription: row.Description != nil,
					CategoryID:        &row.CategoryID,
					Status:            row.Status,
					Supply:            clearableOfSupply(row.Supply, row.ClearSupply),
					UnitCost:          row.UnitCost,
					Sales:             row.Sales.Input(),
					Material:          row.Material.Input(),
				}
				if row.HasAttributes {
					update.AttributeIDs = field.Some(attributeIDs)
				}
				updated, apiErr := updateItemInTx(spCtx, txRepos, update)
				if apiErr != nil {
					return apiErr
				}
				itemID = updated.ID
				return nil
			}

			status := constants.ArchiveStatusActive
			if row.Status != nil {
				status = *row.Status
			}
			created, apiErr := createItemInTx(spCtx, txRepos, domain.CreateItemParams{
				AccountID:    accountID,
				Type:         row.Type,
				SKU:          row.SKU,
				Description:  row.Description,
				CategoryID:   row.CategoryID,
				AttributeIDs: attributeIDs,
				Status:       status,
				Supply:       row.Supply,
				UnitCost:     row.UnitCost,
				Sales:        row.Sales.Input(),
				Material:     row.Material.Input(),
			})
			if apiErr != nil {
				return apiErr
			}
			itemID, isCreate = created.ID, true
			return nil
		})
		if rowErr != nil {
			rowErrors = append(rowErrors, apierror.RowError{Index: i, Error: rowErr.Object()})
			continue
		}
		results = append(results, newRowResult(i, itemID, isCreate))
	}

	return BulkWriteResult{Results: results, Errors: rowErrors, WrittenIDs: resultIDs(results)}, nil
}

func clearableOfSupply(v *constants.Supply, clear bool) field.Clearable[constants.Supply] {
	switch {
	case v != nil:
		return field.Set(*v)
	case clear:
		return field.Clear[constants.Supply]()
	default:
		return field.Unset[constants.Supply]()
	}
}

// wires items into the async bulk engine
func (s *itemSvcImpl) bulkUpsertSpec() bulkOperationSpec[domain.UpsertItemParams, domain.UpsertItemParams] {
	return bulkOperationSpec[domain.UpsertItemParams, domain.UpsertItemParams]{
		JobType:          constants.JobTypeBulkUpsert,
		ResourceType:     constants.ObjectTypeItem,
		RoutingKey:       messaging.BulkUpsertItems.RoutingKey(),
		PermissionDomain: types.PermissionDomainItems,
		Actions:          []types.Action{types.ActionCreate, types.ActionUpdate},
		EntityName:       "items",
		Validate:         validateBulkUpsertItemRows,
		Resolve:          resolveBulkUpsertItemRows,
		Write:            s.writeBulkUpsertItems,
	}
}

// BulkUpsertItems validates and resolves the rows, records them on a job and returns the job to poll.
func (s *itemSvcImpl) BulkUpsertItems(ctx context.Context, params domain.BulkUpsertItemsParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), params.Items)
}

// ExecuteBulkUpsertItems writes an accepted bulk upsert. Delivery is at-least-once; the inbox de-dup and the engine's
// terminal-job guard make it effectively-once.
func (s *itemSvcImpl) ExecuteBulkUpsertItems(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), event)
}
