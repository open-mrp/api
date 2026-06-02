package contracts

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

// hasWaitForReady reports whether the call options request WaitForReady, i.e. a
// FailFastCallOption with FailFast disabled (grpc.WaitForReady(true)).
func hasWaitForReady(opts []grpc.CallOption) bool {
	for _, opt := range opts {
		if ff, ok := opt.(grpc.FailFastCallOption); ok && !ff.FailFast {
			return true
		}
	}
	return false
}

func TestRetryOnTransientUnaryClientInterceptor(t *testing.T) {
	transientErr := grpcstatus.Error(grpccodes.Unavailable, "boom")
	// NotFound maps to a non-transient resource-not-found error; a bare InvalidArgument
	// status (no API-error marker) would fall through to the transient Internal default.
	nonTransientErr := grpcstatus.Error(grpccodes.NotFound, "missing")

	t.Run("succeeds on first attempt without retrying", func(t *testing.T) {
		calls := 0
		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			calls++
			return nil
		}

		err := retryOnTransientUnaryClientInterceptor("svc")(
			context.Background(), "/svc/Method", nil, nil, nil, invoker)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if calls != 1 {
			t.Fatalf("expected 1 invoker call, got %d", calls)
		}
	})

	t.Run("does not retry non-transient errors", func(t *testing.T) {
		calls := 0
		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			calls++
			return nonTransientErr
		}

		err := retryOnTransientUnaryClientInterceptor("svc")(
			context.Background(), "/svc/Method", nil, nil, nil, invoker)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if calls != 1 {
			t.Fatalf("expected 1 invoker call (no retry), got %d", calls)
		}
	})

	t.Run("retries transient error with WaitForReady and succeeds", func(t *testing.T) {
		calls := 0
		var retryHadWaitForReady bool
		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			calls++
			if calls == 1 {
				if hasWaitForReady(opts) {
					t.Error("first attempt should not set WaitForReady (fast-fail semantics)")
				}
				return transientErr
			}
			retryHadWaitForReady = hasWaitForReady(opts)
			return nil
		}

		err := retryOnTransientUnaryClientInterceptor("svc")(
			context.Background(), "/svc/Method", nil, nil, nil, invoker)
		if err != nil {
			t.Fatalf("expected nil error after successful retry, got %v", err)
		}
		if calls != 2 {
			t.Fatalf("expected 2 invoker calls, got %d", calls)
		}
		if !retryHadWaitForReady {
			t.Error("retry attempt should set WaitForReady(true)")
		}
	})

	t.Run("returns retry error when both attempts fail with a live context", func(t *testing.T) {
		calls := 0
		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			calls++
			return transientErr
		}

		err := retryOnTransientUnaryClientInterceptor("svc")(
			context.Background(), "/svc/Method", nil, nil, nil, invoker)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if calls != 2 {
			t.Fatalf("expected 2 invoker calls, got %d", calls)
		}
	})

	t.Run("does not retry when caller context is already canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		calls := 0
		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			calls++
			return grpcstatus.Error(grpccodes.Canceled, "canceled")
		}

		_ = retryOnTransientUnaryClientInterceptor("svc")(
			ctx, "/svc/Method", nil, nil, nil, invoker)
		// A canceled context yields a non-transient client-cancellation error, so the
		// interceptor must not retry.
		if calls != 1 {
			t.Fatalf("expected 1 invoker call for canceled context, got %d", calls)
		}
	})
}
