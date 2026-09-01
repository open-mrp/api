//go:build ledger

// Package repository's ledger tests run two or more real transactions against a real InnoDB and
// control the interleaving between them.
//
// They exist because nothing else in this repo can. The sqlmock suites have no rows and no locks,
// the gomock consumer tests never run SQL, and the e2e concurrency tests fire parallel HTTP
// requests and take whatever interleaving the scheduler gives them. Every claim about lock order in
// this service has been an argument from InnoDB semantics; these tests make it a measurement.
//
// Separate build tag from `integration` on purpose: that tag means "prepare every statement, touch
// no rows, finish in 30s". These write rows, hold locks, and must never run concurrently with each
// other — two of them at once would see each other's lock waits in innodb_trx. `make test-ledger`
// runs them with -p 1 -parallel 1, and that is a correctness requirement, not a tuning choice.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
	"github.com/open-mrp/api/shared/ledger"
)

// The local dev MySQL from docker-compose.yml — the same instance make test-sql-prepare-smoke uses
// and the same one post-merge.yml stands up as a service container.
//
// interpolateParams=false matches shared/db/db_pool.go, and it matters here rather than being
// cosmetic: it is what makes every statement a prepare+execute pair, which is what the statement
// budget is counted in.
const defaultLedgerDSN = "root:Testing123!@tcp(127.0.0.1:3306)/openmrp?parseTime=true&loc=UTC&interpolateParams=false"

var (
	ledgerDBOnce sync.Once
	ledgerSQLDB  *sql.DB
	ledgerDBErr  error
)

func ledgerDB(t *testing.T) *sql.DB {
	t.Helper()
	ledgerDBOnce.Do(func() {
		dsn := os.Getenv("LEDGER_TEST_DSN")
		if dsn == "" {
			dsn = defaultLedgerDSN
		}
		ledgerSQLDB, ledgerDBErr = sql.Open("mysql", dsn)
		if ledgerDBErr == nil {
			// One pinned connection per actor plus probes. The default idle count would recycle a
			// connection out from under an open transaction.
			ledgerSQLDB.SetMaxOpenConns(32)
			ledgerSQLDB.SetMaxIdleConns(32)
			ledgerDBErr = ledgerSQLDB.Ping()
		}
	})
	require.NoError(t, ledgerDBErr,
		"connecting to the ledger test database (make local-db, or set LEDGER_TEST_DSN)")
	return ledgerSQLDB
}

// The background corpus exists to make the local optimizer choose the plan production runs.
//
// A fixture's handful of rows is not enough. With seven issues in the whole table, account_id and
// item_id have a cardinality of one and MySQL answers the paged read from
// inventory_issue_status_code_idx — an index on status_code alone, whose locks and gaps span every
// account in the database. That is not the production plan, and a test that ran against it would be
// measuring the wrong index's footprint: green after the fix would prove nothing about the index the
// consumer actually walks.
//
// So the table is given enough shape for (account_id, item_id, status_code) to be the selective
// choice. The rows are inert — no receipts, no allocations, never read by a fixture's queries, which
// all filter on the fixture's own account. They are left in place between runs because building them
// is the slow part and their whole purpose is to make the table look lived-in.
const (
	ledgerBackgroundAccountPrefix = "ac_ledgerbg_"
	ledgerBackgroundAccounts      = 300
	ledgerBackgroundItemsPer      = 4
	ledgerBackgroundIssuesPer     = 3
	ledgerBackgroundRows          = ledgerBackgroundAccounts * ledgerBackgroundItemsPer * ledgerBackgroundIssuesPer
)

var ledgerBackgroundOnce sync.Once

func ensureLedgerBackground(t *testing.T, db *sql.DB) {
	t.Helper()
	ledgerBackgroundOnce.Do(func() {
		var have int
		require.NoError(t, db.QueryRow(
			`SELECT COUNT(*) FROM inventory_issue WHERE account_id LIKE ?`,
			ledgerBackgroundAccountPrefix+"%").Scan(&have))
		if have >= ledgerBackgroundRows {
			return
		}
		t.Logf("seeding the ledger background corpus (%d issues) so the optimizer picks "+
			"inventory_issue_open_paging_idx; this runs once and is reused by later runs",
			ledgerBackgroundRows)

		_, err := db.Exec(
			`INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, account_id,
			                          ratio_numerator, ratio_denominator, is_base_unit, created_at, updated_at)
			 VALUES ('un_ledgerbg', 'ledgertest-background', 'un_ledgerbg', 'quantity', NULL, 1, 1, 0, NOW(3), NOW(3))`)
		require.NoError(t, err)

		base := time.Now().UTC().Add(-30 * 24 * time.Hour)
		for acct := range ledgerBackgroundAccounts {
			var qArgs, iArgs []any
			var qVals, iVals []string
			for item := range ledgerBackgroundItemsPer {
				for n := range ledgerBackgroundIssuesPer {
					suffix := fmt.Sprintf("%d_%d_%d", acct, item, n)
					qID := "qy_ledgerbg_" + suffix
					iID := "ivis_ledgerbg_" + suffix
					createdAt := base.Add(time.Duration(acct*97+item*13+n) * time.Minute)
					qVals = append(qVals, "(?, '5', 'un_ledgerbg', ?, ?)")
					qArgs = append(qArgs, qID, createdAt, createdAt)
					iVals = append(iVals, "(?, ?, ?, 'open', ?, ?, ?)")
					iArgs = append(iArgs, iID,
						fmt.Sprintf("%s%03d", ledgerBackgroundAccountPrefix, acct),
						fmt.Sprintf("it_ledgerbg_%03d_%d", acct, item),
						qID, createdAt, createdAt)
				}
			}
			_, err := db.Exec(`INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES `+
				strings.Join(qVals, ","), qArgs...)
			require.NoError(t, err)
			_, err = db.Exec(`INSERT IGNORE INTO inventory_issue
				(id, account_id, item_id, status_code, quantity_id, created_at, updated_at) VALUES `+
				strings.Join(iVals, ","), iArgs...)
			require.NoError(t, err)
		}
		_, err = db.Exec(`ANALYZE TABLE inventory_issue`)
		require.NoError(t, err)
	})
}

// requireProductionPlan fails when the local optimizer did not answer the paged read the way
// production does.
//
// It reads the index names out of the locks the scan is holding rather than parsing EXPLAIN, so it
// cannot drift from the statement the test actually ran. Without it these tests degrade silently: a
// scan answered from inventory_issue_status_code_idx locks a different index, over a range covering
// every account, and every assertion below would be about a footprint the consumer never takes.
func requireProductionPlan(t *testing.T, a *actor, db *sql.DB) {
	t.Helper()
	for _, l := range a.locksHeld(t, db) {
		if l.Object == "inventory_issue" && l.Index != "" &&
			l.Index != "inventory_issue_open_paging_idx" && l.Index != "PRIMARY" {
			t.Fatalf("the paged read was answered from %s, not inventory_issue_open_paging_idx: the "+
				"background corpus is no longer making (account_id, item_id, status_code) the selective "+
				"choice, and every lock assertion here would be about the wrong index", l.Index)
		}
	}
}

// fixture owns one account/item pair and every row written under it.
//
// There are no foreign keys in this schema (grep -c "FOREIGN KEY" shared/db/migrations/00001_initial.sql
// is 0), so a fixture mints its own account and item ids without provisioning either table — the
// ledger queries filter on those columns, they never join item or account. The `unit` rows ARE
// needed, by GetUnitRatios, and are created per fixture rather than borrowed from the seed so a test
// can never be perturbed by, or perturb, rows anything else depends on.
type fixture struct {
	db        *sql.DB
	accountID string
	itemID    string
	each      string // ratio 1
	pair      string // ratio 2 — every cross-unit assertion needs a second ratio
	dollar    string
	seq       int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := ledgerDB(t)
	ensureLedgerBackground(t, db)
	suffix := fmt.Sprintf("%d_%d", time.Now().UnixNano(), os.Getpid())
	f := &fixture{
		db:        db,
		accountID: "ac_lt" + suffix,
		itemID:    "it_lt" + suffix,
		each:      "un_lte" + suffix,
		pair:      "un_ltp" + suffix,
		dollar:    "un_ltd" + suffix,
	}
	for _, u := range []struct {
		id, name string
		num, den int
	}{
		{f.each, "ledgertest-each", 1, 1},
		{f.pair, "ledgertest-pair", 2, 1},
		{f.dollar, "ledgertest-dollar", 1, 1},
	} {
		_, err := db.Exec(
			`INSERT INTO unit (id, name, abbreviation, unit_dimension_code, account_id,
			                   ratio_numerator, ratio_denominator, is_base_unit, created_at, updated_at)
			 VALUES (?, ?, ?, 'quantity', ?, ?, ?, 0, NOW(3), NOW(3))`,
			u.id, u.name+suffix, u.id, f.accountID, u.num, u.den)
		require.NoError(t, err, "seeding unit %s", u.id)
	}
	// Every row this fixture writes carries its account or item id, so the teardown is exact and two
	// fixtures never see each other's rows.
	t.Cleanup(func() { f.teardown(t) })
	f.seedClosedBallast(t)
	return f
}

// seedClosedBallast gives the fixture's item the status distribution a real item has: a long history
// of closed issues and a handful of open ones.
//
// Without it the fixture's item_id is unique in the entire table, so inventory_issue_item_id_idx
// looks perfectly selective and the optimizer prefers it over inventory_issue_open_paging_idx — the
// same wrong-index problem the background corpus fixes one level up. What makes the composite index
// the right answer in production is that status_code is in it: an item has thousands of closed
// issues and a few open ones, so filtering on status inside the index is what avoids reading the
// history. These rows reproduce that, and nothing reads them: every fixture query filters
// status_code = 'open'.
//
// They sit a year before anything a test writes, so they cannot land inside a range a test locks.
func (f *fixture) seedClosedBallast(t *testing.T) {
	t.Helper()
	const ballast = 400
	base := time.Now().UTC().Add(-365 * 24 * time.Hour)

	var qArgs, iArgs []any
	var qVals, iVals []string
	for n := range ballast {
		qID, iID := f.nextID("qy"), f.nextID("ivis")
		createdAt := base.Add(time.Duration(n) * time.Minute)
		qVals = append(qVals, "(?, '1', ?, ?, ?)")
		qArgs = append(qArgs, qID, f.each, createdAt, createdAt)
		iVals = append(iVals, "(?, ?, ?, 'closed', ?, ?, ?)")
		iArgs = append(iArgs, iID, f.accountID, f.itemID, qID, createdAt, createdAt)
	}
	_, err := f.db.Exec(`INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES `+
		strings.Join(qVals, ","), qArgs...)
	require.NoError(t, err)
	_, err = f.db.Exec(`INSERT INTO inventory_issue
		(id, account_id, item_id, status_code, quantity_id, created_at, updated_at) VALUES `+
		strings.Join(iVals, ","), iArgs...)
	require.NoError(t, err)
}

// teardown removes the fixture's rows satellite-first, because the ledger's satellites are reachable
// only through the row that owns them: delete the allocation before its quantity and rate and the
// satellite ids are gone with it.
func (f *fixture) teardown(t *testing.T) {
	t.Helper()
	exec := func(stmt string, args ...any) {
		if _, err := f.db.Exec(stmt, args...); err != nil {
			t.Logf("ledger fixture teardown (%s): %v", strings.Fields(stmt)[0], err)
		}
	}
	exec(`DELETE q FROM quantity q
	        JOIN inventory_allocation ia ON ia.quantity_id = q.id OR ia.total_cost_id = q.id
	        JOIN inventory_issue ii ON ii.id = ia.inventory_issue_id
	       WHERE ii.item_id = ?`, f.itemID)
	exec(`DELETE r FROM rate r
	        JOIN inventory_allocation ia ON ia.unit_cost_id = r.id
	        JOIN inventory_issue ii ON ii.id = ia.inventory_issue_id
	       WHERE ii.item_id = ?`, f.itemID)
	exec(`DELETE ia FROM inventory_allocation ia
	        JOIN inventory_issue ii ON ii.id = ia.inventory_issue_id
	       WHERE ii.item_id = ?`, f.itemID)
	exec(`DELETE q FROM quantity q JOIN inventory_issue ii ON ii.quantity_id = q.id WHERE ii.item_id = ?`, f.itemID)
	exec(`DELETE q FROM quantity q JOIN inventory_receipt ir ON ir.quantity_id = q.id WHERE ir.item_id = ?`, f.itemID)
	exec(`DELETE r FROM rate r JOIN inventory_receipt ir ON ir.unit_cost_id = r.id WHERE ir.item_id = ?`, f.itemID)
	exec(`DELETE FROM inventory_issue WHERE item_id = ?`, f.itemID)
	exec(`DELETE FROM inventory_receipt WHERE item_id = ?`, f.itemID)
	exec(`DELETE FROM rate WHERE numerator_unit_id = ? OR denominator_unit_id = ?`, f.dollar, f.dollar)
	// The one DELETE this table ever sees, and only here: production must never issue one, because a
	// delete takes the gap locks the ON DUPLICATE KEY shape exists to avoid. Nothing is acquiring
	// concurrently at teardown, and leaving a row per run to accumulate in a dev database is worse.
	exec(`DELETE FROM inventory_item_lock WHERE item_id LIKE ?`, f.itemID+"%")
	exec(`DELETE FROM unit WHERE account_id = ?`, f.accountID)
}

func (f *fixture) nextID(prefix string) string {
	f.seq++
	return fmt.Sprintf("%s_lt%d_%d", prefix, time.Now().UnixNano(), f.seq)
}

// insertIssue writes an issue with an explicit created_at, because created_at is the fourth column
// of inventory_issue_open_paging_idx and therefore decides both the paging order and where in the
// locked index range a row lands.
func (f *fixture) insertIssue(t *testing.T, status, value, unitID string, createdAt time.Time) string {
	t.Helper()
	return f.insertIssueWithID(t, f.nextID("ivis"), status, value, unitID, createdAt)
}

// insertIssueWithID exists for the tests that need the clustered PK order to disagree with the
// created_at order — production ids are 12-char nanoids (shared/id/utils.go), so the two orders are
// uncorrelated and a test that wants the inverted case has to name the ids itself.
func (f *fixture) insertIssueWithID(t *testing.T, issueID, status, value, unitID string, createdAt time.Time) string {
	t.Helper()
	qID := f.nextID("qy")
	_, err := f.db.Exec(
		`INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		qID, value, unitID, createdAt, createdAt)
	require.NoError(t, err)
	_, err = f.db.Exec(
		`INSERT INTO inventory_issue (id, account_id, item_id, status_code, quantity_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		issueID, f.accountID, f.itemID, status, qID, createdAt, createdAt)
	require.NoError(t, err)
	return issueID
}

func (f *fixture) insertReceipt(t *testing.T, status, value, unitID string, receivedAt time.Time) string {
	t.Helper()
	qID, rateID, rID := f.nextID("qy"), f.nextID("rt"), f.nextID("ivrc")
	_, err := f.db.Exec(
		`INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		qID, value, unitID, receivedAt, receivedAt)
	require.NoError(t, err)
	_, err = f.db.Exec(
		`INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at)
		 VALUES (?, '7', ?, ?, ?, ?)`,
		rateID, f.dollar, unitID, receivedAt, receivedAt)
	require.NoError(t, err)
	_, err = f.db.Exec(
		`INSERT INTO inventory_receipt (id, owner_account_id, holder_account_id, item_id, received_at,
		                                quantity_id, unit_cost_id, status_code, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rID, f.accountID, f.accountID, f.itemID, receivedAt, qID, rateID, status, receivedAt, receivedAt)
	require.NoError(t, err)
	return rID
}

// scope takes the fixture item's ordering root for this actor and returns the evidence, the way a
// conforming ledger transaction opens. Tests that want to observe a LATE acquisition pass a bare
// &ledgerlock.Scope{} instead.
func (a *actor) scope(t *testing.T, f *fixture) *ledgerlock.Scope {
	t.Helper()
	s, apiErr := ledgerlock.Acquire(context.Background(), &inventoryReservationRepo{queries: a.q}, []string{f.itemID})
	require.Nil(t, apiErr, "%s: acquiring the item's ledger root", a.name)
	return s
}

// seedItemLockRow warms the fixture's lock row, so a test can exercise the acquisition's duplicate-key
// branch rather than its insert branch. Both branches matter and are tested separately.
func (f *fixture) seedItemLockRow(t *testing.T, itemID string) {
	t.Helper()
	_, err := f.db.Exec(`INSERT IGNORE INTO inventory_item_lock (item_id, created_at) VALUES (?, NOW(3))`, itemID)
	require.NoError(t, err)
}

// acquireItemLock takes the ledger root exactly as production must: one statement, always this one.
// See 00016_inventory_item_lock — every other shape deadlocks on the path this table exists to protect.
func (a *actor) acquireItemLock(itemID string) error {
	_, err := a.tx.ExecContext(context.Background(),
		`INSERT INTO inventory_item_lock (item_id, created_at) VALUES (?, NOW(3))
		 ON DUPLICATE KEY UPDATE item_id = item_id`, itemID)
	return err
}

// writeRawAllocation commits an allocation the way a writer outside this service does: straight in,
// on its own connection, taking no locking read and holding no ledger lock.
//
// This is dashboard/apps/api's Prisma allocator (inventory-issue.repo.ts:946, :1216) reduced to what
// matters — inventoryAllocation.createMany against the same keyspace, on a live request path. Nothing
// in the Go service can serialise against it, which is why the allocator's arithmetic has to be
// robust to it rather than merely locked against it.
func (f *fixture) writeRawAllocation(t *testing.T, issueID, receiptID, value, unitID string) string {
	t.Helper()
	qID, rateID, costID, aID := f.nextID("qy"), f.nextID("rt"), f.nextID("qy"), f.nextID("ivia")

	_, err := f.db.Exec(
		`INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, ?, NOW(3), NOW(3))`,
		qID, value, unitID)
	require.NoError(t, err)
	_, err = f.db.Exec(
		`INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at)
		 VALUES (?, '7', ?, ?, NOW(3), NOW(3))`, rateID, f.dollar, unitID)
	require.NoError(t, err)
	_, err = f.db.Exec(
		`INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, '0', ?, NOW(3), NOW(3))`,
		costID, f.dollar)
	require.NoError(t, err)
	_, err = f.db.Exec(
		`INSERT INTO inventory_allocation (id, inventory_receipt_id, inventory_issue_id,
		                                   quantity_id, unit_cost_id, total_cost_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, NOW(3), NOW(3))`,
		aID, receiptID, issueID, qID, rateID, costID)
	require.NoError(t, err)
	return aID
}

// receiptOverDrawnBy reports how far past its own quantity a receipt has been allocated, in base
// units. It is the detector cmd/repair-overallocated-receipts uses, in one statement: both sides
// through their own unit's ratio, and the positive-quantity guard, because a migration wrote opening
// balances as negative receipts.
func (f *fixture) receiptOverDrawnBy(t *testing.T, receiptID string) decimal.Decimal {
	t.Helper()
	var overBy string
	err := f.db.QueryRow(`
		SELECT CAST(COALESCE((
		    SELECT SUM(CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator))
		    FROM inventory_allocation ia
		    JOIN quantity aq ON aq.id = ia.quantity_id
		    JOIN unit au ON au.id = aq.unit_id
		    WHERE ia.inventory_receipt_id = ir.id
		  ), 0)
		  - (CAST(rq.value AS DECIMAL(65,30)) * (ru.ratio_numerator / ru.ratio_denominator)) AS CHAR)
		FROM inventory_receipt ir
		JOIN quantity rq ON rq.id = ir.quantity_id
		JOIN unit ru ON ru.id = rq.unit_id
		WHERE ir.id = ? AND rq.value > 0`, receiptID).Scan(&overBy)
	require.NoError(t, err, "reading the over-draw for receipt %s", receiptID)
	return decimal.RequireFromString(overBy)
}

// baseAllocatedForIssue reports what an issue has been covered by, in base units, through each
// allocation's own unit ratio.
func (f *fixture) baseAllocatedForIssue(t *testing.T, issueID string) decimal.Decimal {
	t.Helper()
	var total string
	err := f.db.QueryRow(`
		SELECT CAST(COALESCE(SUM(CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator)), 0) AS CHAR)
		FROM inventory_allocation ia
		JOIN quantity q ON q.id = ia.quantity_id
		JOIN unit u ON u.id = q.unit_id
		WHERE ia.inventory_issue_id = ?`, issueID).Scan(&total)
	require.NoError(t, err, "reading the coverage for issue %s", issueID)
	return decimal.RequireFromString(total)
}

// assertReceiptNotOverDrawn is the invariant the whole exercise exists to protect: a receipt may never
// have more allocated against it than it holds. The tolerance is the monitor's, from the same
// constant, so nothing this permits is something the scheduled check later alarms on.
func assertReceiptNotOverDrawn(t *testing.T, f *fixture, receiptID string) {
	t.Helper()
	overBy := f.receiptOverDrawnBy(t, receiptID)
	if overBy.GreaterThan(ledger.Epsilon) {
		t.Errorf("receipt %s is over-drawn by %s base units: more stock has been allocated off it than "+
			"it ever held, which is the corruption class the 2026-08-26 incident left behind",
			receiptID, overBy.String())
	}
}

// actor is one transaction on one pinned connection, driving the real repository code. It is the
// unit of interleaving control: its statements run when the test calls them and not before.
type actor struct {
	name string
	conn *sql.Conn
	tx   *sql.Tx
	// psThreadID, not the connection id, is how this actor is found in performance_schema.data_locks.
	//
	// The obvious route — join data_locks to information_schema.innodb_trx on the engine transaction
	// id and match trx_mysql_thread_id — does not work here. Most actors in these tests are read-only
	// transactions: they take locking reads and never write, and such a transaction is not reliably
	// listed in innodb_trx. Its locks are still in data_locks, so a barrier or a footprint assertion
	// routed through innodb_trx silently sees nothing and the test passes for the wrong reason.
	psThreadID uint64
	repo       *inventoryReservationRepo
	q          *sqlc.Queries
}

func (f *fixture) actor(t *testing.T, name string) *actor {
	t.Helper()
	ctx := context.Background()
	conn, err := f.db.Conn(ctx)
	require.NoError(t, err, "%s: pinning a connection", name)

	// Two seconds, not the 50s default: a "must not block" assertion has to resolve inside the test
	// timeout, and a genuine block then shows up as 1205 rather than a hang nobody can diagnose.
	_, err = conn.ExecContext(ctx, "SET SESSION innodb_lock_wait_timeout = 2")
	require.NoError(t, err)

	tx, err := conn.BeginTx(ctx, nil) // nil options: exactly what shared/db/transaction.go does
	require.NoError(t, err, "%s: begin", name)

	var psThreadID uint64
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT PS_CURRENT_THREAD_ID()").Scan(&psThreadID))

	q := sqlc.New(f.db).WithTx(tx)
	a := &actor{name: name, conn: conn, tx: tx, psThreadID: psThreadID, q: q,
		repo: &inventoryReservationRepo{queries: q}}
	t.Cleanup(func() {
		_ = a.tx.Rollback()
		_ = a.conn.Close()
	})
	return a
}

func (a *actor) commit(t *testing.T) { t.Helper(); require.NoError(t, a.tx.Commit()) }

// waitUntilBlocked returns once InnoDB itself reports this actor waiting on a lock.
//
// This replaces every sleep in these tests. Polling the server's own view of the transaction turns
// "probably blocked by now" into "the server says it is blocked", which is the difference between a
// deterministic test and one that passes on a fast machine.
func (a *actor) waitUntilBlocked(t *testing.T, db *sql.DB) {
	t.Helper()
	a.waitUntilBlockedOr(t, db, nil)
}

// waitUntilBlockedOr is waitUntilBlocked for a statement whose outcome the test is holding a channel
// for. It gives up the moment that statement finishes, because a statement that returned is a
// statement that is not waiting — and reporting what it returned is far more use than reporting that
// a barrier timed out. A deadlock detected on the spot rolls the transaction back and removes it from
// innodb_trx, so without this the most interesting outcome in these tests looks like a hung barrier.
func (a *actor) waitUntilBlockedOr(t *testing.T, db *sql.DB, done <-chan error) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			if isDeadlock(err) {
				t.Fatalf("%s was rolled back as a deadlock victim (MySQL 1213) before it could even be "+
					"observed waiting: the inversion resolved instantly", a.name)
			}
			t.Fatalf("%s did not block: its statement returned %v. The interleaving this test depends on "+
				"did not happen, so nothing below it is being exercised.%s",
				a.name, err, a.describeLocks(t, db))
		default:
		}
		var waiting int
		err := db.QueryRow(
			`SELECT COUNT(*) FROM performance_schema.data_locks
			  WHERE THREAD_ID = ? AND LOCK_STATUS = 'WAITING'`, a.psThreadID).Scan(&waiting)
		if err == nil && waiting > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s never reached LOCK WAIT; the interleaving this test depends on did not happen.%s",
		a.name, a.describeLocks(t, db))
}

type lockRow struct{ Object, Index, Type, Mode, Data string }

// locksHeld reports what InnoDB says this transaction is holding, so assertions about lock footprint
// are made against the server rather than argued from the query text.
func (a *actor) locksHeld(t *testing.T, db *sql.DB) []lockRow {
	t.Helper()
	rows, err := db.Query(
		`SELECT dl.OBJECT_NAME, IFNULL(dl.INDEX_NAME,''), dl.LOCK_TYPE, dl.LOCK_MODE, IFNULL(dl.LOCK_DATA,'')
		   FROM performance_schema.data_locks dl
		  WHERE dl.THREAD_ID = ? AND dl.LOCK_STATUS = 'GRANTED'`, a.psThreadID)
	require.NoError(t, err)
	defer rows.Close()

	var out []lockRow
	for rows.Next() {
		var l lockRow
		require.NoError(t, rows.Scan(&l.Object, &l.Index, &l.Type, &l.Mode, &l.Data))
		out = append(out, l)
	}
	require.NoError(t, rows.Err())
	return out
}

// describeLocks dumps every lock the server is holding or waiting on, so a barrier that times out
// says why rather than only that it did. It is keyed on performance_schema thread ids for the same
// reason the barrier is: a read-only transaction has locks but no innodb_trx row.
func (a *actor) describeLocks(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query(
		`SELECT dl.THREAD_ID, dl.OBJECT_NAME, IFNULL(dl.INDEX_NAME,''),
		        dl.LOCK_TYPE, dl.LOCK_MODE, dl.LOCK_STATUS, IFNULL(dl.LOCK_DATA,'')
		   FROM performance_schema.data_locks dl
		  WHERE dl.LOCK_TYPE = 'RECORD'
		  ORDER BY dl.LOCK_STATUS DESC, dl.THREAD_ID, dl.OBJECT_NAME`)
	if err != nil {
		return fmt.Sprintf(" (could not read data_locks: %v)", err)
	}
	defer rows.Close()

	var b strings.Builder
	b.WriteString("\n  record locks:")
	for rows.Next() {
		var thread uint64
		var object, index, lockType, mode, status, data string
		if err := rows.Scan(&thread, &object, &index, &lockType, &mode, &status, &data); err != nil {
			return b.String() + fmt.Sprintf("\n  (could not scan data_locks: %v)", err)
		}
		marker := ""
		if thread == a.psThreadID {
			marker = "  <- " + a.name
		}
		fmt.Fprintf(&b, "\n    thread %d  %-7s  %s.%s  %s  %s%s",
			thread, status, object, index, mode, data, marker)
	}
	return b.String()
}

// probeCanInsertOpenIssue asks, from a separate connection with a 2s lock timeout, whether a new open
// issue for the item can be recorded right now. False means somebody is holding the gap it lands in.
func (f *fixture) probeCanInsertOpenIssue(t *testing.T, createdAt time.Time) bool {
	t.Helper()
	ctx := context.Background()
	conn, err := f.db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.ExecContext(ctx, "SET SESSION innodb_lock_wait_timeout = 2")
	require.NoError(t, err)
	tx, err := conn.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck // probe only; nothing is meant to survive it

	qID, iID := f.nextID("qy"), f.nextID("ivis")
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, '1', ?, ?, ?)`,
		qID, f.each, createdAt, createdAt); err != nil {
		t.Fatalf("probe quantity insert: %v", err)
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO inventory_issue (id, account_id, item_id, status_code, quantity_id, created_at, updated_at)
		 VALUES (?, ?, ?, 'open', ?, ?, ?)`,
		iID, f.accountID, f.itemID, qID, createdAt, createdAt)
	if err == nil {
		return true
	}
	if isLockWaitTimeout(err) {
		return false
	}
	t.Fatalf("probe insert failed for an unexpected reason: %v", err)
	return false
}

// discoveryParams drives the discovery read from the epoch cursor, the way both producers of
// core.cmd.allocate_open_issues do — every chain starts over from the beginning of the range.
func discoveryParams(f *fixture, limit int32) sqlc.ListOpenIssueIDsForItemPagedParams {
	return discoveryParamsAfter(f, openIssueCursorEpoch, "", limit)
}

func discoveryParamsAfter(f *fixture, after time.Time, afterID string, limit int32) sqlc.ListOpenIssueIDsForItemPagedParams {
	return sqlc.ListOpenIssueIDsForItemPagedParams{
		AccountID:       f.accountID,
		ItemID:          f.itemID,
		CursorCreatedAt: after,
		CursorID:        afterID,
		Limit:           limit,
	}
}

// claim drives the real locking read an allocate transaction takes on one issue.
func (a *actor) claim(t *testing.T, f *fixture, issueID string) error {
	t.Helper()
	_, err := a.q.ClaimOpenIssueForAllocation(context.Background(), sqlc.ClaimOpenIssueForAllocationParams{
		ID: issueID, AccountID: f.accountID,
	})
	return err
}

func isLockWaitTimeout(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1205
}

func isDeadlock(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1213
}

// assertNeitherDeadlocked drains two in-flight statements and fails if InnoDB rolled either back to
// break a cycle, or if neither ever finished — a stall is the same inversion resolving as a wait, and
// is no more acceptable than the deadlock.
func assertNeitherDeadlocked(t *testing.T, first, second <-chan error, msg string) {
	t.Helper()
	deadline := time.After(20 * time.Second)
	for range 2 {
		select {
		case err := <-first:
			checkNotDeadlock(t, err, msg)
		case err := <-second:
			checkNotDeadlock(t, err, msg)
		case <-deadline:
			t.Fatalf("%s — neither side completed: both are stalled on each other", msg)
		}
	}
}

func checkNotDeadlock(t *testing.T, err error, msg string) {
	t.Helper()
	if isDeadlock(err) {
		t.Fatalf("%s (MySQL 1213)", msg)
	}
	if isLockWaitTimeout(err) {
		t.Fatalf("%s — resolved as a 2s lock wait timeout instead of a deadlock, which is the same "+
			"inversion with a slower victim", msg)
	}
	require.NoError(t, err)
}
