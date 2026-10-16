package repository

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
)

// restateChunk bounds each restating statement to a short primary-key update.
const restateChunk = 500

// restateSource finds the item's rows of one kind still stated in the old unit. find selects their ids given
// (item id, old unit id, limit); the ids are rewritten by primary key on table.
type restateSource struct {
	find string
	// table is quantity or rate; perUnit rates divide by the factor (their denominator is the unit), others
	// multiply (a quantity, or a rate whose numerator is the unit).
	table   string
	perUnit bool
}

// The item's quantities and per-unit amounts kept in its stocking unit: its inventory ledger and logs, the
// quantities its production steps make and consume, its order point, and its price, cost and burn rate.
var restateSources = []restateSource{
	{table: "quantity", find: "SELECT q.id FROM quantity q JOIN inventory_receipt x ON x.quantity_id = q.id WHERE x.item_id = ? AND q.unit_id = ? LIMIT ?"},
	{table: "quantity", find: "SELECT q.id FROM quantity q JOIN inventory_issue x ON x.quantity_id = q.id WHERE x.item_id = ? AND q.unit_id = ? LIMIT ?"},
	{table: "quantity", find: "SELECT q.id FROM quantity q JOIN inventory_allocation a ON a.quantity_id = q.id JOIN inventory_receipt x ON x.id = a.inventory_receipt_id WHERE x.item_id = ? AND q.unit_id = ? LIMIT ?"},
	{table: "quantity", find: "SELECT q.id FROM quantity q JOIN inventory_log x ON x.quantity_id = q.id WHERE x.item_id = ? AND q.unit_id = ? LIMIT ?"},
	{table: "quantity", find: "SELECT q.id FROM quantity q JOIN inventory_change_log x ON x.quantity_id = q.id WHERE x.item_id = ? AND q.unit_id = ? LIMIT ?"},
	{table: "quantity", find: "SELECT q.id FROM quantity q JOIN production x ON x.quantity_id = q.id WHERE x.item_id = ? AND q.unit_id = ? LIMIT ?"},
	{table: "quantity", find: "SELECT q.id FROM quantity q JOIN consumption x ON x.quantity_id = q.id WHERE x.item_id = ? AND q.unit_id = ? LIMIT ?"},
	{table: "quantity", find: "SELECT q.id FROM quantity q JOIN consumption x ON x.waste_quantity_id = q.id WHERE x.item_id = ? AND q.unit_id = ? LIMIT ?"},
	{table: "quantity", find: "SELECT q.id FROM quantity q JOIN material x ON x.order_point_id = q.id WHERE x.item_id = ? AND q.unit_id = ? LIMIT ?"},
	{table: "rate", perUnit: true, find: "SELECT r.id FROM rate r JOIN inventory_receipt x ON x.unit_cost_id = r.id WHERE x.item_id = ? AND r.denominator_unit_id = ? LIMIT ?"},
	{table: "rate", perUnit: true, find: "SELECT r.id FROM rate r JOIN inventory_allocation a ON a.unit_cost_id = r.id JOIN inventory_receipt x ON x.id = a.inventory_receipt_id WHERE x.item_id = ? AND r.denominator_unit_id = ? LIMIT ?"},
	{table: "rate", perUnit: true, find: "SELECT r.id FROM rate r JOIN item x ON x.unit_value_id = r.id WHERE x.id = ? AND r.denominator_unit_id = ? LIMIT ?"},
	{table: "rate", perUnit: true, find: "SELECT r.id FROM rate r JOIN item x ON x.unit_cost_id = r.id WHERE x.id = ? AND r.denominator_unit_id = ? LIMIT ?"},
	{table: "rate", find: "SELECT r.id FROM rate r JOIN item x ON x.burn_rate_id = r.id WHERE x.id = ? AND r.numerator_unit_id = ? LIMIT ?"},
}

// Money columns copied off the rates above (00088, 00089): id-keyed tables with an amount per a unit.
var restateMoneyColumns = []struct{ find, table, amount, perUnit string }{
	{"SELECT x.id FROM inventory_receipt x WHERE x.item_id = ? AND x.unit_cost_per_unit_id = ? LIMIT ?", "inventory_receipt", "unit_cost_amount", "unit_cost_per_unit_id"},
	{"SELECT a.id FROM inventory_allocation a JOIN inventory_receipt x ON x.id = a.inventory_receipt_id WHERE x.item_id = ? AND a.unit_cost_per_unit_id = ? LIMIT ?", "inventory_allocation", "unit_cost_amount", "unit_cost_per_unit_id"},
	{"SELECT x.id FROM item x WHERE x.id = ? AND x.sales_unit_price_per_unit_id = ? LIMIT ?", "item", "sales_unit_price_amount", "sales_unit_price_per_unit_id"},
	{"SELECT x.id FROM item x WHERE x.id = ? AND x.unit_cost_per_unit_id = ? LIMIT ?", "item", "unit_cost_amount", "unit_cost_per_unit_id"},
}

type stockingUnitRestateRepo struct {
	queries *sqlc.Queries
}

// NewStockingUnitRestateRepo rewrites an item's numbers when its stocking unit changes.
func NewStockingUnitRestateRepo(queries *sqlc.Queries) domain.StockingUnitRestateRepo {
	return &stockingUnitRestateRepo{queries: queries}
}

func (r *stockingUnitRestateRepo) Restate(ctx context.Context, _, itemID, fromUnitID, toUnitID string, factor decimal.Decimal) (int64, *apierror.APIError) {
	if fromUnitID == toUnitID || factor.IsZero() {
		return 0, nil
	}
	q := r.queries.DB()
	f := factor.String()
	var total int64
	for _, src := range restateSources {
		var set string
		switch {
		case src.table == "quantity":
			set = "UPDATE quantity SET value = value * ?, unit_id = ?, updated_at = NOW(3) WHERE unit_id = ? AND id IN "
		case src.perUnit:
			set = "UPDATE rate SET value = value / ?, denominator_unit_id = ?, updated_at = NOW(3) WHERE denominator_unit_id = ? AND id IN "
		default:
			set = "UPDATE rate SET value = value * ?, numerator_unit_id = ?, updated_at = NOW(3) WHERE numerator_unit_id = ? AND id IN "
		}
		n, apiErr := restateInChunks(ctx, q, src.find, set, itemID, fromUnitID, f, toUnitID)
		if apiErr != nil {
			return total, apiErr
		}
		total += n
	}
	for _, mc := range restateMoneyColumns {
		set := "UPDATE " + mc.table + " SET " + mc.amount + " = " + mc.amount + " / ?, " + mc.perUnit + " = ? WHERE " + mc.perUnit + " = ? AND id IN "
		n, apiErr := restateInChunks(ctx, q, mc.find, set, itemID, fromUnitID, f, toUnitID)
		if apiErr != nil {
			return total, apiErr
		}
		total += n
	}
	return total, nil
}

// restateInChunks rewrites the rows find returns, a chunk at a time, until none is left in the old unit.
func restateInChunks(ctx context.Context, q sqlc.DBTX, find, set, itemID, fromUnitID, factor, toUnitID string) (int64, *apierror.APIError) {
	var total int64
	for {
		ids, err := selectStrings(ctx, q, find, itemID, fromUnitID, restateChunk)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return total, apiErr
		}
		if len(ids) == 0 {
			return total, nil
		}
		args := append([]any{factor, toUnitID, fromUnitID}, stringsToAny(ids)...)
		res, err := q.ExecContext(ctx, set+"("+placeholders(len(ids))+")", args...)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return total, apiErr
		}
		n, _ := res.RowsAffected()
		total += n
		if len(ids) < restateChunk {
			return total, nil
		}
	}
}
