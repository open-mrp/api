// Package metadata is the client-owned string map on sales orders, their lines and invoices. OpenMRP
// stores it and returns it unchanged; it never reads it.
//
// Writes follow the clearable-field contract: on update, keys not sent are kept, a key sent as null is
// removed, and `metadata: null` removes every key. An empty string is a value like any other.
package metadata

import (
	"database/sql"
	"encoding/json"
	"fmt"

	apierror "github.com/open-mrp/api/shared/errors"
)

const (
	MaxKeys        = 50
	MaxKeyLength   = 40
	MaxValueLength = 500
)

// Update is a change to a record's metadata. Clear empties the map before Set is applied; keys in Remove
// are dropped. The zero value changes nothing.
type Update struct {
	Clear  bool
	Set    map[string]string
	Remove []string
}

// IsZero reports whether the update leaves the metadata as it is.
func (u Update) IsZero() bool {
	return !u.Clear && len(u.Set) == 0 && len(u.Remove) == 0
}

// Encode is the column value for a new row: nil (NULL) when nothing is set.
func Encode(m map[string]string) json.RawMessage {
	if len(m) == 0 {
		return nil
	}
	b, _ := json.Marshal(m) // a map[string]string always marshals
	return b
}

// Patch is the JSON_MERGE_PATCH document for an update, where a null value removes its key. It is NULL
// when the update changes nothing, so the column is left alone. Clear is applied by the SQL, which then
// merges into an empty object instead of the stored one.
func Patch(u Update) sql.NullString {
	if u.IsZero() {
		return sql.NullString{}
	}
	patch := make(map[string]*string, len(u.Set)+len(u.Remove))
	for _, k := range u.Remove {
		patch[k] = nil
	}
	for k, v := range u.Set {
		patch[k] = &v
	}
	b, _ := json.Marshal(patch)
	return sql.NullString{String: string(b), Valid: true}
}

// Decode reads the column as sqlc scans `CAST(metadata AS CHAR)`: nil, []byte or string. NULL decodes to
// an empty map, never nil, so a resource always shows `{}`. OpenMRP only ever writes strings; a value of
// another JSON type (a hand-edited row) comes back as its JSON text rather than failing the read.
func Decode(v any) map[string]string {
	var raw []byte
	switch typed := v.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	}
	m := map[string]string{}
	var fields map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &fields) != nil {
		return m
	}
	for k, field := range fields {
		var s string
		if json.Unmarshal(field, &s) == nil {
			m[k] = s
		} else {
			m[k] = string(field)
		}
	}
	return m
}

// CheckLimit rejects a merged map over MaxKeys. It runs on the row after the update, inside its
// transaction, since only the merge result knows how many keys the object holds.
func CheckLimit(m map[string]string, param string) *apierror.APIError {
	if len(m) > MaxKeys {
		return apierror.NewParameterInvalidError(
			fmt.Sprintf("An object can hold at most %d metadata keys; this update would leave %d. Set keys you no longer need to null to remove them.", MaxKeys, len(m)),
			param,
		)
	}
	return nil
}
