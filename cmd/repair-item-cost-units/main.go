// Command repair-item-cost-units puts item unit costs back into the unit their inventory is counted
// in, and restates the receipt layers that copied one while it was wrong.
//
// # What it repairs
//
// An item's unit cost is a rate: a value and the unit that value is per. The cost rollup used to
// write both halves at once, taking the denominator from whatever unit the production step producing
// the item happened to output in, and never checking that unit against the one the item is stocked
// in — its category's unit group base unit. Nothing rescaled the value when the denominator moved
// underneath it, so a cost that was per carton stayed the same number and started reading as per
// each. Inventory valuation converts the quantity into the cost's denominator before multiplying, so
// an eight-to-one carton reads eight times its worth from that moment on.
//
// Receipts carry a copy of the item's cost as it stood when the stock landed. Any receipt written
// while an item's cost was mislabelled holds the same mislabelled pair, and goes on valuing that
// stock wrongly after the item itself is fixed.
//
// # How it repairs
//
// By moving the denominator back, and leaving the value alone. That is the exact inverse of the
// write that broke it: the value was never rescaled, so it is still the number that was correct for
// the stocking unit. Rescaling it instead would preserve the wrong valuation rather than undo it.
//
// The same rule covers receipt layers, because a layer's cost is only ever a verbatim copy of an
// item cost — a layer whose denominator is not its item's stocking unit was copied from an item cost
// that was already wrong.
//
// # Run it in this order
//
// GET /v1/catalog/items/{id}/costs recomputes the rollup and writes the result back, so a single page
// view rewrites every row this command fixed. The sequence is not optional:
//
//  1. Deploy the rollup fix. Until it is live, the repair survives until the next page view.
//  2. Run this command. It relabels item costs leaves first, then the receipt layers.
//  3. Recompute the items it lists as produced by a production step, leaves first, by opening each
//     one's costs or by moving an input's price so the cost-basis consumer walks outwards from it.
//
// # Why the order matters
//
// The rollup prices a step from the unit costs of everything it consumes, so an item inherits its
// inputs' errors and no item is right until its inputs are. Repairs are applied leaves first, in
// production-flow dependency order, and the plan prints in that order too — repairing a finished
// good before the sub-assemblies under it only recomputes it from costs that are still wrong.
//
// A relabel is the whole repair for an item whose cost was mislabelled once. An item that compounded
// past that — one whose corrupted inputs were rolled into it — needs a recompute to converge on a
// value, and only the service can run one. Those items are listed on their own for step 3.
//
// Usage:
//
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/repair-item-cost-units
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/repair-item-cost-units --account ac_...
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/repair-item-cost-units --account ac_... --item itm_...
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/repair-item-cost-units --account ac_... --dry-run=false
//
// Flags:
//
//	--dry-run   report what would change; make no writes (default true — pass --dry-run=false to write).
//	--account   restrict to one account id.
//	--item      restrict to one item id, for verifying the effect on a single SKU first.
//	--limit     cap the number of item costs repaired (0 = no cap); receipt layers are not capped.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/env"
)

func main() {
	ctx := context.Background()
	if err := Run(ctx, os.Args, os.Getenv, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err) // #nosec G705 -- CLI stderr output, not web context
		os.Exit(1)
	}
}

// mislabelledCost is one rate whose denominator is not the stocking unit of the item it prices,
// together with what that costs: ValuationFactor is how many of the denominator's units one stocking
// unit holds, which is the factor by which the stored pair overstates a stocking unit's worth.
type mislabelledCost struct {
	AccountID     string
	ItemID        string
	SKU           string
	RateID        string
	Value         decimal.Decimal
	DenominatorID string
	DenominatorAb string
	StockingID    string
	StockingAb    string

	ValuationFactor decimal.Decimal

	// Derived items are produced by a production step, so their cost is the rollup's to restate; a
	// purchased item's cost is entered by hand and a relabel is the end of it.
	Derived bool
}

// mislabelledLayer is one receipt holding a copied cost that needs the same relabel, with enough of
// the receipt on it to check the money against a physical count before writing.
type mislabelledLayer struct {
	ReceiptID  string
	StatusCode string
	ReceivedAt time.Time
	Quantity   decimal.Decimal
	QuantityAb string

	// QuantityInStockingUnits converts the receipt's own quantity onto the item's stocking unit, which
	// is where the restated valuation is worked out; a receipt need not be counted in that unit.
	QuantityInStockingUnits decimal.Decimal

	Cost mislabelledCost
}

func Run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("repair-item-cost-units", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", true, "report what would change; make no writes")
	accountID := fs.String("account", "", "restrict to a single account id")
	itemID := fs.String("item", "", "restrict to a single item id")
	limit := fs.Int("limit", 0, "cap item costs repaired (0 = no cap); receipt layers are not capped")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	dbURL := env.GetEnv("DB_URL", getenv)
	if dbURL == "" {
		return fmt.Errorf("DB_URL is required")
	}
	dsn, err := normalizeDSN(dbURL)
	if err != nil {
		return err
	}

	sqlDB, err := db.NewDbPool(&db.Config{DBURI: dsn})
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer sqlDB.Close()

	p := &progress{out: stdout, start: time.Now()}
	if *accountID != "" {
		p.stagef("Restricted to account %s", *accountID)
	}
	if *itemID != "" {
		p.stagef("Restricted to item %s", *itemID)
	}

	items, err := findMislabelledItemCosts(ctx, sqlDB, *accountID, *itemID, p)
	if err != nil {
		return err
	}

	items, err = orderLeavesFirst(ctx, sqlDB, items, p)
	if err != nil {
		return err
	}
	if *limit > 0 && len(items) > *limit {
		items = items[:*limit]
		p.stagef("Capped at %d item(s) by --limit", *limit)
	}

	layers, err := findMislabelledLayers(ctx, sqlDB, *accountID, *itemID, p)
	if err != nil {
		return err
	}

	p.stage("Summary")
	fmt.Fprintf(stdout, "  Item costs to relabel:     %d\n", len(items))
	fmt.Fprintf(stdout, "  Receipt layers to restate: %d\n", len(layers))
	fmt.Fprintf(stdout, "  Items needing a recompute: %d\n", countDerived(items))

	printItemPlan(stdout, items)
	printLayerPlan(stdout, layers)
	printRecomputeList(stdout, items)

	if len(items) == 0 && len(layers) == 0 {
		p.stage("Nothing to do")
		return nil
	}

	if *dryRun {
		p.stage("--dry-run: no writes made (pass --dry-run=false to apply)")
		return nil
	}

	if err := apply(ctx, sqlDB, items, layers, p); err != nil {
		return err
	}

	p.stage("Done")
	fmt.Fprintf(stdout, "  Relabelled %d item unit cost(s).\n", len(items))
	fmt.Fprintf(stdout, "  Restated %d receipt layer(s).\n", len(layers))
	fmt.Fprintf(stdout, "  %d item(s) still need a cost recompute; see the list above.\n", countDerived(items))
	return nil
}

// selectMislabelledCosts is the shared projection behind both scans. Both ask the same question of a
// rate — is its denominator the stocking unit of the item it prices — and differ only in which rate
// they reach it through, so the columns and their order are fixed here and read back by scanCost.
const selectMislabelledCosts = `
       i.account_id, i.id, i.sku, r.id, r.value,
       r.denominator_unit_id, du.abbreviation,
       ug.base_unit_id, bu.abbreviation,
       CAST((bu.ratio_numerator / bu.ratio_denominator) / (du.ratio_numerator / du.ratio_denominator) AS DECIMAL(65,30)),
       EXISTS (SELECT 1 FROM production pr WHERE pr.item_id = i.id AND pr.production_step_id IS NOT NULL)`

const joinItemStockingUnit = `
JOIN item_category ic ON ic.id = i.item_category_id
JOIN unit_group ug ON ug.id = ic.unit_group_id
JOIN unit bu ON bu.id = ug.base_unit_id
JOIN unit du ON du.id = r.denominator_unit_id`

func scanCost(rows *sql.Rows, extra ...any) (mislabelledCost, error) {
	var c mislabelledCost
	var value, factor string
	dest := []any{
		&c.AccountID, &c.ItemID, &c.SKU, &c.RateID, &value,
		&c.DenominatorID, &c.DenominatorAb,
		&c.StockingID, &c.StockingAb,
		&factor, &c.Derived,
	}
	dest = append(dest, extra...)
	if err := rows.Scan(dest...); err != nil {
		return mislabelledCost{}, fmt.Errorf("scan mislabelled cost: %w", err)
	}
	var err error
	if c.Value, err = decimal.NewFromString(value); err != nil {
		return mislabelledCost{}, fmt.Errorf("parse cost value for %s: %w", c.RateID, err)
	}
	if c.ValuationFactor, err = decimal.NewFromString(factor); err != nil {
		return mislabelledCost{}, fmt.Errorf("parse valuation factor for %s: %w", c.RateID, err)
	}
	return c, nil
}

// findMislabelledItemCosts lists the item unit costs denominated in something other than the unit the
// item is stocked in.
func findMislabelledItemCosts(ctx context.Context, sqlDB *sql.DB, accountID, itemID string, p *progress) ([]mislabelledCost, error) {
	var where strings.Builder
	var params []any
	if accountID != "" {
		where.WriteString(" AND i.account_id = ?")
		params = append(params, accountID)
	}
	if itemID != "" {
		where.WriteString(" AND i.id = ?")
		params = append(params, itemID)
	}

	// #nosec G202 -- the only interpolation is `where`, built above from fixed literals; the account
	// and item ids it filters on are bound as parameters.
	query := `SELECT` + selectMislabelledCosts + `
FROM item i
JOIN rate r ON r.id = i.unit_cost_id` + joinItemStockingUnit + `
WHERE i.deleted_at IS NULL
  AND r.denominator_unit_id <> ug.base_unit_id` + where.String() + `
ORDER BY i.sku`

	p.stage("Scanning item unit costs for a denominator outside the item's stocking unit...")

	rows, err := sqlDB.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("find mislabelled item costs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []mislabelledCost
	for rows.Next() {
		c, scanErr := scanCost(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mislabelled item costs: %w", err)
	}

	p.stagef("Found %d mislabelled item cost(s)", len(out))
	return out, nil
}

// findMislabelledLayers lists the receipt layers whose copied cost carries the same defect.
//
// Scanned against the whole account rather than against the items this run repairs: a layer written
// while its item was mislabelled goes on valuing stock wrongly after the item's own cost is put
// right, so it has to be reachable on a later run too.
func findMislabelledLayers(ctx context.Context, sqlDB *sql.DB, accountID, itemID string, p *progress) ([]mislabelledLayer, error) {
	var where strings.Builder
	var params []any
	if accountID != "" {
		where.WriteString(" AND i.account_id = ?")
		params = append(params, accountID)
	}
	if itemID != "" {
		where.WriteString(" AND i.id = ?")
		params = append(params, itemID)
	}

	p.stage("Scanning receipt layers for costs copied while the item was mislabelled...")

	// #nosec G202 -- the only interpolation is `where`, built above from fixed literals; the account
	// and item ids it filters on are bound as parameters.
	query := `SELECT` + selectMislabelledCosts + `,
       ir.id, ir.status_code, ir.received_at, q.value, qu.abbreviation,
       CAST((qu.ratio_numerator / qu.ratio_denominator) / (bu.ratio_numerator / bu.ratio_denominator) AS DECIMAL(65,30))
FROM inventory_receipt ir
JOIN item i ON i.id = ir.item_id
JOIN rate r ON r.id = ir.unit_cost_id` + joinItemStockingUnit + `
JOIN quantity q ON q.id = ir.quantity_id
JOIN unit qu ON qu.id = q.unit_id
WHERE i.deleted_at IS NULL
  AND r.denominator_unit_id <> ug.base_unit_id` + where.String() + `
ORDER BY ir.received_at`

	rows, err := sqlDB.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("find mislabelled receipt layers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []mislabelledLayer
	for rows.Next() {
		var l mislabelledLayer
		var quantity, inStockingUnits string
		cost, scanErr := scanCost(rows, &l.ReceiptID, &l.StatusCode, &l.ReceivedAt, &quantity, &l.QuantityAb, &inStockingUnits)
		if scanErr != nil {
			return nil, scanErr
		}
		l.Cost = cost
		if l.Quantity, err = decimal.NewFromString(quantity); err != nil {
			return nil, fmt.Errorf("parse receipt quantity for %s: %w", l.ReceiptID, err)
		}
		if l.QuantityInStockingUnits, err = decimal.NewFromString(inStockingUnits); err != nil {
			return nil, fmt.Errorf("parse receipt quantity conversion for %s: %w", l.ReceiptID, err)
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mislabelled receipt layers: %w", err)
	}

	p.stagef("Found %d mislabelled receipt layer(s)", len(out))
	return out, nil
}

// orderLeavesFirst sorts the repairs so an item follows everything its production steps consume.
//
// Only edges between items being repaired matter: an input that is already correctly denominated
// constrains nothing. Items left in a cycle — a routing that consumes its own output — keep their
// scan order and are appended last, since no order satisfies them and refusing to repair them would
// leave the rest of the chain reading against a mislabelled input.
func orderLeavesFirst(ctx context.Context, sqlDB *sql.DB, items []mislabelledCost, p *progress) ([]mislabelledCost, error) {
	if len(items) < 2 {
		return items, nil
	}

	index := make(map[string]int, len(items))
	accounts := make(map[string]struct{}, len(items))
	for i, it := range items {
		index[it.ItemID] = i
		accounts[it.AccountID] = struct{}{}
	}

	p.stage("Ordering repairs by production-flow depth (leaves first)...")

	inputsOf := make(map[string]map[string]struct{}, len(items))
	for accountID := range accounts {
		const query = `
SELECT DISTINCT c.item_id, pr.item_id
FROM consumption c
JOIN production pr ON pr.production_step_id = c.production_step_id
JOIN production_step ps ON ps.id = c.production_step_id
WHERE ps.account_id = ?`
		rows, err := sqlDB.QueryContext(ctx, query, accountID)
		if err != nil {
			return nil, fmt.Errorf("load item dependency edges: %w", err)
		}
		for rows.Next() {
			var input, output string
			if err := rows.Scan(&input, &output); err != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("scan item dependency edge: %w", err)
			}
			if input == output {
				continue
			}
			if _, ok := index[input]; !ok {
				continue
			}
			if _, ok := index[output]; !ok {
				continue
			}
			if inputsOf[output] == nil {
				inputsOf[output] = map[string]struct{}{}
			}
			inputsOf[output][input] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("iterate item dependency edges: %w", err)
		}
		_ = rows.Close()
	}

	ordered := make([]mislabelledCost, 0, len(items))
	placed := make(map[string]struct{}, len(items))
	for len(ordered) < len(items) {
		ready := make([]string, 0, len(items))
		for _, it := range items {
			if _, done := placed[it.ItemID]; done {
				continue
			}
			blocked := false
			for input := range inputsOf[it.ItemID] {
				if _, done := placed[input]; !done {
					blocked = true
					break
				}
			}
			if !blocked {
				ready = append(ready, it.ItemID)
			}
		}
		if len(ready) == 0 {
			for _, it := range items {
				if _, done := placed[it.ItemID]; !done {
					ordered = append(ordered, it)
					placed[it.ItemID] = struct{}{}
				}
			}
			p.stage("  Some items consume each other; appending them in scan order")
			break
		}
		sort.Strings(ready)
		for _, itemID := range ready {
			ordered = append(ordered, items[index[itemID]])
			placed[itemID] = struct{}{}
		}
	}

	return ordered, nil
}

func countDerived(items []mislabelledCost) int {
	n := 0
	for _, it := range items {
		if it.Derived {
			n++
		}
	}
	return n
}

// printItemPlan lists the relabels in the order they will be applied, with the factor each takes off
// the item's valuation — which is what a reviewer checks against a known-good price before writing.
func printItemPlan(out io.Writer, items []mislabelledCost) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(out, "\n  Item unit costs to relabel (leaves first; SKU, value, denominator -> stocking unit, valuation /factor):\n")
	for i, it := range items {
		if i >= planLines {
			fmt.Fprintf(out, "    ... and %d more\n", len(items)-planLines)
			break
		}
		fmt.Fprintf(out, "    %-24s %-22s %s -> %s   valuation /%s\n",
			it.SKU, it.Value.String(), it.DenominatorAb, it.StockingAb, it.ValuationFactor.String())
	}
	fmt.Fprintln(out)
}

// printLayerPlan lists the receipt layers and what each is worth before and after, so the balance
// sheet effect of the run is on the page rather than inferred from it.
func printLayerPlan(out io.Writer, layers []mislabelledLayer) {
	if len(layers) == 0 {
		return
	}
	fmt.Fprintf(out, "  Receipt layers to restate (receipt, SKU, quantity, valued at -> valued at):\n")
	for i, l := range layers {
		if i >= planLines {
			fmt.Fprintf(out, "    ... and %d more\n", len(layers)-planLines)
			break
		}
		after := l.Quantity.Mul(l.QuantityInStockingUnits).Mul(l.Cost.Value)
		before := after.Mul(l.Cost.ValuationFactor)
		fmt.Fprintf(out, "    %-22s %-24s %s %s   %s -> %s   [%s]\n",
			l.ReceiptID, l.Cost.SKU, l.Quantity.String(), l.QuantityAb,
			before.StringFixed(2), after.StringFixed(2), l.StatusCode)
	}
	fmt.Fprintln(out)
}

// printRecomputeList names the items a relabel does not finish. Their cost was rolled up from inputs
// that were themselves mislabelled, so the number is wrong by more than the label and only a
// recompute against repaired inputs settles it.
func printRecomputeList(out io.Writer, items []mislabelledCost) {
	derived := make([]mislabelledCost, 0, len(items))
	for _, it := range items {
		if it.Derived {
			derived = append(derived, it)
		}
	}
	if len(derived) == 0 {
		return
	}
	fmt.Fprintf(out, "  Produced by a production step — recompute these after the run, leaves first:\n")
	for i, it := range derived {
		if i >= planLines {
			fmt.Fprintf(out, "    ... and %d more\n", len(derived)-planLines)
			break
		}
		fmt.Fprintf(out, "    %-24s %s\n", it.SKU, it.ItemID)
	}
	fmt.Fprintln(out)
}

const planLines = 40

// apply relabels every rate in one transaction, items before the layers that copied them, so a
// failure leaves the costs as they were rather than half-moved.
func apply(ctx context.Context, sqlDB *sql.DB, items []mislabelledCost, layers []mislabelledLayer, p *progress) error {
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	p.stage("Applying (single transaction; nothing commits until the end)")

	p.stagef("  Relabelling %d item unit cost(s), leaves first...", len(items))
	for i, it := range items {
		if err := relabel(ctx, tx, it.RateID, it.StockingID); err != nil {
			return err
		}
		p.step(i+1, len(items), "item costs relabelled")
	}

	p.stagef("  Restating %d receipt layer(s)...", len(layers))
	for i, l := range layers {
		if err := relabel(ctx, tx, l.Cost.RateID, l.Cost.StockingID); err != nil {
			return err
		}
		p.step(i+1, len(layers), "receipt layers restated")
	}

	p.stage("  Committing...")
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// relabel moves a rate's denominator without touching its value, which is the whole repair: the value
// was already the number the stocking unit wanted.
func relabel(ctx context.Context, tx *sql.Tx, rateID, stockingUnitID string) error {
	const query = `UPDATE rate SET denominator_unit_id = ?, updated_at = NOW(3) WHERE id = ?`
	if _, err := tx.ExecContext(ctx, query, stockingUnitID, rateID); err != nil {
		return fmt.Errorf("relabel rate %s: %w", rateID, err)
	}
	return nil
}

// progress reports what the command is doing as it goes: a full run against a remote database spends
// minutes in these loops and is indistinguishable from a hang without it. Every line carries elapsed
// time so a run that is merely slow can be told apart from one that is stuck.
type progress struct {
	out   io.Writer
	start time.Time
}

func (p *progress) elapsed() string {
	return time.Since(p.start).Truncate(time.Second).String()
}

func (p *progress) stage(msg string) {
	fmt.Fprintf(p.out, "[%s] %s\n", p.elapsed(), msg)
}

func (p *progress) stagef(format string, args ...any) {
	p.stage(fmt.Sprintf(format, args...))
}

// step prints a counter line every progressEvery items, and always on the final item, so the last
// line of a loop matches its total rather than stopping short at the nearest interval.
func (p *progress) step(done, total int, noun string) {
	if done%progressEvery != 0 && done != total {
		return
	}
	fmt.Fprintf(p.out, "[%s]   %d/%d %s\n", p.elapsed(), done, total, noun)
}

const progressEvery = 100

func chunks(ids []string, size int) func(func([]string) bool) {
	return func(yield func([]string) bool) {
		for start := 0; start < len(ids); start += size {
			end := min(start+size, len(ids))
			if !yield(ids[start:end]) {
				return
			}
		}
	}
}

// normalizeDSN accepts either a driver DSN (user:pass@tcp(host:port)/db) — returned unchanged — or a
// PlanetScale/URL form (mysql://user:pass@host[:port]/db[?...]), which it rewrites into the DSN form
// db.NewDbPool expects. Remote hosts get tls=true, since PlanetScale requires TLS. Mirrors the helper
// in cmd/dedupe-inventory-allocations; both are one-off commands in package main and cannot share it.
func normalizeDSN(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("DB_URL is empty")
	}
	if strings.Contains(raw, "@tcp(") || strings.Contains(raw, "@unix(") {
		return raw, nil
	}
	if !strings.HasPrefix(raw, "mysql://") {
		return "", fmt.Errorf("unrecognized DB_URL form; expected a mysql://... URL or a user:pass@tcp(host:port)/db DSN")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parsing DB_URL: %w", err)
	}
	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("DB_URL has no host")
	}
	port := u.Port()
	if port == "" {
		port = "3306"
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	userInfo := u.User.Username()
	if pass, ok := u.User.Password(); ok && pass != "" {
		userInfo += ":" + pass
	}

	dsn := fmt.Sprintf("%s@tcp(%s:%s)/%s", userInfo, host, port, dbName)
	if host != "localhost" && host != "127.0.0.1" {
		dsn += "?tls=true"
	}
	return dsn, nil
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func toAny(ids []string) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}
