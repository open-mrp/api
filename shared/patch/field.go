package patch

// Field represents a PATCH field with three states: unset (omit), clear (null), or set (value).
type Field[T any] struct {
	state fieldState
	value T
}

type fieldState uint8

const (
	stateUnset fieldState = iota
	stateClear
	stateSet
)

// Unset returns a field that was not provided in the request.
func Unset[T any]() Field[T] {
	return Field[T]{state: stateUnset}
}

// Clear returns a field explicitly set to null.
func Clear[T any]() Field[T] {
	return Field[T]{state: stateClear}
}

// Set returns a field set to the given value.
func Set[T any](v T) Field[T] {
	return Field[T]{state: stateSet, value: v}
}

// SetPtr returns Clear when v is nil, otherwise Set(*v).
func SetPtr[T any](v *T) Field[T] {
	if v == nil {
		return Clear[T]()
	}
	return Set(*v)
}

// IsUnset reports whether the field was absent from the request.
func (f Field[T]) IsUnset() bool { return f.state == stateUnset }

// WasProvided reports whether the field was present in the JSON body (even if null).
func (f Field[T]) WasProvided() bool { return !f.IsUnset() }

// IsNull reports whether the field was explicitly set to null in JSON.
func (f Field[T]) IsNull() bool { return f.IsClear() }

// IsClear reports whether the field was explicitly cleared.
func (f Field[T]) IsClear() bool { return f.state == stateClear }

// IsSet reports whether the field has a concrete value.
func (f Field[T]) IsSet() bool { return f.state == stateSet }

// Value returns the value and true when IsSet.
func (f Field[T]) Value() (T, bool) {
	if !f.IsSet() {
		var zero T
		return zero, false
	}
	return f.value, true
}

// ValuePtr returns a pointer to the value when IsSet, otherwise nil.
func (f Field[T]) ValuePtr() *T {
	if !f.IsSet() {
		return nil
	}
	v := f.value
	return &v
}

// BackfillUnset replaces unset fields with the provided existing value.
func (f Field[T]) BackfillUnset(existing T) Field[T] {
	if f.IsUnset() {
		return Set(existing)
	}
	return f
}

// BackfillUnsetPtr replaces unset fields with the existing pointer value when non-nil.
func (f Field[T]) BackfillUnsetPtr(existing *T) Field[T] {
	if f.IsUnset() {
		if existing == nil {
			return Unset[T]()
		}
		return Set(*existing)
	}
	return f
}

// Coalesce returns Unset when f is nil; otherwise it returns *f.
func Coalesce[T any](f *Field[T]) Field[T] {
	if f == nil {
		return Unset[T]()
	}
	return *f
}

// Ptr returns a pointer to f for use in request examples and tests.
//
//go:fix inline
func Ptr[T any](f Field[T]) *Field[T] {
	return new(f)
}

// StringPtrAfterBackfill returns *string for repository use after backfilling unset from existing.
// Clear yields nil (SQL NULL); set yields a pointer to the value.
func (f Field[string]) StringPtrAfterBackfill(existing *string) *string {
	f = f.BackfillUnsetPtr(existing)
	if f.IsClear() {
		return nil
	}
	return f.ValuePtr()
}
