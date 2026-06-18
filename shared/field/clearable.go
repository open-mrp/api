package field

// Clearable represents a PATCH field with three states: unset (omit), clear (null), or set (value).
//
// Use it on PATCH/update request structs as a value (Clearable[T], never *Clearable[T]) with json:"<name>,omitzero", where a caller may send a value, omit the key to leave the field unchanged, or send null to clear it. It must be a value, not a pointer: encoding/json short-circuits an explicit null on a pointer field to a nil pointer without calling UnmarshalJSON, which would make "clear" indistinguishable from "unset". As a value the addressable field's UnmarshalJSON is always invoked, so null is recorded as clear.
type Clearable[T any] struct {
	state clearableState
	value T
}

type clearableState uint8

const (
	stateUnset clearableState = iota
	stateClear
	stateSet
)

// Unset returns a field that was not provided in the request.
func Unset[T any]() Clearable[T] {
	return Clearable[T]{state: stateUnset}
}

// Clear returns a field explicitly set to null.
func Clear[T any]() Clearable[T] {
	return Clearable[T]{state: stateClear}
}

// Set returns a field set to the given value.
func Set[T any](v T) Clearable[T] {
	return Clearable[T]{state: stateSet, value: v}
}

// IsUnset reports whether the field was absent from the request.
func (f Clearable[T]) IsUnset() bool { return f.state == stateUnset }

// WasProvided reports whether the field was present in the JSON body (even if null).
func (f Clearable[T]) WasProvided() bool { return !f.IsUnset() }

// IsNull reports whether the field was explicitly set to null in JSON.
func (f Clearable[T]) IsNull() bool { return f.IsClear() }

// IsClear reports whether the field was explicitly cleared.
func (f Clearable[T]) IsClear() bool { return f.state == stateClear }

// IsSet reports whether the field has a concrete value.
func (f Clearable[T]) IsSet() bool { return f.state == stateSet }

// Value returns the value and true when IsSet.
func (f Clearable[T]) Value() (T, bool) {
	if !f.IsSet() {
		var zero T
		return zero, false
	}
	return f.value, true
}

// ValuePtr returns a pointer to the value when IsSet, otherwise nil.
func (f Clearable[T]) ValuePtr() *T {
	if !f.IsSet() {
		return nil
	}
	v := f.value
	return &v
}

// BackfillUnset replaces unset fields with the provided existing value.
func (f Clearable[T]) BackfillUnset(existing T) Clearable[T] {
	if f.IsUnset() {
		return Set(existing)
	}
	return f
}

// BackfillUnsetPtr replaces unset fields with the existing pointer value when non-nil.
func (f Clearable[T]) BackfillUnsetPtr(existing *T) Clearable[T] {
	if f.IsUnset() {
		if existing == nil {
			return Unset[T]()
		}
		return Set(*existing)
	}
	return f
}

// StringPtrAfterBackfill returns *string for repository use after backfilling unset from existing.
// Clear yields nil (SQL NULL); set yields a pointer to the value.
func (f Clearable[string]) StringPtrAfterBackfill(existing *string) *string {
	f = f.BackfillUnsetPtr(existing)
	if f.IsClear() {
		return nil
	}
	return f.ValuePtr()
}
