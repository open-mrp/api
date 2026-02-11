package grpc

import (
	"context"
	"time"

	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/rpc"

	apierror "github.com/augno/api/shared/errors"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

const (
	PasswordOperationTimeout = 15 * time.Second
)

// RPCOption configures the behavior of [CallRPC]. Use the With* functions
// in this package to obtain option values.
type RPCOption func(*rpcConfig)

type rpcConfig struct {
	timeout time.Duration
}

// WithTimeout overrides the default RPC deadline for this call.
func WithTimeout(t time.Duration) RPCOption {
	return func(c *rpcConfig) { c.timeout = t }
}

// CallRPC executes a gRPC call with gateway-specific metadata (identity,
// idempotency key, API version, request ID) and standard boilerplate
// (tracing, timeout, error conversion, replayed detection).
//
// Callers within the gateway continue using this function with the same
// signature — it delegates to [rpc.CallRPC] after preparing metadata.
func CallRPC[T any](
	ctx context.Context,
	tracer trace.Tracer,
	spanName string,
	serviceName string,
	call func(ctx context.Context, opts ...grpc.CallOption) (T, error),
	opts ...RPCOption,
) (T, *apierror.APIError) {
	var cfg rpcConfig
	for _, o := range opts {
		o(&cfg)
	}

	// Build gateway-specific outgoing metadata.
	ctx = prepareGatewayMetadata(ctx)

	// Map gateway options to shared rpc options.
	var rpcOpts []rpc.Option
	if cfg.timeout > 0 {
		rpcOpts = append(rpcOpts, rpc.WithTimeout(cfg.timeout))
	}
	rpcOpts = append(rpcOpts, rpc.WithOnReplayed(func() {
		appctx.SetHTTPReplayed(ctx, true)
	}))

	return rpc.CallRPC(ctx, tracer, spanName, serviceName, call, rpcOpts...)
}

// prepareGatewayMetadata builds outgoing gRPC metadata from gateway context
// values: identity, idempotency key, idempotency key ID, API version, and
// request ID.
func prepareGatewayMetadata(ctx context.Context) context.Context {
	mdOpts := []rpc.MetadataOption{
		rpc.WithIdentity(ctx),
	}

	idempotencyKey := resolveIdempotencyKey(ctx)
	mdOpts = append(mdOpts, rpc.WithMetadata(contracts.IdempotencyKeyHeader, idempotencyKey))

	if keyID, ok := appctx.GetIdempotencyKeyID(ctx); ok && keyID != "" {
		mdOpts = append(mdOpts, rpc.WithMetadata(contracts.IdempotencyKeyIDHeader, keyID))
	}

	if version, ok := appctx.GetAPIVersionFromContext(ctx); ok {
		mdOpts = append(mdOpts, rpc.WithMetadata(contracts.APIVersionHeader, version.String()))
	}

	if rl, ok := appctx.GetRequestLog(ctx); ok && rl != nil && rl.ID != "" {
		mdOpts = append(mdOpts, rpc.WithMetadata(contracts.RequestIDHeader, rl.ID))
	}

	return rpc.PrepareRPCCtx(ctx, mdOpts...)
}

func resolveIdempotencyKey(ctx context.Context) string {
	if userKey, ok := appctx.GetIdempotencyKey(ctx); ok && userKey != "" {
		return userKey
	}
	if rl, ok := appctx.GetRequestLog(ctx); ok && rl != nil && rl.ID != "" {
		return rl.ID
	}
	panic("no idempotency key found in context")
}
