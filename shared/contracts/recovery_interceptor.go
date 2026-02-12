package contracts

import (
	"context"
	"fmt"
	"runtime"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryUnaryInterceptor recovers from panics in gRPC unary handlers
// and converts them to internal server errors with details
func RecoveryUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if rec := recover(); rec != nil {
				// Capture stack trace
				stackTrace := make([]byte, 32768) // 32KB
				length := runtime.Stack(stackTrace, false)
				stackStr := string(stackTrace[:length])

				// Convert panic to error message
				var errMsg string
				if panicErr, ok := rec.(error); ok {
					errMsg = panicErr.Error()
				} else {
					errMsg = fmt.Sprintf("%v", rec)
				}

				// Include stack trace in error details
				details := fmt.Sprintf("Panic occurred: %s\n\nStack trace:\n%s", errMsg, stackStr)
				err = status.Error(codes.Internal, details)
			}
		}()

		return handler(ctx, req)
	}
}
