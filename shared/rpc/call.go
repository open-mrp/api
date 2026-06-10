package rpc

import (
	"context"
	"time"

	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const DefaultTimeout = 10 * time.Second

// Option configures the behavior of [CallRPC].
type Option func(*config)

type config struct {
	// timeout (optional; default: 0, meaning DefaultTimeout of 10s) overrides the
	// RPC deadline. Set via WithTimeout.
	timeout time.Duration

	// onReplayed (optional; default: nil) is invoked when the response was an
	// idempotent replay; skipped when nil. Set via WithOnReplayed.
	onReplayed func()
}

// WithTimeout overrides the default RPC deadline ([DefaultTimeout]).
func WithTimeout(t time.Duration) Option {
	return func(c *config) { c.timeout = t }
}

// WithOnReplayed registers a callback that is invoked when the response
// headers indicate the server replayed a cached idempotent response.
func WithOnReplayed(fn func()) Option {
	return func(c *config) { c.onReplayed = fn }
}

// CallRPC executes a gRPC call with standard boilerplate:
//
//  1. Starts a tracing span named spanName on tracer, deferred End.
//  2. Applies a timeout (default [DefaultTimeout], overridable via [WithTimeout]).
//  3. Captures response header metadata.
//  4. Invokes call with the prepared context and grpc.Header option.
//  5. Converts any gRPC error to an [apierror.APIError] using serviceName.
//  6. Checks the response headers for the idempotent-replayed marker and
//     invokes the [WithOnReplayed] callback when present.
func CallRPC[T any](
	ctx context.Context,
	tracer trace.Tracer,
	spanName string,
	serviceName string,
	call func(ctx context.Context, opts ...grpc.CallOption) (T, error),
	opts ...Option,
) (T, *apierror.APIError) {
	ctx, span := tracer.Start(ctx, spanName)
	defer span.End()

	var cfg config
	for _, o := range opts {
		o(&cfg)
	}

	timeout := DefaultTimeout
	if cfg.timeout > 0 {
		timeout = cfg.timeout
	}
	rpcCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var headerMD metadata.MD
	resp, err := call(rpcCtx, grpc.Header(&headerMD))

	if apiErr := contracts.ConvertGRPCError(ctx, err, serviceName); apiErr != nil {
		var zero T
		return zero, tracing.Trace(span, apiErr)
	}

	if cfg.onReplayed != nil && contracts.IsIdempotentReplayed(headerMD) {
		cfg.onReplayed()
	}

	return resp, nil
}
