package contracts

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// interceptorName returns the fully-qualified name of the function that produced
// the interceptor closure, which is the only way to identify an entry in the
// default chain.
func interceptorName(i grpc.UnaryServerInterceptor) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

// chainUnaryInterceptors composes interceptors the way grpc.ChainUnaryInterceptor
// does: the first entry is the outermost wrapper, the last runs closest to the
// handler.
func chainUnaryInterceptors(interceptors []grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		var next func(i int) grpc.UnaryHandler
		next = func(i int) grpc.UnaryHandler {
			if i == len(interceptors) {
				return handler
			}
			return func(ctx context.Context, req any) (any, error) {
				return interceptors[i](ctx, req, info, next(i+1))
			}
		}
		return next(0)(ctx, req)
	}
}

func TestGRPCServerConfig_WithDefaults_InterceptorChain(t *testing.T) {
	t.Parallel()
	wantInterceptors := []string{
		"tracing.UnarySpanRenamer",
		"contracts.RecoveryUnaryInterceptor",
		"contracts.IdentityUnaryServerInterceptor",
		"contracts.IdempotencyKeyUnaryServerInterceptor",
		"contracts.RequestIDUnaryServerInterceptor",
		"contracts.ClientIPUnaryServerInterceptor",
		"logging.CanonicalLogInterceptor",
	}

	var nilConfig *GRPCServerConfig
	config := nilConfig.WithDefaults(nil)

	if len(config.UnaryInterceptors) != len(wantInterceptors) {
		t.Fatalf("expected %d default interceptors, got %d", len(wantInterceptors), len(config.UnaryInterceptors))
	}

	present := make(map[string]bool, len(config.UnaryInterceptors))
	for _, i := range config.UnaryInterceptors {
		present[interceptorName(i)] = true
	}
	for _, want := range wantInterceptors {
		found := false
		for name := range present {
			if strings.Contains(name, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected the default chain to contain %s, got %v", want, present)
		}
	}
}

func TestGRPCServerConfig_WithDefaults_CustomInterceptorsReplaceDefaults(t *testing.T) {
	t.Parallel()
	called := false
	custom := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		called = true
		return handler(ctx, req)
	}

	config := (&GRPCServerConfig{UnaryInterceptors: []grpc.UnaryServerInterceptor{custom}}).WithDefaults(nil)

	if len(config.UnaryInterceptors) != 1 {
		t.Fatalf("expected a custom slice to replace all defaults, got %d interceptors", len(config.UnaryInterceptors))
	}

	handler := func(ctx context.Context, req any) (any, error) { return nil, nil }
	if _, err := config.UnaryInterceptors[0](context.Background(), nil, nil, handler); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Error("expected the custom interceptor to be the one retained")
	}
}

func TestGRPCServerConfig_WithDefaults_KeepalivePartialOverride(t *testing.T) {
	t.Parallel()
	config := (&GRPCServerConfig{
		KeepaliveParams:   keepalive.ServerParameters{Time: 7 * time.Second},
		EnforcementPolicy: keepalive.EnforcementPolicy{PermitWithoutStream: false},
	}).WithDefaults(nil)

	if config.KeepaliveParams.Time != 7*time.Second {
		t.Errorf("expected the supplied keepalive time to survive, got %v", config.KeepaliveParams.Time)
	}
	if config.KeepaliveParams.Timeout != defaultServerKeepaliveTimeout {
		t.Errorf("expected default keepalive timeout %v, got %v", defaultServerKeepaliveTimeout, config.KeepaliveParams.Timeout)
	}
	// PermitWithoutStream is forced on regardless of what the caller supplied.
	if !config.EnforcementPolicy.PermitWithoutStream {
		t.Error("expected PermitWithoutStream to be forced on")
	}
}

func TestNewGRPCServer_InvalidKeepalive(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		config *GRPCServerConfig
	}{
		{
			name:   "negative keepalive time",
			config: &GRPCServerConfig{KeepaliveParams: keepalive.ServerParameters{Time: -1}},
		},
		{
			name:   "negative keepalive timeout",
			config: &GRPCServerConfig{KeepaliveParams: keepalive.ServerParameters{Timeout: -1}},
		},
		{
			name:   "negative enforcement min time",
			config: &GRPCServerConfig{EnforcementPolicy: keepalive.EnforcementPolicy{MinTime: -1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, err := NewGRPCServer("svc", nil, tt.config)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if srv != nil {
				t.Errorf("expected nil server, got %+v", srv)
			}
		})
	}
}

// The default chain must extract the identity before the canonical log line is
// built, otherwise every log line across the platform loses its actor fields.
func TestGRPCServerConfig_DefaultChain_LogsIdentity(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	config := (&GRPCServerConfig{}).WithDefaults(logger)

	md := metadata.New(nil)
	SetIdentityInMetadata(md, identityForLogging())
	md.Set(RequestIDHeader, "req_123")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(ctx context.Context, req any) (any, error) { return "ok", nil }
	info := &grpc.UnaryServerInfo{FullMethod: "/orders.OrderService/CreateOrder"}

	if _, err := chainUnaryInterceptors(config.UnaryInterceptors)(ctx, nil, info, handler); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var line map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &line); err != nil {
		t.Fatalf("expected a canonical log line, got %q (%v)", buf.String(), err)
	}
	if line["user_id"] != "usr_1" {
		t.Errorf("expected user_id usr_1 in the log line, got %v", line["user_id"])
	}
	if line["request_id"] != "req_123" {
		t.Errorf("expected request_id req_123 in the log line, got %v", line["request_id"])
	}
	if line["grpc_method"] != info.FullMethod {
		t.Errorf("expected grpc_method %q, got %v", info.FullMethod, line["grpc_method"])
	}
}

func identityForLogging() *types.Identity {
	return &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: "acc_1"},
		Actor:  &types.IdentityActor{ID: "usr_1", RelationType: types.IdentityRelationTypeInternal},
	}
}

func TestGRPCServer_Serve_ListenFailure(t *testing.T) {
	t.Parallel()
	srv, err := NewGRPCServer("svc", nil, nil)
	if err != nil {
		t.Fatalf("expected a server, got %v", err)
	}

	err = srv.Serve(context.Background(), -1)
	if err == nil {
		t.Fatal("expected a listen error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to listen on port -1") {
		t.Errorf("expected the wrapped listen error, got %v", err)
	}
}

func TestGRPCServer_Serve_StopsOnContextCancellation(t *testing.T) {
	t.Parallel()
	srv, err := NewGRPCServer("svc", nil, nil)
	if err != nil {
		t.Fatalf("expected a server, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Port 0 binds an ephemeral local port; the cancelled context means Serve
	// shuts down without accepting a connection.
	if err := srv.Serve(ctx, 0); err != nil {
		t.Fatalf("expected a clean shutdown, got %v", err)
	}
}
