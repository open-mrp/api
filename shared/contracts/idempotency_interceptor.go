package contracts

import (
	"context"

	"github.com/augno/api/shared/appctx"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// IdempotentReplayedHeader is the gRPC response metadata key that signals a
	// cached (replayed) response. Clients can inspect this to know whether the
	// server executed the request or returned a stored result.
	IdempotentReplayedHeader = "x-idempotent-replayed"
	// IdempotentReplayedHeaderValue is the value set on IdempotentReplayedHeader
	// when the response was served from cache.
	IdempotentReplayedHeaderValue = "true"
)

// SetIdempotentReplayed sets the idempotent replayed header on the gRPC response.
func SetIdempotentReplayed(ctx context.Context) {
	_ = grpc.SetHeader(ctx, metadata.Pairs(IdempotentReplayedHeader, IdempotentReplayedHeaderValue))
}

// IsIdempotentReplayed checks if the idempotent replayed header is set in the gRPC metadata.
func IsIdempotentReplayed(md metadata.MD) bool {
	values := md.Get(IdempotentReplayedHeader)
	return len(values) > 0 && values[0] == IdempotentReplayedHeaderValue
}

// WithIdempotencyTracking sets up idempotency response tracking for a gRPC handler.
// It returns the updated context and a finalize function that should be deferred.
// The finalize function will set the appropriate gRPC header if the response was replayed.
//
// Usage:
//
//	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
//	defer finalizeIdempotency()
func WithIdempotencyTracking(ctx context.Context) (context.Context, func()) {
	meta := &appctx.IdempotencyResponseMetadata{}
	ctx = appctx.WithIdempotencyResponseMetadata(ctx, meta)
	return ctx, func() {
		if meta.Replayed {
			SetIdempotentReplayed(ctx)
		}
	}
}
