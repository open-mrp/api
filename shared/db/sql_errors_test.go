package db

import (
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"

	apierror "github.com/augno/api/shared/errors"
)

func TestExtractKeyName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "standard format with table prefix",
			message: "Duplicate entry 'x' for key 'unit.unit_account_id_name_key'",
			want:    "unit_account_id_name_key",
		},
		{
			name:    "without table prefix",
			message: "Duplicate entry 'x' for key 'unit_account_id_name_key'",
			want:    "unit_account_id_name_key",
		},
		{
			name:    "empty message",
			message: "",
			want:    "",
		},
		{
			name:    "no for key marker",
			message: "some other error message",
			want:    "",
		},
		{
			name:    "malformed — missing closing quote",
			message: "Duplicate entry 'x' for key 'unit.unit_account_id_name_key",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractKeyName(tt.message)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMapSQLErrorWithDuplicateKeys(t *testing.T) {
	t.Parallel()
	mapping := DuplicateKeyMapping{
		"unit_account_id_name_key": func() *apierror.APIError {
			return apierror.NewConflictErrorWithParam("A unit with this name already exists.", "name")
		},
	}

	t.Run("nil error returns nil", func(t *testing.T) {
		result := MapSQLErrorWithDuplicateKeys(nil, mapping)
		assert.Nil(t, result)
	})

	t.Run("1062 with matching mapping returns custom error", func(t *testing.T) {
		err := &mysql.MySQLError{
			Number:  1062,
			Message: "Duplicate entry 'val' for key 'unit.unit_account_id_name_key'",
		}
		result := MapSQLErrorWithDuplicateKeys(err, mapping)
		assert.NotNil(t, result)
		assert.Equal(t, apierror.ErrorCodeResourceConflict, result.Code)
		assert.Equal(t, "name", result.Param)
	})

	t.Run("1062 with no matching mapping falls back to generic", func(t *testing.T) {
		err := &mysql.MySQLError{
			Number:  1062,
			Message: "Duplicate entry 'val' for key 'unit.some_other_key'",
		}
		result := MapSQLErrorWithDuplicateKeys(err, mapping)
		assert.NotNil(t, result)
		assert.Equal(t, apierror.ErrorCodeResourceExists, result.Code)
	})

	t.Run("non-1062 error falls through to MapSQLError", func(t *testing.T) {
		err := &mysql.MySQLError{
			Number:  1205,
			Message: "Lock wait timeout exceeded",
		}
		result := MapSQLErrorWithDuplicateKeys(err, mapping)
		assert.NotNil(t, result)
		assert.Equal(t, apierror.ErrorCodeInternalError, result.Code)
	})
}

func TestMapSQLError_Postgres(t *testing.T) {
	t.Parallel()
	t.Run("unique_violation returns ResourceExists", func(t *testing.T) {
		err := &pgconn.PgError{Code: "23505", Message: "duplicate key value"}
		result := MapSQLError(err)
		assert.NotNil(t, result)
		assert.Equal(t, apierror.ErrorCodeResourceExists, result.Code)
	})

	t.Run("foreign_key_violation returns Validation", func(t *testing.T) {
		err := &pgconn.PgError{Code: "23503", Message: "foreign key violation"}
		result := MapSQLError(err)
		assert.NotNil(t, result)
		assert.Equal(t, apierror.ErrorCodeValidationFailed, result.Code)
	})

	t.Run("not_null_violation returns Validation", func(t *testing.T) {
		err := &pgconn.PgError{Code: "23502", Message: "not null violation"}
		result := MapSQLError(err)
		assert.NotNil(t, result)
		assert.Equal(t, apierror.ErrorCodeValidationFailed, result.Code)
	})

	t.Run("deadlock_detected returns InternalError", func(t *testing.T) {
		err := &pgconn.PgError{Code: "40P01", Message: "deadlock detected"}
		result := MapSQLError(err)
		assert.NotNil(t, result)
		assert.Equal(t, apierror.ErrorCodeInternalError, result.Code)
	})

	t.Run("too_many_connections returns InternalError", func(t *testing.T) {
		err := &pgconn.PgError{Code: "53300", Message: "too many connections"}
		result := MapSQLError(err)
		assert.NotNil(t, result)
		assert.Equal(t, apierror.ErrorCodeInternalError, result.Code)
	})

	t.Run("undefined_table returns InternalError", func(t *testing.T) {
		err := &pgconn.PgError{Code: "42P01", Message: "relation does not exist"}
		result := MapSQLError(err)
		assert.NotNil(t, result)
		assert.Equal(t, apierror.ErrorCodeInternalError, result.Code)
		assert.Equal(t, "Database table does not exist.", result.InternalMessage)
	})

	t.Run("unrecognized pg error falls through to unknown", func(t *testing.T) {
		err := &pgconn.PgError{Code: "XX000", Message: "internal error"}
		result := MapSQLError(err)
		assert.NotNil(t, result)
		assert.Equal(t, apierror.ErrorCodeInternalError, result.Code)
		assert.Equal(t, "Database request failed for unknown reason.", result.InternalMessage)
	})
}

func TestMapSQLErrorWithDuplicateKeys_Postgres(t *testing.T) {
	t.Parallel()
	mapping := DuplicateKeyMapping{
		"agent_account_status_account_id_agent_definition_id_key": func() *apierror.APIError {
			return apierror.NewConflictErrorWithParam("Status already exists for this agent.", "agent_definition_id")
		},
	}

	t.Run("23505 with matching constraint returns custom error", func(t *testing.T) {
		err := &pgconn.PgError{
			Code:           "23505",
			ConstraintName: "agent_account_status_account_id_agent_definition_id_key",
		}
		result := MapSQLErrorWithDuplicateKeys(err, mapping)
		assert.NotNil(t, result)
		assert.Equal(t, apierror.ErrorCodeResourceConflict, result.Code)
		assert.Equal(t, "agent_definition_id", result.Param)
	})

	t.Run("23505 with no matching constraint falls back to generic", func(t *testing.T) {
		err := &pgconn.PgError{
			Code:           "23505",
			ConstraintName: "some_other_constraint",
		}
		result := MapSQLErrorWithDuplicateKeys(err, mapping)
		assert.NotNil(t, result)
		assert.Equal(t, apierror.ErrorCodeResourceExists, result.Code)
	})
}

func TestIsDuplicateEntry_Postgres(t *testing.T) {
	t.Parallel()
	t.Run("returns true for PgError 23505", func(t *testing.T) {
		err := &pgconn.PgError{Code: "23505"}
		assert.True(t, IsDuplicateEntry(err))
	})

	t.Run("returns false for other PgError codes", func(t *testing.T) {
		err := &pgconn.PgError{Code: "23503"}
		assert.False(t, IsDuplicateEntry(err))
	})
}

func TestIsDuplicateEntry(t *testing.T) {
	t.Parallel()
	t.Run("returns true for 1062", func(t *testing.T) {
		err := &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}
		assert.True(t, IsDuplicateEntry(err))
	})

	t.Run("returns false for other MySQL errors", func(t *testing.T) {
		err := &mysql.MySQLError{Number: 1205, Message: "Lock wait timeout"}
		assert.False(t, IsDuplicateEntry(err))
	})

	t.Run("returns false for non-MySQL errors", func(t *testing.T) {
		err := assert.AnError
		assert.False(t, IsDuplicateEntry(err))
	})
}
