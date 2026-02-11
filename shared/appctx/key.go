package appctx

// contextKey is an unexported type used for all context keys in this package to
// prevent collisions with keys defined in other packages.
type contextKey string

// noTraceKeyType is the unexported context-key type for the "no trace" flag.
// Using a dedicated struct type avoids collisions with other context values.
type noTraceKeyType struct{}
