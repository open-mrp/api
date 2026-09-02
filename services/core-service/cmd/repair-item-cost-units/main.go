// Command repair-item-cost-units puts item unit costs back into the unit their inventory is counted
// in, restates the receipt layers that copied one while it was wrong, and recomputes every cost the
// rollup derives, so one run leaves the account's costing correct rather than merely relabelled.
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
// The same release fixed a second defect with no visible signature at all: the material term
// multiplied consumption quantities by consumed items' unit costs without normalising either side,
// so a carton drawn against a per-each cost priced at an eighth. An item hit by that has a correct
// denominator and a wrong number, which no scan of the rate row can distinguish from a real price.
// The only repair for it is to recompute, which is why the recompute pass covers every derived item
// in the account rather than only the ones found mislabelled.
//
// # How it repairs
//
// Mislabelled denominators move back, and the value is left alone. That is the exact inverse of the
// write that broke it: the value was never rescaled, so it is still the number that was correct for
// the stocking unit. Rescaling it instead would preserve the wrong valuation rather than undo it.
// The same rule covers receipt layers, because a layer's cost is only ever a verbatim copy of an item
// cost — a layer whose denominator is not its item's stocking unit was copied from an item cost that
// was already wrong.
//
// Derived costs are then recomputed through the service's own rollup rather than any arithmetic of
// this command's, so there is one costing implementation and this cannot drift from it.
//
// # Order, and why it is not negotiable
//
// The rollup prices a step from the unit costs of everything it consumes, so an item inherits its
// inputs' errors and no item is right until its inputs are. Both passes run leaves first, in
// production-flow dependency order: repairing or recomputing a finished good before the
// sub-assemblies under it only restates it from costs that are still wrong.
//
// Relabelling comes before recomputing for the same reason. A recompute reads its inputs' stored
// costs, so the labels have to be right first or the rollup consumes the corruption it is meant to
// clear.
//
// # Deploy before you run this
//
// GET /v1/catalog/items/{id}/costs recomputes the rollup and writes the result back, so a single page
// view against a service still running the old rollup re-corrupts every row this fixed. The binary
// carries the corrected rollup, but the deployment is what page views hit. Deploy first.
//
// # What it cannot repair
//
// A receipt layer whose denominator is correct but whose value came from a mis-scaled item cost.
// There is no second source of truth to reconcile it against, and it is indistinguishable from a real
// price change. Historical valuations that used one stay as they are; only live item costs and
// label-corrupted layers are recoverable.
//
// Usage:
//
//	DB_URL=<dsn-or-mysql-url> go run ./services/core-service/cmd/repair-item-cost-units --account ac_...
//	DB_URL=<dsn-or-mysql-url> go run ./services/core-service/cmd/repair-item-cost-units --account ac_... --item itm_...
//	DB_URL=<dsn-or-mysql-url> go run ./services/core-service/cmd/repair-item-cost-units --account ac_... --dry-run=false
//
// Flags:
//
//	--dry-run          report what would change; make no writes (default true — pass --dry-run=false to write).
//	--account          the account to repair (required unless --skip-recompute).
//	--item             restrict to one item id, for verifying the effect on a single SKU first.
//	--limit            cap the number of item costs relabelled (0 = no cap); layers and recomputes are not capped.
//	--skip-recompute   relabel only, and print the recompute plan instead of running it.
//	--halt-on-error    stop at the first item the rollup refuses, rather than recording it and going on.
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

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	"github.com/open-mrp/api/services/core-service/internal/service"
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

// itemRef names an item in the two places order matters: the recompute plan and its report.
type itemRef struct {
	ID  string
	SKU string
}

// recomputeResult is what one item's rollup did, as the operator needs to read it: the cost before
// and after, or the reason the rollup declined to write.
type recomputeResult struct {
	Item   itemRef
	Before string
	After  string
	Err    string
}

func Run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("repair-item-cost-units", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", true, "report what would change; make no writes")
	accountID := fs.String("account", "", "the account to repair (required unless --skip-recompute)")
	itemID := fs.String("item", "", "restrict to a single item id")
	limit := fs.Int("limit", 0, "cap item costs relabelled (0 = no cap); layers and recomputes are not capped")
	skipRecompute := fs.Bool("skip-recompute", false, "relabel only; print the recompute plan instead of running it")
	haltOnError := fs.Bool("halt-on-error", false, "stop at the first item the rollup refuses")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	// Recomputing every derived item the database holds, across every tenant, is not something to
	// arrive at by forgetting a flag.
	if !*skipRecompute && *accountID == "" {
		return fmt.Errorf("--account is required for the recompute pass; pass --skip-recompute to relabel only")
	}

	dbURL := env.GetEnv("DB_URL", getenv)
	if dbURL == "" {
		return fmt.Errorf("DB_URL is required")
	}
	dsn, err := normalizeDSN(dbURL)
	if err != nil {
		return err
	}

	pool, err := db.NewDbPool(&db.Config{DBURI: dsn})
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	p := &progress{out: stdout, start: time.Now()}
	if *accountID != "" {
		p.stagef("Restricted to account %s", *accountID)
	}
	if *itemID != "" {
		p.stagef("Restricted to item %s", *itemID)
		if !*skipRecompute {
			p.stage("  NOTE: one item recomputes against inputs this run is not repairing, so its cost is only")
			p.stage("        as good as they are. Run the account whole before trusting the number.")
		}
	}

	items, err := findMislabelledItemCosts(ctx, pool, *accountID, *itemID, p)
	if err != nil {
		return err
	}
	items, err = orderCostsLeavesFirst(ctx, pool, items, p)
	if err != nil {
		return err
	}
	if *limit > 0 && len(items) > *limit {
		items = items[:*limit]
		p.stagef("Capped at %d item cost(s) by --limit", *limit)
	}

	layers, err := findMislabelledLayers(ctx, pool, *accountID, *itemID, p)
	if err != nil {
		return err
	}

	var derived []itemRef
	if !*skipRecompute {
		derived, err = findDerivedItems(ctx, pool, *accountID, *itemID, p)
		if err != nil {
			return err
		}
	}

	p.stage("Summary")
	fmt.Fprintf(stdout, "  Item costs to relabel:     %d\n", len(items))
	fmt.Fprintf(stdout, "  Receipt layers to restate: %d\n", len(layers))
	fmt.Fprintf(stdout, "  Derived items to recompute: %d\n", len(derived))

	printItemPlan(stdout, items)
	printLayerPlan(stdout, layers)
	printRecomputePlan(stdout, derived, *skipRecompute)

	if len(items) == 0 && len(layers) == 0 && len(derived) == 0 {
		p.stage("Nothing to do")
		return nil
	}

	if *dryRun {
		p.stage("--dry-run: no writes made (pass --dry-run=false to apply)")
		return nil
	}

	if len(items) > 0 || len(layers) > 0 {
		if err := applyRelabels(ctx, pool, items, layers, p); err != nil {
			return err
		}
	}

	var results []recomputeResult
	if len(derived) > 0 {
		results, err = recomputeAll(ctx, pool, *accountID, derived, *haltOnError, p)
		if err != nil {
			return err
		}
	}

	remaining, err := findMislabelledItemCosts(ctx, pool, *accountID, *itemID, p)
	if err != nil {
		return err
	}

	p.stage("Done")
	fmt.Fprintf(stdout, "  Relabelled %d item unit cost(s).\n", len(items))
	fmt.Fprintf(stdout, "  Restated %d receipt layer(s).\n", len(layers))
	printRecomputeResults(stdout, results)
	if len(remaining) > 0 {
		fmt.Fprintf(stdout, "  WARNING: %d item cost(s) still denominated outside their stocking unit — re-run.\n", len(remaining))
	} else {
		fmt.Fprintf(stdout, "  Verified: every item cost in scope is denominated in its stocking unit.\n")
	}
	return nil
}

// selectMislabelledCosts is the shared projection behind both scans. Both ask the same question of a
// rate — is its denominator the stocking unit of the item it prices — and differ only in which rate
// they reach it through, so the columns and their order are fixed here and read back by scanCost.
const selectMislabelledCosts = `
       i.account_id, i.id, i.sku, r.id, r.value,
       r.denominator_unit_id, du.abbreviation,
       ug.base_unit_id, bu.abbreviation,
       CAST((bu.ratio_numerator / bu.ratio_denominator) / (du.ratio_numerator / du.ratio_denominator) AS DECIMAL(65,30))`

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
		&factor,
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

// scopeFilter builds the account/item predicate both scans share, as SQL text plus its bound values.
func scopeFilter(accountID, itemID string) (string, []any) {
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
	return where.String(), params
}

// findMislabelledItemCosts lists the item unit costs denominated in something other than the unit the
// item is stocked in.
func findMislabelledItemCosts(ctx context.Context, pool *sql.DB, accountID, itemID string, p *progress) ([]mislabelledCost, error) {
	where, params := scopeFilter(accountID, itemID)

	// #nosec G202 -- the only interpolation is `where`, built from fixed literals by scopeFilter; the
	// account and item ids it filters on are bound as parameters.
	query := `SELECT` + selectMislabelledCosts + `
FROM item i
JOIN rate r ON r.id = i.unit_cost_id` + joinItemStockingUnit + `
WHERE i.deleted_at IS NULL
  AND r.denominator_unit_id <> ug.base_unit_id` + where + `
ORDER BY i.sku`

	p.stage("Scanning item unit costs for a denominator outside the item's stocking unit...")

	rows, err := pool.QueryContext(ctx, query, params...)
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
func findMislabelledLayers(ctx context.Context, pool *sql.DB, accountID, itemID string, p *progress) ([]mislabelledLayer, error) {
	where, params := scopeFilter(accountID, itemID)

	p.stage("Scanning receipt layers for costs copied while the item was mislabelled...")

	// #nosec G202 -- the only interpolation is `where`, built from fixed literals by scopeFilter; the
	// account and item ids it filters on are bound as parameters.
	query := `SELECT` + selectMislabelledCosts + `,
       ir.id, ir.status_code, ir.received_at, q.value, qu.abbreviation,
       CAST((qu.ratio_numerator / qu.ratio_denominator) / (bu.ratio_numerator / bu.ratio_denominator) AS DECIMAL(65,30))
FROM inventory_receipt ir
JOIN item i ON i.id = ir.item_id
JOIN rate r ON r.id = ir.unit_cost_id` + joinItemStockingUnit + `
JOIN quantity q ON q.id = ir.quantity_id
JOIN unit qu ON qu.id = q.unit_id
WHERE i.deleted_at IS NULL
  AND r.denominator_unit_id <> ug.base_unit_id` + where + `
ORDER BY ir.received_at`

	rows, err := pool.QueryContext(ctx, query, params...)
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

// findDerivedItems lists every item the rollup owns a cost for — one produced by a production step.
//
// Deliberately the whole account rather than the mislabelled ones. The material term's missing
// normalisation left costs with a correct denominator and a wrong number, which no scan of the rate
// row can find; recomputing everything derived is the only thing that reaches them.
func findDerivedItems(ctx context.Context, pool *sql.DB, accountID, itemID string, p *progress) ([]itemRef, error) {
	where, params := scopeFilter(accountID, itemID)

	// #nosec G202 -- the only interpolation is `where`, built from fixed literals by scopeFilter; the
	// account and item ids it filters on are bound as parameters.
	query := `
SELECT DISTINCT i.id, i.sku
FROM item i
JOIN production pr ON pr.item_id = i.id AND pr.production_step_id IS NOT NULL
JOIN production_step ps ON ps.id = pr.production_step_id AND ps.account_id = i.account_id
WHERE i.deleted_at IS NULL` + where + `
ORDER BY i.sku`

	p.stage("Listing derived items to recompute...")

	rows, err := pool.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("find derived items: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var refs []itemRef
	for rows.Next() {
		var r itemRef
		if err := rows.Scan(&r.ID, &r.SKU); err != nil {
			return nil, fmt.Errorf("scan derived item: %w", err)
		}
		refs = append(refs, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate derived items: %w", err)
	}

	ordered, err := leavesFirst(ctx, pool, []string{accountID}, refs, p)
	if err != nil {
		return nil, err
	}

	p.stagef("Found %d derived item(s)", len(ordered))
	return ordered, nil
}

// orderCostsLeavesFirst puts the relabels into the same dependency order the recompute pass uses, so
// a run read top to bottom repairs a chain from its leaves up.
func orderCostsLeavesFirst(ctx context.Context, pool *sql.DB, items []mislabelledCost, p *progress) ([]mislabelledCost, error) {
	if len(items) < 2 {
		return items, nil
	}
	byID := make(map[string]mislabelledCost, len(items))
	refs := make([]itemRef, 0, len(items))
	seenAccounts := map[string]struct{}{}
	accounts := make([]string, 0, 1)
	for _, it := range items {
		byID[it.ItemID] = it
		refs = append(refs, itemRef{ID: it.ItemID, SKU: it.SKU})
		if _, ok := seenAccounts[it.AccountID]; !ok {
			seenAccounts[it.AccountID] = struct{}{}
			accounts = append(accounts, it.AccountID)
		}
	}

	ordered, err := leavesFirst(ctx, pool, accounts, refs, p)
	if err != nil {
		return nil, err
	}

	out := make([]mislabelledCost, 0, len(ordered))
	for _, ref := range ordered {
		out = append(out, byID[ref.ID])
	}
	return out, nil
}

// leavesFirst sorts items so each follows everything its production steps consume.
//
// Only edges between the items being ordered matter; an input outside the set constrains nothing.
// Items left in a cycle — a routing that consumes its own output — keep their scan order and are
// appended last, since no order satisfies them and dropping them would strand the rest of the chain.
func leavesFirst(ctx context.Context, pool *sql.DB, accountIDs []string, items []itemRef, p *progress) ([]itemRef, error) {
	if len(items) < 2 {
		return items, nil
	}

	index := make(map[string]itemRef, len(items))
	for _, it := range items {
		index[it.ID] = it
	}

	p.stagef("Ordering %d item(s) by production-flow depth (leaves first)...", len(items))

	inputsOf := make(map[string]map[string]struct{}, len(items))
	for _, accountID := range accountIDs {
		if accountID == "" {
			continue
		}
		const query = `
SELECT DISTINCT c.item_id, pr.item_id
FROM consumption c
JOIN production pr ON pr.production_step_id = c.production_step_id
JOIN production_step ps ON ps.id = c.production_step_id
WHERE ps.account_id = ?`
		rows, err := pool.QueryContext(ctx, query, accountID)
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

	ordered := make([]itemRef, 0, len(items))
	placed := make(map[string]struct{}, len(items))
	for len(ordered) < len(items) {
		ready := make([]string, 0, len(items))
		for _, it := range items {
			if _, done := placed[it.ID]; done {
				continue
			}
			blocked := false
			for input := range inputsOf[it.ID] {
				if _, done := placed[input]; !done {
					blocked = true
					break
				}
			}
			if !blocked {
				ready = append(ready, it.ID)
			}
		}
		if len(ready) == 0 {
			for _, it := range items {
				if _, done := placed[it.ID]; !done {
					ordered = append(ordered, it)
					placed[it.ID] = struct{}{}
				}
			}
			p.stage("  Some items consume each other; appending them in scan order")
			break
		}
		sort.Strings(ready)
		for _, id := range ready {
			ordered = append(ordered, index[id])
			placed[id] = struct{}{}
		}
	}

	return ordered, nil
}

// applyRelabels moves every mislabelled denominator in one transaction, items before the layers that
// copied them, so a failure leaves the costs as they were rather than half-moved.
func applyRelabels(ctx context.Context, pool *sql.DB, items []mislabelledCost, layers []mislabelledLayer, p *progress) error {
	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	p.stage("Relabelling (single transaction; nothing commits until the end)")

	p.stagef("  %d item unit cost(s), leaves first...", len(items))
	for i, it := range items {
		if err := relabel(ctx, tx, it.RateID, it.StockingID); err != nil {
			return err
		}
		p.step(i+1, len(items), "item costs relabelled")
	}

	p.stagef("  %d receipt layer(s)...", len(layers))
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

// recomputeAll restates each derived cost through the service's own rollup, in dependency order.
//
// Each item commits on its own, matching how the cost-basis consumer drives the same call: these are
// independent calculations, and one item the rollup refuses — a step producing in a unit outside the
// item's group, say — should not withhold every other corrected cost. Refusals are collected and
// reported, since they name real misconfiguration for someone to fix by hand.
func recomputeAll(ctx context.Context, pool *sql.DB, accountID string, items []itemRef, haltOnError bool, p *progress) ([]recomputeResult, error) {
	queries := sqlc.New(pool)
	itemSvc := service.NewItemSvc(&service.ItemSvcConfig{
		Repos:           repository.NewRepoFactory(queries),
		MediatorFactory: mediator.NewMediatorFactory(),
		TxManager:       service.NewTransactionManager(pool, queries),
	})

	p.stagef("Recomputing %d derived item cost(s), leaves first...", len(items))

	results := make([]recomputeResult, 0, len(items))
	for i, ref := range items {
		before, err := readUnitCost(ctx, pool, accountID, ref.ID)
		if err != nil {
			return nil, err
		}

		res := recomputeResult{Item: ref, Before: before}
		if _, apiErr := itemSvc.RecomputeItemCosts(ctx, accountID, ref.ID); apiErr != nil {
			res.Err = apiErr.PublicMessage
			results = append(results, res)
			if haltOnError {
				return results, fmt.Errorf("halted at item %s (%s): %s", ref.SKU, ref.ID, apiErr.PublicMessage)
			}
			p.step(i+1, len(items), "items recomputed")
			continue
		}

		after, err := readUnitCost(ctx, pool, accountID, ref.ID)
		if err != nil {
			return nil, err
		}
		res.After = after
		results = append(results, res)
		p.step(i+1, len(items), "items recomputed")
	}

	return results, nil
}

// readUnitCost renders an item's stored cost the way the report compares them, value and denominator
// together — a value on its own says nothing about whether the pair moved.
func readUnitCost(ctx context.Context, pool *sql.DB, accountID, itemID string) (string, error) {
	const query = `
SELECT r.value, du.abbreviation
FROM item i
JOIN rate r ON r.id = i.unit_cost_id
JOIN unit du ON du.id = r.denominator_unit_id
WHERE i.id = ? AND i.account_id = ? AND i.deleted_at IS NULL`

	var value, abbreviation string
	if err := pool.QueryRowContext(ctx, query, itemID, accountID).Scan(&value, &abbreviation); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("read unit cost for %s: %w", itemID, err)
	}
	trimmed, err := decimal.NewFromString(value)
	if err != nil {
		return value + "/" + abbreviation, nil
	}
	return trimmed.String() + "/" + abbreviation, nil
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

func printRecomputePlan(out io.Writer, items []itemRef, skipped bool) {
	if len(items) == 0 {
		return
	}
	verb := "to recompute, in this order"
	if skipped {
		verb = "needing a recompute this run will not do (--skip-recompute), in this order"
	}
	fmt.Fprintf(out, "  Derived items %s:\n", verb)
	for i, it := range items {
		if i >= planLines {
			fmt.Fprintf(out, "    ... and %d more\n", len(items)-planLines)
			break
		}
		fmt.Fprintf(out, "    %-24s %s\n", it.SKU, it.ID)
	}
	fmt.Fprintln(out)
}

// printRecomputeResults reports only the costs that moved and the items the rollup refused. An
// unchanged cost is the expected outcome for most of an account and listing them buries the rest.
func printRecomputeResults(out io.Writer, results []recomputeResult) {
	if len(results) == 0 {
		return
	}
	var changed, unchanged int
	var refused []recomputeResult
	var moved []recomputeResult
	for _, r := range results {
		switch {
		case r.Err != "":
			refused = append(refused, r)
		case r.Before != r.After:
			changed++
			moved = append(moved, r)
		default:
			unchanged++
		}
	}

	fmt.Fprintf(out, "  Recomputed %d item(s): %d changed, %d unchanged, %d refused.\n",
		len(results), changed, unchanged, len(refused))

	if len(moved) > 0 {
		fmt.Fprintf(out, "\n  Costs that moved (SKU, before -> after):\n")
		for i, r := range moved {
			if i >= planLines {
				fmt.Fprintf(out, "    ... and %d more\n", len(moved)-planLines)
				break
			}
			fmt.Fprintf(out, "    %-24s %s -> %s\n", r.Item.SKU, r.Before, r.After)
		}
	}

	if len(refused) > 0 {
		fmt.Fprintf(out, "\n  Refused by the rollup — these need a person (SKU, reason):\n")
		for i, r := range refused {
			if i >= planLines {
				fmt.Fprintf(out, "    ... and %d more\n", len(refused)-planLines)
				break
			}
			fmt.Fprintf(out, "    %-24s %s\n", r.Item.SKU, r.Err)
		}
	}
	fmt.Fprintln(out)
}

const planLines = 40

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
