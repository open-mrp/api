// Command backfill-operating-calendars gives existing accounts the shipping and
// receiving calendars new accounts are provisioned with.
//
// Registration and sandbox creation seed these going forward, so this exists only to
// catch up accounts that predate the feature. Until an account has them, its ship-by
// dates resolve against a plain Monday-to-Friday week with no holidays — which is what
// the system did before calendars existed, and is why running late is harmless rather
// than urgent.
//
// It reuses the same seed the registration path runs, so the calendars an existing
// account gets are identical to a new one's. That seed is idempotent by code, which
// makes this safe to re-run: an account that already has the calendars only has its
// closure horizon topped up, and a holiday somebody has renamed or deleted stays that
// way.
//
// Usage:
//
//	DB_URL=<dsn-or-mysql-url> go run ./services/core-service/cmd/backfill-operating-calendars --dry-run
//	DB_URL=<dsn-or-mysql-url> go run ./services/core-service/cmd/backfill-operating-calendars
//
// Flags:
//
//	--dry-run  report which accounts would be seeded; make no writes.
//	--account  seed one account only, by ID.
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

	"github.com/open-mrp/api/services/core-service/internal/calendarseed"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/env"
)

func main() {
	ctx := context.Background()
	if err := Run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err) // #nosec G705 -- CLI stderr output, not web context
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
	fs := flag.NewFlagSet("backfill-operating-calendars", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", false, "report which accounts would be seeded; make no writes")
	onlyAccount := fs.String("account", "", "seed one account only, by ID")
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

	pool, err := db.NewDbPool(&db.Config{DBURI: dsn})
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	accountIDs, err := accountsMissingCalendars(ctx, pool, *onlyAccount)
	if err != nil {
		return err
	}

	if len(accountIDs) == 0 {
		fmt.Fprintln(stdout, "every account already has operating calendars; nothing to do")
		return nil
	}

	if *dryRun {
		fmt.Fprintf(stdout, "would seed operating calendars for %d account(s):\n", len(accountIDs))
		for _, accountID := range accountIDs {
			fmt.Fprintf(stdout, "  %s\n", accountID)
		}
		return nil
	}

	repos := repository.NewRepoFactory(sqlc.New(pool))
	asOf := time.Now()

	// Seeded one account at a time, outside a transaction, because each account's calendars are independent and a single failure should not roll back the accounts already done. Re-running picks up whatever is left.
	var seeded int
	for _, accountID := range accountIDs {
		if apiErr := calendarseed.Seed(ctx, repos, accountID, asOf); apiErr != nil {
			fmt.Fprintf(stderr, "  %s: %s\n", accountID, apiErr.PublicMessage)
			continue
		}
		seeded++
		fmt.Fprintf(stdout, "  %s seeded\n", accountID)
	}

	fmt.Fprintf(stdout, "seeded %d of %d account(s)\n", seeded, len(accountIDs))
	if seeded < len(accountIDs) {
		return fmt.Errorf("%d account(s) failed; re-run to retry them", len(accountIDs)-seeded)
	}
	return nil
}

// accountsMissingCalendars lists accounts with no operating calendar at all.
//
// Missing entirely rather than missing one kind: the seed writes both together, so a half-seeded account can only come from a failure mid-run, and re-running the seed on it is harmless anyway.
func accountsMissingCalendars(ctx context.Context, pool *sql.DB, onlyAccount string) ([]string, error) {
	query := `
		SELECT a.id
		FROM account a
		LEFT JOIN operating_calendar oc ON oc.account_id = a.id AND oc.deleted_at IS NULL
		WHERE oc.id IS NULL`
	queryArgs := []any{}
	if onlyAccount != "" {
		query += " AND a.id = ?"
		queryArgs = append(queryArgs, onlyAccount)
	}
	query += " GROUP BY a.id ORDER BY a.id"

	rows, err := pool.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("listing accounts: %w", err)
	}
	defer rows.Close()

	var accountIDs []string
	for rows.Next() {
		var accountID string
		if err := rows.Scan(&accountID); err != nil {
			return nil, fmt.Errorf("scanning account: %w", err)
		}
		accountIDs = append(accountIDs, accountID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing accounts: %w", err)
	}
	return accountIDs, nil
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
