// Command backfill-customer-role provisions the global "Customer" role in the
// database and assigns it to customer-portal account users.
//
// It is idempotent (fixed IDs; syncs to an exact state) and safe to re-run. It:
//  1. creates the global Customer role (role_type "user", account_id NULL) if absent,
//  2. syncs its role_permission rows to EXACTLY customerRolePermissions (adds missing,
//     removes stale),
//  3. backfills account_user.role_id = <Customer role> for every ACTIVE account_user
//     of a customer (counterparty) account,
//  4. migrates any lingering references off the legacy role id and removes it.
//
// Usage:
//
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/backfill-customer-role --dry-run
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/backfill-customer-role
//
// Flags:
//
//	--dry-run          report what would change; make no writes.
//	--skip-subscribed  also skip accounts that have their own active/trialing
//	                   subscription (real merchants who are also someone's customer).
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

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/env"
)

func main() {
	ctx := context.Background()
	if err := Run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err) // #nosec G705 -- CLI stderr output, not web context
		os.Exit(1)
	}
}

// customerRolePermission mirrors one row of the Customer role's permission set.
// The fixed IDs match shared/db/seed/0004_auth.sql.
type customerRolePermission struct {
	id                        string
	code                      string
	create, read, update, del bool
}

// legacyCustomerRoleID is the original hand-picked Customer role id, superseded by
// the native-looking constants.GlobalCustomerRoleID. This command migrates any
// account_user still pointing at it onto the new id and deletes the legacy role +
// its permissions. It is a no-op once no legacy row exists.
const legacyCustomerRoleID = "rl_customer"

// customerRolePermissions is the EXACT permission set for the Customer role. Only
// addresses (c/r/u) and purchase_orders:create are functionally required by the
// portal; the remaining reads are inert (relation-scoped portal calls never consult
// them) and kept only as future-proofing. apply() syncs the role to exactly this set.
var customerRolePermissions = []customerRolePermission{
	{"rlpm_customer_addr000", "addresses", true, true, true, false},
	{"rlpm_customer_purord0", "purchase_orders", true, true, false, false},
	{"rlpm_customer_salord0", "sales_orders", false, true, false, false},
	{"rlpm_customer_prods00", "products", false, true, false, false},
	{"rlpm_customer_invoic0", "invoices", false, true, false, false},
	{"rlpm_customer_ship000", "shipments", false, true, false, false},
	{"rlpm_customer_disc000", "discounts", false, true, false, false},
	{"rlpm_customer_msg0000", "messaging", false, true, false, false},
}

// customerAccountUserPredicate matches every active account_user of a customer
// (counterparty) account. Kept as a shared fragment so the dry-run counts and the
// real UPDATE target exactly the same rows.
const customerAccountUserPredicate = `
	au.status_code = 'active'
	AND au.account_id IN (
		SELECT ar.counterparty_account_id
		FROM account_relation ar
		WHERE ar.account_relation_role_code = 'customer'
	)`

// subscribedAccountExclusion additionally excludes accounts that have their own
// active/trialing subscription (real merchants who are also someone's customer).
const subscribedAccountExclusion = `
	AND au.account_id NOT IN (
		SELECT a.id FROM account a WHERE a.subscription_status IN ('active', 'trialing')
	)`

func Run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	fs := flag.NewFlagSet("backfill-customer-role", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", false, "report what would change; make no writes")
	skipSubscribed := fs.Bool("skip-subscribed", false, "skip accounts with their own active/trialing subscription")
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

	backfillWhere := customerAccountUserPredicate
	if *skipSubscribed {
		backfillWhere += subscribedAccountExclusion
	}

	if *dryRun {
		return reportDryRun(ctx, pool, backfillWhere, stdout)
	}

	return apply(ctx, pool, backfillWhere, stdout)
}

// reportDryRun prints the counts of what the real run would change, without writing.
func reportDryRun(ctx context.Context, pool *sql.DB, backfillWhere string, stdout io.Writer) error {
	roleExists, err := countRows(ctx, pool, "SELECT COUNT(*) FROM role WHERE id = ?", constants.GlobalCustomerRoleID)
	if err != nil {
		return err
	}
	if roleExists == 0 {
		fmt.Fprintf(stdout, "role %q (%s): would be CREATED\n", constants.GlobalCustomerRoleName, constants.GlobalCustomerRoleID)
	} else {
		fmt.Fprintf(stdout, "role %q (%s): already exists\n", constants.GlobalCustomerRoleName, constants.GlobalCustomerRoleID)
	}
	fmt.Fprintf(stdout, "permissions: would be synced to exactly %d rows\n", len(customerRolePermissions))

	total, err := countRows(ctx, pool, "SELECT COUNT(*) FROM account_user au WHERE"+backfillWhere)
	if err != nil {
		return err
	}
	changed, err := countRows(ctx, pool, "SELECT COUNT(*) FROM account_user au WHERE"+backfillWhere+" AND (au.role_id IS NULL OR au.role_id <> ?)", constants.GlobalCustomerRoleID)
	if err != nil {
		return err
	}
	subscribedOverlap, err := countRows(ctx, pool,
		"SELECT COUNT(*) FROM account_user au WHERE"+customerAccountUserPredicate+
			" AND au.account_id IN (SELECT a.id FROM account a WHERE a.subscription_status IN ('active', 'trialing'))")
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "customer account_users matched: %d (of which %d would change role_id)\n", total, changed)
	fmt.Fprintf(stdout, "of matched users, %d belong to an account with its own active/trialing subscription (use --skip-subscribed to leave those untouched)\n", subscribedOverlap)

	legacyExists, err := countRows(ctx, pool, "SELECT COUNT(*) FROM role WHERE id = ?", legacyCustomerRoleID)
	if err != nil {
		return err
	}
	if legacyExists > 0 {
		legacyRefs, err := countRows(ctx, pool, "SELECT COUNT(*) FROM account_user WHERE role_id = ?", legacyCustomerRoleID)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "legacy role %q present: %d user(s) reference it; it would be migrated to %s and DELETED\n", legacyCustomerRoleID, legacyRefs, constants.GlobalCustomerRoleID)
	}

	fmt.Fprintln(stdout, "dry run: no changes written")
	return nil
}

// apply creates the role, syncs its permissions, backfills account_user roles, and
// removes the legacy role — all in one transaction.
func apply(ctx context.Context, pool *sql.DB, backfillWhere string, stdout io.Writer) error {
	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		"INSERT IGNORE INTO role (id, name, role_type_code, account_id, created_at, updated_at) VALUES (?, ?, 'user', NULL, NOW(3), NOW(3))",
		constants.GlobalCustomerRoleID, constants.GlobalCustomerRoleName,
	); err != nil {
		return fmt.Errorf("insert role: %w", err)
	}

	// Sync permissions to EXACTLY customerRolePermissions: clear then re-insert, so
	// re-running removes rows that were dropped from the set.
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_permission WHERE role_id = ?", constants.GlobalCustomerRoleID); err != nil {
		return fmt.Errorf("clear role permissions: %w", err)
	}
	for _, p := range customerRolePermissions {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO role_permission (id, role_id, permission_code, `create`, `read`, `update`, `delete`, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))",
			p.id, constants.GlobalCustomerRoleID, p.code, p.create, p.read, p.update, p.del,
		); err != nil {
			return fmt.Errorf("insert role_permission %s: %w", p.code, err)
		}
	}

	res, err := tx.ExecContext(ctx,
		"UPDATE account_user au SET au.role_id = ?, au.updated_at = NOW(3) WHERE"+backfillWhere+" AND (au.role_id IS NULL OR au.role_id <> ?)",
		constants.GlobalCustomerRoleID, constants.GlobalCustomerRoleID,
	)
	if err != nil {
		return fmt.Errorf("backfill account_user roles: %w", err)
	}
	affected, _ := res.RowsAffected()

	// Migrate any lingering references off the legacy role id, then delete it.
	migRes, err := tx.ExecContext(ctx,
		"UPDATE account_user SET role_id = ?, updated_at = NOW(3) WHERE role_id = ?",
		constants.GlobalCustomerRoleID, legacyCustomerRoleID,
	)
	if err != nil {
		return fmt.Errorf("migrate legacy role references: %w", err)
	}
	migrated, _ := migRes.RowsAffected()

	if _, err := tx.ExecContext(ctx, "DELETE FROM role_permission WHERE role_id = ?", legacyCustomerRoleID); err != nil {
		return fmt.Errorf("delete legacy role permissions: %w", err)
	}
	delRes, err := tx.ExecContext(ctx, "DELETE FROM role WHERE id = ?", legacyCustomerRoleID)
	if err != nil {
		return fmt.Errorf("delete legacy role: %w", err)
	}
	legacyDeleted, _ := delRes.RowsAffected()

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	fmt.Fprintf(stdout, "role %q (%s) ensured; permissions synced to %d rows\n", constants.GlobalCustomerRoleName, constants.GlobalCustomerRoleID, len(customerRolePermissions))
	fmt.Fprintf(stdout, "assigned Customer role to %d customer account_user(s)\n", affected)
	if migrated > 0 || legacyDeleted > 0 {
		fmt.Fprintf(stdout, "migrated %d user(s) off legacy role %q and removed it\n", migrated, legacyCustomerRoleID)
	}
	return nil
}

// normalizeDSN accepts either a go-sql-driver DSN (user:pass@tcp(host:port)/db[?params])
// — returned unchanged — or a PlanetScale/URL form (mysql://user:pass@host[:port]/db[?...])
// which it rewrites into the driver DSN form that db.NewDbPool expects. For remote hosts it
// forces tls=true (PlanetScale requires TLS) and drops UI-only query params (e.g.
// sslaccept=strict) that the driver would reject; db.NewDbPool re-adds parseTime/loc/etc.
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

func countRows(ctx context.Context, pool *sql.DB, query string, queryArgs ...any) (int64, error) {
	var n int64
	if err := pool.QueryRowContext(ctx, query, queryArgs...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count query: %w", err)
	}
	return n, nil
}
