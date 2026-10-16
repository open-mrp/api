package repository

import (
	"context"
	gosql "database/sql"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/tracing"
)

func supplyNullString(s *constants.Supply) gosql.NullString {
	if s == nil {
		return gosql.NullString{}
	}
	return gosql.NullString{String: string(*s), Valid: true}
}

// nullable is the value to bind for a nullable column: nil for NULL.
func nullable(v gosql.NullString) any {
	if !v.Valid {
		return nil
	}
	return v.String
}

func (r *itemRepoImpl) SetFields(ctx context.Context, params domain.SetItemFieldsParams) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.set_fields")
	defer span.End()

	isActive := gosql.NullBool{}
	if params.Status != nil {
		isActive = gosql.NullBool{Bool: params.Status.IsActive(), Valid: true}
	}
	err := r.queries.SetItemFields(ctx, sqlc.SetItemFieldsParams{
		Sku:            toNullString(params.SKU),
		SetDescription: params.SetDescription,
		Description:    nullable(toNullString(params.Description)),
		ItemCategoryID: toNullString(params.CategoryID),
		IsActive:       isActive,
		SetSupply:      params.SetSupply,
		SupplyCode:     nullable(supplyNullString(params.Supply)),
		ID:             params.ItemID,
		AccountID:      params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *itemRepoImpl) Insert(ctx context.Context, params domain.InsertItemParams) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.insert")
	defer span.End()

	err := r.queries.InsertItem(ctx, sqlc.InsertItemParams{
		ID:             params.ItemID,
		Sku:            params.SKU,
		Description:    toNullString(params.Description),
		ItemTypeCode:   string(params.Type),
		ItemCategoryID: params.CategoryID,
		UnitValueID:    params.UnitValueRateID,
		UnitCostID:     params.UnitCostRateID,
		BurnRateID:     params.BurnRateRateID,
		AccountID:      params.AccountID,
		IsActive:       params.Status.IsActive(),
		SupplyCode:     supplyNullString(params.Supply),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := db.MapSQLError(syncItemMoney(ctx, r.queries.DB(), params.ItemID)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *itemRepoImpl) InsertRate(ctx context.Context, id string, rate domain.CreateRateParams) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.insert_rate")
	defer span.End()

	err := r.queries.MaterialInsertRate(ctx, sqlc.MaterialInsertRateParams{
		ID:                id,
		Value:             rate.Value,
		NumeratorUnitID:   rate.NumeratorUnitID,
		DenominatorUnitID: rate.DenominatorUnitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *itemRepoImpl) InsertQuantity(ctx context.Context, id string, q domain.QuantityInput) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.insert_quantity")
	defer span.End()

	err := r.queries.MaterialInsertQuantity(ctx, sqlc.MaterialInsertQuantityParams{ID: id, Value: q.Value, UnitID: q.UnitID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *itemRepoImpl) UpdateQuantity(ctx context.Context, id string, q domain.QuantityInput) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.update_quantity")
	defer span.End()

	_, err := r.queries.MaterialUpdateQuantity(ctx, sqlc.MaterialUpdateQuantityParams{ID: id, Value: q.Value, UnitID: q.UnitID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *itemRepoImpl) DeleteQuantities(ctx context.Context, ids []string) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.delete_quantities")
	defer span.End()

	if len(ids) == 0 {
		return nil
	}
	if apiErr := db.MapSQLError(r.queries.DeleteQuantitiesByIDs(ctx, ids)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *itemRepoImpl) ReplaceAttributes(ctx context.Context, itemID string, attributeIDs []string) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.replace_attributes")
	defer span.End()

	if apiErr := db.MapSQLError(r.queries.DeleteItemAttributes(ctx, itemID)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	for _, attributeID := range attributeIDs {
		err := r.queries.AddItemAttribute(ctx, sqlc.AddItemAttributeParams{AttributeID: attributeID, ItemID: itemID})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
}

func (r *itemRepoImpl) InsertProduct(ctx context.Context, row domain.ItemProductRow) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.insert_product")
	defer span.End()

	err := r.queries.InsertItemProduct(ctx, sqlc.InsertItemProductParams{
		ID:            row.ProductID,
		ItemID:        row.ItemID,
		ProductLineID: toNullString(row.ProductLineID),
		IsPortalReady: row.IsPortalReady,
		LeadTimeID:    toNullString(row.LeadTimeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *itemRepoImpl) SetProduct(ctx context.Context, params domain.SetItemProductParams) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.set_product")
	defer span.End()

	portal := gosql.NullBool{}
	if params.IsPortalReady != nil {
		portal = gosql.NullBool{Bool: *params.IsPortalReady, Valid: true}
	}
	err := r.queries.SetItemProductFields(ctx, sqlc.SetItemProductFieldsParams{
		SetProductLine: params.SetProductLine,
		ProductLineID:  nullable(toNullString(params.ProductLineID)),
		IsPortalReady:  portal,
		SetLeadTime:    params.SetLeadTime,
		LeadTimeID:     nullable(toNullString(params.LeadTimeID)),
		ItemID:         params.ItemID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *itemRepoImpl) DeleteProduct(ctx context.Context, itemID string) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.delete_product")
	defer span.End()

	if apiErr := db.MapSQLError(r.queries.DeleteItemProduct(ctx, itemID)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *itemRepoImpl) InsertPart(ctx context.Context, partID, itemID string) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.insert_part")
	defer span.End()

	if apiErr := db.MapSQLError(r.queries.InsertItemPart(ctx, sqlc.InsertItemPartParams{ID: partID, ItemID: itemID})); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *itemRepoImpl) HardDelete(ctx context.Context, item *domain.Item) *apierror.APIError {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.hard_delete")
	defer span.End()

	q := r.queries
	logQuantityIDs, err := q.ListItemInventoryLogQuantityIDs(ctx, sqlc.ListItemInventoryLogQuantityIDsParams{ItemID: item.ID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	quantityIDs := append([]string{}, logQuantityIDs...)
	if item.Sales != nil && item.Sales.LeadTimeID != nil {
		quantityIDs = append(quantityIDs, *item.Sales.LeadTimeID)
	}
	if m := item.MaterialSection; m != nil {
		quantityIDs = append(quantityIDs, m.OrderPointID, m.LeadTimeID)
	}

	steps := []func() error{
		func() error { return q.DeleteItemInventoryChangeLogs(ctx, item.ID) },
		func() error { return q.DeleteItemInventoryLogs(ctx, item.ID) },
		func() error { return q.DeleteItemInventoryLock(ctx, item.ID) },
		func() error { return q.DeleteItemAttributes(ctx, item.ID) },
		func() error {
			return q.DeleteItemMachineItems(ctx, sqlc.DeleteItemMachineItemsParams{AccountID: item.AccountID, ItemID: item.ID})
		},
		func() error {
			return q.DeleteItemPlanningSettings(ctx, sqlc.DeleteItemPlanningSettingsParams{AccountID: item.AccountID, ItemID: item.ID})
		},
		func() error {
			return q.DeleteItemPlanningPolicies(ctx, sqlc.DeleteItemPlanningPoliciesParams{AccountID: item.AccountID, ItemID: item.ID})
		},
		func() error {
			return q.DeleteItemFinishedPolicies(ctx, sqlc.DeleteItemFinishedPoliciesParams{AccountID: item.AccountID, ItemID: item.ID})
		},
		func() error {
			return q.DeleteItemDemandOverrides(ctx, sqlc.DeleteItemDemandOverridesParams{AccountID: item.AccountID, ItemID: item.ID})
		},
		func() error { return q.DeleteItemProduct(ctx, item.ID) },
		func() error { return q.DeleteItemMaterial(ctx, item.ID) },
		func() error { return q.DeleteItemPart(ctx, item.ID) },
		func() error {
			res, err := q.DeleteItemByID(ctx, sqlc.DeleteItemByIDParams{ID: item.ID, AccountID: item.AccountID})
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n == 0 {
				return gosql.ErrNoRows
			}
			return nil
		},
		func() error {
			return q.DeleteRatesByIDs(ctx, []string{item.UnitValueID, item.UnitCostID, item.BurnRateID})
		},
		func() error {
			if len(quantityIDs) == 0 {
				return nil
			}
			return q.DeleteQuantitiesByIDs(ctx, quantityIDs)
		},
	}
	for _, step := range steps {
		if apiErr := db.MapSQLError(step()); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
}

func (r *itemRepoImpl) GetAssignability(ctx context.Context, accountID string, ids []string) ([]domain.ItemAssignability, *apierror.APIError) {
	ctx, span := itemRepoTracer.Start(ctx, "repository.item.get_assignability")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetItemAssignabilityByIDs(ctx, sqlc.GetItemAssignabilityByIDsParams{Ids: ids, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]domain.ItemAssignability, len(rows))
	for i, row := range rows {
		out[i] = domain.ItemAssignability{ItemID: row.ID, SKU: row.Sku, IsAssignable: row.IsAssignable == 1}
	}
	return out, nil
}

// stitchItemSections attaches each item's sales and material details. Every item read for the API carries them.
func stitchItemSections(ctx context.Context, queries *sqlc.Queries, items []*domain.Item) *apierror.APIError {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, len(items))
	byID := make(map[string]*domain.Item, len(items))
	for i, item := range items {
		ids[i] = item.ID
		byID[item.ID] = item
	}

	sales, err := queries.GetItemSalesByItemIDs(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	for _, row := range sales {
		item := byID[row.ItemID]
		if item == nil {
			continue
		}
		section := &domain.ItemSales{ProductID: row.ID, IsPortalReady: row.IsPortalReady}
		if row.ProductLineID.Valid {
			section.ProductLineID = &row.ProductLineID.String
		}
		if row.LeadTimeID.Valid {
			section.LeadTimeID = &row.LeadTimeID.String
			if row.LeadTimeValue.Valid {
				section.LeadTime = &domain.QuantityValue{Value: row.LeadTimeValue.String, UnitID: row.LeadTimeUnitID.String, UnitType: row.LeadTimeUnitType.String}
			}
		}
		item.Sales = section
	}

	materials, err := queries.GetItemMaterialsByItemIDs(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	for _, row := range materials {
		item := byID[row.ItemID]
		if item == nil {
			continue
		}
		item.MaterialSection = &domain.ItemMaterialSection{
			MaterialID:   row.ID,
			OrderPointID: row.OrderPointID,
			LeadTimeID:   row.LeadTimeID,
			OrderPoint:   nonZeroQuantity(row.OrderPointValue, row.OrderPointUnitID, row.OrderPointUnitType),
			LeadTime:     nonZeroQuantity(row.LeadTimeValue, row.LeadTimeUnitID, row.LeadTimeUnitType),
		}
	}
	return nil
}

// nonZeroQuantity is a stored material quantity, or nil for the zero a material stores when it has none.
func nonZeroQuantity(value, unitID, unitType string) *domain.QuantityValue {
	if d, err := decimal.NewFromString(value); err == nil && d.IsZero() {
		return nil
	}
	return &domain.QuantityValue{Value: value, UnitID: unitID, UnitType: unitType}
}
