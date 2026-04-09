package apirequest

import (
	"encoding/json"
	"reflect"
)

// NullableInput represents a JSON field that can be absent, explicitly null, or a value.
// Use this for nested objects on PATCH endpoints where you need to distinguish all three states.
type NullableInput[T any] struct {
	value *T
	set   bool
}

// OpenAPINullableInner returns the reflect.Type of the wrapped value type.
// The OpenAPI generator uses this to render NullableInput[T] as a nullable T schema.
func (n NullableInput[T]) OpenAPINullableInner() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

// IsSet returns true when the field was present in the JSON body (even if null).
func (n NullableInput[T]) IsSet() bool { return n.set }

// IsNull returns true when the field was explicitly set to null.
func (n NullableInput[T]) IsNull() bool { return n.set && n.value == nil }

// Value returns the parsed value, or nil if absent or null.
func (n NullableInput[T]) Value() *T { return n.value }

func (n *NullableInput[T]) UnmarshalJSON(data []byte) error {
	n.set = true
	if string(data) == "null" {
		n.value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	n.value = &v
	return nil
}
