package field

import (
	"encoding/json"
	"errors"
	"reflect"
)

var errMarshalUnset = errors.New("patch: cannot marshal unset field")

// OpenAPIInnerType returns the wrapped value type for OpenAPI schema generation.
func (Clearable[T]) OpenAPIInnerType() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

// OpenAPIKind identifies this type for OpenAPI generation (clearable PATCH field).
func (Clearable[T]) OpenAPIKind() string { return "clearable" }

// IsZero reports whether the field is unset so encoding/json omitempty omits it.
func (f Clearable[T]) IsZero() bool {
	return f.IsUnset()
}

// MarshalJSON encodes set values as JSON and clear as null.
// Unset fields must use json omitzero (IsZero reports unset) so they are omitted.
func (f Clearable[T]) MarshalJSON() ([]byte, error) {
	switch f.state {
	case stateClear:
		return []byte("null"), nil
	case stateSet:
		return json.Marshal(f.value)
	default:
		return nil, errMarshalUnset
	}
}

// UnmarshalJSON decodes PATCH JSON: absent keys stay unset; null clears; values set.
func (f *Clearable[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		f.state = stateClear
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	f.state = stateSet
	f.value = v
	return nil
}
