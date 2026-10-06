// Command backfill-sales-line-items links sales order lines written without an item_id to their product's
// item. The dashboard wrote every line that way until it began connecting the item, and anything that
// finds lines by item (cost restatements, the item on an order, pick or shipment line) skipped them.
//
// Shipping and credit lines stay unlinked, as both the Go API and the dashboard create them. updated_at is
// left alone: the line itself has not changed, only the link the dashboard failed to write.
//
//	DATABASE_DSN='user:pass@tcp(127.0.0.1:3306)/augno_core' go run ./cmd/backfill-sales-line-items \
//	  --account ac_... [--apply] [--batch 500]
//
// Without --apply it only counts the lines it would link.
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

	_ "github.com/go-sql-driver/mysql"
)

// errBadFlags signals a flag-parse failure; the FlagSet already printed the
// message and usage to stderr, so main exits nonzero without re-printing.
var errBadFlags = errors.New("invalid command-line flags")

type options struct {
	accountID string
	batch     int
	apply     bool
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

	if err := printSummary(ctx, pool, opts, stdout); err != nil {
		return err
	}
	if !opts.apply {
		fmt.Fprintln(stdout, "Dry run: nothing written. Re-run with --apply to link these lines.")
		return nil
	}
	return backfill(ctx, pool, opts, stdout)
}

func parseFlags(args []string, stderr io.Writer) (options, error) {
	var opts options

	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.accountID, "account", "", "Account whose sales order lines to link")
	flags.IntVar(&opts.batch, "batch", 500, "Lines linked per transaction")
	flags.BoolVar(&opts.apply, "apply", false, "Write the links; without it the lines are only counted")
	if err := flags.Parse(args[1:]); err != nil {
		return opts, errBadFlags
	}

	if opts.accountID == "" {
		return opts, errors.New("missing required --account")
	}
	if opts.batch < 1 || opts.batch > 5000 {
		return opts, fmt.Errorf("--batch must be between 1 and 5000, got %d", opts.batch)
	}
	return opts, nil
}

// unlinkedLines selects the account's lines with no item_id whose product carries one of the account's
// items. Keyset on sol.id, so each batch starts where the last ended instead of rescanning linked lines.
const unlinkedLines = `
	FROM sales_order_line sol
	JOIN sales_order so ON so.id = sol.sales_order_id AND so.owner_account_id = ?
	JOIN product p ON p.id = sol.product_id AND p.product_type_code NOT IN ('shipping', 'credit')
	JOIN item i ON i.id = p.item_id AND i.account_id = so.owner_account_id
	WHERE sol.item_id IS NULL`

func printSummary(ctx context.Context, pool *sql.DB, opts options, stdout io.Writer) error {
	rows, err := pool.QueryContext(ctx, `SELECT p.product_type_code, YEAR(sol.created_at), COUNT(*)`+unlinkedLines+`
		GROUP BY p.product_type_code, YEAR(sol.created_at)
		ORDER BY p.product_type_code, YEAR(sol.created_at)`, opts.accountID)
	if err != nil {
		return fmt.Errorf("count unlinked lines: %w", err)
	}
	defer rows.Close()

	w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PRODUCT TYPE\tYEAR\tLINES")
	total := 0
	for rows.Next() {
		var productType string
		var year, count int
		if err := rows.Scan(&productType, &year, &count); err != nil {
			return fmt.Errorf("scan count: %w", err)
		}
		fmt.Fprintf(w, "%s\t%d\t%d\n", productType, year, count)
		total += count
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("count unlinked lines: %w", err)
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("print counts: %w", err)
	}
	fmt.Fprintf(stdout, "%d lines without an item_id can be linked to their product's item.\n", total)
	return nil
}

func backfill(ctx context.Context, pool *sql.DB, opts options, stdout io.Writer) error {
	after := ""
	linked := int64(0)
	for {
		ids, err := nextBatch(ctx, pool, opts, after)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			break
		}
		args := make([]any, 0, len(ids))
		for _, id := range ids {
			args = append(args, id)
		}
		// Re-checks item_id IS NULL, so a line linked since it was listed is left as it is.
		res, err := pool.ExecContext(ctx, `UPDATE sales_order_line sol
			JOIN product p ON p.id = sol.product_id
			SET sol.item_id = p.item_id
			WHERE sol.item_id IS NULL AND sol.id IN (`+strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")+`)`, args...)
		if err != nil {
			return fmt.Errorf("link batch after %q: %w", after, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("count linked lines: %w", err)
		}
		linked += n
		after = ids[len(ids)-1]
		fmt.Fprintf(stdout, "Linked %d lines so far.\n", linked)
	}
	fmt.Fprintf(stdout, "Linked %d lines.\n", linked)
	return nil
}

func nextBatch(ctx context.Context, pool *sql.DB, opts options, after string) ([]string, error) {
	rows, err := pool.QueryContext(ctx, `SELECT sol.id`+unlinkedLines+` AND sol.id > ?
		ORDER BY sol.id
		LIMIT ?`, opts.accountID, after, opts.batch)
	if err != nil {
		return nil, fmt.Errorf("list unlinked lines: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan line id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list unlinked lines: %w", err)
	}
	return ids, nil
}
