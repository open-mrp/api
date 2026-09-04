package ptrutil

// Deref returns the value of the given pointer as long as it's non-nil, and the zero value of T otherwise.
func Deref[T any](ptr *T) T {
	if ptr != nil {
		return *ptr
	}
	var zero T
	return zero
}

// ValOrDefault returns the value of the given pointer as long as it's non-nil, and the specified default value otherwise.
func ValOrDefault[T any](ptr *T, defaultVal T) T {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

// ValOrDefaultFunc returns the value of the given pointer as long as it's non-nil, or invokes the given function to produce a default value otherwise.
func ValOrDefaultFunc[T any](ptr *T, defaultFunc func() T) T {
	if ptr != nil {
		return *ptr
	}
	return defaultFunc()
}

// ApplyIfSet sets *dst to *src if src is non-nil. Use this to merge optional (pointer) fields from a PATCH input into an existing plain-value struct without overwriting fields that were not provided in the request.
func ApplyIfSet[T any](dst *T, src *T) {
	if src != nil {
		*dst = *src
	}
}

// NonEmptyPtr returns a pointer to s, or nil when s is empty, for proto optional fields that should be absent rather than blank.
func NonEmptyPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
