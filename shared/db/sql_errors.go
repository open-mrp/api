package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"

	apierror "github.com/open-mrp/api/shared/errors"
)

// DuplicateKeyMapping maps MySQL unique constraint names to custom APIError constructors.
type DuplicateKeyMapping map[string]func() *apierror.APIError

// MapSQLError converts common SQL/driver errors into an APIError so callers can differentiate expected cases (e.g. not found) from infrastructural failures (timeouts, connection issues, unknown errors).
func MapSQLError(err error) *apierror.APIError {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return apierror.NewResourceNotFoundError("Resource not found.")
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return apierror.NewInternalError(err, "Database query deadline exceeded.")
	}

	if errors.Is(err, context.Canceled) {
		return apierror.NewClientClosedRequestError("Request was canceled.")
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return apierror.NewInternalError(err, "Database request timed out.")
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1062: // duplicate entry
			return apierror.NewResourceExistsError("Resource already exists.")
		case 1205: // lock wait timeout
			return apierror.NewInternalError(err, "Database lock wait timed out.")
		case 1213: // deadlock
			return apierror.NewInternalError(err, "Database deadlock; the transaction was rolled back.")
		case 1105: // Vitess catch-all — only the message says what happened
			if isVitessTxKill(mysqlErr.Message) {
				return apierror.NewInternalError(err, "Database transaction exceeded the server time limit and was rolled back.")
			}
		case 1040, 2002, 2006: // too many connections / conn refused / server gone
			return apierror.NewInternalError(err, "Database unavailable.")
		case 1053, 1927, 2013: // server shutdown / connection killed / lost conn during query
			// Vitess/PlanetScale surfaces tablet failovers and dropped vttablet
			// connections as these codes (e.g. 2013 "Lost connection to MySQL
			// server during query" wrapping a gRPC Canceled/EOF).
			return apierror.NewInternalError(err, "Database connection lost.")
		}
	}

	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, mysql.ErrInvalidConn) {
		return apierror.NewInternalError(err, "Database connection lost.")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return apierror.NewResourceExistsError("Resource already exists.")
		case "23503": // foreign_key_violation
			return apierror.NewValidationError("Referenced resource does not exist.")
		case "23502": // not_null_violation
			return apierror.NewValidationError("A required field is missing.")
		case "40001", "40P01": // serialization_failure / deadlock_detected
			return apierror.NewInternalError(err, "Database request timed out.")
		case "53300", "53400": // too_many_connections / configuration_limit_exceeded
			return apierror.NewInternalError(err, "Database unavailable.")
		case "42P01": // undefined_table
			return apierror.NewInternalError(err, "Database table does not exist.")
		case "42703": // undefined_column
			return apierror.NewInternalError(err, "Database column does not exist.")
		}
	}

	return apierror.NewInternalError(err, "Database request failed for unknown reason.")
}

// MapSQLErrorWithDuplicateKeys works like MapSQLError but, for MySQL 1062 errors, looks up the violated constraint name in the provided mapping to return a domain-specific error. If no mapping matches, it falls through to the generic ResourceExistsError from MapSQLError.
func MapSQLErrorWithDuplicateKeys(err error, mapping DuplicateKeyMapping) *apierror.APIError {
	if err == nil {
		return nil
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		keyName := extractKeyName(mysqlErr.Message)
		if fn, ok := mapping[keyName]; ok {
			return fn()
		}
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if fn, ok := mapping[pgErr.ConstraintName]; ok {
			return fn()
		}
	}

	return MapSQLError(err)
}

// isVitessTxKill reports whether a MySQL 1105 carries vttablet's transaction-killer message.
//
// 1105 is ER_UNKNOWN_ERROR, which Vitess uses as a catch-all, so the code alone says nothing and the
// message is the only discriminator. The one worth naming is the per-transaction time limit —
// PlanetScale rolls a transaction back once it has been open too long, and the driver error reads
//
//	Error 1105 (HY000): target: <keyspace>.-.primary: vttablet: rpc error: code = Aborted
//	desc = transaction 1787579861050300718: in use: in use: for tx killer rollback
//
// which mapped to "Database request failed for unknown reason." That is how a core-service message
// sat in message_inbox from 2026-08-25 to 2026-09-01 looking like a mystery rather than like a
// transaction that was simply too big. Matching on the message is unlovely, and it is what there is.
func isVitessTxKill(msg string) bool {
	return strings.Contains(msg, "tx killer") ||
		(strings.Contains(msg, "transaction") && strings.Contains(msg, "in use"))
}

// IsDeadlock reports whether err is a MySQL 1213 (deadlock) or PostgreSQL 40P01 (deadlock_detected) / 40001 (serialization_failure) error.
func IsDeadlock(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1213 {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && (pgErr.Code == "40P01" || pgErr.Code == "40001")
}

// IsRetryableLockConflict reports whether err is a transient database lock conflict that is safe to retry around a small, idempotent database operation.
func IsRetryableLockConflict(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1205 || mysqlErr.Number == 1213
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && (pgErr.Code == "40P01" || pgErr.Code == "40001")
}

// IsRetryableConnectionError reports whether err is a transient connection-level failure (connection refused, server gone away, connection killed, or lost mid-query — e.g. a Vitess tablet failover) that is safe to retry for an idempotent operation. It returns false when the caller's own context is canceled or past its deadline, since retrying then is pointless.
func IsRetryableConnectionError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, mysql.ErrInvalidConn) {
		return true
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1053, 1927, 2002, 2006, 2013: // server shutdown / conn killed / conn refused / server gone / lost conn during query
			return true
		}
	}
	return false
}

// IsDuplicateEntry reports whether err is a MySQL 1062 (duplicate entry) error.
func IsDuplicateEntry(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// extractKeyName parses "Duplicate entry '...' for key '<table>.<key_name>'" and returns the key_name portion. Returns "" if the format doesn't match.
func extractKeyName(message string) string {
	const marker = "for key '"
	idx := strings.LastIndex(message, marker)
	if idx == -1 {
		return ""
	}
	rest := message[idx+len(marker):]
	before, _, ok := strings.Cut(rest, "'")
	if !ok {
		return ""
	}
	keyName := before
	// Strip table prefix: "unit.unit_account_id_name_key" → "unit_account_id_name_key"
	if dot := strings.LastIndexByte(keyName, '.'); dot != -1 {
		keyName = keyName[dot+1:]
	}
	return keyName
}
