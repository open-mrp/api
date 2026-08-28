package contracts

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	apierror "github.com/open-mrp/api/shared/errors"
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

// TestRetryOnTransientUnaryClientInterceptor_ErrorClassification pins exactly
// which error classes cause the RPC to be executed a second time. Widening this
// set silently re-runs mutations, so every new entry must be a deliberate choice.
func TestRetryOnTransientUnaryClientInterceptor_ErrorClassification(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		err       error
		wantCalls int
	}{
		{
			name:      "bare Internal (e.g. a recovered panic) is retried",
			err:       grpcstatus.Error(grpccodes.Internal, "panic"),
			wantCalls: 2,
		},
		{
			name:      "encoded invariant violation is retried",
			err:       ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("identity missing")),
			wantCalls: 2,
		},
		{
			name:      "DeadlineExceeded is retried",
			err:       grpcstatus.Error(grpccodes.DeadlineExceeded, "too slow"),
			wantCalls: 2,
		},
		{
			name:      "encoded rate limit is retried",
			err:       ConvertAPIErrorToGRPC(apierror.NewRateLimitExceededError("slow down")),
			wantCalls: 2,
		},
		{
			name:      "encoded idempotency in progress is retried",
			err:       ConvertAPIErrorToGRPC(apierror.NewIdempotencyInProgressError("idem_1")),
			wantCalls: 2,
		},
		{
			name:      "Unavailable is retried",
			err:       grpcstatus.Error(grpccodes.Unavailable, "connection refused"),
			wantCalls: 2,
		},
		{
			name:      "encoded validation failure is not retried",
			err:       ConvertAPIErrorToGRPC(apierror.NewValidationError("bad input")),
			wantCalls: 1,
		},
		{
			name:      "encoded authorization failure is not retried",
			err:       ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("forbidden")),
			wantCalls: 1,
		},
		{
			name:      "encoded resource conflict is not retried",
			err:       ConvertAPIErrorToGRPC(apierror.NewResourceConflictError("already exists")),
			wantCalls: 1,
		},
		{
			name:      "encoded client closed request is not retried",
			err:       ConvertAPIErrorToGRPC(apierror.NewClientClosedRequestError("client gone")),
			wantCalls: 1,
		},
		{
			name:      "NotFound is not retried",
			err:       grpcstatus.Error(grpccodes.NotFound, "missing"),
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				calls++
				return tt.err
			}

			err := retryOnTransientUnaryClientInterceptor("svc")(
				context.Background(), "/svc/Method", nil, nil, nil, invoker)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if calls != tt.wantCalls {
				t.Fatalf("expected %d invoker calls, got %d", tt.wantCalls, calls)
			}
		})
	}
}

// The retry must carry its own bounded deadline: a WaitForReady attempt against
// a server that never comes back would otherwise queue for as long as the
// caller's context allows.
func TestRetryOnTransientUnaryClientInterceptor_RetryIsBounded(t *testing.T) {
	t.Parallel()
	calls := 0
	var firstHadDeadline, retryHadDeadline bool
	var retryRemaining time.Duration
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		calls++
		if calls == 1 {
			_, firstHadDeadline = ctx.Deadline()
			return grpcstatus.Error(grpccodes.Unavailable, "connection refused")
		}
		var deadline time.Time
		deadline, retryHadDeadline = ctx.Deadline()
		retryRemaining = time.Until(deadline)
		return nil
	}

	err := retryOnTransientUnaryClientInterceptor("svc")(
		context.Background(), "/svc/Method", nil, nil, nil, invoker)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 invoker calls, got %d", calls)
	}
	if firstHadDeadline {
		t.Error("expected the first attempt to inherit the caller's deadline unchanged")
	}
	if !retryHadDeadline {
		t.Fatal("expected the retry attempt to carry a deadline")
	}
	if retryRemaining <= 0 || retryRemaining > retryWaitForReadyTimeout {
		t.Errorf("expected the retry deadline within %v, got %v remaining", retryWaitForReadyTimeout, retryRemaining)
	}
}

// pausableCtx stands in for a caller context in the one state that is otherwise
// only reachable by waiting out the real retryWaitForReadyTimeout: the retry
// context has expired while the caller's context is still alive. expireChildren
// cancels the derived context and then reports the caller alive again.
type pausableCtx struct {
	done chan struct{}
	mu   sync.Mutex
	err  error
}

func (c *pausableCtx) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *pausableCtx) Done() <-chan struct{}       { return c.done }
func (c *pausableCtx) Value(any) any               { return nil }

func (c *pausableCtx) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *pausableCtx) expireChildren(child context.Context) {
	c.mu.Lock()
	c.err = context.DeadlineExceeded
	c.mu.Unlock()

	close(c.done)
	<-child.Done()

	c.mu.Lock()
	c.err = nil
	c.mu.Unlock()
}

// When only the interceptor's own bounded wait elapsed, the caller must see the
// error that actually describes the problem, not a DeadlineExceeded manufactured
// by the retry.
func TestRetryOnTransientUnaryClientInterceptor_BoundedWaitSurfacesOriginalError(t *testing.T) {
	t.Parallel()
	originalErr := grpcstatus.Error(grpccodes.Unavailable, "connection refused")
	retryErr := grpcstatus.Error(grpccodes.DeadlineExceeded, "context deadline exceeded")

	caller := &pausableCtx{done: make(chan struct{})}
	calls := 0
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		calls++
		if calls == 1 {
			return originalErr
		}
		caller.expireChildren(ctx)
		return retryErr
	}

	err := retryOnTransientUnaryClientInterceptor("svc")(
		caller, "/svc/Method", nil, nil, nil, invoker)
	if calls != 2 {
		t.Fatalf("expected 2 invoker calls, got %d", calls)
	}
	if !errors.Is(err, originalErr) {
		t.Errorf("expected the original %v, got %v", originalErr, err)
	}
}
