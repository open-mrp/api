package fieldpb

import "github.com/open-mrp/apikit/field"

// A PATCH merges a section (defaults, contact, freight, commitment) field by field: the request carries the section as
// field.Optional[XxxInput] whose own fields are field.Optional (never null) or field.Clearable (null clears), and the
// service applies each field onto the stored section with ApplyOptional and ApplyClearable. A field the request left
// out keeps its stored value, so sending one field of a section never resets its siblings. Across the RPC, each field
// travels as its own patch message (StringPatch, QuantityPatch, DatePatch…) inside the section's message, and an
// omitted section is an unset message.

// ApplyOptional is the value after a PATCH: current when the field was left out, otherwise the sent value.
func ApplyOptional[T any](current T, f field.Optional[T]) T {
	if v, ok := f.Value(); ok {
		return v
	}
	return current
}

// ApplyClearable is the value after a PATCH: current when the field was left out, nil when it was sent as null, otherwise the sent value.
func ApplyClearable[T any](current *T, f field.Clearable[T]) *T {
	switch {
	case f.IsUnset():
		return current
	case f.IsClear():
		return nil
	default:
		return f.ValuePtr()
	}
}
