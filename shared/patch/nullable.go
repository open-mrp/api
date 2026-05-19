package patch

import "reflect"

// Nullable is an optional request/input value documented as nullable in OpenAPI.
// Absent keys are unset; explicit JSON null is rejected at unmarshal.
//
// Use the value type on request structs with json:"field,omitzero" (not a pointer).
// Do not use *Nullable[T]: encoding/json would treat explicit null as a nil pointer
// without invoking UnmarshalJSON, so null would not be rejected.
type Nullable[T any] struct {
	set   bool
	value T
}

// UnsetNullable returns a nullable field that was not provided in the request.
func UnsetNullable[T any]() Nullable[T] {
	return Nullable[T]{}
}

// SetNullable returns a nullable field set to the given value.
func SetNullable[T any](v T) Nullable[T] {
	return Nullable[T]{set: true, value: v}
}

// PtrNullable returns UnsetNullable when p is nil, otherwise SetNullable(*p).
func PtrNullable[T any](p *T) Nullable[T] {
	if p == nil {
		return UnsetNullable[T]()
	}
	return SetNullable(*p)
}

// IsUnset reports whether the field was absent from the request.
func (n Nullable[T]) IsUnset() bool { return !n.set }

// IsSet reports whether the field has a concrete value.
func (n Nullable[T]) IsSet() bool { return n.set }

// Value returns the value and true when IsSet.
func (n Nullable[T]) Value() (T, bool) {
	if !n.IsSet() {
		var zero T
		return zero, false
	}
	return n.value, true
}

// Ptr returns a pointer to the value when IsSet, otherwise nil.
func (n Nullable[T]) Ptr() *T {
	if !n.IsSet() {
		return nil
	}
	v := n.value
	return &v
}

// OpenAPINullableInner returns the wrapped value type for OpenAPI schema generation.
func (Nullable[T]) OpenAPINullableInner() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

// OpenAPIKind identifies this type for OpenAPI generation (nullable input, not clearable).
func (Nullable[T]) OpenAPIKind() string { return "nullable_input" }
