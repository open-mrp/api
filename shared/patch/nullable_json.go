package patch

import (
	"bytes"
	"encoding/json"
	"errors"
)

// ErrExplicitNull is returned when JSON null is sent for a Nullable field.
var ErrExplicitNull = errors.New("patch: explicit null is not allowed")

// IsZero reports whether the field is unset so encoding/json omitempty omits it.
func (n Nullable[T]) IsZero() bool {
	return n.IsUnset()
}

// MarshalJSON encodes set values; unset fields must use json omitempty.
func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if !n.IsSet() {
		return nil, errMarshalUnset
	}
	return json.Marshal(n.value)
}

// UnmarshalJSON decodes JSON: null is rejected; values set the field.
func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return ErrExplicitNull
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	n.set = true
	n.value = v
	return nil
}
