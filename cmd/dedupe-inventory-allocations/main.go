// Command dedupe-inventory-allocations removes duplicate inventory allocation rows written by
// replayed production-step scans, and returns the receipts they falsely closed out to `available`.
//
// A production-step delivery is retried with backoff and replayed from the top. Before the step was
// made transactional, a failure partway through left its allocations committed, so the replay drew
// the same receipts a second time. The result is stock consumed on paper that never left the floor,
// and an on-hand figure well below what the plant actually holds.
//
// # What counts as a duplicate
//
// Rows sharing (inventory_issue_id, inventory_receipt_id, unit, value). One allocation pass visits a
// receipt at most once and takes min(available, remaining), so two draws of an identical quantity
// from the same receipt to the same issue is the replay signature rather than anything the allocator
// produces on its own.
//
// That test alone is not sufficient. An item consumed in many small equal increments accumulates
// identical (issue, receipt, value) rows legitimately, and deleting them would leave the issue short
// of what it actually drew. So a group is only deduplicated when the issue's allocations still cover
// its ordered quantity afterwards; issues that would end up under-allocated are reported and skipped.
//
// The earliest row in each group is kept, along with its quantity, unit-cost and total-cost rows. The
// satellite rows of deleted allocations go with them: total_cost_id and quantity_id reference
// `quantity`, unit_cost_id references `rate`, and each is unique to its allocation.
//
// # Receipt re-statusing
//
// An allocation that fully covered a receipt flipped it to `allocated`, which drops it out of on-hand
// entirely. Once the duplicates are gone the surviving allocations no longer cover those receipts, so
// any that are no longer fully drawn go back to `available` — the same rule FreeReleasedReceipts
// applies when a batch scan is undone.
//
// Usage:
//
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/dedupe-inventory-allocations --dry-run
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/dedupe-inventory-allocations
//
// Flags:
//
//	--dry-run   report what would change; make no writes.
//	--item      restrict to a single item id, for verifying the effect on one SKU first.
//	--limit     cap the number of duplicate groups processed (0 = no cap).
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

// dupGroup is one set of identical allocations for an (issue, receipt) pair. KeepID is the earliest
// row; DeleteIDs are the replays behind it.
type dupGroup struct {
	IssueID   string
	ReceiptID string
	Value     string
	UnitID    string
	KeepID    string
	DeleteIDs []string
}

func Run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("dedupe-inventory-allocations", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", false, "report what would change; make no writes")
	itemID := fs.String("item", "", "restrict to a single item id")
	limit := fs.Int("limit", 0, "cap duplicate groups processed (0 = no cap)")
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
	if *itemID != "" {
		p.stagef("Restricted to item %s", *itemID)
	}

	groups, skipped, err := findDuplicateGroups(ctx, sqlDB, *itemID, *limit, p)
	if err != nil {
		return err
	}

	var toDelete []string
	receipts := map[string]struct{}{}
	for _, g := range groups {
		toDelete = append(toDelete, g.DeleteIDs...)
		receipts[g.ReceiptID] = struct{}{}
	}

	p.stage("Summary")
	fmt.Fprintf(stdout, "  Duplicate groups:          %d\n", len(groups))
	fmt.Fprintf(stdout, "  Allocation rows to delete: %d\n", len(toDelete))
	fmt.Fprintf(stdout, "  Receipts to re-check:      %d\n", len(receipts))
	fmt.Fprintf(stdout, "  Issues skipped (would go under-allocated): %d\n", skipped)

	if len(toDelete) == 0 {
		p.stage("Nothing to do")
		return nil
	}

	if *dryRun {
		p.stage("--dry-run: no writes made")
		return nil
	}

	freed, err := apply(ctx, sqlDB, toDelete, receipts, p)
	if err != nil {
		return err
	}

	p.stage("Done")
	fmt.Fprintf(stdout, "  Deleted %d duplicate allocation(s).\n", len(toDelete))
	fmt.Fprintf(stdout, "  Returned %d receipt(s) to available.\n", freed)
	return nil
}

// progress reports what the command is doing as it goes. Resolving a group and deleting a chunk are
// each one round trip, so a full run against a remote database spends minutes in those loops and is
// indistinguishable from a hang without this. Every line carries elapsed time so a run that is merely
// slow can be told apart from one that is stuck.
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

// progressEvery is how many items pass between counter lines. Small enough that a stalled run shows
// it within seconds, large enough not to bury the stage headings.
const progressEvery = 100

// findDuplicateGroups returns the duplicate groups that are safe to collapse, and the count of issues
// excluded because collapsing them would leave the issue allocated below its ordered quantity.
//
// The exclusion is what separates a replayed scan from an item drawn down in equal increments, so it
// is applied per issue: every group belonging to an excluded issue is left alone.
func findDuplicateGroups(ctx context.Context, sqlDB *sql.DB, itemID string, limit int, p *progress) ([]dupGroup, int, error) {
	var where strings.Builder
	var params []any
	if itemID != "" {
		where.WriteString(" AND ii.item_id = ?")
		params = append(params, itemID)
	}

	// The per-issue guard: an issue is eligible only when its allocations, minus the duplicates this
	// command would remove, still cover the quantity the issue asked for. Compared in the issue's own
	// unit — allocations recorded in a different unit are not summed against it.
	//
	// #nosec G202 -- the only interpolation is `where`, built above from a fixed literal; the item id
	// it filters on is bound as a parameter.
	query := `
WITH dups AS (
    SELECT ia.inventory_issue_id, ia.inventory_receipt_id, q.unit_id, q.value,
           COUNT(*) - 1 AS extra,
           (COUNT(*) - 1) * CAST(q.value AS DECIMAL(65,30)) AS extra_qty
    FROM inventory_allocation ia
    JOIN quantity q ON q.id = ia.quantity_id
    GROUP BY ia.inventory_issue_id, ia.inventory_receipt_id, q.unit_id, q.value
    HAVING COUNT(*) > 1
),
issue_extra AS (
    SELECT inventory_issue_id, SUM(extra_qty) AS total_extra
    FROM dups GROUP BY inventory_issue_id
),
issue_state AS (
    SELECT ii.id AS issue_id,
           CAST(iq.value AS DECIMAL(65,30)) AS issue_qty,
           COALESCE(SUM(CAST(aq.value AS DECIMAL(65,30))), 0) AS allocated_same_unit,
           MAX(ie.total_extra) AS total_extra
    FROM inventory_issue ii
    JOIN quantity iq ON iq.id = ii.quantity_id
    JOIN issue_extra ie ON ie.inventory_issue_id = ii.id
    LEFT JOIN inventory_allocation ia ON ia.inventory_issue_id = ii.id
    LEFT JOIN quantity aq ON aq.id = ia.quantity_id AND aq.unit_id = iq.unit_id
    WHERE 1 = 1` + where.String() + `
    GROUP BY ii.id, iq.value
)
SELECT d.inventory_issue_id, d.inventory_receipt_id, d.value, d.unit_id,
       s.allocated_same_unit - s.total_extra >= s.issue_qty AS eligible
FROM dups d
JOIN issue_state s ON s.issue_id = d.inventory_issue_id
ORDER BY d.inventory_issue_id, d.inventory_receipt_id`

	p.stage("Scanning inventory_allocation for duplicate groups (one pass, may take a minute)...")

	rows, err := sqlDB.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, 0, fmt.Errorf("find duplicate groups: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var groups []dupGroup
	skippedIssues := map[string]struct{}{}
	for rows.Next() {
		var g dupGroup
		var eligible bool
		if err := rows.Scan(&g.IssueID, &g.ReceiptID, &g.Value, &g.UnitID, &eligible); err != nil {
			return nil, 0, fmt.Errorf("scan duplicate group: %w", err)
		}
		if !eligible {
			skippedIssues[g.IssueID] = struct{}{}
			continue
		}
		groups = append(groups, g)
		if limit > 0 && len(groups) >= limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate duplicate groups: %w", err)
	}

	p.stagef("Found %d eligible group(s), %d issue(s) skipped", len(groups), len(skippedIssues))
	if len(groups) == 0 {
		return nil, len(skippedIssues), nil
	}

	p.stage("Resolving which row in each group to keep...")
	for i := range groups {
		if err := loadGroupMembers(ctx, sqlDB, &groups[i]); err != nil {
			return nil, 0, err
		}
		p.step(i+1, len(groups), "groups resolved")
	}

	return groups, len(skippedIssues), nil
}

// loadGroupMembers fills in which row of the group survives and which are deleted. Ordering by
// created_at then id keeps the first write and discards the replays, and makes the choice stable
// across runs.
func loadGroupMembers(ctx context.Context, sqlDB *sql.DB, g *dupGroup) error {
	const query = `
SELECT ia.id
FROM inventory_allocation ia
JOIN quantity q ON q.id = ia.quantity_id
WHERE ia.inventory_issue_id = ?
  AND ia.inventory_receipt_id = ?
  AND q.unit_id = ?
  AND q.value = ?
ORDER BY ia.created_at ASC, ia.id ASC`

	rows, err := sqlDB.QueryContext(ctx, query, g.IssueID, g.ReceiptID, g.UnitID, g.Value)
	if err != nil {
		return fmt.Errorf("load group members: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scan group member: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate group members: %w", err)
	}
	if len(ids) < 2 {
		return nil
	}

	g.KeepID = ids[0]
	g.DeleteIDs = ids[1:]
	return nil
}

// apply deletes the duplicate allocations with their satellite rows and re-statuses the receipts they
// had closed out, all in one transaction so a failure leaves the ledger as it was.
func apply(ctx context.Context, sqlDB *sql.DB, allocationIDs []string, receipts map[string]struct{}, p *progress) (int, error) {
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	p.stage("Applying (single transaction; nothing commits until the end)")

	// Capture the satellites before the allocations that point at them are gone.
	p.stagef("  Collecting satellite rows for %d allocation(s)...", len(allocationIDs))
	quantityIDs, rateIDs, err := satelliteIDs(ctx, tx, allocationIDs, p)
	if err != nil {
		return 0, err
	}

	p.stagef("  Deleting %d allocation(s)...", len(allocationIDs))
	if err := deleteByIDs(ctx, tx, "inventory_allocation", allocationIDs, p); err != nil {
		return 0, err
	}
	p.stagef("  Deleting %d quantity row(s)...", len(quantityIDs))
	if err := deleteByIDs(ctx, tx, "quantity", quantityIDs, p); err != nil {
		return 0, err
	}
	p.stagef("  Deleting %d rate row(s)...", len(rateIDs))
	if err := deleteByIDs(ctx, tx, "rate", rateIDs, p); err != nil {
		return 0, err
	}

	p.stagef("  Re-checking %d receipt(s) for release...", len(receipts))
	freed, err := freeReleasedReceipts(ctx, tx, receipts, p)
	if err != nil {
		return 0, err
	}

	p.stage("  Committing...")
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return freed, nil
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

// freeReleasedReceipts returns receipts to `available` once the surviving allocations no longer cover
// them. The correlated sum mirrors FreeReleasedReceipts in batch_scan_undo.sql: a receipt another
// issue still fills stays as it is.
//
// Both sides go through their own unit's ratio. Allocations against one receipt carry whatever unit
// the code that wrote them chose, so an allocation stamped in each against a receipt in pairs counts
// double against it, and the raw comparison left receipts closed out that had stock left on them.
func freeReleasedReceipts(ctx context.Context, tx *sql.Tx, receipts map[string]struct{}, p *progress) (int, error) {
	ids := make([]string, 0, len(receipts))
	for id := range receipts {
		ids = append(ids, id)
	}

	total := 0
	done := 0
	for chunk := range chunks(ids, 500) {
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
		p.step(done, len(ids), "receipts checked")
	}
	return total, nil
}

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
// in cmd/backfill-customer-role; both are one-off commands in package main and cannot share it.
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
