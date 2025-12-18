package db

import (
	"context"
	"database/sql"
	"errors"
	"net"

	"github.com/go-sql-driver/mysql"

	"github.com/augno/api/shared/contracts"
)

// MapSQLError converts common SQL/driver errors into an APIError so callers
// can differentiate expected cases (e.g. not found) from infrastructural
// failures (timeouts, connection issues, unknown errors).
func MapSQLError(err error) *contracts.APIError {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return contracts.NewResourceNotFoundError("Resource not found.")
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return contracts.NewInternalError(err, "Database query deadline exceeded.")
	}

	if errors.Is(err, context.Canceled) {
		return contracts.NewClientClosedRequestError("Request was canceled.")
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return contracts.NewInternalError(err, "Database request timed out.")
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1062: // duplicate entry
			return contracts.NewResourceConflictError("Resource already exists.")
		case 1205, 1213: // lock wait timeout / deadlock
			return contracts.NewInternalError(err, "Database request timed out.")
		case 1040, 2002, 2006: // too many connections / conn refused / server gone
			return contracts.NewInternalError(err, "Database unavailable.")
		}
	}

	return contracts.NewInternalError(err, "Database request failed for unknown reason.")
}
