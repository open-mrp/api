// Command restate-sales-line-costs points sales order lines still carrying an item's stale unit cost at
// the item's current one, and marks them for the sales fact refresher.
//
// A line's cost is a copy of its item's, taken when the line was written, so correcting the item leaves
// the copies stale. Lines added or edited one at a time round the copy to the cent, so they are matched on
// the stale value rounded to the cent. Some lines carry no item_id and reach their item only through their
// product. Identifiers are flags, never committed:
//
//	DATABASE_DSN='user:pass@tcp(127.0.0.1:3306)/augno_core' go run ./cmd/restate-sales-line-costs \
//	  --account ac_... --item <item id> --stale-cost 8.40 --since 2026-01-01 [--stale-unit <unit id>] [--apply]
//
// Without --apply it only lists the lines it would change.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/shopspring/decimal"
)

// errBadFlags signals a flag-parse failure; the FlagSet already printed the
// message and usage to stderr, so main exits nonzero without re-printing.
var errBadFlags = errors.New("invalid command-line flags")

const dateLayout = "2006-01-02"

type options struct {
	accountID string
	itemIDs   []string
	staleCost decimal.Decimal
	staleUnit string
	since     time.Time
	until     time.Time
	apply     bool
}

// staleLine is one sales order line whose cost is a copy of the item's stale cost.
type staleLine struct {
	LineID      string
	OrderNumber string
	ItemID      string
	CreatedOn   string
	Cost        string
	CostUnit    string
	ItemCost    string
	ItemUnit    string
}

func main() {
	ctx := context.Background()
	if err := Run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		if !errors.Is(err, errBadFlags) {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		os.Exit(1)
	}
}

func Run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	opts, err := parseFlags(args, stderr)
	if err != nil {
		return err
	}

	dsn := getenv("DATABASE_DSN")
	if dsn == "" {
		return errors.New("missing required DATABASE_DSN")
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	return restate(ctx, pool, opts, stdout)
}

func parseFlags(args []string, stderr io.Writer) (options, error) {
	var opts options
	var items, staleCost, since, until string

	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.accountID, "account", "", "Account that owns the items")
	flags.StringVar(&items, "item", "", "Comma-separated item IDs whose cost was corrected")
	flags.StringVar(&staleCost, "stale-cost", "", "The item's cost before it was corrected; lines are matched on it rounded to the cent")
	flags.StringVar(&opts.staleUnit, "stale-unit", "", "Unit the stale cost was per (default: the unit the item's cost is per now)")
	flags.StringVar(&since, "since", "", "Earliest line creation date to restate, YYYY-MM-DD (UTC)")
	flags.StringVar(&until, "until", "", "Line creation date to stop before, YYYY-MM-DD (UTC; default: now)")
	flags.BoolVar(&opts.apply, "apply", false, "Write the changes; without it the lines are only listed")
	if err := flags.Parse(args[1:]); err != nil {
		return opts, errBadFlags
	}

	if opts.accountID == "" {
		return opts, errors.New("missing required --account")
	}
	for _, id := range strings.Split(items, ",") {
		if id = strings.TrimSpace(id); id != "" {
			opts.itemIDs = append(opts.itemIDs, id)
		}
	}
	if len(opts.itemIDs) == 0 {
		return opts, errors.New("missing required --item")
	}
	if staleCost == "" {
		return opts, errors.New("missing required --stale-cost")
	}
	cost, err := decimal.NewFromString(staleCost)
	if err != nil || !cost.IsPositive() {
		return opts, fmt.Errorf("--stale-cost must be a positive number, got %q", staleCost)
	}
	opts.staleCost = cost
	if since == "" {
		return opts, errors.New("missing required --since")
	}
	if opts.since, err = time.Parse(dateLayout, since); err != nil {
		return opts, fmt.Errorf("--since must be YYYY-MM-DD, got %q", since)
	}
	opts.until = time.Now().UTC()
	if until != "" {
		if opts.until, err = time.Parse(dateLayout, until); err != nil {
			return opts, fmt.Errorf("--until must be YYYY-MM-DD, got %q", until)
		}
	}
	if !opts.since.Before(opts.until) {
		return opts, errors.New("--since must be before --until")
	}

	return opts, nil
}

// staleLineFilter is shared by the listing and the update, so a line that moves between them is left alone.
// A line's own rate is never the item's (r.id <> ir.id), so the item's cost is not touched.
const staleLineFilter = `
	(sol.item_id IN (%[1]s) OR (sol.item_id IS NULL AND sol.product_id IN (SELECT p.id FROM product p WHERE p.item_id IN (%[1]s))))
	AND sol.created_at >= ? AND sol.created_at < ?
	AND r.id <> ir.id
	AND r.numerator_unit_id = ir.numerator_unit_id
	AND r.denominator_unit_id = COALESCE(?, ir.denominator_unit_id)
	AND ROUND(r.value, 2) = ROUND(?, 2)
	AND NOT (r.value = ir.value AND r.denominator_unit_id = ir.denominator_unit_id)`

func filterArgs(opts options) []any {
	args := make([]any, 0, 2*len(opts.itemIDs)+5)
	args = append(args, opts.accountID)
	// The filter names the items twice: on the line, and on the product of a line without one.
	for range 2 {
		for _, id := range opts.itemIDs {
			args = append(args, id)
		}
	}
	var staleUnit any
	if opts.staleUnit != "" {
		staleUnit = opts.staleUnit
	}
	return append(args, opts.since, opts.until, staleUnit, opts.staleCost.String())
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func restate(ctx context.Context, pool *sql.DB, opts options, stdout io.Writer) error {
	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	lines, err := listStaleLines(ctx, tx, opts)
	if err != nil {
		return err
	}
	printLines(stdout, lines)

	if len(lines) == 0 || !opts.apply {
		if len(lines) > 0 {
			fmt.Fprintln(stdout, "Dry run: nothing written. Re-run with --apply to restate these lines.")
		}
		return nil
	}

	// Marked first, matching the data migrations, so the refresher re-prices every line this changes.
	ids := make([]any, 0, len(lines))
	for _, l := range lines {
		ids = append(ids, l.LineID)
	}
	marks := `INSERT INTO sales_fact_dirty (scope_type, scope_id, account_id, marked_at)
		SELECT 'sales_order_line', sol.id, ?, NOW(3) FROM sales_order_line sol WHERE sol.id IN (` + placeholders(len(ids)) + `)
		ON DUPLICATE KEY UPDATE marked_at = VALUES(marked_at)`
	if _, err := tx.ExecContext(ctx, marks, append([]any{opts.accountID}, ids...)...); err != nil {
		return fmt.Errorf("mark lines for the sales fact refresher: %w", err)
	}

	update := `UPDATE rate r
		JOIN sales_order_line sol ON sol.unit_cost_id = r.id
		LEFT JOIN product lp ON lp.id = sol.product_id
		JOIN item i ON i.id = COALESCE(sol.item_id, lp.item_id) AND i.account_id = ?
		JOIN rate ir ON ir.id = i.unit_cost_id
		SET r.value = ir.value, r.denominator_unit_id = ir.denominator_unit_id, r.updated_at = NOW(3)
		WHERE ` + fmt.Sprintf(staleLineFilter, placeholders(len(opts.itemIDs))) + `
		AND sol.id IN (` + placeholders(len(ids)) + `)`
	res, err := tx.ExecContext(ctx, update, append(filterArgs(opts), ids...)...)
	if err != nil {
		return fmt.Errorf("restate line costs: %w", err)
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("count restated lines: %w", err)
	}
	if changed != int64(len(lines)) {
		return fmt.Errorf("restated %d lines, expected %d; rolled back", changed, len(lines))
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	fmt.Fprintf(stdout, "Restated %d lines.\n", changed)
	return nil
}

func listStaleLines(ctx context.Context, tx *sql.Tx, opts options) ([]staleLine, error) {
	query := `SELECT sol.id, so.number, i.id, DATE_FORMAT(sol.created_at, '%Y-%m-%d'), r.value, r.denominator_unit_id, ir.value, ir.denominator_unit_id
		FROM sales_order_line sol
		JOIN sales_order so ON so.id = sol.sales_order_id
		LEFT JOIN product lp ON lp.id = sol.product_id
		JOIN item i ON i.id = COALESCE(sol.item_id, lp.item_id) AND i.account_id = ?
		JOIN rate ir ON ir.id = i.unit_cost_id
		JOIN rate r ON r.id = sol.unit_cost_id
		WHERE ` + fmt.Sprintf(staleLineFilter, placeholders(len(opts.itemIDs))) + `
		ORDER BY sol.created_at, sol.id
		FOR UPDATE`
	rows, err := tx.QueryContext(ctx, query, filterArgs(opts)...)
	if err != nil {
		return nil, fmt.Errorf("list stale lines: %w", err)
	}
	defer rows.Close()

	var lines []staleLine
	for rows.Next() {
		var l staleLine
		if err := rows.Scan(&l.LineID, &l.OrderNumber, &l.ItemID, &l.CreatedOn, &l.Cost, &l.CostUnit, &l.ItemCost, &l.ItemUnit); err != nil {
			return nil, fmt.Errorf("scan stale line: %w", err)
		}
		lines = append(lines, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list stale lines: %w", err)
	}
	return lines, nil
}

func printLines(stdout io.Writer, lines []staleLine) {
	if len(lines) == 0 {
		fmt.Fprintln(stdout, "No sales order lines carry the stale cost.")
		return
	}
	w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ORDER\tLINE\tITEM\tCREATED\tCOST NOW\tPER\tRESTATED TO\tPER")
	for _, l := range lines {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			l.OrderNumber, l.LineID, l.ItemID, l.CreatedOn, l.Cost, l.CostUnit, l.ItemCost, l.ItemUnit)
	}
	w.Flush()
	fmt.Fprintf(stdout, "%d lines.\n", len(lines))
}
