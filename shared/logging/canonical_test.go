package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --- buildCanonicalAttrs tests ---

func TestBuildCanonicalAttrs_BaseFields(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	attrs := buildCanonicalAttrs(ctx, "/auth.AuthService/Login", nil, 5*time.Millisecond)

	m := attrsToMap(attrs)

	assertAttr(t, m, "type", "canonical-log-line")
	assertAttr(t, m, "grpc_method", "/auth.AuthService/Login")
	assertAttr(t, m, "grpc_code", "OK")

	if _, ok := m["duration_ms"]; !ok {
		t.Error("expected duration_ms attribute")
	}
}

func TestBuildCanonicalAttrs_WithError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := status.Error(codes.NotFound, "not found")
	attrs := buildCanonicalAttrs(ctx, "/svc/Method", err, time.Millisecond)

	m := attrsToMap(attrs)

	assertAttr(t, m, "grpc_code", "NotFound")
	if _, ok := m["error"]; !ok {
		t.Error("expected error attribute when err is non-nil")
	}
}

func TestBuildCanonicalAttrs_NoErrorField_WhenNil(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	attrs := buildCanonicalAttrs(ctx, "/svc/Method", nil, time.Millisecond)

	m := attrsToMap(attrs)
	if _, ok := m["error"]; ok {
		t.Error("expected no error attribute when err is nil")
	}
}

func TestBuildCanonicalAttrs_WithRequestID(t *testing.T) {
	t.Parallel()
	ctx := appctx.WithRequestID(context.Background(), "req-abc-123")
	attrs := buildCanonicalAttrs(ctx, "/svc/Method", nil, time.Millisecond)

	m := attrsToMap(attrs)
	assertAttr(t, m, "request_id", "req-abc-123")
}

func TestBuildCanonicalAttrs_NoRequestID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	attrs := buildCanonicalAttrs(ctx, "/svc/Method", nil, time.Millisecond)

	m := attrsToMap(attrs)
	if _, ok := m["request_id"]; ok {
		t.Error("expected no request_id attribute when not in context")
	}
}

func TestBuildCanonicalAttrs_DurationIsPositive(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	duration := 1234 * time.Microsecond
	attrs := buildCanonicalAttrs(ctx, "/svc/Method", nil, duration)

	m := attrsToMap(attrs)
	val, ok := m["duration_ms"]
	if !ok {
		t.Fatal("expected duration_ms attribute")
	}
	if val.Float64() < 1.0 || val.Float64() > 2.0 {
		t.Errorf("expected duration_ms ~1.234, got %f", val.Float64())
	}
}

func TestBuildCanonicalAttrs_WithRecordingSpan(t *testing.T) {
	t.Parallel()
	tracer := sdktrace.NewTracerProvider().Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	attrs := buildCanonicalAttrs(ctx, "/svc/Method", nil, time.Millisecond)

	m := attrsToMap(attrs)
	assertAttr(t, m, "trace_id", span.SpanContext().TraceID().String())
	assertAttr(t, m, "span_id", span.SpanContext().SpanID().String())
}

func TestBuildCanonicalAttrs_NoTraceIDs_WithoutSpan(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	attrs := buildCanonicalAttrs(ctx, "/svc/Method", nil, time.Millisecond)

	m := attrsToMap(attrs)
	if _, ok := m["trace_id"]; ok {
		t.Error("expected no trace_id attribute when no span in context")
	}
	if _, ok := m["span_id"]; ok {
		t.Error("expected no span_id attribute when no span in context")
	}
}

func TestBuildCanonicalAttrs_NoTraceIDs_NonRecordingSpan(t *testing.T) {
	t.Parallel()
	tracer := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.NeverSample())).Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	attrs := buildCanonicalAttrs(ctx, "/svc/Method", nil, time.Millisecond)

	m := attrsToMap(attrs)
	if _, ok := m["trace_id"]; ok {
		t.Error("expected no trace_id attribute when span is not recording")
	}
	if _, ok := m["span_id"]; ok {
		t.Error("expected no span_id attribute when span is not recording")
	}
}

// --- extractIdentityAttrs tests ---

func TestExtractIdentityAttrs_UserIdentity(t *testing.T) {
	t.Parallel()
	identity := &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			ID: "usr_123",
		},
		Target:      &types.IdentityTarget{AccountID: "acct_456"},
		AccountMode: constants.AccountModeProduction,
	}

	attrs := extractIdentityAttrs(identity)
	m := attrsToMap(attrs)

	assertAttr(t, m, "auth_type", "user")
	assertAttr(t, m, "user_id", "usr_123")
	assertAttr(t, m, "target_account_id", "acct_456")
	assertAttr(t, m, "account_mode", string(constants.AccountModeProduction))
}

func TestExtractIdentityAttrs_APIKeyIdentity(t *testing.T) {
	t.Parallel()
	identity := &types.Identity{
		Type: types.IdentityActorTypeAPIKey,
		Actor: &types.IdentityActor{
			ID: "apke_789",
		},
	}

	attrs := extractIdentityAttrs(identity)
	m := attrsToMap(attrs)

	assertAttr(t, m, "auth_type", "api_key")
	assertAttr(t, m, "key_id", "apke_789")

	if _, ok := m["user_id"]; ok {
		t.Error("expected no user_id for API key identity")
	}
}

func TestExtractIdentityAttrs_AgentIdentity(t *testing.T) {
	t.Parallel()
	identity := &types.Identity{
		Type: types.IdentityActorTypeAgent,
		Actor: &types.IdentityActor{
			ID: "agnt_456",
		},
	}

	attrs := extractIdentityAttrs(identity)
	m := attrsToMap(attrs)

	assertAttr(t, m, "auth_type", "agent")
	assertAttr(t, m, "agent_id", "agnt_456")

	if _, ok := m["user_id"]; ok {
		t.Error("expected no user_id for agent identity")
	}
	if _, ok := m["key_id"]; ok {
		t.Error("expected no key_id for agent identity")
	}
}

func TestExtractIdentityAttrs_NilActor(t *testing.T) {
	t.Parallel()
	identity := &types.Identity{
		Type:  types.IdentityActorTypeUnauthenticated,
		Actor: nil,
	}

	attrs := extractIdentityAttrs(identity)
	m := attrsToMap(attrs)

	assertAttr(t, m, "auth_type", "unauthenticated")

	if _, ok := m["user_id"]; ok {
		t.Error("expected no user_id when actor is nil")
	}
	if _, ok := m["key_id"]; ok {
		t.Error("expected no key_id when actor is nil")
	}
}

func TestExtractIdentityAttrs_NilTarget(t *testing.T) {
	t.Parallel()
	identity := &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Actor:  &types.IdentityActor{ID: "usr_1"},
		Target: nil,
	}

	attrs := extractIdentityAttrs(identity)
	m := attrsToMap(attrs)

	if _, ok := m["target_account_id"]; ok {
		t.Error("expected no target_account_id when nil")
	}
}

func TestExtractIdentityAttrs_EmptyAccountMode(t *testing.T) {
	t.Parallel()
	identity := &types.Identity{
		Type:        types.IdentityActorTypeUser,
		Actor:       &types.IdentityActor{ID: "usr_1"},
		AccountMode: "",
	}

	attrs := extractIdentityAttrs(identity)
	m := attrsToMap(attrs)

	if _, ok := m["account_mode"]; ok {
		t.Error("expected no account_mode when empty")
	}
}

func TestExtractIdentityAttrs_FullIdentity(t *testing.T) {
	t.Parallel()
	identity := &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			ID: "usr_full",
		},
		Target:      &types.IdentityTarget{AccountID: "acct_full"},
		AccountMode: constants.AccountModeProduction,
	}

	attrs := extractIdentityAttrs(identity)
	m := attrsToMap(attrs)

	expected := []string{"auth_type", "user_id", "target_account_id", "account_mode"}
	for _, key := range expected {
		if _, ok := m[key]; !ok {
			t.Errorf("expected attribute %q to be present", key)
		}
	}
}

// --- buildCanonicalAttrs with identity in context ---

func TestBuildCanonicalAttrs_WithIdentityInContext(t *testing.T) {
	t.Parallel()
	identity := &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			ID: "usr_ctx",
		},
	}
	ctx := appctx.WithIdentity(context.Background(), identity)
	attrs := buildCanonicalAttrs(ctx, "/svc/Method", nil, time.Millisecond)

	m := attrsToMap(attrs)
	assertAttr(t, m, "auth_type", "user")
	assertAttr(t, m, "user_id", "usr_ctx")
}

func TestBuildCanonicalAttrs_NoIdentityInContext(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	attrs := buildCanonicalAttrs(ctx, "/svc/Method", nil, time.Millisecond)

	m := attrsToMap(attrs)
	if _, ok := m["auth_type"]; ok {
		t.Error("expected no auth_type when no identity in context")
	}
}

// --- CanonicalLogInterceptor tests ---

func TestCanonicalLogInterceptor_CallsHandler(t *testing.T) {
	t.Parallel()
	logger := slog.Default()
	interceptor := CanonicalLogInterceptor(logger)

	called := false
	handler := func(ctx context.Context, req any) (any, error) {
		called = true
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	resp, err := interceptor(context.Background(), "request", info, handler)

	if !called {
		t.Error("expected handler to be called")
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp != "response" {
		t.Errorf("expected %q, got %v", "response", resp)
	}
}

func TestCanonicalLogInterceptor_PropagatesError(t *testing.T) {
	t.Parallel()
	logger := slog.Default()
	interceptor := CanonicalLogInterceptor(logger)

	handlerErr := errors.New("handler failed")
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, handlerErr
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	resp, err := interceptor(context.Background(), "request", info, handler)

	if resp != nil {
		t.Errorf("expected nil response, got %v", resp)
	}
	if err != handlerErr {
		t.Errorf("expected handler error to be propagated, got %v", err)
	}
}

func TestCanonicalLogInterceptor_PassesContextToHandler(t *testing.T) {
	t.Parallel()
	logger := slog.Default()
	interceptor := CanonicalLogInterceptor(logger)

	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("test"), "value")

	handler := func(ctx context.Context, req any) (any, error) {
		if v, ok := ctx.Value(ctxKey("test")).(string); !ok || v != "value" {
			t.Error("expected context value to be propagated to handler")
		}
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	_, _ = interceptor(ctx, nil, info, handler)
}

// --- Helpers ---

func attrsToMap(attrs []slog.Attr) map[string]slog.Value {
	m := make(map[string]slog.Value, len(attrs))
	for _, a := range attrs {
		m[a.Key] = a.Value
	}
	return m
}

func assertAttr(t *testing.T, m map[string]slog.Value, key string, expected string) {
	t.Helper()
	val, ok := m[key]
	if !ok {
		t.Errorf("expected attribute %q to be present", key)
		return
	}
	got := fmt.Sprintf("%v", val.Any())
	if got != expected {
		t.Errorf("attribute %q: expected %q, got %q", key, expected, got)
	}
}
