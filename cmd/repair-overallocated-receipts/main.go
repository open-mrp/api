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
	"github.com/open-mrp/api/shared/ledger"
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
}

// allocationRow is one allocation being considered for removal, with the satellite rows it owns.
type allocationRow struct {
	ID          string
	IssueID     string
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

func Run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("repair-overallocated-receipts", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", false, "report what would change; make no writes")
	accountID := fs.String("account", "", "restrict to a single account id")
	itemID := fs.String("item", "", "restrict to a single item id")
	limit := fs.Int("limit", 0, "cap receipts repaired (0 = no cap)")
	epsilon := fs.String("epsilon", ledger.EpsilonFlagDefault, "ignore overshoots at or below this many base units")
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

	p.stagef("Choosing allocations to remove for %d receipt(s)...", len(receipts))
	var repairs []repair
	for i, r := range receipts {
		rep, planErr := planRepair(ctx, sqlDB, r)
		if planErr != nil {
			return planErr
		}
		if len(rep.Delete) > 0 {
			repairs = append(repairs, rep)
		}
		p.step(i+1, len(receipts), "receipts planned")
	}

	allocationIDs, issueIDs, receiptIDs, totalFreed := collect(repairs)

	p.stage("Summary")
	fmt.Fprintf(stdout, "  Over-allocated receipts:   %d\n", len(receipts))
	fmt.Fprintf(stdout, "  Receipts to repair:        %d\n", len(repairs))
	fmt.Fprintf(stdout, "  Allocation rows to delete: %d\n", len(allocationIDs))
	fmt.Fprintf(stdout, "  Issues to re-check:        %d\n", len(issueIDs))
	fmt.Fprintf(stdout, "  Phantom base units removed: %s\n", totalFreed.String())

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
	var params []any
	if accountID != "" {
		where.WriteString(" AND i2.account_id = ?")
		params = append(params, accountID)
	}
	if itemID != "" {
		where.WriteString(" AND ir2.item_id = ?")
		params = append(params, itemID)
	}

	// The filters are applied inside the grouped subquery as well as outside it: without them the
	// aggregate covers every allocation in the database and the join throws all but a handful away.
	//
	// #nosec G202 -- the only interpolation is `where`, built above from fixed literals; the account
	// and item ids it filters on are bound as parameters.
	query := `
SELECT ir.id, ir.item_id, i.sku,
       CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator) AS receipt_base,
       a.alloc_base
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
		if err := rows.Scan(&r.ReceiptID, &r.ItemID, &r.SKU, &receiptBase, &allocBase); err != nil {
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

// planRepair picks the allocations to remove from one receipt: newest first, stopping as soon as the
// survivors fit inside the receipt. An allocation that would take the receipt from over-drawn to
// under-drawn by more than it is over is still removed — whole rows only — because the reopened
// demand draws the remainder back down.
func planRepair(ctx context.Context, sqlDB *sql.DB, r overDrawnReceipt) (repair, error) {
	const query = `
SELECT ia.id, ia.inventory_issue_id, ia.quantity_id, ia.unit_cost_id, ia.total_cost_id,
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
		if err := rows.Scan(&a.ID, &a.IssueID, &a.QuantityID, &a.UnitCostID, &a.TotalCostID, &base); err != nil {
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
	for _, rep := range repairs {
		receiptIDs = append(receiptIDs, rep.Receipt.ReceiptID)
		totalFreed = totalFreed.Add(rep.Freed)
		for _, a := range rep.Delete {
			allocationIDs = append(allocationIDs, a.ID)
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
