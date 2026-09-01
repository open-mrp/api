// Command repair-unreleased-receipts returns receipts that are still marked `allocated` but are no
// longer drawn to their capacity back to `available`, and asks for the demand they can now cover.
//
// # What it repairs
//
// A receipt is `allocated` when its allocations total its whole quantity, and `available` otherwise.
// Releasing an order's reservations deletes the allocations covering them, which gives the receipt its
// stock back — but until 2026-09 the four order paths that release reservations (unissue, issue's
// stale-reservation cleanup, order delete, production-run delete) deleted the allocations and stopped
// there. The receipt stayed `allocated` holding stock nothing had a claim on.
//
// FindReceiptsForAllocation matches `status_code = 'available'` and nothing else, so that stock is
// invisible to every later allocation: the item reads short while the units sit on the shelf, and
// there is no bad row to find — the receipt is right, the allocations are right, only the status is
// stale. ReleaseReservedIssuesForOrder now frees the receipt as part of the release. This command
// cleans up what the old paths left.
//
// # How it decides
//
// Per receipt, every allocation against it is taken through its own unit's ratio and totalled, and
// the receipt's own quantity through its. A receipt at `allocated` whose total is below its quantity —
// by more than --epsilon, which keeps DECIMAL(65,30) noise out of the results — has that much free and
// is returned to `available`. This is FreeReleasedReceipts from batch_scan_undo.sql, run over the
// whole table instead of over the receipts one release touched.
//
// The receipt's own quantity must be positive: a migration wrote opening balances as receipts with
// negative quantities, and one of those is under its capacity by definition without anything having
// gone wrong.
//
// # What happens to the stock
//
// Nothing else is written. An allocation is not created, an issue is not touched, and no quantity is
// changed — the stock was always on the receipt and this makes the allocator able to see it again.
//
// What the allocator does with it is then asked for rather than done here: one
// core.cmd.allocate_open_issues per (account, item) goes into the outbox, which is the same request
// every other release makes, and the consumer covers whatever open demand the item has. Without it a
// dormant item's freed stock would sit unoffered until something else happened to that item.
//
// Requests go to both the owner and the holder account when they differ, because demand for consigned
// stock sits under whichever of the two placed it and the consumer's discovery filters on one.
//
// # Order of operations
//
// Run repair-overallocated-receipts first if it has anything to do. It deletes allocations, which can
// take a receipt from fully drawn to partially drawn — a receipt this command should then free.
// Running in the other order leaves those receipts for the next run.
//
// # Re-running
//
// Safe, and cheap: a receipt already at `available` no longer matches. The apply loop commits per
// batch rather than as one transaction, so an interrupted run leaves the batches it finished repaired
// and the rest untouched, for the same reason — every receipt is independent of every other, and
// re-running picks up where it stopped. That is a deliberate difference from
// repair-overallocated-receipts, whose deletes have to be all-or-nothing.
//
// Usage:
//
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/repair-unreleased-receipts --dry-run
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/repair-unreleased-receipts --account ac_... --dry-run
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/repair-unreleased-receipts --account ac_...
//
// Flags:
//
//	--dry-run      report what would change; make no writes.
//	--account      restrict to one account id (matches owner or holder).
//	--item         restrict to one item id, for verifying the effect on a single SKU first.
//	--limit        cap the number of receipts freed (0 = no cap).
//	--epsilon      ignore shortfalls at or below this many base units (default 0.000001).
//	--no-enqueue   free the receipts but do not ask for allocation; the stock waits for the item's next pass.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/env"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/ledger"
	"github.com/open-mrp/api/shared/messaging"
)

func main() {
	ctx := context.Background()
	if err := Run(ctx, os.Args, os.Getenv, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err) // #nosec G705 -- CLI stderr output, not web context
		os.Exit(1)
	}
}

// unreleasedReceipt is one receipt held at `allocated` with stock still on it. Both totals are in base
// units, which is the only footing on which allocations recorded in different units can be compared to
// the receipt that carries them.
type unreleasedReceipt struct {
	ReceiptID      string
	ItemID         string
	SKU            string
	OwnerAccountID string
	HolderAccount  string
	ReceiptBase    decimal.Decimal
	AllocatedBase  decimal.Decimal
}

// Free is what the receipt is holding that nothing has a claim on.
func (r unreleasedReceipt) Free() decimal.Decimal { return r.ReceiptBase.Sub(r.AllocatedBase) }

// allocationRequest is one (account, item) whose open demand should be offered the freed stock.
type allocationRequest struct {
	AccountID string
	ItemID    string
}

// core-service is the service name the consumer's inbox and the enqueuer's metrics key on. It is a
// literal here because services/core-service/internal is not importable from cmd.
const coreServiceName = "core-service"

// allocateOpenIssuesEvent mirrors domain.AllocateOpenIssuesEvent, which lives under an internal
// package this command cannot import. The cursor fields are deliberately omitted: absent from the
// JSON they unmarshal to their zero values, which is what starts a chain at the beginning of the
// item's demand — exactly what every other producer of this command sends.
type allocateOpenIssuesEvent struct {
	AccountID string `json:"account_id"`
	ItemID    string `json:"item_id"`
}

func Run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("repair-unreleased-receipts", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", false, "report what would change; make no writes")
	accountID := fs.String("account", "", "restrict to a single account id (owner or holder)")
	itemID := fs.String("item", "", "restrict to a single item id")
	limit := fs.Int("limit", 0, "cap receipts freed (0 = no cap)")
	epsilon := fs.String("epsilon", ledger.EpsilonFlagDefault, "ignore shortfalls at or below this many base units")
	noEnqueue := fs.Bool("no-enqueue", false, "free the receipts but do not ask for allocation")
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

	receipts, err := findUnreleasedReceipts(ctx, sqlDB, *accountID, *itemID, *limit, eps, p)
	if err != nil {
		return err
	}

	requests := allocationRequests(receipts)
	totalFree := decimal.Zero
	neverDrawn := 0
	for _, r := range receipts {
		totalFree = totalFree.Add(r.Free())
		if r.AllocatedBase.IsZero() {
			neverDrawn++
		}
	}

	p.stage("Summary")
	fmt.Fprintf(stdout, "  Receipts to free:          %d\n", len(receipts))
	fmt.Fprintf(stdout, "    of which never drawn:    %d\n", neverDrawn)
	fmt.Fprintf(stdout, "  Base units made visible:   %s\n", totalFree.String())
	fmt.Fprintf(stdout, "  Allocation requests:       %d\n", len(requests))

	printLargestHoldings(stdout, receipts)

	if len(receipts) == 0 {
		p.stage("Nothing to do")
		return nil
	}

	if *dryRun {
		p.stage("--dry-run: no writes made")
		return nil
	}

	freed, err := freeReceipts(ctx, sqlDB, receipts, eps, p)
	if err != nil {
		return err
	}

	enqueued := 0
	if *noEnqueue {
		p.stage("--no-enqueue: freed stock will be offered on the item's next allocation pass")
	} else {
		if enqueued, err = enqueueAllocationRequests(ctx, sqlDB, requests, p); err != nil {
			return err
		}
	}

	p.stage("Done")
	fmt.Fprintf(stdout, "  Returned %d receipt(s) to available.\n", freed)
	fmt.Fprintf(stdout, "  Enqueued %d allocation request(s).\n", enqueued)
	return nil
}

// findUnreleasedReceipts lists the receipts held at `allocated` that are no longer drawn to capacity.
//
// LEFT JOIN, not JOIN: a receipt whose allocations were all deleted has no row in the grouped total at
// all, and it is the clearest case of what this repairs — an inner join would drop precisely the
// receipts that most need freeing.
//
// `status_code = 'allocated'` rather than `<> 'available'`: those are the only two statuses the table
// has, and naming the one being repaired says what the command looks for rather than what it skips.
func findUnreleasedReceipts(ctx context.Context, sqlDB *sql.DB, accountID, itemID string, limit int, eps decimal.Decimal, p *progress) ([]unreleasedReceipt, error) {
	var where strings.Builder
	var params []any
	if accountID != "" {
		where.WriteString(" AND (ir.owner_account_id = ? OR ir.holder_account_id = ?)")
		params = append(params, accountID, accountID)
	}
	if itemID != "" {
		where.WriteString(" AND ir.item_id = ?")
		params = append(params, itemID)
	}

	// #nosec G202 -- the only interpolation is `where`, built above from fixed literals; the account
	// and item ids it filters on are bound as parameters.
	query := `
SELECT ir.id, ir.item_id, i.sku, ir.owner_account_id, ir.holder_account_id,
       CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator) AS receipt_base,
       COALESCE(a.alloc_base, 0) AS alloc_base
FROM inventory_receipt ir
JOIN quantity q ON q.id = ir.quantity_id
JOIN unit u ON u.id = q.unit_id
JOIN item i ON i.id = ir.item_id
LEFT JOIN (
    SELECT ia.inventory_receipt_id AS rid,
           SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator)) AS alloc_base
    FROM inventory_allocation ia
    JOIN quantity aq ON aq.id = ia.quantity_id
    JOIN unit au ON au.id = aq.unit_id
    GROUP BY ia.inventory_receipt_id
) a ON a.rid = ir.id
WHERE ir.status_code = 'allocated'
  AND CAST(q.value AS DECIMAL(65,30)) > 0` + where.String() + `
  AND COALESCE(a.alloc_base, 0) < CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator) - ?
ORDER BY (CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator) - COALESCE(a.alloc_base, 0)) DESC`

	params = append(params, eps.String())

	p.stage("Scanning inventory_receipt for stock held at `allocated` (one pass, may take a minute)...")

	rows, err := sqlDB.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("find unreleased receipts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []unreleasedReceipt
	for rows.Next() {
		var r unreleasedReceipt
		var receiptBase, allocBase string
		if err := rows.Scan(&r.ReceiptID, &r.ItemID, &r.SKU, &r.OwnerAccountID, &r.HolderAccount,
			&receiptBase, &allocBase); err != nil {
			return nil, fmt.Errorf("scan unreleased receipt: %w", err)
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
		return nil, fmt.Errorf("iterate unreleased receipts: %w", err)
	}

	p.stagef("Found %d receipt(s) holding stock at `allocated`", len(out))
	return out, nil
}

// allocationRequests reduces the receipts to the (account, item) pairs worth asking allocation for,
// deduplicated and ordered so two runs over the same data produce the same sequence of outbox rows.
//
// Both accounts on a receipt are asked when they differ: consigned stock sits under a holder while
// another account owns it, and the demand it should cover may have been recorded under either.
func allocationRequests(receipts []unreleasedReceipt) []allocationRequest {
	seen := map[allocationRequest]struct{}{}
	var out []allocationRequest
	add := func(accountID, itemID string) {
		if accountID == "" {
			return
		}
		req := allocationRequest{AccountID: accountID, ItemID: itemID}
		if _, dup := seen[req]; dup {
			return
		}
		seen[req] = struct{}{}
		out = append(out, req)
	}
	for _, r := range receipts {
		add(r.OwnerAccountID, r.ItemID)
		add(r.HolderAccount, r.ItemID)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].AccountID != out[j].AccountID {
			return out[i].AccountID < out[j].AccountID
		}
		return out[i].ItemID < out[j].ItemID
	})
	return out
}

// printLargestHoldings lists the receipts giving back the most, which is what a reviewer checks
// against a physical count before letting the run write anything.
func printLargestHoldings(out io.Writer, receipts []unreleasedReceipt) {
	if len(receipts) == 0 {
		return
	}
	fmt.Fprintf(out, "\n  Largest holdings (receipt, SKU, held / allocated / freed, base units):\n")
	for i, r := range receipts {
		if i >= largestHoldingLines {
			fmt.Fprintf(out, "    ... and %d more\n", len(receipts)-largestHoldingLines)
			break
		}
		fmt.Fprintf(out, "    %-22s %-24s %s / %s / +%s\n",
			r.ReceiptID, r.SKU, r.ReceiptBase.String(), r.AllocatedBase.String(), r.Free().String())
	}
	fmt.Fprintln(out)
}

const largestHoldingLines = 20

// freeReceipts returns the receipts to `available`, a batch per transaction.
//
// The condition is re-evaluated in the UPDATE rather than trusted from the scan: the scan ran outside
// any transaction and an allocation may have landed on one of these receipts since, which would make
// freeing it wrong. A receipt that no longer qualifies is simply not updated, and the count reported
// at the end is what actually changed.
func freeReceipts(ctx context.Context, sqlDB *sql.DB, receipts []unreleasedReceipt, eps decimal.Decimal, p *progress) (int, error) {
	ids := make([]string, 0, len(receipts))
	for _, r := range receipts {
		ids = append(ids, r.ReceiptID)
	}

	p.stagef("Freeing %d receipt(s), %d per transaction...", len(ids), freeBatchSize)

	total := 0
	done := 0
	for chunk := range chunks(ids, freeBatchSize) {
		// #nosec G202 -- placeholders() emits only "?,?,…"; the ids themselves are bound below.
		query := `
UPDATE inventory_receipt ir
JOIN quantity q ON q.id = ir.quantity_id
JOIN unit u ON u.id = q.unit_id
SET ir.status_code = 'available', ir.updated_at = NOW(3)
WHERE ir.id IN (` + placeholders(len(chunk)) + `)
AND ir.status_code = 'allocated'
AND CAST(q.value AS DECIMAL(65,30)) > 0
AND COALESCE((
    SELECT SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator))
    FROM inventory_allocation ia
    JOIN quantity aq ON aq.id = ia.quantity_id
    JOIN unit au ON au.id = aq.unit_id
    WHERE ia.inventory_receipt_id = ir.id
), 0) < CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator) - ?`

		args := append(toAny(chunk), eps.String())
		res, err := sqlDB.ExecContext(ctx, query, args...)
		if err != nil {
			return total, fmt.Errorf("free receipts (%d already freed and committed): %w", total, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return total, fmt.Errorf("free receipts rows affected: %w", err)
		}
		total += int(n)
		done += len(chunk)
		p.step(done, len(ids), "receipts checked")
	}
	return total, nil
}

const freeBatchSize = 500

// enqueueAllocationRequests writes one core.cmd.allocate_open_issues per (account, item) into the
// outbox, which the enqueuer publishes on its next poll.
//
// Written straight to message_outbox because the repositories that own this are under
// services/core-service/internal and cannot be imported here. The row is the same one
// mediator.EnqueueAllocateOpenIssues produces: pending, no headers, no request or parent id, a fresh
// random message id per request so each is its own chain.
func enqueueAllocationRequests(ctx context.Context, sqlDB *sql.DB, requests []allocationRequest, p *progress) (int, error) {
	p.stagef("Enqueuing %d allocation request(s)...", len(requests))

	const query = `
INSERT INTO message_outbox (
    message_id, service_name, message_type, destination, routing_key,
    headers, payload, status, max_attempts, next_run_at, request_id, parent_message_id
) VALUES (?, ?, ?, ?, ?, NULL, ?, 'pending', ?, NOW(3), NULL, NULL)`

	total := 0
	for i, req := range requests {
		// A conversion, not a literal: the two structs are the same pair of fields, and the compiler
		// says so here the day the wire shape stops matching.
		data, err := json.Marshal(allocateOpenIssuesEvent(req))
		if err != nil {
			return total, fmt.Errorf("marshal allocate open issues event: %w", err)
		}
		payload, err := json.Marshal(contracts.AmqpMessage{Data: data})
		if err != nil {
			return total, fmt.Errorf("marshal outbox payload: %w", err)
		}
		messageID, apiErr := id.GenID(id.MessageIDPrefix, nil)
		if apiErr != nil {
			return total, fmt.Errorf("generate message id: %w", apiErr)
		}

		if _, err := sqlDB.ExecContext(ctx, query,
			messageID, coreServiceName, string(contracts.CoreCmdAllocateOpenIssues),
			messaging.ApplicationExchange, string(contracts.CoreCmdAllocateOpenIssues),
			payload, outboxMaxAttempts,
		); err != nil {
			return total, fmt.Errorf("enqueue allocation request for %s/%s (%d already enqueued): %w",
				req.AccountID, req.ItemID, total, err)
		}
		total++
		p.step(i+1, len(requests), "requests enqueued")
	}
	return total, nil
}

// outboxMaxAttempts matches the column default the service's own outbox insert relies on.
const outboxMaxAttempts = 25

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
// in cmd/repair-overallocated-receipts; both are one-off commands in package main and cannot share it.
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
