package contracts

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

func TestRecoveryUnaryInterceptor_ConvertsPanicToInternal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		panicValue  any
		wantMessage string
	}{
		{
			name:        "string panic uses the value verbatim",
			panicValue:  "boom",
			wantMessage: "Panic occurred: boom",
		},
		{
			name:        "error panic uses Error()",
			panicValue:  errors.New("nil map write"),
			wantMessage: "Panic occurred: nil map write",
		},
		{
			name:        "non-error panic is formatted with %v",
			panicValue:  struct{ Code int }{42},
			wantMessage: "Panic occurred: {42}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := RecoveryUnaryInterceptor()
			handler := func(ctx context.Context, req any) (any, error) {
				panic(tt.panicValue)
			}

			resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, handler)
			if resp != nil {
				t.Errorf("expected nil response, got %v", resp)
			}
			if err == nil {
				t.Fatal("expected an error, got nil")
			}

			st, ok := grpcstatus.FromError(err)
			if !ok {
				t.Fatalf("expected a gRPC status error, got %T", err)
			}
			if st.Code() != grpccodes.Internal {
				t.Errorf("expected code %v, got %v", grpccodes.Internal, st.Code())
			}
			if !strings.Contains(st.Message(), tt.wantMessage) {
				t.Errorf("expected message to contain %q, got %q", tt.wantMessage, st.Message())
			}
			if !strings.Contains(st.Message(), "Stack trace:") {
				t.Errorf("expected a stack trace in the message, got %q", st.Message())
			}
			// The captured stack must be the panicking goroutine's, not an empty
			// placeholder, so it has to name the interceptor frame.
			if !strings.Contains(st.Message(), "RecoveryUnaryInterceptor") {
				t.Errorf("expected the stack trace to include the interceptor frame, got %q", st.Message())
			}
		})
	}
}

func TestRecoveryUnaryInterceptor_PassesThroughWithoutPanic(t *testing.T) {
	t.Parallel()
	handlerErr := grpcstatus.Error(grpccodes.NotFound, "missing")

	tests := []struct {
		name     string
		resp     any
		err      error
		wantResp any
		wantErr  error
	}{
		{
			name:     "success",
			resp:     "ok",
			err:      nil,
			wantResp: "ok",
		},
		{
			name:     "handler error is not swallowed",
			resp:     nil,
			err:      handlerErr,
			wantResp: nil,
			wantErr:  handlerErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := RecoveryUnaryInterceptor()
			handler := func(ctx context.Context, req any) (any, error) {
				return tt.resp, tt.err
			}

			resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, handler)
			if resp != tt.wantResp {
				t.Errorf("expected response %v, got %v", tt.wantResp, resp)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}
