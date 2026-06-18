package contracts

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// retryWaitForReadyTimeout bounds how long the single retry attempt waits for a broken subchannel to re-establish before giving up. This bridges the brief window when a server process is restarting (e.g. a Tilt in-place hot reload via docker_build_with_restart) without letting calls hang indefinitely when a service is genuinely down.
const retryWaitForReadyTimeout = 5 * time.Second

// retryOnTransientUnaryClientInterceptor retries a request once if a transient error is encountered. The first attempt keeps the caller's fast-fail semantics; the retry sets WaitForReady so it queues until the connection re-establishes (bounded by retryWaitForReadyTimeout) instead of fast-failing against a subchannel that is still reconnecting after a server restart.
func retryOnTransientUnaryClientInterceptor(targetName string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err == nil {
			return nil
		}

		apiErr := ConvertGRPCError(ctx, err, targetName)
		if apiErr == nil || !apiErr.IsTransient {
			return err
		}

		retryCtx, cancel := context.WithTimeout(ctx, retryWaitForReadyTimeout)
		defer cancel()

		// Build a fresh options slice so we don't mutate the caller's backing array.
		retryOpts := make([]grpc.CallOption, 0, len(opts)+1)
		retryOpts = append(retryOpts, opts...)
		retryOpts = append(retryOpts, grpc.WaitForReady(true))

		retryErr := invoker(retryCtx, method, req, reply, cc, retryOpts...)
		if retryErr == nil {
			return nil
		}

		// If the retry only failed because our bounded wait elapsed (the caller's context is still alive), surface the original error — Unavailable is more representative of the real problem than a DeadlineExceeded from our timeout.
		if retryCtx.Err() != nil && ctx.Err() == nil {
			return err
		}

		return retryErr
	}
}
