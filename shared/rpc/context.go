package rpc

import (
	"context"

	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"

	"google.golang.org/grpc/metadata"
)

// MetadataOption configures the outgoing gRPC metadata built by [PrepareRPCCtx].
type MetadataOption func(metadata.MD)

// WithIdentity adds the caller's identity from the context to the outgoing
// gRPC metadata. If the context does not carry an identity the option is a no-op.
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

// PrepareRPCCtx returns a new context with outgoing gRPC metadata configured
// by the supplied options. Existing outgoing metadata on ctx is preserved.
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
