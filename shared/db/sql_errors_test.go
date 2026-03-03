package db

import (
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"

	apierror "github.com/augno/api/shared/errors"
)

func TestExtractKeyName(t *testing.T) {
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

func TestIsDuplicateEntry(t *testing.T) {
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
