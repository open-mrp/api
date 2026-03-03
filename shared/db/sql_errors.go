package db

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"strings"

	"github.com/go-sql-driver/mysql"

	apierror "github.com/augno/api/shared/errors"
)

// DuplicateKeyMapping maps MySQL unique constraint names to custom APIError constructors.
type DuplicateKeyMapping map[string]func() *apierror.APIError

// MapSQLError converts common SQL/driver errors into an APIError so callers
// can differentiate expected cases (e.g. not found) from infrastructural
// failures (timeouts, connection issues, unknown errors).
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

	return apierror.NewInternalError(err, "Database request failed for unknown reason.")
}

// MapSQLErrorWithDuplicateKeys works like MapSQLError but, for MySQL 1062 errors,
// looks up the violated constraint name in the provided mapping to return a
// domain-specific error. If no mapping matches, it falls through to the generic
// ResourceExistsError from MapSQLError.
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

	return MapSQLError(err)
}

// IsDuplicateEntry reports whether err is a MySQL 1062 (duplicate entry) error.
func IsDuplicateEntry(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// extractKeyName parses "Duplicate entry '...' for key '<table>.<key_name>'"
// and returns the key_name portion. Returns "" if the format doesn't match.
func extractKeyName(message string) string {
	const marker = "for key '"
	idx := strings.LastIndex(message, marker)
	if idx == -1 {
		return ""
	}
	rest := message[idx+len(marker):]
	end := strings.IndexByte(rest, '\'')
	if end == -1 {
		return ""
	}
	keyName := rest[:end]
	// Strip table prefix: "unit.unit_account_id_name_key" → "unit_account_id_name_key"
	if dot := strings.LastIndexByte(keyName, '.'); dot != -1 {
		keyName = keyName[dot+1:]
	}
	return keyName
}
