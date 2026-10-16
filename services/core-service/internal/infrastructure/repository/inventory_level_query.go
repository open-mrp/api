package repository

import (
	"context"
	gosql "database/sql"
	"errors"
	"time"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/tracing"
)

// netAgainst is a ledger row's quantity in base units less what has been allocated against it; side names the
// allocation column that points at the row (inventory_receipt_id or inventory_issue_id).
func netAgainst(alias, side string) string {
	return "CAST(" + alias + "q.value AS DECIMAL(65,30)) * (" + alias + "u.ratio_numerator / " + alias + "u.ratio_denominator) - COALESCE((" +
		"SELECT SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator)) " +
		"FROM inventory_allocation ia JOIN quantity aq ON aq.id = ia.quantity_id JOIN unit au ON au.id = aq.unit_id " +
		"WHERE ia." + side + " = " + alias + ".id), 0)"
}

// buildStockLevelsQuery is each item's on-hand, reserved and short stock, in its stocking unit, from the ledger rows
// at locationIDs (all rows when empty). Each sum is correlated per item so it ranges the ledger's (item_id, status)
// keys rather than aggregating the account; FetchOnHandInventoryBulk measured that shape fastest for a page.
func buildStockLevelsQuery(accountID string, itemIDs, locationIDs []string) (string, []any) {
	receiptLocation, issueLocation := "", ""
	var locArgs []any
	if len(locationIDs) > 0 {
		receiptLocation = " AND ir.storage_location_id IN (" + iclPlaceholders(len(locationIDs)) + ")"
		issueLocation = " AND ii.storage_location_id IN (" + iclPlaceholders(len(locationIDs)) + ")"
		for _, id := range locationIDs {
			locArgs = append(locArgs, id)
		}
	}
	issueSum := func(status string) string {
		return "COALESCE((SELECT SUM(" + netAgainst("ii", "inventory_issue_id") + ") FROM inventory_issue ii " +
			"JOIN quantity iiq ON iiq.id = ii.quantity_id JOIN unit iiu ON iiu.id = iiq.unit_id " +
			"WHERE ii.item_id = i.id AND ii.account_id = ? AND ii.status_code = '" + status + "'" + issueLocation + "), 0)"
	}
	ratio := "COALESCE(NULLIF(bu.ratio_numerator / bu.ratio_denominator, 0), 1)"
	query := "SELECT i.id, COALESCE(bu.id, ''), " +
		"CAST(COALESCE((SELECT SUM(" + netAgainst("ir", "inventory_receipt_id") + ") FROM inventory_receipt ir " +
		"JOIN quantity irq ON irq.id = ir.quantity_id JOIN unit iru ON iru.id = irq.unit_id " +
		"WHERE ir.item_id = i.id AND (ir.owner_account_id = ? OR ir.holder_account_id = ?) AND ir.status_code = 'available'" + receiptLocation + "), 0) / " + ratio + " AS DECIMAL(65,30)), " +
		"CAST(" + issueSum("reserved") + " / " + ratio + " AS DECIMAL(65,30)), " +
		"CAST(" + issueSum("open") + " / " + ratio + " AS DECIMAL(65,30)) " +
		"FROM item i " +
		"LEFT JOIN item_category ic ON ic.id = i.item_category_id " +
		"LEFT JOIN unit_group ug ON ug.id = ic.unit_group_id " +
		"LEFT JOIN unit bu ON bu.id = COALESCE(i.stocking_unit_id, ug.base_unit_id) " +
		"WHERE i.account_id = ? AND i.id IN (" + iclPlaceholders(len(itemIDs)) + ")"
	args := []any{accountID, accountID}
	args = append(args, locArgs...)
	args = append(args, accountID)
	args = append(args, locArgs...)
	args = append(args, accountID)
	args = append(args, locArgs...)
	args = append(args, accountID)
	for _, id := range itemIDs {
		args = append(args, id)
	}
	return query, args
}

// FetchStockLevels is each item's stock position at locationIDs (every location when empty), in its stocking unit:
// on hand (available receipts net of what was drawn from them, owned or held), reserved (reserved issues net of their
// allocations) and short (open issues net of theirs). Available to promise is what's left.
func (r *inventoryQueryRepoImpl) FetchStockLevels(ctx context.Context, accountID string, itemIDs, locationIDs []string) (map[string]*domain.StockLevel, *apierror.APIError) {
	ctx, span := inventoryQueryRepoTracer.Start(ctx, "repository.inventory_query.fetch_stock_levels")
	defer span.End()

	out := make(map[string]*domain.StockLevel, len(itemIDs))
	if len(itemIDs) == 0 {
		return out, nil
	}
	query, args := buildStockLevelsQuery(accountID, itemIDs, locationIDs)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer rows.Close()

	for rows.Next() {
		var itemID, unitID, onHandStr, reservedStr, shortStr string
		if err := rows.Scan(&itemID, &unitID, &onHandStr, &reservedStr, &shortStr); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		onHand, err1 := decimal.NewFromString(onHandStr)
		reserved, err2 := decimal.NewFromString(reservedStr)
		short, err3 := decimal.NewFromString(shortStr)
		if err := errors.Join(err1, err2, err3); err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Invalid inventory level."))
		}
		available := onHand.Sub(reserved).Sub(short)
		out[itemID] = &domain.StockLevel{
			ItemID: itemID, UnitID: unitID,
			OnHand: onHand, Reserved: &reserved, AvailableToPromise: &available, Short: &short,
		}
	}
	if apiErr := db.MapSQLError(rows.Err()); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return out, nil
}

// FetchStockLevelsAsOf is each item's last logged level at or before asOf, in its stocking unit, as an on-hand figure
// with nothing else known; zero for an item with no log by then.
func (r *inventoryQueryRepoImpl) FetchStockLevelsAsOf(ctx context.Context, accountID string, itemIDs []string, asOf time.Time) (map[string]*domain.StockLevel, *apierror.APIError) {
	ctx, span := inventoryQueryRepoTracer.Start(ctx, "repository.inventory_query.fetch_stock_levels_as_of")
	defer span.End()

	out := make(map[string]*domain.StockLevel, len(itemIDs))
	if len(itemIDs) == 0 {
		return out, nil
	}
	rows, err := r.queries.FetchInventoryLevelsAsOf(ctx, sqlc.FetchInventoryLevelsAsOfParams{
		AccountID: accountID, AsOf: asOf, LogItemIds: itemIDs, ItemIds: itemIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, row := range rows {
		level, parseErr := decimal.NewFromString(row.Level)
		if parseErr != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(parseErr, "Invalid inventory level."))
		}
		out[row.ItemID] = &domain.StockLevel{ItemID: row.ItemID, UnitID: row.UnitID, OnHand: level}
	}
	return out, nil
}

// LotBelongsToItem reports whether lotID is a lot of the account's item.
func (r *inventoryQueryRepoImpl) LotBelongsToItem(ctx context.Context, accountID, itemID, lotID string) (bool, *apierror.APIError) {
	ctx, span := inventoryQueryRepoTracer.Start(ctx, "repository.inventory_query.lot_belongs_to_item")
	defer span.End()

	var one int
	err := r.queries.DB().QueryRowContext(ctx,
		"SELECT 1 FROM lot WHERE id = ? AND account_id = ? AND item_id = ?", lotID, accountID, itemID).Scan(&one)
	if errors.Is(err, gosql.ErrNoRows) {
		return false, nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return true, nil
}
