//go:build e2e

package api_test

import (
	"database/sql"
	"sync"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

// defaultE2EDBURL points at the e2e MySQL published on the host (3306 is the
// local dev DB, so the e2e stack publishes 3307). CI can override via E2E_DB_URL
// — e.g. mysql-e2e:3306 when tests run inside the compose network.
const defaultE2EDBURL = "root:Testing123!@tcp(127.0.0.1:3307)/openmrp?parseTime=true"

var (
	e2eDBOnce sync.Once
	e2eDB     *sql.DB
	e2eDBErr  error
)

// authDB returns a lazily-opened connection to the e2e backend database. Tests
// use it to read state that is deliberately not exposed through the API, such as
// the registration email verification token.
func authDB(t *testing.T) *sql.DB {
	t.Helper()
	e2eDBOnce.Do(func() {
		dsn := envOr("E2E_DB_URL", defaultE2EDBURL)
		e2eDB, e2eDBErr = sql.Open("mysql", dsn)
		if e2eDBErr == nil {
			e2eDBErr = e2eDB.Ping()
		}
	})
	require.NoError(t, e2eDBErr, "connecting to e2e database (is the stack up with mysql-e2e published on 3307?)")
	return e2eDB
}

// registrationVerificationToken reads the verification token for a registration
// session straight from the database, standing in for the link the user would
// receive by email. The token is never returned by the API.
func registrationVerificationToken(t *testing.T, sessionID string) string {
	t.Helper()
	var token string
	err := authDB(t).QueryRow(
		"SELECT verification_token FROM registration_session WHERE type_id = ?",
		sessionID,
	).Scan(&token)
	require.NoError(t, err, "querying verification_token for session %s", sessionID)
	require.NotEmpty(t, token, "verification_token should be populated for session %s", sessionID)
	return token
}

// sandboxForOwnerAccount returns the id and name of the sandbox account owned by
// ownerAccountID, asserting exactly one exists. Completing registration
// provisions a sandbox alongside the production account; ownership lives in the
// sandbox_account link table rather than a column on account.
func sandboxForOwnerAccount(t *testing.T, ownerAccountID string) (id, name string) {
	t.Helper()
	err := authDB(t).QueryRow(
		`SELECT a.id, a.name
		   FROM account a
		   JOIN sandbox_account sa ON sa.account_id = a.id
		  WHERE sa.owner_account_id = ? AND a.account_type_code = 'sandbox'`,
		ownerAccountID,
	).Scan(&id, &name)
	require.NoError(t, err, "querying sandbox for owner account %s", ownerAccountID)
	return id, name
}

// backdateIssuedAt moves an order's issue date back, so a test can exercise the difference
// between the day an order was issued and the day somebody edits it. There is no API for it:
// issued_at is stamped by the issue action and never settable, which is exactly why an order
// that has been open for a while can only be built this way.
func backdateIssuedAt(t *testing.T, salesOrderID string, days int) {
	t.Helper()
	_, err := authDB(t).Exec(
		"UPDATE sales_order SET issued_at = DATE_SUB(issued_at, INTERVAL ? DAY) WHERE id = ?",
		days, salesOrderID,
	)
	require.NoError(t, err, "backdating issued_at for order %s", salesOrderID)
}

// defaultE2EAgentDBURL points at the agent-service Postgres published on the host (5432 is the
// local dev Postgres). CI can override via E2E_AGENT_DB_URL.
const defaultE2EAgentDBURL = "postgres://openmrp@127.0.0.1:5433/openmrp_agents?sslmode=disable"

var (
	agentDBOnce sync.Once
	agentDBConn *sql.DB
	agentDBErr  error
)

// agentDB returns a lazily-opened connection to the e2e agent-service database, which tests use to
// put a one-shot fixture back to its seeded state so the suite can run again on the same stack.
func agentDB(t *testing.T) *sql.DB {
	t.Helper()
	agentDBOnce.Do(func() {
		agentDBConn, agentDBErr = sql.Open("pgx", envOr("E2E_AGENT_DB_URL", defaultE2EAgentDBURL))
		if agentDBErr == nil {
			agentDBErr = agentDBConn.Ping()
		}
	})
	require.NoError(t, agentDBErr, "connecting to e2e agent database (is the stack up with postgres-e2e published on 5433?)")
	return agentDBConn
}

// resetAgentRun puts a seeded run back to the status a one-shot test expects to find it in.
func resetAgentRun(t *testing.T, runID, status string) {
	t.Helper()
	_, err := agentDB(t).Exec(`UPDATE agent_run
		SET status_code = $1, output = '{}', error_message = NULL, completed_at = NULL, retry_count = 0, updated_at = now()
		WHERE id = $2`, status, runID)
	require.NoError(t, err, "resetting agent run %s", runID)
}

// trimPublishedOutbox deletes outbox messages earlier runs already delivered. Services purge them only
// after a week, and every API call adds one, so a stack reused for several runs carries hundreds of
// thousands of dead rows that slow the outbox poller and fill Docker's disk. Undelivered rows stay.
func trimPublishedOutbox() error {
	conn, err := sql.Open("mysql", envOr("E2E_DB_URL", defaultE2EDBURL))
	if err != nil {
		return err
	}
	defer conn.Close()

	for {
		result, err := conn.Exec("DELETE FROM message_outbox WHERE status = 'published' LIMIT 5000")
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n == 0 {
			return nil
		}
	}
}

// trimRuntimeRequestLogs deletes the request logs earlier runs produced, keeping the seeded rqlog_
// rows. request_log has no retention, and on a stack reused for several runs a filter that matches
// nothing walks every row the account has ever logged, which outgrows the request deadline.
func trimRuntimeRequestLogs() error {
	conn, err := sql.Open("mysql", envOr("E2E_DB_URL", defaultE2EDBURL))
	if err != nil {
		return err
	}
	defer conn.Close()

	// Batched so the delete never holds locks the platform consumer is waiting on for long.
	for {
		result, err := conn.Exec("DELETE FROM request_log WHERE id LIKE 'rq\\_%' LIMIT 5000")
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n == 0 {
			return nil
		}
	}
}
