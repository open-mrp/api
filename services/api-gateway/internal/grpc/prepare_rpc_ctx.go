package grpc

import (
	"context"
	"time"

	apicontext "github.com/augno/api/services/api-gateway/internal/context"

	grpcidentity "github.com/augno/api/services/auth-service/pkg/grpc"

	"google.golang.org/grpc/metadata"
)

const (
	DefaultRPCTimeout        = 5 * time.Second
	PasswordOperationTimeout = 15 * time.Second
)

func PrepareRPCCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return PrepareRPCCtxWithTimeout(ctx, DefaultRPCTimeout)
}

// PrepareRPCCtxWithTimeout creates an RPC context with a custom timeout.
func PrepareRPCCtxWithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	rpcCtx, cancel := context.WithTimeout(ctx, timeout)

	if identity, ok := apicontext.GetIdentityFromContext(ctx); ok && identity != nil {
		md, ok := metadata.FromOutgoingContext(rpcCtx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}

		grpcidentity.SetIdentityInMetadata(md, identity)
		rpcCtx = metadata.NewOutgoingContext(rpcCtx, md)
	}

	return rpcCtx, cancel
}
