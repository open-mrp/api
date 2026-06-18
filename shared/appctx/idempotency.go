package appctx

import "context"

const (
	idempotencyKeyKey              contextKey = "idempotency_key"
	idempotencyKeyIDKey            contextKey = "idempotency_key_id"
	idempotencyResponseMetadataKey contextKey = "idempotency_response_metadata"
)

// WithIdempotencyKey returns a child context carrying the client-supplied idempotency key.
func WithIdempotencyKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, idempotencyKeyKey, key)
}

// GetIdempotencyKey retrieves the idempotency key from the context.
func GetIdempotencyKey(ctx context.Context) (string, bool) {
	key, ok := ctx.Value(idempotencyKeyKey).(string)
	return key, ok && key != ""
}

// WithIdempotencyKeyID returns a child context carrying the idempotency key database ID.
func WithIdempotencyKeyID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, idempotencyKeyIDKey, id)
}

// GetIdempotencyKeyID retrieves the idempotency key ID from the context.
func GetIdempotencyKeyID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(idempotencyKeyIDKey).(string)
	return v, ok && v != ""
}

// IdempotencyResponseMetadata holds mutable, response-scoped flags that allow service-layer code to communicate information back to the transport layer.
type IdempotencyResponseMetadata struct {
	Replayed bool
}

// WithIdempotencyResponseMetadata returns a child context carrying the given metadata pointer.
func WithIdempotencyResponseMetadata(ctx context.Context, meta *IdempotencyResponseMetadata) context.Context {
	return context.WithValue(ctx, idempotencyResponseMetadataKey, meta)
}

// GetIdempotencyResponseMetadata retrieves the metadata pointer from the context.
func GetIdempotencyResponseMetadata(ctx context.Context) (*IdempotencyResponseMetadata, bool) {
	meta, ok := ctx.Value(idempotencyResponseMetadataKey).(*IdempotencyResponseMetadata)
	return meta, ok && meta != nil
}

// MarkIdempotencyReplayed sets the Replayed flag on the context's idempotency response metadata. Safe to call when no metadata is present.
func MarkIdempotencyReplayed(ctx context.Context) {
	if meta, ok := GetIdempotencyResponseMetadata(ctx); ok {
		meta.Replayed = true
	}
}
