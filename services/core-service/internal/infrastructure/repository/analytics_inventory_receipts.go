package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// inventoryReceiptSummaryQuery groups an account's available receipts by item, location, lot, owner and holder, oldest receipt first.
//
// Each receipt counts for what is left of it after its allocations, in the item's base unit and never below zero, at its unit cost per that unit. A receipt without a quantity or a unit cost is left out, as the dashboard left it out. The cost's currency is the oldest receipt's.
func inventoryReceiptSummaryQuery(params domain.AnalyzeInventoryReceiptsParams) (string, []any) {
	// Status repeats in each branch so each ranges its own (account, status) key; factored out, the union reads every status.
	preds := []string{"((ir.owner_account_id = ? AND ir.status_code = 'available') OR (ir.holder_account_id = ? AND ir.status_code = 'available'))"}
	args := []any{params.AccountID, params.AccountID}
	// The receipts are read through the account's owner and holder keys, or a filter's: left to choose, the planner reads every tenant's available receipts by status.
	keys := []string{"inventory_receipt_owner_status_idx", "inventory_receipt_holder_status_idx"}
	in := func(column, key string, ids []string) {
		preds = append(preds, column+" IN ("+placeholders(len(ids))+")")
		args = append(args, stringsToAny(ids)...)
		keys = append(keys, key)
	}
	if len(params.ItemIDs) > 0 {
		in("ir.item_id", "inventory_receipt_item_id_status_code_idx", params.ItemIDs)
	}
	if len(params.LocationIDs) > 0 {
		in("ir.storage_location_id", "inventory_receipt_storage_location_id_idx", params.LocationIDs)
	}
	if len(params.LotIDs) > 0 {
		in("ir.lot_id", "inventory_receipt_lot_id_idx", params.LotIDs)
	}

	// r is materialized (NO_MERGE) so each receipt's allocations are totalled once: clamped in place, GREATEST would evaluate the subquery twice.
	query := `SELECT
    g.item_id, it.sku, it.description,
    g.storage_location_id, sl.name, g.lot_id, l.lot_number,
    g.owner_account_id, oa.name, g.holder_account_id, ha.name,
    g.remaining, g.value, g.oldest, g.newest,
    bu.abbreviation, bu.name, bu.unit_dimension_code,
    cn.abbreviation, cn.name, cn.unit_dimension_code
FROM (
    SELECT /*+ NO_MERGE(r) */
        r.item_id, r.storage_location_id, r.lot_id, r.owner_account_id, r.holder_account_id,
        CAST(SUM(GREATEST(r.received - r.allocated, 0) / r.base_ratio) AS DECIMAL(65,30)) AS remaining,
        CAST(SUM(GREATEST(r.received - r.allocated, 0) / r.base_ratio * r.cost) AS DECIMAL(65,30)) AS value,
        MIN(r.received_at) AS oldest,
        MAX(r.received_at) AS newest,
        MIN(r.base_unit_id) AS base_unit_id,
        SUBSTRING_INDEX(GROUP_CONCAT(r.cost_numerator_unit_id ORDER BY r.received_at, r.id SEPARATOR ','), ',', 1) AS cost_numerator_unit_id
    FROM (
        SELECT
            ir.id, ir.item_id, ir.storage_location_id, ir.lot_id, ir.owner_account_id, ir.holder_account_id, ir.received_at,
            bu.id AS base_unit_id,
            (bu.ratio_numerator / bu.ratio_denominator) AS base_ratio,
            (q.value * (qu.ratio_numerator / qu.ratio_denominator)) + (qu.offset_numerator / qu.offset_denominator) AS received,
            COALESCE((
                -- Correlated per receipt: a grouped derived table cannot take the account filter and so totals all of inventory_allocation.
                SELECT SUM((aq.value * (au.ratio_numerator / au.ratio_denominator)) + (au.offset_numerator / au.offset_denominator))
                FROM inventory_allocation ia
                JOIN quantity aq ON aq.id = ia.quantity_id
                JOIN unit au ON au.id = aq.unit_id
                WHERE ia.inventory_receipt_id = ir.id
            ), 0) AS allocated,
            CASE WHEN cd.unit_dimension_code = bu.unit_dimension_code
                THEN rc.value * (bu.ratio_numerator / bu.ratio_denominator) / (cd.ratio_numerator / cd.ratio_denominator)
                ELSE rc.value END AS cost,
            rc.numerator_unit_id AS cost_numerator_unit_id
        FROM inventory_receipt ir FORCE INDEX (` + strings.Join(keys, ", ") + `)
        JOIN quantity q ON q.id = ir.quantity_id
        JOIN unit qu ON qu.id = q.unit_id
        JOIN rate rc ON rc.id = ir.unit_cost_id
        JOIN unit cd ON cd.id = rc.denominator_unit_id
        JOIN item it ON it.id = ir.item_id
        JOIN item_category ic ON ic.id = it.item_category_id
        JOIN unit_group ug ON ug.id = ic.unit_group_id
        JOIN unit bu ON bu.id = ug.base_unit_id
        WHERE ` + strings.Join(preds, " AND ") + `
    ) r
    GROUP BY r.item_id, r.storage_location_id, r.lot_id, r.owner_account_id, r.holder_account_id
) g
JOIN item it ON it.id = g.item_id
LEFT JOIN storage_location sl ON sl.id = g.storage_location_id
LEFT JOIN lot l ON l.id = g.lot_id
JOIN account oa ON oa.id = g.owner_account_id
JOIN account ha ON ha.id = g.holder_account_id
LEFT JOIN unit bu ON bu.id = g.base_unit_id
LEFT JOIN unit cn ON cn.id = g.cost_numerator_unit_id
ORDER BY g.oldest ASC, g.item_id ASC, g.storage_location_id ASC, g.lot_id ASC, g.owner_account_id ASC, g.holder_account_id ASC`
	return query, args
}

func (r *analyticsRepoImpl) GetInventoryReceiptAnalytics(ctx context.Context, params domain.AnalyzeInventoryReceiptsParams) ([]domain.InventoryReceiptEntry, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_inventory_receipt_analytics")
	defer span.End()

	query, args := inventoryReceiptSummaryQuery(params)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	defer rows.Close()

	entries := []domain.InventoryReceiptEntry{}
	for rows.Next() {
		var (
			e                                 domain.InventoryReceiptEntry
			description, locationID, location sql.NullString
			lotID, lotNumber                  sql.NullString
			remaining, value                  sql.NullString
			oldest, newest                    sql.NullTime
			unitAbbr, unitName, unitType      sql.NullString
			costAbbr, costName, costType      sql.NullString
		)
		if err := rows.Scan(&e.ItemID, &e.ProductSku, &description, &locationID, &location, &lotID, &lotNumber,
			&e.OwnerAccountID, &e.OwnerAccountName, &e.HolderAccountID, &e.HolderAccountName,
			&remaining, &value, &oldest, &newest,
			&unitAbbr, &unitName, &unitType, &costAbbr, &costName, &costType); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		e.ProductDescription = nullStringPtr(description)
		e.LocationID, e.LocationName = nullStringPtr(locationID), nullStringPtr(location)
		e.LotID, e.LotNumber = nullStringPtr(lotID), nullStringPtr(lotNumber)
		e.RemainingQuantity = decimalToFloat(remaining)
		// Nothing left has no value and no weighted cost, as the dashboard reported it.
		if e.RemainingQuantity > 0 {
			v := decimalToFloat(value)
			e.InventoryValue = &v
			e.WeightedAverageUnitCost = v / e.RemainingQuantity
		}
		e.OldestReceiptAt, e.NewestReceiptAt = nullTimePtr(oldest), nullTimePtr(newest)
		e.Unit, e.UnitName, e.UnitType = unitAbbr.String, unitName.String, unitType.String
		e.CostNumeratorUnitAbbreviation, e.CostNumeratorUnitName, e.CostNumeratorUnitType = costAbbr.String, costName.String, costType.String
		// The cost is carried per base unit, so its denominator is the base unit.
		e.CostDenominatorUnitAbbreviation, e.CostDenominatorUnitName, e.CostDenominatorUnitType = unitAbbr.String, unitName.String, unitType.String
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	return entries, nil
}
