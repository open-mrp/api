package db

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"

	apierror "github.com/augno/api/shared/errors"
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
		case 1205, 1213: // lock wait timeout / deadlock
			return apierror.NewInternalError(err, "Database request timed out.")
		case 1040, 2002, 2006: // too many connections / conn refused / server gone
			return apierror.NewInternalError(err, "Database unavailable.")
		}
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
