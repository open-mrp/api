// Command repair-overallocated-receipts returns receipts that have been drawn on for more than they
// hold back to the quantity they actually received, and reopens the demand that leaves uncovered.
//
// # What it repairs
//
// A receipt's allocations must never total more than the receipt. Two allocation transactions running
// against the same item could both read a receipt as undrawn — one because the paged read of open
// demand was not a locking read and so froze the transaction's view of the ledger before the other
// committed, one because the whole step predated being wrapped in a transaction at all — and each
// then drew the same stock. The result is allocations covering demand out of a receipt that never
// held the units, so the plant reads short against stock that is physically on the shelf.
//
// The allocator no longer does this. This command cleans up what it already wrote.
//
// # How it decides what to remove
//
// Per receipt, every allocation against it is taken through its own unit's ratio and totalled. A
// receipt whose total exceeds its own quantity — by more than --epsilon, which keeps DECIMAL(65,30)
// noise out of the results — is over-allocated by the difference.
//
// Allocations are then removed newest first until the receipt is back within its quantity. Newest
// first because the earliest draw is the one that had the stock: the later ones are the duplicates
// the race produced. Only whole allocations are removed, never part of one — leaving a receipt
// slightly under-drawn is self-correcting, since the reopened demand draws it down again, while
// rewriting a quantity and its costed satellites in place is a second thing to get wrong.
//
// # What happens to the demand
//
// An issue that loses an allocation may no longer be covered, and an issue that is not covered is not
// finished: it goes back to `open` so the allocator can cover it from stock that exists. Issues whose
// surviving allocations still cover them are left closed. Receipts that are no longer fully drawn go
// back to `available`, the same rule a batch-scan undo applies.
//
// Reopened demand is picked up by the next allocation pass for that item — a receipt landing, a
// receiving order being stocked, a shipment. Until then it reads as short, which is what it is.
//
// # Order of operations
//
// Run dedupe-inventory-allocations first. Where the over-draw is an exact replay — the same issue
// allocated twice from the same receipt for the same quantity — that command removes the duplicate
// and leaves the issue covered, which is the better repair. This command handles what is left: draws
// from two different issues that each took stock the receipt did not have.
//
// Usage:
//
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/repair-overallocated-receipts --dry-run
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/repair-overallocated-receipts --account ac_... --dry-run
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/repair-overallocated-receipts --account ac_...
//
// Flags:
//
//	--dry-run   report what would change; make no writes.
//	--account   restrict to one account id.
//	--item      restrict to one item id, for verifying the effect on a single SKU first.
//	--limit     cap the number of receipts repaired (0 = no cap).
//	--epsilon   ignore overshoots at or below this many base units (default 0.000001).
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
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

// overDrawnReceipt is one receipt whose allocations total more than it holds. Both totals are in base
// units, which is the only footing on which allocations recorded in different units can be added.
type overDrawnReceipt struct {
	ReceiptID     string
	ItemID        string
	SKU           string
	ReceiptBase   decimal.Decimal
	AllocatedBase decimal.Decimal
	// MixedUnits marks a receipt carrying an allocation stamped in a different unit from its own AND
	// inconsistent with it — the row alone covers more than the order it is against, which is the
	// mislabel signature. A differing unit on its own means nothing: the dashboard's allocator records
	// a draw in whichever unit bounded it, so a draw limited by the issue is written in the issue's
	// unit with a value to match. Skipping on the unit alone left genuinely over-drawn receipts
	// unrepaired. Reported and skipped, never repaired — see skipMixedUnitsNote.
	MixedUnits bool
	// Oversized marks a receipt carrying a single allocation larger than the whole receipt. Also
	// reported and skipped — see skipOversizedNote.
	Oversized bool
}

// allocationRow is one allocation being considered for removal, with the satellite rows it owns.
type allocationRow struct {
	ID          string
	IssueID     string
	ReceiptID   string
	QuantityID  string
	UnitCostID  string
	TotalCostID string
	Base        decimal.Decimal
}

// repair is the decision for one receipt: which allocations go, and which issues that touches.
type repair struct {
	Receipt overDrawnReceipt
	Delete  []allocationRow
	Freed   decimal.Decimal
}

// overCoveredIssue is the other side of the same damage: an order carrying more allocation than it
// ever asked for. It is what an inflated or duplicated draw looks like when the receipt it came from
// was large enough to absorb it — a 697 pr receipt swallows a draw of ten times 20 pairs without ever
// reading as over-drawn, while the order it covers plainly shows ten times what it ordered.
//
// The excess is not free stock sitting idle: it is still charged against real receipts, so it holds
// down on-hand exactly as an over-drawn receipt does.
type overCoveredIssue struct {
	IssueID       string
	ItemID        string
	SKU           string
	DemandBase    decimal.Decimal
	AllocatedBase decimal.Decimal
	// Oversized marks an issue carrying a single allocation larger than the whole order. Allocation
	// takes min(receipt remaining, issue remaining), so no race writes one; the quantity is wrong and
	// trimming would delete a real draw. Reported and skipped, as on the receipt side.
	Oversized bool
}

func Run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("repair-overallocated-receipts", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", false, "report what would change; make no writes")
	accountID := fs.String("account", "", "restrict to a single account id")
	itemID := fs.String("item", "", "restrict to a single item id")
	limit := fs.Int("limit", 0, "cap receipts repaired (0 = no cap)")
	epsilon := fs.String("epsilon", "0.000001", "ignore overshoots at or below this many base units")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	eps, err := decimal.NewFromString(*epsilon)
	if err != nil {
		return fmt.Errorf("parsing --epsilon: %w", err)
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

	receipts, err := findOverDrawnReceipts(ctx, sqlDB, *accountID, *itemID, *limit, eps, p)
	if err != nil {
		return err
	}

	// A receipt drawn on in a unit other than its own is not evidence of over-allocation at all: the
	// allocator stamps every allocation with its receipt's unit, so a differing unit means the row was
	// written by something else with the wrong label, and the overshoot is the ratio between the two
	// labels rather than stock. Removing those allocations would delete draws that really happened.
	// They are reported and left alone; see skipMixedUnitsNote.
	var eligible []overDrawnReceipt
	var mixed []overDrawnReceipt
	var oversized []overDrawnReceipt
	for _, r := range receipts {
		switch {
		case r.MixedUnits:
			mixed = append(mixed, r)
		case r.Oversized:
			oversized = append(oversized, r)
		default:
			eligible = append(eligible, r)
		}
	}

	p.stagef("Choosing allocations to remove for %d receipt(s)...", len(eligible))
	var repairs []repair
	for i, r := range eligible {
		rep, planErr := planRepair(ctx, sqlDB, r)
		if planErr != nil {
			return planErr
		}
		if len(rep.Delete) > 0 {
			repairs = append(repairs, rep)
		}
		p.step(i+1, len(eligible), "receipts planned")
	}

	allocationIDs, issueIDs, receiptIDs, totalFreed := collect(repairs)

	issues, err := findOverCoveredIssues(ctx, sqlDB, *accountID, *itemID, eps, p)
	if err != nil {
		return err
	}
	var oversizedIssues []overCoveredIssue
	var issueRepairs []repair
	for _, iss := range issues {
		if iss.Oversized {
			oversizedIssues = append(oversizedIssues, iss)
			continue
		}
		rep, planErr := planIssueRepair(ctx, sqlDB, iss)
		if planErr != nil {
			return planErr
		}
		if len(rep.Delete) > 0 {
			issueRepairs = append(issueRepairs, rep)
		}
	}
	issueAllocIDs, issueTouched, issueReceipts, issueFreed := collect(issueRepairs)
	allocationIDs = append(allocationIDs, issueAllocIDs...)
	issueIDs = append(issueIDs, issueTouched...)
	receiptIDs = append(receiptIDs, issueReceipts...)
	totalFreed = totalFreed.Add(issueFreed)

	p.stage("Summary")
	fmt.Fprintf(stdout, "  Over-drawn receipts found: %d\n", len(receipts))
	fmt.Fprintf(stdout, "  Skipped, mixed units:      %d\n", len(mixed))
	fmt.Fprintf(stdout, "  Skipped, oversized alloc:  %d\n", len(oversized))
	fmt.Fprintf(stdout, "  Receipts to repair:        %d\n", len(repairs))
	fmt.Fprintf(stdout, "  Over-covered issues found: %d\n", len(issues))
	fmt.Fprintf(stdout, "  Skipped, oversized draw:   %d\n", len(oversizedIssues))
	fmt.Fprintf(stdout, "  Issues to repair:          %d\n", len(issueRepairs))
	fmt.Fprintf(stdout, "  Allocation rows to delete: %d\n", len(allocationIDs))
	fmt.Fprintf(stdout, "  Issues to re-check:        %d\n", len(issueIDs))
	fmt.Fprintf(stdout, "  Phantom base units removed: %s\n", totalFreed.String())

	if len(mixed) > 0 {
		fmt.Fprint(stdout, skipMixedUnitsNote)
		printReceiptLines(stdout, mixed)
	}
	if len(oversized) > 0 {
		fmt.Fprint(stdout, skipOversizedNote)
		printReceiptLines(stdout, oversized)
	}
	if len(oversizedIssues) > 0 {
		fmt.Fprint(stdout, skipOversizedIssueNote)
		for i, iss := range oversizedIssues {
			if i >= worstOffenderLines {
				fmt.Fprintf(stdout, "    ... and %d more\n", len(oversizedIssues)-worstOffenderLines)
				break
			}
			fmt.Fprintf(stdout, "    %-22s %-24s ordered %s / allocated %s\n",
				iss.IssueID, iss.SKU, iss.DemandBase.StringFixed(2), iss.AllocatedBase.StringFixed(2))
		}
		fmt.Fprintln(stdout)
	}

	printWorstOffenders(stdout, repairs)

	if len(allocationIDs) == 0 {
		p.stage("Nothing to do")
		return nil
	}

	if *dryRun {
		p.stage("--dry-run: no writes made")
		return nil
	}

	reopened, freed, err := apply(ctx, sqlDB, allocationIDs, issueIDs, receiptIDs, p)
	if err != nil {
		return err
	}

	p.stage("Done")
	fmt.Fprintf(stdout, "  Deleted %d over-drawn allocation(s).\n", len(allocationIDs))
	fmt.Fprintf(stdout, "  Reopened %d issue(s) that are no longer covered.\n", reopened)
	fmt.Fprintf(stdout, "  Returned %d receipt(s) to available.\n", freed)
	return nil
}

// findOverDrawnReceipts lists the receipts whose allocations exceed what they hold.
//
// The receipt's own quantity must be positive. A migration wrote opening balances as receipts with
// negative quantities, and any allocation at all against one of those reads as an overshoot without
// anything having gone wrong.
func findOverDrawnReceipts(ctx context.Context, sqlDB *sql.DB, accountID, itemID string, limit int, eps decimal.Decimal, p *progress) ([]overDrawnReceipt, error) {
	var where strings.Builder
	var filterParams []any
	if accountID != "" {
		where.WriteString(" AND i2.account_id = ?")
		filterParams = append(filterParams, accountID)
	}
	if itemID != "" {
		where.WriteString(" AND ir2.item_id = ?")
		filterParams = append(filterParams, itemID)
	}

	// Bound by position in the SQL text, not by role: the `mixed_units` and `oversized` subqueries sit
	// in the SELECT list and so bind before the derived table's filters, and the overshoot last.
	params := append([]any{eps.String(), eps.String()}, filterParams...)

	// The filters are applied inside the grouped subquery as well as outside it: without them the
	// aggregate covers every allocation in the database and the join throws all but a handful away.
	//
	// #nosec G202 -- the only interpolation is `where`, built above from fixed literals; the account
	// and item ids it filters on are bound as parameters.
	query := `
SELECT ir.id, ir.item_id, i.sku,
       CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator) AS receipt_base,
       a.alloc_base,
       EXISTS (
           SELECT 1 FROM inventory_allocation ia3
           JOIN quantity aq3 ON aq3.id = ia3.quantity_id
           JOIN unit au3 ON au3.id = aq3.unit_id
           JOIN inventory_issue ii3 ON ii3.id = ia3.inventory_issue_id
           JOIN quantity iq3 ON iq3.id = ii3.quantity_id
           JOIN unit iu3 ON iu3.id = iq3.unit_id
           WHERE ia3.inventory_receipt_id = ir.id
             AND aq3.unit_id <> q.unit_id
             AND CAST(aq3.value AS DECIMAL(65,30)) * (au3.ratio_numerator / au3.ratio_denominator)
                 > CAST(iq3.value AS DECIMAL(65,30)) * (iu3.ratio_numerator / iu3.ratio_denominator) + ?
       ) AS mixed_units,
       EXISTS (
           SELECT 1 FROM inventory_allocation ia4
           JOIN quantity aq4 ON aq4.id = ia4.quantity_id
           JOIN unit au4 ON au4.id = aq4.unit_id
           WHERE ia4.inventory_receipt_id = ir.id
             AND CAST(aq4.value AS DECIMAL(65,30)) * (au4.ratio_numerator / au4.ratio_denominator)
                 > CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator) + ?
       ) AS oversized
FROM inventory_receipt ir
JOIN quantity q ON q.id = ir.quantity_id
JOIN unit u ON u.id = q.unit_id
JOIN item i ON i.id = ir.item_id
JOIN (
    SELECT ia.inventory_receipt_id AS rid,
           SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator)) AS alloc_base
    FROM inventory_allocation ia
    JOIN inventory_receipt ir2 ON ir2.id = ia.inventory_receipt_id
    JOIN item i2 ON i2.id = ir2.item_id
    JOIN quantity aq ON aq.id = ia.quantity_id
    JOIN unit au ON au.id = aq.unit_id
    WHERE 1 = 1` + where.String() + `
    GROUP BY ia.inventory_receipt_id
) a ON a.rid = ir.id
WHERE CAST(q.value AS DECIMAL(65,30)) > 0
  AND a.alloc_base > CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator) + ?
ORDER BY (a.alloc_base - CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator)) DESC`

	params = append(params, eps.String())

	p.stage("Scanning inventory_allocation for over-drawn receipts (one pass, may take a minute)...")

	rows, err := sqlDB.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("find over-drawn receipts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []overDrawnReceipt
	for rows.Next() {
		var r overDrawnReceipt
		var receiptBase, allocBase string
		if err := rows.Scan(&r.ReceiptID, &r.ItemID, &r.SKU, &receiptBase, &allocBase, &r.MixedUnits, &r.Oversized); err != nil {
			return nil, fmt.Errorf("scan over-drawn receipt: %w", err)
		}
		if r.ReceiptBase, err = decimal.NewFromString(receiptBase); err != nil {
			return nil, fmt.Errorf("parse receipt quantity for %s: %w", r.ReceiptID, err)
		}
		if r.AllocatedBase, err = decimal.NewFromString(allocBase); err != nil {
			return nil, fmt.Errorf("parse allocated total for %s: %w", r.ReceiptID, err)
		}
		out = append(out, r)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate over-drawn receipts: %w", err)
	}

	p.stagef("Found %d over-drawn receipt(s)", len(out))
	return out, nil
}

// findOverCoveredIssues lists the orders carrying more allocation than they asked for.
//
// Demand must be positive: an issue for nothing is not something this can reason about.
func findOverCoveredIssues(ctx context.Context, sqlDB *sql.DB, accountID, itemID string, eps decimal.Decimal, p *progress) ([]overCoveredIssue, error) {
	var where strings.Builder
	var params []any
	if accountID != "" {
		where.WriteString(" AND i.account_id = ?")
		params = append(params, accountID)
	}
	if itemID != "" {
		where.WriteString(" AND ii.item_id = ?")
		params = append(params, itemID)
	}
	params = append(params, eps.String())

	// #nosec G202 -- the only interpolation is `where`, built above from fixed literals; the account
	// and item ids it filters on are bound as parameters.
	query := `
SELECT ii.id, ii.item_id, i.sku,
       CAST(iq.value AS DECIMAL(65,30)) * (iu.ratio_numerator / iu.ratio_denominator) AS demand_base,
       COALESCE((
           SELECT SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator))
           FROM inventory_allocation ia
           JOIN quantity aq ON aq.id = ia.quantity_id
           JOIN unit au ON au.id = aq.unit_id
           WHERE ia.inventory_issue_id = ii.id), 0) AS alloc_base,
       EXISTS (
           SELECT 1 FROM inventory_allocation ia2
           JOIN quantity aq2 ON aq2.id = ia2.quantity_id
           JOIN unit au2 ON au2.id = aq2.unit_id
           WHERE ia2.inventory_issue_id = ii.id
             AND CAST(aq2.value AS DECIMAL(65,30)) * (au2.ratio_numerator / au2.ratio_denominator)
                 > CAST(iq.value AS DECIMAL(65,30)) * (iu.ratio_numerator / iu.ratio_denominator) + 0.000001
       ) AS oversized
FROM inventory_issue ii
JOIN quantity iq ON iq.id = ii.quantity_id
JOIN unit iu ON iu.id = iq.unit_id
JOIN item i ON i.id = ii.item_id
WHERE CAST(iq.value AS DECIMAL(65,30)) > 0` + where.String() + `
HAVING alloc_base > demand_base + ?
ORDER BY (alloc_base - demand_base) DESC`

	p.stage("Scanning inventory_issue for over-covered orders...")

	rows, err := sqlDB.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("find over-covered issues: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []overCoveredIssue
	for rows.Next() {
		var iss overCoveredIssue
		var demand, alloc string
		if err := rows.Scan(&iss.IssueID, &iss.ItemID, &iss.SKU, &demand, &alloc, &iss.Oversized); err != nil {
			return nil, fmt.Errorf("scan over-covered issue: %w", err)
		}
		if iss.DemandBase, err = decimal.NewFromString(demand); err != nil {
			return nil, fmt.Errorf("parse demand for %s: %w", iss.IssueID, err)
		}
		if iss.AllocatedBase, err = decimal.NewFromString(alloc); err != nil {
			return nil, fmt.Errorf("parse allocated total for %s: %w", iss.IssueID, err)
		}
		out = append(out, iss)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate over-covered issues: %w", err)
	}

	p.stagef("Found %d over-covered issue(s)", len(out))
	return out, nil
}

// planIssueRepair picks the allocations to drop from one over-covered order: newest first, stopping
// as soon as what is left fits inside what was ordered. Newest first for the same reason as on the
// receipt side — the earliest draw is the one that filled the order, and the later ones are what the
// duplication added.
func planIssueRepair(ctx context.Context, sqlDB *sql.DB, iss overCoveredIssue) (repair, error) {
	const query = `
SELECT ia.id, ia.inventory_issue_id, ia.inventory_receipt_id, ia.quantity_id, ia.unit_cost_id, ia.total_cost_id,
       CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator) AS base_value
FROM inventory_allocation ia
JOIN quantity q ON q.id = ia.quantity_id
JOIN unit u ON u.id = q.unit_id
WHERE ia.inventory_issue_id = ?
ORDER BY ia.created_at DESC, ia.id DESC`

	rows, err := sqlDB.QueryContext(ctx, query, iss.IssueID)
	if err != nil {
		return repair{}, fmt.Errorf("load allocations for issue %s: %w", iss.IssueID, err)
	}
	defer func() { _ = rows.Close() }()

	rep := repair{Freed: decimal.Zero}
	remaining := iss.AllocatedBase

	for rows.Next() {
		if remaining.LessThanOrEqual(iss.DemandBase) {
			break
		}
		var a allocationRow
		var base string
		if err := rows.Scan(&a.ID, &a.IssueID, &a.ReceiptID, &a.QuantityID, &a.UnitCostID, &a.TotalCostID, &base); err != nil {
			return repair{}, fmt.Errorf("scan allocation: %w", err)
		}
		if a.Base, err = decimal.NewFromString(base); err != nil {
			return repair{}, fmt.Errorf("parse allocation quantity for %s: %w", a.ID, err)
		}
		if a.Base.LessThanOrEqual(decimal.Zero) {
			continue
		}
		rep.Delete = append(rep.Delete, a)
		rep.Freed = rep.Freed.Add(a.Base)
		remaining = remaining.Sub(a.Base)
	}
	if err := rows.Err(); err != nil {
		return repair{}, fmt.Errorf("iterate allocations: %w", err)
	}

	return rep, nil
}

// planRepair picks the allocations to remove from one receipt: newest first, stopping as soon as the
// survivors fit inside the receipt. An allocation that would take the receipt from over-drawn to
// under-drawn by more than it is over is still removed — whole rows only — because the reopened
// demand draws the remainder back down.
func planRepair(ctx context.Context, sqlDB *sql.DB, r overDrawnReceipt) (repair, error) {
	const query = `
SELECT ia.id, ia.inventory_issue_id, ia.inventory_receipt_id, ia.quantity_id, ia.unit_cost_id, ia.total_cost_id,
       CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator) AS base_value
FROM inventory_allocation ia
JOIN quantity q ON q.id = ia.quantity_id
JOIN unit u ON u.id = q.unit_id
WHERE ia.inventory_receipt_id = ?
ORDER BY ia.created_at DESC, ia.id DESC`

	rows, err := sqlDB.QueryContext(ctx, query, r.ReceiptID)
	if err != nil {
		return repair{}, fmt.Errorf("load allocations for %s: %w", r.ReceiptID, err)
	}
	defer func() { _ = rows.Close() }()

	rep := repair{Receipt: r, Freed: decimal.Zero}
	remaining := r.AllocatedBase

	for rows.Next() {
		if remaining.LessThanOrEqual(r.ReceiptBase) {
			break
		}
		var a allocationRow
		var base string
		if err := rows.Scan(&a.ID, &a.IssueID, &a.ReceiptID, &a.QuantityID, &a.UnitCostID, &a.TotalCostID, &base); err != nil {
			return repair{}, fmt.Errorf("scan allocation: %w", err)
		}
		if a.Base, err = decimal.NewFromString(base); err != nil {
			return repair{}, fmt.Errorf("parse allocation quantity for %s: %w", a.ID, err)
		}
		// A zero or negative allocation frees nothing, so removing it would loop without progress.
		if a.Base.LessThanOrEqual(decimal.Zero) {
			continue
		}
		rep.Delete = append(rep.Delete, a)
		rep.Freed = rep.Freed.Add(a.Base)
		remaining = remaining.Sub(a.Base)
	}
	if err := rows.Err(); err != nil {
		return repair{}, fmt.Errorf("iterate allocations: %w", err)
	}

	return rep, nil
}

func collect(repairs []repair) (allocationIDs, issueIDs, receiptIDs []string, totalFreed decimal.Decimal) {
	totalFreed = decimal.Zero
	seenIssues := map[string]struct{}{}
	seenReceipts := map[string]struct{}{}
	addReceipt := func(id string) {
		if id == "" {
			return
		}
		if _, seen := seenReceipts[id]; seen {
			return
		}
		seenReceipts[id] = struct{}{}
		receiptIDs = append(receiptIDs, id)
	}
	for _, rep := range repairs {
		addReceipt(rep.Receipt.ReceiptID)
		totalFreed = totalFreed.Add(rep.Freed)
		for _, a := range rep.Delete {
			allocationIDs = append(allocationIDs, a.ID)
			// An issue-side repair has no single receipt of its own; the rows it drops each name one.
			addReceipt(a.ReceiptID)
			if _, seen := seenIssues[a.IssueID]; !seen {
				seenIssues[a.IssueID] = struct{}{}
				issueIDs = append(issueIDs, a.IssueID)
			}
		}
	}
	return allocationIDs, issueIDs, receiptIDs, totalFreed
}

// printWorstOffenders lists the receipts giving back the most, which is what a reviewer checks against
// a physical count before letting the run write anything.
func printWorstOffenders(out io.Writer, repairs []repair) {
	if len(repairs) == 0 {
		return
	}
	fmt.Fprintf(out, "\n  Largest repairs (receipt, SKU, held / allocated / removed, base units):\n")
	for i, rep := range repairs {
		if i >= worstOffenderLines {
			fmt.Fprintf(out, "    ... and %d more\n", len(repairs)-worstOffenderLines)
			break
		}
		fmt.Fprintf(out, "    %-22s %-24s %s / %s / -%s\n",
			rep.Receipt.ReceiptID, rep.Receipt.SKU,
			rep.Receipt.ReceiptBase.String(), rep.Receipt.AllocatedBase.String(), rep.Freed.String())
	}
	fmt.Fprintln(out)
}

const worstOffenderLines = 20

// skipMixedUnitsNote explains the one class this command deliberately will not touch. It is printed
// in full rather than summarised because the receipts behind it need a person: there is no mechanical
// rule that recovers the intent, and the obvious one — restamp the allocation with the receipt's unit
// — is only right where the raw values already agree.
const skipMixedUnitsNote = `
  NOT REPAIRED — these receipts carry an allocation whose unit and value disagree.
  The allocator writes every allocation in its receipt's unit, so a differing unit means the row came
  from somewhere else with the wrong label and the overshoot is the ratio between the two labels, not
  missing stock. The draws themselves are real. Deleting them would remove allocations that happened
  and reopen demand that was filled, so they are left for a person to judge.
`

// skipOversizedNote covers the other class a race cannot produce. Allocation takes
// min(what the receipt has left, what the issue still wants), so no allocation the allocator writes
// is ever larger than the receipt it draws from, however many transactions raced. One that is came
// from a wrong quantity, and the row to correct is that quantity — not the allocation.
const skipOversizedIssueNote = `
  NOT REPAIRED — these orders carry a single allocation larger than the whole order.
  A draw is min(receipt remaining, issue remaining), so no duplication produces one bigger than the
  order it covers; that quantity is wrong in itself. Trimming would delete a draw that happened, so
  they are left for a person to judge.
`

const skipOversizedNote = `
  NOT REPAIRED — these receipts carry a single allocation larger than the whole receipt.
  Allocation takes min(receipt remaining, issue remaining), so no race can produce a draw bigger than
  the receipt itself; the quantity on that allocation is simply wrong. Trimming here would delete a
  real draw and reopen filled demand, so they are left for a person to judge.
`

// printReceiptLines lists receipts with what they hold against what is allocated, in base units.
func printReceiptLines(out io.Writer, receipts []overDrawnReceipt) {
	for i, r := range receipts {
		if i >= worstOffenderLines {
			fmt.Fprintf(out, "    ... and %d more\n", len(receipts)-worstOffenderLines)
			break
		}
		fmt.Fprintf(out, "    %-32s %-24s %s / %s\n",
			r.ReceiptID, r.SKU, r.ReceiptBase.String(), r.AllocatedBase.String())
	}
	fmt.Fprintln(out)
}

// apply deletes the over-drawn allocations with their satellite rows, reopens the demand they were
// covering and re-statuses the receipts, all in one transaction so a failure leaves the ledger as it
// was.
func apply(ctx context.Context, sqlDB *sql.DB, allocationIDs, issueIDs, receiptIDs []string, p *progress) (reopened, freed int, err error) {
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	p.stage("Applying (single transaction; nothing commits until the end)")

	p.stagef("  Collecting satellite rows for %d allocation(s)...", len(allocationIDs))
	quantityIDs, rateIDs, err := satelliteIDs(ctx, tx, allocationIDs, p)
	if err != nil {
		return 0, 0, err
	}

	p.stagef("  Deleting %d allocation(s)...", len(allocationIDs))
	if err := deleteByIDs(ctx, tx, "inventory_allocation", allocationIDs, p); err != nil {
		return 0, 0, err
	}
	p.stagef("  Deleting %d quantity row(s)...", len(quantityIDs))
	if err := deleteByIDs(ctx, tx, "quantity", quantityIDs, p); err != nil {
		return 0, 0, err
	}
	p.stagef("  Deleting %d rate row(s)...", len(rateIDs))
	if err := deleteByIDs(ctx, tx, "rate", rateIDs, p); err != nil {
		return 0, 0, err
	}

	p.stagef("  Re-checking %d issue(s) for reopening...", len(issueIDs))
	reopened, err = reopenUncoveredIssues(ctx, tx, issueIDs, p)
	if err != nil {
		return 0, 0, err
	}

	p.stagef("  Re-checking %d receipt(s) for release...", len(receiptIDs))
	freed, err = freeReleasedReceipts(ctx, tx, receiptIDs, p)
	if err != nil {
		return 0, 0, err
	}

	p.stage("  Committing...")
	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("commit: %w", err)
	}
	return reopened, freed, nil
}

// satelliteIDs collects the quantity and rate rows owned by the allocations being deleted.
// quantity_id and total_cost_id reference `quantity`; unit_cost_id references `rate`.
func satelliteIDs(ctx context.Context, tx *sql.Tx, allocationIDs []string, p *progress) (quantityIDs, rateIDs []string, err error) {
	done := 0
	for chunk := range chunks(allocationIDs, 500) {
		// #nosec G202 -- placeholders() emits only "?,?,…"; the ids themselves are bound below.
		query := "SELECT quantity_id, total_cost_id, unit_cost_id FROM inventory_allocation WHERE id IN (" +
			placeholders(len(chunk)) + ")"
		rows, qErr := tx.QueryContext(ctx, query, toAny(chunk)...)
		if qErr != nil {
			return nil, nil, fmt.Errorf("load satellite ids: %w", qErr)
		}
		for rows.Next() {
			var qty, totalCost, unitCost string
			if sErr := rows.Scan(&qty, &totalCost, &unitCost); sErr != nil {
				_ = rows.Close()
				return nil, nil, fmt.Errorf("scan satellite ids: %w", sErr)
			}
			quantityIDs = append(quantityIDs, qty, totalCost)
			rateIDs = append(rateIDs, unitCost)
		}
		if rErr := rows.Err(); rErr != nil {
			_ = rows.Close()
			return nil, nil, fmt.Errorf("iterate satellite ids: %w", rErr)
		}
		_ = rows.Close()
		done += len(chunk)
		p.step(done, len(allocationIDs), "allocations read")
	}
	return quantityIDs, rateIDs, nil
}

func deleteByIDs(ctx context.Context, tx *sql.Tx, table string, ids []string, p *progress) error {
	done := 0
	for chunk := range chunks(ids, 500) {
		// #nosec G202 -- table is a package-level literal, never caller input; ids are placeheld.
		query := "DELETE FROM " + table + " WHERE id IN (" + placeholders(len(chunk)) + ")"
		if _, err := tx.ExecContext(ctx, query, toAny(chunk)...); err != nil {
			return fmt.Errorf("delete from %s: %w", table, err)
		}
		done += len(chunk)
		p.step(done, len(ids), "rows deleted from "+table)
	}
	return nil
}

// reopenUncoveredIssues hands demand back to the allocator once the surviving allocations no longer
// cover it. An issue the remaining rows still cover keeps whatever status it had.
func reopenUncoveredIssues(ctx context.Context, tx *sql.Tx, issueIDs []string, p *progress) (int, error) {
	total := 0
	done := 0
	for chunk := range chunks(issueIDs, 500) {
		// #nosec G202 -- placeholders() emits only "?,?,…"; the ids themselves are bound below.
		query := `
UPDATE inventory_issue ii
JOIN quantity iq ON iq.id = ii.quantity_id
JOIN unit iu ON iu.id = iq.unit_id
SET ii.status_code = 'open', ii.updated_at = NOW(3)
WHERE ii.id IN (` + placeholders(len(chunk)) + `)
AND ii.status_code = 'closed'
AND COALESCE((
    SELECT SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator))
    FROM inventory_allocation ia
    JOIN quantity aq ON aq.id = ia.quantity_id
    JOIN unit au ON au.id = aq.unit_id
    WHERE ia.inventory_issue_id = ii.id
), 0) < CAST(iq.value AS DECIMAL(65,30)) * (iu.ratio_numerator / iu.ratio_denominator)`
		res, err := tx.ExecContext(ctx, query, toAny(chunk)...)
		if err != nil {
			return 0, fmt.Errorf("reopen uncovered issues: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("reopen uncovered issues rows affected: %w", err)
		}
		total += int(n)
		done += len(chunk)
		p.step(done, len(issueIDs), "issues checked")
	}
	return total, nil
}

// freeReleasedReceipts returns receipts to `available` once the surviving allocations no longer cover
// them, mirroring FreeReleasedReceipts in batch_scan_undo.sql. Both sides go through their own unit's
// ratio: an allocation stamped in each against a receipt in pairs is half the number it looks like,
// and comparing the raw values leaves a receipt closed out that has stock left on it.
func freeReleasedReceipts(ctx context.Context, tx *sql.Tx, receiptIDs []string, p *progress) (int, error) {
	total := 0
	done := 0
	for chunk := range chunks(receiptIDs, 500) {
		// #nosec G202 -- placeholders() emits only "?,?,…"; the ids themselves are bound below.
		query := `
UPDATE inventory_receipt ir
JOIN quantity q ON q.id = ir.quantity_id
JOIN unit u ON u.id = q.unit_id
SET ir.status_code = 'available', ir.updated_at = NOW(3)
WHERE ir.id IN (` + placeholders(len(chunk)) + `)
AND ir.status_code <> 'available'
AND COALESCE((
    SELECT SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator))
    FROM inventory_allocation ia
    JOIN quantity aq ON aq.id = ia.quantity_id
    JOIN unit au ON au.id = aq.unit_id
    WHERE ia.inventory_receipt_id = ir.id
), 0) < CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator)`
		res, err := tx.ExecContext(ctx, query, toAny(chunk)...)
		if err != nil {
			return 0, fmt.Errorf("free released receipts: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("free released receipts rows affected: %w", err)
		}
		total += int(n)
		done += len(chunk)
		p.step(done, len(receiptIDs), "receipts checked")
	}
	return total, nil
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
