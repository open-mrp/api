package rpc

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func testIdentity(actorID string) *types.Identity {
	accountID := "acct_1"
	return &types.Identity{
		Type:        types.IdentityActorTypeUser,
		Actor:       &types.IdentityActor{ID: actorID, AccountID: &accountID},
		Target:      &types.IdentityTarget{AccountID: accountID},
		AccountMode: constants.AccountModeProduction,
	}
}

func outgoingMD(t *testing.T, ctx context.Context) metadata.MD {
	t.Helper()
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("expected outgoing metadata on the prepared context")
	}
	return md
}

func assertMDValue(t *testing.T, md metadata.MD, key, expected string) {
	t.Helper()
	values := md.Get(key)
	if len(values) != 1 {
		t.Fatalf("expected exactly one %q value, got %v", key, values)
	}
	if values[0] != expected {
		t.Errorf("metadata %q: expected %q, got %q", key, expected, values[0])
	}
}

func assertMDAbsent(t *testing.T, md metadata.MD, key string) {
	t.Helper()
	if values := md.Get(key); len(values) != 0 {
		t.Errorf("expected no %q in metadata, got %v", key, values)
	}
}

// --- Metadata assembly ---

func TestPrepareServiceCallCtx_ForwardsContextValues(t *testing.T) {
	t.Parallel()
	ctx := appctx.WithIdentity(context.Background(), testIdentity("usr_123"))
	ctx = appctx.WithIdempotencyKey(ctx, "key-abc")
	ctx = appctx.WithRequestID(ctx, "req_1")
	ctx = appctx.WithPropagatedClientIP(ctx, "203.0.113.7")

	md := outgoingMD(t, PrepareServiceCallCtx(ctx))

	assertMDValue(t, md, contracts.IdempotencyKeyHeader, "key-abc")
	assertMDValue(t, md, contracts.RequestIDHeader, "req_1")
	assertMDValue(t, md, contracts.ClientIPHeader, "203.0.113.7")

	identity, apiErr := contracts.GetIdentityFromMetadata(md)
	if apiErr != nil {
		t.Fatalf("unexpected error reading identity: %v", apiErr)
	}
	if identity.Actor == nil || identity.Actor.ID != "usr_123" {
		t.Errorf("expected actor usr_123, got %+v", identity.Actor)
	}
}

func TestPrepareServiceCallCtx_OmitsMissingValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ctx  func() context.Context
	}{
		{
			name: "nothing in context",
			ctx:  context.Background,
		},
		{
			name: "nil identity and empty strings",
			ctx: func() context.Context {
				ctx := appctx.WithIdentity(context.Background(), nil)
				ctx = appctx.WithIdempotencyKey(ctx, "")
				ctx = appctx.WithRequestID(ctx, "")
				return appctx.WithPropagatedClientIP(ctx, "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			md := outgoingMD(t, PrepareServiceCallCtx(tt.ctx()))

			for _, key := range []string{
				contracts.IdentityHeader,
				contracts.IdempotencyKeyHeader,
				contracts.RequestIDHeader,
				contracts.ClientIPHeader,
			} {
				assertMDAbsent(t, md, key)
			}
		})
	}
}

func TestPrepareServiceCallCtx_IdempotencyKeyOverride(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		contextKey string
		opts       []ServiceCallOption
		want       string
		wantAbsent bool
	}{
		{
			name:       "override beats the context key",
			contextKey: "key-from-context",
			opts:       []ServiceCallOption{WithIdempotencyKeyOverride("key-override")},
			want:       "key-override",
		},
		{
			name:       "override applies when the context has no key",
			contextKey: "",
			opts:       []ServiceCallOption{WithIdempotencyKeyOverride("key-override")},
			want:       "key-override",
		},
		{
			name:       "empty override falls back to the context key",
			contextKey: "key-from-context",
			opts:       []ServiceCallOption{WithIdempotencyKeyOverride("")},
			want:       "key-from-context",
		},
		{
			name:       "no key anywhere leaves the header unset",
			contextKey: "",
			wantAbsent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			if tt.contextKey != "" {
				ctx = appctx.WithIdempotencyKey(ctx, tt.contextKey)
			}

			md := outgoingMD(t, PrepareServiceCallCtx(ctx, tt.opts...))

			if tt.wantAbsent {
				assertMDAbsent(t, md, contracts.IdempotencyKeyHeader)
				return
			}
			assertMDValue(t, md, contracts.IdempotencyKeyHeader, tt.want)
		})
	}
}

func TestPrepareServiceCallCtx_PreservesExistingOutgoingMetadata(t *testing.T) {
	t.Parallel()
	parent := metadata.Pairs("x-custom", "kept")
	ctx := metadata.NewOutgoingContext(context.Background(), parent)
	ctx = appctx.WithRequestID(ctx, "req_1")

	md := outgoingMD(t, PrepareServiceCallCtx(ctx))

	assertMDValue(t, md, "x-custom", "kept")
	assertMDValue(t, md, contracts.RequestIDHeader, "req_1")
}

func TestPrepareServiceCallCtx_DoesNotMutateParentMetadata(t *testing.T) {
	t.Parallel()
	parent := metadata.New(nil)
	contracts.SetIdentityInMetadata(parent, testIdentity("usr_parent"))
	parent.Set(contracts.IdempotencyKeyHeader, "key-parent")

	ctx := metadata.NewOutgoingContext(context.Background(), parent)
	ctx = appctx.WithIdentity(ctx, testIdentity("usr_child"))
	ctx = appctx.WithIdempotencyKey(ctx, "key-child")

	child := outgoingMD(t, PrepareServiceCallCtx(ctx))
	assertMDValue(t, child, contracts.IdempotencyKeyHeader, "key-child")

	assertMDValue(t, parent, contracts.IdempotencyKeyHeader, "key-parent")
	parentIdentity, apiErr := contracts.GetIdentityFromMetadata(parent)
	if apiErr != nil {
		t.Fatalf("unexpected error reading parent identity: %v", apiErr)
	}
	if parentIdentity.Actor == nil || parentIdentity.Actor.ID != "usr_parent" {
		t.Errorf("expected the parent metadata to keep usr_parent, got %+v", parentIdentity.Actor)
	}
}

func TestPrepareRPCCtx_AppliesOptionsWithoutExistingMetadata(t *testing.T) {
	t.Parallel()
	ctx := PrepareRPCCtx(context.Background(), WithMetadata("x-one", "1"), WithMetadata("x-many", "a", "b"))

	md := outgoingMD(t, ctx)
	assertMDValue(t, md, "x-one", "1")
	if values := md.Get("x-many"); len(values) != 2 || values[0] != "a" || values[1] != "b" {
		t.Errorf("expected both x-many values, got %v", values)
	}
}

func TestWithIdentity_NoIdentityInContext(t *testing.T) {
	t.Parallel()
	md := outgoingMD(t, PrepareRPCCtx(context.Background(), WithIdentity(context.Background())))
	assertMDAbsent(t, md, contracts.IdentityHeader)
}

// --- One hop across the metadata boundary ---

// hop moves the prepared outgoing metadata onto an incoming context and runs the
// server interceptor chain a real service is built with, so a header with no
// reader on the far side fails the test.
func hop(t *testing.T, ctx context.Context, method string, handler grpc.UnaryHandler) {
	t.Helper()
	md := outgoingMD(t, ctx)
	serverCtx := metadata.NewIncomingContext(context.Background(), md)

	info := &grpc.UnaryServerInfo{FullMethod: method}
	interceptors := []grpc.UnaryServerInterceptor{
		contracts.IdentityUnaryServerInterceptor(),
		contracts.IdempotencyKeyUnaryServerInterceptor(),
		contracts.RequestIDUnaryServerInterceptor(),
		contracts.ClientIPUnaryServerInterceptor(),
	}

	next := handler
	for i := len(interceptors) - 1; i >= 0; i-- {
		interceptor, inner := interceptors[i], next
		next = func(ctx context.Context, req any) (any, error) {
			return interceptor(ctx, req, info, inner)
		}
	}

	if _, err := next(serverCtx, nil); err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
}

func TestPrepareServiceCallCtx_ValuesReadableDownstream(t *testing.T) {
	t.Parallel()
	const method = "/core.v1.CoreService/CreateTransaction"

	ctx := appctx.WithIdentity(context.Background(), testIdentity("usr_123"))
	ctx = appctx.WithIdempotencyKey(ctx, "key-abc")
	ctx = appctx.WithRequestID(ctx, "req_1")
	ctx = appctx.WithPropagatedClientIP(ctx, "203.0.113.7")

	hop(t, PrepareServiceCallCtx(ctx), method, func(downstream context.Context, req any) (any, error) {
		identity, ok := appctx.GetIdentityFromContext(downstream)
		if !ok || identity == nil || identity.Actor == nil || identity.Actor.ID != "usr_123" {
			t.Errorf("expected identity usr_123 downstream, got %+v", identity)
		}
		if key, ok := appctx.GetIdempotencyKey(downstream); !ok || key != "key-abc" {
			t.Errorf("expected idempotency key %q downstream, got %q", "key-abc", key)
		}
		if requestID, ok := appctx.GetRequestID(downstream); !ok || requestID != "req_1" {
			t.Errorf("expected request ID %q downstream, got %q", "req_1", requestID)
		}
		if ip, ok := appctx.GetPropagatedClientIP(downstream); !ok || ip != "203.0.113.7" {
			t.Errorf("expected client IP %q downstream, got %q", "203.0.113.7", ip)
		}
		if handler, ok := appctx.GetHandler(downstream); !ok || handler != method {
			t.Errorf("expected handler %q downstream, got %q", method, handler)
		}
		return nil, nil
	})
}

func TestPrepareServiceCallCtx_OverriddenKeyReadableDownstream(t *testing.T) {
	t.Parallel()
	ctx := appctx.WithIdempotencyKey(context.Background(), "key-original")

	prepared := PrepareServiceCallCtx(ctx, WithIdempotencyKeyOverride("key-original:retry"))
	hop(t, prepared, "/core.v1.CoreService/CreateTransaction", func(downstream context.Context, req any) (any, error) {
		if key, ok := appctx.GetIdempotencyKey(downstream); !ok || key != "key-original:retry" {
			t.Errorf("expected the overridden key downstream, got %q", key)
		}
		return nil, nil
	})
}
