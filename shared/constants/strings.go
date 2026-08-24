package constants

// Strings converts a slice of enum values into plain strings, for handing a validated
// list filter down to a layer that takes strings.
func Strings[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

// EnumPtr converts an optional plain string from a lower layer into the optional enum value the API surface exposes.
func EnumPtr[T ~string](v *string) *T {
	if v == nil {
		return nil
	}
	e := T(*v)
	return &e
}
