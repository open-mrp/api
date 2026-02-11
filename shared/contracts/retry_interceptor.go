package contracts

import (
	"context"

	"google.golang.org/grpc"
)

// retryOnTransientUnaryClientInterceptor will retry a request if a transient error is
// encountered.
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

		retryErr := invoker(ctx, method, req, reply, cc, opts...)
		if retryErr != nil {
			return retryErr
		}

		return nil
	}
}
