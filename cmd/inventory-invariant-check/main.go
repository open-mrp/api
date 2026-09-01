// Command inventory-invariant-check reports inventory ledger rows that violate an invariant the
// allocator is supposed to maintain. It is read-only and never repairs anything.
//
// # Why it exists
//
// Every invariant in the ledger is arithmetic in Go — read the allocations, subtract, write more —
// and there is not one foreign key or capacity constraint in the schema to catch it when that
// arithmetic runs on a stale read. Twice now a concurrency bug has corrupted the ledger and nobody
// found out from the ledger: 2026-08-26 surfaced as a shortage report, and the over-drawn receipt
// still present when this command was written surfaced only because someone went looking during an
// unrelated deadlock investigation. The repair commands next to this one can fix what they find, but
// they only run when a human already suspects something. This runs on a schedule and says so first.
//
// # Exit status
//
// 0 when every detector comes back empty, 1 when any of them finds a row, 2 when the run itself
// failed. A scheduler should alarm on 1 and page on 2 — a detector that cannot run is worse than one
// that finds something, because it is the silence that has cost us twice.
//
// # The detectors
//
// Each one names an invariant and returns the rows that break it. They are deliberately independent:
// a single receipt over-draw shows up in over_drawn_receipts, and if it also left an issue wrongly
// closed it shows up again in closed_issue_not_covered. Overlap is cheaper than a gap.
//
// Two guards appear throughout and both matter. Quantities must be positive before a comparison
// means anything — a migration wrote opening balances as receipts with negative quantities, and any
// allocation at all against one of those reads as an overshoot with nothing wrong. And every
// comparison carries an epsilon, because these are DECIMAL(65,30) sums through unit ratios and the
// last places are noise, not shortage.
//
// Usage:
//
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/inventory-invariant-check
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/inventory-invariant-check --account ac_...
//	go run ./cmd/inventory-invariant-check --print-sql
//
// Flags:
//
//	--account     restrict to one account id.
//	--item        restrict to one item id.
//	--epsilon     ignore differences at or below this many base units (default 0.000001).
//	--samples     how many offending rows to print per detector (default 5, 0 = none).
//	--only        run one detector by name.
//	--print-sql   print each detector's SQL and exit; makes no connection.
package main

import (
	"context"
	"database/sql"
	"errors"
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

// exitViolations is returned when the ledger is intact except for what the detectors found. It is
// separate from a run failure so a scheduler can tell "the ledger is wrong" from "the check broke".
const exitViolations = 1

func main() {
	err := run(context.Background(), os.Args[1:], os.Getenv, os.Stdout)
	var v violationsFound
	switch {
	case errors.As(err, &v):
		os.Exit(exitViolations)
	case err != nil:
		fmt.Fprintf(os.Stderr, "inventory-invariant-check: %v\n", err)
		os.Exit(2)
	}
}

// violationsFound reports a clean run that found broken rows, so main can exit 1 rather than 2.
type violationsFound struct{ total int }

func (v violationsFound) Error() string {
	return fmt.Sprintf("%d ledger invariant violation(s) found", v.total)
}

// detector is one invariant and the query that finds rows breaking it.
//
// accountFilter and itemFilter are the column expressions the optional --account/--item flags
// compare against. They differ per detector because the account reaches the row by a different path
// depending on which table the detector drives from, and getting that wrong silently returns
// everything rather than erroring.
type detector struct {
	name          string
	invariant     string
	query         string
	accountFilter string
	itemFilter    string
	// reportOnly keeps a detector out of the exit status. See the note on the one that sets it: a
	// check that alarms on a backlog nobody is going to clear teaches people to ignore the alarm,
	// which costs more than the check is worth.
	reportOnly bool
}

// detectors builds the full set with eps already substituted.
//
// eps is interpolated rather than bound because several queries use it more than once and in
// positions that would make the parameter order a thing to get wrong on every future edit. It is
// parsed as a decimal before it gets here, so it is a number and nothing else.
func detectors(eps decimal.Decimal) []detector {
	e := eps.String()

	// Allocations totalled per receipt and per issue, each row through its own unit's ratio. An
	// allocation against one receipt may be recorded in a different unit from the receipt itself, so
	// the raw column values do not add up to anything.
	perReceipt := `
    SELECT ia.inventory_receipt_id AS rid,
           SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator)) AS alloc_base
    FROM inventory_allocation ia
    JOIN quantity aq ON aq.id = ia.quantity_id
    JOIN unit au ON au.id = aq.unit_id
    GROUP BY ia.inventory_receipt_id`

	perIssue := `
    SELECT ia.inventory_issue_id AS iid,
           SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator)) AS alloc_base
    FROM inventory_allocation ia
    JOIN quantity aq ON aq.id = ia.quantity_id
    JOIN unit au ON au.id = aq.unit_id
    GROUP BY ia.inventory_issue_id`

	receiptBase := `CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator)`
	issueBase := `CAST(iq.value AS DECIMAL(65,30)) * (iu.ratio_numerator / iu.ratio_denominator)`

	return []detector{
		{
			name:      "over_drawn_receipts",
			invariant: "a receipt's allocations never total more than the receipt holds",
			// #nosec G202 -- eps is a parsed decimal; the account and item filters are bound parameters.
			query: `
SELECT ir.id AS receipt_id, ir.item_id, ir.status_code,
       ` + receiptBase + ` AS receipt_base,
       a.alloc_base,
       a.alloc_base - ` + receiptBase + ` AS over_by
FROM inventory_receipt ir
JOIN quantity q ON q.id = ir.quantity_id
JOIN unit u ON u.id = q.unit_id
JOIN (` + perReceipt + `
) a ON a.rid = ir.id
WHERE CAST(q.value AS DECIMAL(65,30)) > 0
  AND a.alloc_base > ` + receiptBase + ` + ` + e + `
{{FILTER}}
ORDER BY over_by DESC`,
			accountFilter: "ir.owner_account_id",
			itemFilter:    "ir.item_id",
		},
		{
			name:      "over_allocated_issues",
			invariant: "an issue's allocations never total more than the issue demanded",
			query: `
SELECT ii.id AS issue_id, ii.item_id, ii.status_code,
       ` + issueBase + ` AS demand_base,
       a.alloc_base,
       a.alloc_base - ` + issueBase + ` AS over_by
FROM inventory_issue ii
JOIN quantity iq ON iq.id = ii.quantity_id
JOIN unit iu ON iu.id = iq.unit_id
JOIN (` + perIssue + `
) a ON a.iid = ii.id
WHERE CAST(iq.value AS DECIMAL(65,30)) > 0
  AND a.alloc_base > ` + issueBase + ` + ` + e + `
{{FILTER}}
ORDER BY over_by DESC`,
			accountFilter: "ii.account_id",
			itemFilter:    "ii.item_id",
		},
		{
			name:      "non_positive_allocations",
			invariant: "an allocation always moves a positive quantity",
			query: `
SELECT ia.id AS allocation_id, ia.inventory_issue_id, ia.inventory_receipt_id, q.value
FROM inventory_allocation ia
JOIN quantity q ON q.id = ia.quantity_id
JOIN inventory_issue ii ON ii.id = ia.inventory_issue_id
WHERE CAST(q.value AS DECIMAL(65,30)) <= 0
{{FILTER}}
ORDER BY q.value ASC`,
			accountFilter: "ii.account_id",
			itemFilter:    "ii.item_id",
		},
		{
			name:      "duplicate_allocations",
			invariant: "one issue never draws the same quantity from one receipt twice",
			query: `
SELECT ia.inventory_issue_id, ia.inventory_receipt_id, q.unit_id, q.value, COUNT(*) AS copies
FROM inventory_allocation ia
JOIN quantity q ON q.id = ia.quantity_id
JOIN inventory_issue ii ON ii.id = ia.inventory_issue_id
WHERE 1 = 1
{{FILTER}}
GROUP BY ia.inventory_issue_id, ia.inventory_receipt_id, q.unit_id, q.value
HAVING COUNT(*) > 1
ORDER BY copies DESC`,
			accountFilter: "ii.account_id",
			itemFilter:    "ii.item_id",
		},
		{
			name:      "closed_issue_not_covered",
			invariant: "an issue is closed only once its allocations cover its demand",
			// Report-only, and the numbers are why. On 2026-09-01 this matched 656 issues, 543 of
			// which have no allocations at all — a backlog from paths that close demand without ever
			// allocating it, accumulated long before anyone was watching. The invariant is real and
			// worth seeing, but alarming on a number that starts at 656 and is nobody's current task
			// just trains people to skip the report. Promote it to alarming once the backlog is
			// explained and cleared, and it becomes one of the strongest signals here.
			reportOnly: true,
			query: `
SELECT ii.id AS issue_id, ii.item_id,
       ` + issueBase + ` AS demand_base,
       COALESCE(a.alloc_base, 0) AS alloc_base,
       ` + issueBase + ` - COALESCE(a.alloc_base, 0) AS short_by
FROM inventory_issue ii
JOIN quantity iq ON iq.id = ii.quantity_id
JOIN unit iu ON iu.id = iq.unit_id
LEFT JOIN (` + perIssue + `
) a ON a.iid = ii.id
WHERE ii.status_code = 'closed'
  AND CAST(iq.value AS DECIMAL(65,30)) > 0
  AND COALESCE(a.alloc_base, 0) < ` + issueBase + ` - ` + e + `
{{FILTER}}
ORDER BY short_by DESC`,
			accountFilter: "ii.account_id",
			itemFilter:    "ii.item_id",
		},
		{
			name:      "covered_issue_left_open",
			invariant: "an issue whose demand is covered is closed, not left open to be re-examined forever",
			query: `
SELECT ii.id AS issue_id, ii.item_id,
       ` + issueBase + ` AS demand_base,
       a.alloc_base
FROM inventory_issue ii
JOIN quantity iq ON iq.id = ii.quantity_id
JOIN unit iu ON iu.id = iq.unit_id
JOIN (` + perIssue + `
) a ON a.iid = ii.id
WHERE ii.status_code = 'open'
  AND CAST(iq.value AS DECIMAL(65,30)) > 0
  AND a.alloc_base >= ` + issueBase + ` - ` + e + `
{{FILTER}}
ORDER BY ii.created_at ASC`,
			accountFilter: "ii.account_id",
			itemFilter:    "ii.item_id",
		},
		{
			name:      "drawn_receipt_still_available",
			invariant: "a fully drawn receipt is marked allocated, so later passes stop reconsidering it",
			query: `
SELECT ir.id AS receipt_id, ir.item_id,
       ` + receiptBase + ` AS receipt_base,
       a.alloc_base
FROM inventory_receipt ir
JOIN quantity q ON q.id = ir.quantity_id
JOIN unit u ON u.id = q.unit_id
JOIN (` + perReceipt + `
) a ON a.rid = ir.id
WHERE ir.status_code = 'available'
  AND CAST(q.value AS DECIMAL(65,30)) > 0
  AND a.alloc_base >= ` + receiptBase + ` - ` + e + `
{{FILTER}}
ORDER BY ir.received_at ASC`,
			accountFilter: "ir.owner_account_id",
			itemFilter:    "ir.item_id",
		},
		{
			name:      "orphaned_allocations",
			invariant: "every allocation points at an issue, a receipt, and its three satellite rows",
			// No foreign keys exist on any of these columns, so nothing but this notices.
			query: `
SELECT ia.id AS allocation_id,
       ii.id IS NULL AS missing_issue,
       ir.id IS NULL AS missing_receipt,
       q.id IS NULL AS missing_quantity,
       uc.id IS NULL AS missing_unit_cost,
       tc.id IS NULL AS missing_total_cost
FROM inventory_allocation ia
LEFT JOIN inventory_issue ii ON ii.id = ia.inventory_issue_id
LEFT JOIN inventory_receipt ir ON ir.id = ia.inventory_receipt_id
LEFT JOIN quantity q ON q.id = ia.quantity_id
LEFT JOIN rate uc ON uc.id = ia.unit_cost_id
LEFT JOIN quantity tc ON tc.id = ia.total_cost_id
WHERE ii.id IS NULL OR ir.id IS NULL OR q.id IS NULL OR uc.id IS NULL OR tc.id IS NULL
{{FILTER}}`,
			// Driven from the allocation side precisely to catch rows whose issue is gone, so the
			// account cannot be reached through the issue without discarding those rows.
			accountFilter: "",
			itemFilter:    "",
		},
	}
}

// resolve substitutes the optional account/item filters into a detector's query.
func (d detector) resolve(accountID, itemID string) (string, []any) {
	var filter strings.Builder
	var params []any

	if accountID != "" && d.accountFilter != "" {
		filter.WriteString("  AND " + d.accountFilter + " = ?\n")
		params = append(params, accountID)
	}
	if itemID != "" && d.itemFilter != "" {
		filter.WriteString("  AND " + d.itemFilter + " = ?\n")
		params = append(params, itemID)
	}

	return strings.Replace(d.query, "{{FILTER}}", strings.TrimRight(filter.String(), "\n"), 1), params
}

// skipped reports whether a filter was asked for that this detector cannot honour, so the report can
// say so rather than quietly returning rows from every account.
func (d detector) skipped(accountID, itemID string) bool {
	return (accountID != "" && d.accountFilter == "") || (itemID != "" && d.itemFilter == "")
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout io.Writer) error {
	fs := flag.NewFlagSet("inventory-invariant-check", flag.ContinueOnError)
	fs.SetOutput(stdout)
	accountID := fs.String("account", "", "restrict to one account id")
	itemID := fs.String("item", "", "restrict to one item id")
	epsilon := fs.String("epsilon", "0.000001", "ignore differences at or below this many base units")
	samples := fs.Int("samples", 5, "offending rows to print per detector (0 = none)")
	only := fs.String("only", "", "run one detector by name")
	printSQL := fs.Bool("print-sql", false, "print each detector's SQL and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}

	eps, err := decimal.NewFromString(*epsilon)
	if err != nil {
		return fmt.Errorf("parsing --epsilon: %w", err)
	}

	all := detectors(eps)
	if *only != "" {
		filtered := all[:0:0]
		for _, d := range all {
			if d.name == *only {
				filtered = append(filtered, d)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("no detector named %q", *only)
		}
		all = filtered
	}

	// --print-sql makes no connection: the point is to paste a detector into a console and run it by
	// hand against a database this command has no credentials for.
	if *printSQL {
		for _, d := range all {
			q, _ := d.resolve(*accountID, *itemID)
			fmt.Fprintf(stdout, "-- %s: %s\n%s;\n\n", d.name, d.invariant, strings.TrimSpace(q))
		}
		return nil
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

	start := time.Now()
	total := 0

	for _, d := range all {
		if d.skipped(*accountID, *itemID) {
			fmt.Fprintf(stdout, "~ %-28s SKIPPED (cannot be filtered by account or item)\n", d.name)
			continue
		}

		query, params := d.resolve(*accountID, *itemID)
		cols, rows, qErr := queryAll(ctx, sqlDB, query, params)
		if qErr != nil {
			return fmt.Errorf("detector %s: %w", d.name, qErr)
		}

		if len(rows) == 0 {
			fmt.Fprintf(stdout, "  %-28s ok\n", d.name)
			continue
		}

		marker := "!"
		if d.reportOnly {
			marker = "~"
		} else {
			total += len(rows)
		}
		fmt.Fprintf(stdout, "%s %-28s %d row(s)%s\n", marker, d.name, len(rows), reportOnlySuffix(d))
		fmt.Fprintf(stdout, "      invariant: %s\n", d.invariant)
		printSamples(stdout, cols, rows, *samples)
	}

	fmt.Fprintf(stdout, "\nchecked %d detector(s) in %s\n", len(all), time.Since(start).Round(time.Millisecond))
	if total > 0 {
		fmt.Fprintf(stdout, "FAIL: %d offending row(s)\n", total)
		return violationsFound{total: total}
	}
	fmt.Fprintln(stdout, "OK: no invariant violations")
	return nil
}

func reportOnlySuffix(d detector) string {
	if d.reportOnly {
		return "  (report-only, does not fail the run)"
	}
	return ""
}

// queryAll reads a whole result set as strings. The detectors return different shapes and the report
// only prints them, so nothing is gained by giving each one a struct to scan into.
func queryAll(ctx context.Context, sqlDB *sql.DB, query string, params []any) ([]string, [][]string, error) {
	rows, err := sqlDB.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()

	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	var out [][]string
	for rows.Next() {
		cells := make([]sql.NullString, len(cols))
		dest := make([]any, len(cols))
		for i := range cells {
			dest[i] = &cells[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, nil, err
		}
		rec := make([]string, len(cols))
		for i, c := range cells {
			if c.Valid {
				rec[i] = c.String
			} else {
				rec[i] = "NULL"
			}
		}
		out = append(out, rec)
	}
	return cols, out, rows.Err()
}

func printSamples(stdout io.Writer, cols []string, rows [][]string, samples int) {
	if samples <= 0 {
		return
	}
	fmt.Fprintf(stdout, "      %s\n", strings.Join(cols, " | "))
	for i, r := range rows {
		if i >= samples {
			fmt.Fprintf(stdout, "      ... and %d more\n", len(rows)-samples)
			break
		}
		fmt.Fprintf(stdout, "      %s\n", strings.Join(trimDecimals(r), " | "))
	}
}

// trimDecimals shortens DECIMAL(65,30) values, which arrive with thirty places of which the last
// twenty-odd are always zeros and make the report unreadable.
func trimDecimals(rec []string) []string {
	out := make([]string, len(rec))
	for i, v := range rec {
		if d, err := decimal.NewFromString(v); err == nil && strings.Contains(v, ".") {
			out[i] = d.Truncate(6).String()
			continue
		}
		out[i] = v
	}
	return out
}

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
