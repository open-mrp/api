package rpc

import (
	"context"

	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/contracts"

	"google.golang.org/grpc/metadata"
)

// MetadataOption configures the outgoing gRPC metadata built by [PrepareRPCCtx].
type MetadataOption func(metadata.MD)

// WithIdentity adds the caller's identity from the context to the outgoing gRPC metadata. If the context does not carry an identity the option is a no-op.
func WithIdentity(ctx context.Context) MetadataOption {
	return func(md metadata.MD) {
		if identity, ok := appctx.GetIdentityFromContext(ctx); ok && identity != nil {
			contracts.SetIdentityInMetadata(md, identity)
		}
	}
}

// WithMetadata sets arbitrary key/value pairs on the outgoing gRPC metadata.
func WithMetadata(key string, values ...string) MetadataOption {
	return func(md metadata.MD) {
		md.Set(key, values...)
	}
}

// ServiceCallOption configures the behavior of [PrepareServiceCallCtx].
type ServiceCallOption func(*serviceCallConfig)

type serviceCallConfig struct {
	// idempotencyKey (optional; default: "") explicitly overrides the idempotency key sent in outgoing metadata. When empty, the key carried by the incoming context is forwarded instead.
	idempotencyKey string
}

// WithIdempotencyKeyOverride sets an explicit idempotency key on the outgoing metadata, overriding whatever the context carries. Use this when a caller needs to pass a derived key (e.g. appending a suffix) instead of forwarding the original.
func WithIdempotencyKeyOverride(key string) ServiceCallOption {
	return func(c *serviceCallConfig) { c.idempotencyKey = key }
}

// PrepareServiceCallCtx builds outgoing gRPC metadata for a service-to-service call. It always forwards identity, the idempotency key, the request ID, and the propagated client IP from the incoming context. Callers can supply [ServiceCallOption] values to customise the defaults (e.g. override the idempotency key).
//
// Use this instead of assembling metadata manually in each gRPC client.
func PrepareServiceCallCtx(ctx context.Context, opts ...ServiceCallOption) context.Context {
	var cfg serviceCallConfig
	for _, o := range opts {
		o(&cfg)
	}

	mdOpts := []MetadataOption{
		WithIdentity(ctx),
	}

	// Idempotency key: explicit override wins, otherwise forward from context.
	switch {
	case cfg.idempotencyKey != "":
		mdOpts = append(mdOpts, WithMetadata(contracts.IdempotencyKeyHeader, cfg.idempotencyKey))
	default:
		if key, ok := appctx.GetIdempotencyKey(ctx); ok && key != "" {
			mdOpts = append(mdOpts, WithMetadata(contracts.IdempotencyKeyHeader, key))
		}
	}

	if requestID, ok := appctx.GetRequestID(ctx); ok && requestID != "" {
		mdOpts = append(mdOpts, WithMetadata(contracts.RequestIDHeader, requestID))
	}

	if ip, ok := appctx.GetPropagatedClientIP(ctx); ok {
		mdOpts = append(mdOpts, WithMetadata(contracts.ClientIPHeader, ip))
	}

	return PrepareRPCCtx(ctx, mdOpts...)
}

// PrepareRPCCtx returns a new context with outgoing gRPC metadata configured by the supplied options. Existing outgoing metadata on ctx is preserved.
func PrepareRPCCtx(ctx context.Context, opts ...MetadataOption) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}

	for _, o := range opts {
		o(md)
	}

	return metadata.NewOutgoingContext(ctx, md)
}
