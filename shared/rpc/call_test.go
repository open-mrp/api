package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testResponse struct {
	ID   string
	Name string
}

func testTracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("rpc-test")
}

// setHeader mimics the gRPC invoker writing response headers into the metadata
// address supplied by grpc.Header.
func setHeader(opts []grpc.CallOption, md metadata.MD) {
	for _, o := range opts {
		if h, ok := o.(grpc.HeaderCallOption); ok && h.HeaderAddr != nil {
			*h.HeaderAddr = md
		}
	}
}

// --- Deadline ---

func TestCallRPC_Deadline(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		opts []Option
		want time.Duration
	}{
		{
			name: "no options uses the default timeout",
			want: DefaultTimeout,
		},
		{
			name: "explicit timeout overrides the default",
			opts: []Option{WithTimeout(2 * time.Second)},
			want: 2 * time.Second,
		},
		{
			name: "longer explicit timeout overrides the default",
			opts: []Option{WithTimeout(30 * time.Second)},
			want: 30 * time.Second,
		},
		{
			name: "zero timeout falls back to the default",
			opts: []Option{WithTimeout(0)},
			want: DefaultTimeout,
		},
		{
			name: "negative timeout falls back to the default",
			opts: []Option{WithTimeout(-5 * time.Second)},
			want: DefaultTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var deadline time.Time
			var hasDeadline bool
			call := func(ctx context.Context, opts ...grpc.CallOption) (testResponse, error) {
				deadline, hasDeadline = ctx.Deadline()
				return testResponse{}, nil
			}

			_, apiErr := CallRPC(context.Background(), testTracer(), "span", "svc", call, tt.opts...)
			if apiErr != nil {
				t.Fatalf("unexpected error: %v", apiErr)
			}
			if !hasDeadline {
				t.Fatal("expected the RPC context to carry a deadline")
			}

			remaining := time.Until(deadline)
			if remaining > tt.want || remaining < tt.want-time.Second {
				t.Errorf("expected ~%v remaining on the deadline, got %v", tt.want, remaining)
			}
		})
	}
}

func TestCallRPC_DoesNotExtendShorterParentDeadline(t *testing.T) {
	t.Parallel()
	const parentTimeout = time.Second

	ctx, cancel := context.WithTimeout(context.Background(), parentTimeout)
	defer cancel()

	var deadline time.Time
	var hasDeadline bool
	call := func(ctx context.Context, opts ...grpc.CallOption) (testResponse, error) {
		deadline, hasDeadline = ctx.Deadline()
		return testResponse{}, nil
	}

	_, apiErr := CallRPC(ctx, testTracer(), "span", "svc", call, WithTimeout(time.Minute))
	if apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if !hasDeadline {
		t.Fatal("expected the RPC context to carry a deadline")
	}
	if remaining := time.Until(deadline); remaining > parentTimeout {
		t.Errorf("expected the parent deadline (%v) to be respected, got %v remaining", parentTimeout, remaining)
	}
}

// --- Error mapping ---

func TestCallRPC_ErrorMapping(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		ctx        func(t *testing.T) context.Context
		callErr    error
		wantCode   apierror.ErrorCode
		wantStatus int
	}{
		{
			name:       "not found maps to resource not found",
			ctx:        func(t *testing.T) context.Context { return context.Background() },
			callErr:    status.Error(codes.NotFound, "order not found"),
			wantCode:   apierror.ErrorCodeResourceNotFound,
			wantStatus: 404,
		},
		{
			name: "canceled parent context maps to client closed request",
			ctx: func(t *testing.T) context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			callErr:    status.Error(codes.Canceled, "context canceled"),
			wantCode:   apierror.ErrorCodeClientClosedRequest,
			wantStatus: 499,
		},
		{
			name:       "expired rpc deadline maps to request timeout",
			ctx:        func(t *testing.T) context.Context { return context.Background() },
			callErr:    status.Error(codes.DeadlineExceeded, "context deadline exceeded"),
			wantCode:   apierror.ErrorCodeRequestTimeout,
			wantStatus: 504,
		},
		{
			name:       "unrecognised code maps to internal error",
			ctx:        func(t *testing.T) context.Context { return context.Background() },
			callErr:    status.Error(codes.Unavailable, "connection refused"),
			wantCode:   apierror.ErrorCodeInternalError,
			wantStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			call := func(ctx context.Context, opts ...grpc.CallOption) (testResponse, error) {
				return testResponse{}, tt.callErr
			}

			_, apiErr := CallRPC(tt.ctx(t), testTracer(), "span", "core-service", call)
			if apiErr == nil {
				t.Fatal("expected an APIError")
			}
			if apiErr.Code != tt.wantCode {
				t.Errorf("expected code %q, got %q", tt.wantCode, apiErr.Code)
			}
			if got := apierror.GetHTTPStatusCode(apiErr.Code); got != tt.wantStatus {
				t.Errorf("expected HTTP status %d, got %d", tt.wantStatus, got)
			}
		})
	}
}

func TestCallRPC_ReturnsZeroValueOnError(t *testing.T) {
	t.Parallel()
	call := func(ctx context.Context, opts ...grpc.CallOption) (testResponse, error) {
		return testResponse{ID: "or_123", Name: "half populated"}, status.Error(codes.Internal, "boom")
	}

	resp, apiErr := CallRPC(context.Background(), testTracer(), "span", "core-service", call)
	if apiErr == nil {
		t.Fatal("expected an APIError")
	}
	if resp != (testResponse{}) {
		t.Errorf("expected the zero value of T on error, got %+v", resp)
	}
}

func TestCallRPC_PreservesEncodedAPIError(t *testing.T) {
	t.Parallel()
	original := apierror.NewValidationErrorWithParam("Quantity must be positive.", "quantity")
	call := func(ctx context.Context, opts ...grpc.CallOption) (testResponse, error) {
		return testResponse{}, contracts.ConvertAPIErrorToGRPC(original)
	}

	_, apiErr := CallRPC(context.Background(), testTracer(), "span", "core-service", call)
	if apiErr == nil {
		t.Fatal("expected an APIError")
	}
	if apiErr.Code != original.Code {
		t.Errorf("expected code %q, got %q", original.Code, apiErr.Code)
	}
	if apiErr.Param != "quantity" {
		t.Errorf("expected param %q, got %q", "quantity", apiErr.Param)
	}
}

// --- Replay detection ---

func TestCallRPC_OnReplayed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		header   metadata.MD
		wantFire bool
	}{
		{
			name:     "replayed header fires the callback",
			header:   metadata.Pairs(contracts.IdempotentReplayedHeader, contracts.IdempotentReplayedHeaderValue),
			wantFire: true,
		},
		{
			name:     "header with another value does not fire the callback",
			header:   metadata.Pairs(contracts.IdempotentReplayedHeader, "false"),
			wantFire: false,
		},
		{
			name:     "absent header does not fire the callback",
			header:   metadata.New(nil),
			wantFire: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			call := func(ctx context.Context, opts ...grpc.CallOption) (testResponse, error) {
				setHeader(opts, tt.header)
				return testResponse{ID: "or_1"}, nil
			}

			fired := false
			_, apiErr := CallRPC(context.Background(), testTracer(), "span", "svc", call, WithOnReplayed(func() { fired = true }))
			if apiErr != nil {
				t.Fatalf("unexpected error: %v", apiErr)
			}
			if fired != tt.wantFire {
				t.Errorf("expected onReplayed fired=%v, got %v", tt.wantFire, fired)
			}
		})
	}
}

func TestCallRPC_NilOnReplayedCallbackIsSkipped(t *testing.T) {
	t.Parallel()
	call := func(ctx context.Context, opts ...grpc.CallOption) (testResponse, error) {
		setHeader(opts, metadata.Pairs(contracts.IdempotentReplayedHeader, contracts.IdempotentReplayedHeaderValue))
		return testResponse{}, nil
	}

	if _, apiErr := CallRPC(context.Background(), testTracer(), "span", "svc", call); apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
}
