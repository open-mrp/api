package field

import (
	"bytes"
	"encoding/json"
	"errors"
)

// ErrExplicitNull is returned when JSON null is sent for an Optional field.
// The message is consumer-facing: UnmarshalJSON cannot know the field's JSON
// key, so callers that have the request body should use ExplicitNullField to
// build a parameter-specific message instead of surfacing this text directly.
var ErrExplicitNull = errors.New("this field cannot be null")

// IsZero reports whether the field is unset so encoding/json omitempty omits it.
func (n Optional[T]) IsZero() bool {
	return n.IsUnset()
}

// MarshalJSON encodes set values; unset fields must use json omitempty.
func (n Optional[T]) MarshalJSON() ([]byte, error) {
	if !n.IsSet() {
		return nil, errMarshalUnset
	}
	return json.Marshal(n.value)
}

// UnmarshalJSON decodes JSON: null is rejected; values set the field.
func (n *Optional[T]) UnmarshalJSON(data []byte) error {
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
