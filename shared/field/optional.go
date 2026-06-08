package field

import "reflect"

// Optional is an optional request/input value: present-or-absent, never null.
// It is documented as nullable in OpenAPI (the value domain may be null), but an
// explicit JSON null is rejected at unmarshal — callers express "no value" by
// omitting the key. Absent keys are unset.
//
// Use the value type on create/input request structs with json:"<field>,omitzero"
// (not a pointer). Do not use *Optional[T]: encoding/json would treat explicit null
// as a nil pointer without invoking UnmarshalJSON, so null would not be rejected.
type Optional[T any] struct {
	set   bool
	value T
}

// None returns a nullable field that was not provided in the request.
func None[T any]() Optional[T] {
	return Optional[T]{}
}

// Some returns a nullable field set to the given value.
func Some[T any](v T) Optional[T] {
	return Optional[T]{set: true, value: v}
}

// SomePtr returns None when p is nil, otherwise Some(*p).
func SomePtr[T any](p *T) Optional[T] {
	if p == nil {
		return None[T]()
	}
	return Some(*p)
}

// IsUnset reports whether the field was absent from the request.
func (n Optional[T]) IsUnset() bool { return !n.set }

// IsSet reports whether the field has a concrete value.
func (n Optional[T]) IsSet() bool { return n.set }

// Value returns the value and true when IsSet.
func (n Optional[T]) Value() (T, bool) {
	if !n.IsSet() {
		var zero T
		return zero, false
	}
	return n.value, true
}

// Ptr returns a pointer to the value when IsSet, otherwise nil.
func (n Optional[T]) Ptr() *T {
	if !n.IsSet() {
		return nil
	}
	v := n.value
	return &v
}

// OpenAPIInnerType returns the wrapped value type for OpenAPI schema generation.
func (Optional[T]) OpenAPIInnerType() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

// OpenAPIKind identifies this type for OpenAPI generation (nullable input, not clearable).
func (Optional[T]) OpenAPIKind() string { return "nullable_input" }
