package contracts

import (
	"context"
	"testing"

	"github.com/open-mrp/api/shared/appctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestIdempotencyKeyUnaryServerInterceptor_RestoresContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		md      metadata.MD
		useMD   bool
		wantKey string
	}{
		{
			name:    "key present",
			md:      metadata.Pairs(IdempotencyKeyHeader, "idem_abc"),
			useMD:   true,
			wantKey: "idem_abc",
		},
		{
			name:    "empty key value is ignored",
			md:      metadata.Pairs(IdempotencyKeyHeader, ""),
			useMD:   true,
			wantKey: "",
		},
		{
			name:    "header absent",
			md:      metadata.New(nil),
			useMD:   true,
			wantKey: "",
		},
		{
			name:    "no incoming metadata",
			useMD:   false,
			wantKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := IdempotencyKeyUnaryServerInterceptor()
			ctx := context.Background()
			if tt.useMD {
				ctx = metadata.NewIncomingContext(ctx, tt.md)
			}

			var handlerCtx context.Context
			handler := func(ctx context.Context, req any) (any, error) {
				handlerCtx = ctx
				return nil, nil
			}

			info := &grpc.UnaryServerInfo{FullMethod: "/orders.OrderService/CreateOrder"}
			if _, err := interceptor(ctx, nil, info, handler); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			// The handler name is recorded unconditionally: idempotency replay keys
			// off the method even when the client sent no key.
			handlerName, ok := appctx.GetHandler(handlerCtx)
			if !ok || handlerName != info.FullMethod {
				t.Errorf("expected handler %q, got %q (ok=%v)", info.FullMethod, handlerName, ok)
			}

			key, ok := appctx.GetIdempotencyKey(handlerCtx)
			if tt.wantKey == "" {
				if ok {
					t.Errorf("expected no idempotency key, got %q", key)
				}
				return
			}
			if !ok || key != tt.wantKey {
				t.Errorf("expected idempotency key %q, got %q (ok=%v)", tt.wantKey, key, ok)
			}
		})
	}
}
