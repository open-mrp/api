package repository

import (
	"context"
	gosql "database/sql"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/db/backfill"
)

// Money is moving off quantity and rate rows in a currency unit onto columns of the row that owns it
// (migration 00088): an amount, a currency code and, for a price, the unit it is per. Until every writer
// has moved, the quantity and rate rows stay the record and the columns are a copy of them: each write
// to an owned row, or to the link to it, re-copies in the same transaction, and MoneyColumnBackfill
// fills the rows written before the columns existed. Reading the rows rather than the columns keeps
// every read right while a deploy rolls out, when pods that never write the columns still serve
// writes. An order line's money copies live with its other copies (orderLineCopies).

// currencyOf is the currency of an amount stored in the quantity or rate row alias r, or NULL when the
// row is missing. Every amount is read as usd whatever unit its row is in, exactly as the API has presented
// it, so the copy matches what readers show.
func currencyOf(r string) string {
	return "CASE WHEN " + r + ".id IS NOT NULL THEN 'usd' END"
}

var (
	itemMoneyCopies = copyTable{
		table: "item", alias: "i",
		joins: `LEFT JOIN rate uv ON uv.id = i.unit_value_id
LEFT JOIN rate uc ON uc.id = i.unit_cost_id`,
		copies: []string{"sales_unit_price_amount", "sales_unit_price_currency", "sales_unit_price_per_unit_id",
			"unit_cost_amount", "unit_cost_currency", "unit_cost_per_unit_id"},
		sources: []string{"uv.value", currencyOf("uv"), "uv.denominator_unit_id",
			"uc.value", currencyOf("uc"), "uc.denominator_unit_id"},
		uncopied: []string{"sales_unit_price_currency", "unit_cost_currency"},
	}
	creditLimitCopies = copyTable{
		table: "account_relation", alias: "ar",
		joins:    "LEFT JOIN quantity cl ON cl.id = ar.credit_limit_id",
		copies:   []string{"credit_limit_amount", "credit_limit_currency"},
		sources:  []string{"cl.value", currencyOf("cl")},
		uncopied: []string{"credit_limit_currency"},
	}
	shippingTermMoneyCopies = copyTable{
		table: "shipping_term", alias: "st",
		joins: `LEFT JOIN quantity fr ON fr.id = st.flat_rate_id
LEFT JOIN quantity mo ON mo.id = st.minimum_order_id`,
		copies:   []string{"flat_rate_amount", "flat_rate_currency", "minimum_order_value_amount", "minimum_order_value_currency"},
		sources:  []string{"fr.value", currencyOf("fr"), "mo.value", currencyOf("mo")},
		uncopied: []string{"flat_rate_currency", "minimum_order_value_currency"},
	}
	shippingCaseFreightCopies = copyTable{
		table: "shipping_case", alias: "sc",
		joins:    "LEFT JOIN quantity fa ON fa.id = sc.freight_amount_id",
		copies:   []string{"freight_amount", "freight_currency"},
		sources:  []string{"fa.value", currencyOf("fa")},
		uncopied: []string{"freight_currency"},
	}
	targetAmountCopies = copyTable{
		table: "target", alias: "tg",
		joins:    "LEFT JOIN quantity ta ON ta.id = tg.amount_id",
		copies:   []string{"amount", "currency"},
		sources:  []string{"ta.value", currencyOf("ta")},
		uncopied: []string{"currency"},
	}
	productLineTargetAmountCopies = copyTable{
		table: "product_line_target", alias: "plt",
		joins:    "LEFT JOIN quantity ta ON ta.id = plt.amount_id",
		copies:   []string{"amount", "currency"},
		sources:  []string{"ta.value", currencyOf("ta")},
		uncopied: []string{"currency"},
	}
	// orderLineMoneyCopies backfills every copy of an order line still missing its price's currency: the
	// lines the line-copy backfill filled, or no writer has touched, since the currency copies existed.
	orderLineMoneyCopies = func() copyTable {
		t := orderLineCopies
		t.uncopied = []string{"unit_price_currency"}
		return t
	}()
)

// byID is the predicate for t's rows with the given ids.
func byID(t copyTable, ids []string) (string, []any) {
	return t.alias + ".id IN (" + placeholders(len(ids)) + ")", stringsToAny(ids)
}

// syncItemMoney re-copies the given items' sales price and unit cost.
func syncItemMoney(ctx context.Context, q sqlc.DBTX, itemIDs ...string) error {
	if len(itemIDs) == 0 {
		return nil
	}
	where, args := byID(itemMoneyCopies, itemIDs)
	return syncCopies(ctx, q, itemMoneyCopies, where, args...)
}

// syncItemMoneyByRate re-copies the item owning a rate row as its sales price or unit cost, if one does.
// Each is its own unique-key lookup.
func syncItemMoneyByRate(ctx context.Context, q sqlc.DBTX, rateID string) error {
	if err := syncCopies(ctx, q, itemMoneyCopies, "i.unit_value_id = ?", rateID); err != nil {
		return err
	}
	return syncCopies(ctx, q, itemMoneyCopies, "i.unit_cost_id = ?", rateID)
}

// syncCreditLimit re-copies a customer or supplier relation's credit limit, clearing it when the relation has none.
func syncCreditLimit(ctx context.Context, q sqlc.DBTX, relationID string) error {
	return syncCopies(ctx, q, creditLimitCopies, "ar.id = ?", relationID)
}

// syncShippingTermMoney re-copies a shipping term's flat rate and minimum order value.
func syncShippingTermMoney(ctx context.Context, q sqlc.DBTX, termID string) error {
	return syncCopies(ctx, q, shippingTermMoneyCopies, "st.id = ?", termID)
}

// syncShippingCaseFreight re-copies the given shipping cases' freight amounts.
func syncShippingCaseFreight(ctx context.Context, q sqlc.DBTX, caseIDs ...string) error {
	if len(caseIDs) == 0 {
		return nil
	}
	where, args := byID(shippingCaseFreightCopies, caseIDs)
	return syncCopies(ctx, q, shippingCaseFreightCopies, where, args...)
}

// syncShippingCaseFreightByQuantity re-copies the shipping case owning a quantity row as its freight, if one does.
func syncShippingCaseFreightByQuantity(ctx context.Context, q sqlc.DBTX, quantityID string) error {
	return syncCopies(ctx, q, shippingCaseFreightCopies, "sc.freight_amount_id = ?", quantityID)
}

// syncTargetAmount re-copies a sales target's amount.
func syncTargetAmount(ctx context.Context, q sqlc.DBTX, targetID string) error {
	return syncCopies(ctx, q, targetAmountCopies, "tg.id = ?", targetID)
}

// syncTargetAmountByQuantity re-copies the sales target owning a quantity row as its amount, if one does.
func syncTargetAmountByQuantity(ctx context.Context, q sqlc.DBTX, quantityID string) error {
	return syncCopies(ctx, q, targetAmountCopies, "tg.amount_id = ?", quantityID)
}

// syncAccountMoneyCopies re-copies the money of every item, relation, shipping term, shipping case and
// target in the account: for rows written by SQL that bypasses the repositories (the sandbox seed).
func syncAccountMoneyCopies(ctx context.Context, q sqlc.DBTX, accountID string) error {
	for _, s := range []struct {
		t     copyTable
		where string
	}{
		{itemMoneyCopies, "i.account_id = ?"},
		{creditLimitCopies, "ar.owner_account_id = ?"},
		{shippingTermMoneyCopies, "st.account_id = ?"},
		{shippingCaseFreightCopies, "sc.account_id = ?"},
		{targetAmountCopies, "tg.account_id = ?"},
		{productLineTargetAmountCopies, "plt.account_id = ?"},
	} {
		if err := syncCopies(ctx, q, s.t, s.where, accountID); err != nil {
			return err
		}
	}
	return nil
}

// MoneyColumnBackfill fills one table's money columns for the rows written before they existed, walking
// the table by id. It reads from read (a replica) and writes to write (the primary). A write only fills a
// row that still has no copy, so a row a writer copied since the replica read keeps the writer's newer
// values, and a re-run batch changes nothing.
type MoneyColumnBackfill struct {
	Name        string
	t           copyTable
	read, write sqlc.DBTX
}

// NewMoneyColumnBackfills returns a backfill for each table whose money moved onto columns.
func NewMoneyColumnBackfills(read, write *sqlc.Queries) []*MoneyColumnBackfill {
	tables := []struct {
		name string
		t    copyTable
	}{
		{"money_columns_item", itemMoneyCopies},
		{"money_columns_sales_order_line", orderLineMoneyCopies},
		{"money_columns_account_relation", creditLimitCopies},
		{"money_columns_shipping_term", shippingTermMoneyCopies},
		{"money_columns_shipping_case", shippingCaseFreightCopies},
		{"money_columns_target", targetAmountCopies},
		{"money_columns_product_line_target", productLineTargetAmountCopies},
	}
	out := make([]*MoneyColumnBackfill, len(tables))
	for i, tb := range tables {
		out[i] = &MoneyColumnBackfill{Name: tb.name, t: tb.t, read: read.DB(), write: write.DB()}
	}
	return out
}

// Batch is a backfill.Batch over the table's ids.
func (b *MoneyColumnBackfill) Batch(ctx context.Context, m *backfill.Meter, cursor string, limit int) (string, int, bool, error) {
	var ids []string
	if err := m.Time(func() (err error) {
		ids, err = selectStrings(ctx, b.read, "SELECT id FROM "+b.t.table+" WHERE id > ? ORDER BY id LIMIT ?", cursor, limit)
		return err
	}); err != nil {
		return cursor, 0, false, err
	}
	if len(ids) == 0 {
		return cursor, 0, true, nil
	}
	where, args := byID(b.t, ids)
	var rows []copyRow
	if err := m.Time(func() (err error) {
		rows, err = readCopies(ctx, b.read, b.t, where+" AND "+b.t.uncopiedWhere(b.t.alias), args...)
		return err
	}); err != nil {
		return cursor, 0, false, err
	}
	if err := m.Time(func() error { return fillCopies(ctx, b.write, b.t, rows) }); err != nil {
		return cursor, 0, false, err
	}
	return ids[len(ids)-1], len(ids), len(ids) < limit, nil
}

// moneyCopyTables are every table whose money moved onto columns, for MoneyColumnDrift.
var moneyCopyTables = []copyTable{itemMoneyCopies, orderLineCopies, creditLimitCopies, shippingTermMoneyCopies,
	shippingCaseFreightCopies, targetAmountCopies, productLineTargetAmountCopies}

// MoneyColumnDrift counts, per table, the rows whose money columns disagree with the quantity and rate rows
// they copy, skipping rows the backfill has yet to fill. Every count must be zero before readers move onto
// the columns. It scans whole tables: run it against a replica or a test database, never on a request path.
func MoneyColumnDrift(ctx context.Context, q sqlc.DBTX) (map[string]int64, error) {
	out := make(map[string]int64, len(moneyCopyTables))
	for _, t := range moneyCopyTables {
		var n int64
		if err := q.QueryRowContext(ctx, moneyDriftQuery(t)).Scan(&n); err != nil {
			return nil, err
		}
		out[t.table] = n
	}
	return out, nil
}

func moneyDriftQuery(t copyTable) string {
	same := make([]string, len(t.copies))
	for i, c := range t.copies {
		same[i] = t.alias + "." + c + " <=> " + t.sources[i]
	}
	return "SELECT COUNT(*) FROM " + t.table + " " + t.alias + "\n" + t.joins + "\nWHERE NOT (" + t.uncopiedWhere(t.alias) +
		") AND NOT (" + strings.Join(same, " AND ") + ")"
}

// storedCurrency is a money copy's currency. A row the backfill has yet to reach has no copy, and its
// amount, like every amount in a quantity or rate row, is usd.
func storedCurrency(c gosql.NullString) constants.Currency {
	if c.Valid && c.String != "" {
		return constants.Currency(c.String)
	}
	return constants.CurrencyUSD
}

// storedMoney is an amount read from its quantity row with the currency copied beside it.
func storedMoney(amount string, currency gosql.NullString) domain.Money {
	return domain.Money{Amount: amount, Currency: storedCurrency(currency)}
}

// storedMoneyPtr is storedMoney for an amount that may be absent.
func storedMoneyPtr(amount gosql.NullString, currency gosql.NullString) *domain.Money {
	if !amount.Valid {
		return nil
	}
	m := storedMoney(amount.String, currency)
	return &m
}

// storedRatePrice is storedUnitPrice for a rate row that may be missing.
func storedRatePrice(r *domain.Rate, currency gosql.NullString) *domain.UnitPrice {
	if r == nil {
		return nil
	}
	p := storedUnitPrice(r.Value, currency, r.DenominatorUnitID)
	return &p
}

// storedUnitPrice is a price read from its rate row with the currency copied beside it.
func storedUnitPrice(amount string, currency gosql.NullString, perUnitID string) domain.UnitPrice {
	return domain.UnitPrice{Amount: amount, Currency: storedCurrency(currency), PerUnitID: perUnitID}
}
