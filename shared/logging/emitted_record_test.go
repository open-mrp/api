package logging

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// captureHandler records everything written to the logger so tests can assert on
// the record that actually reaches the backend rather than on the attributes
// buildCanonicalAttrs returns.
type captureHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *captureHandler) WithGroup(string) slog.Handler { return h }

func (h *captureHandler) all() []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]slog.Record(nil), h.records...)
}

func (h *captureHandler) only(t *testing.T) slog.Record {
	t.Helper()
	records := h.all()
	if len(records) != 1 {
		t.Fatalf("expected exactly one canonical log line, got %d", len(records))
	}
	return records[0]
}

func recordAttrs(r slog.Record) map[string]slog.Value {
	m := make(map[string]slog.Value, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value
		return true
	})
	return m
}

func newCapturingInterceptor() (*captureHandler, grpc.UnaryServerInterceptor) {
	h := &captureHandler{}
	return h, CanonicalLogInterceptor(slog.New(h))
}

func TestCanonicalLogInterceptor_EmitsOneRecordPerCall(t *testing.T) {
	t.Parallel()
	handler, interceptor := newCapturingInterceptor()

	info := &grpc.UnaryServerInfo{FullMethod: "/auth.v1.AuthService/LoginUser"}
	for range 3 {
		_, _ = interceptor(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
			return "ok", nil
		})
	}

	if got := len(handler.all()); got != 3 {
		t.Fatalf("expected 3 canonical log lines, got %d", got)
	}
}

func TestCanonicalLogInterceptor_EmittedRecordFields(t *testing.T) {
	t.Parallel()
	handler, interceptor := newCapturingInterceptor()

	ctx := appctx.WithRequestID(context.Background(), "req_1")
	ctx = appctx.WithIdentity(ctx, &types.Identity{
		Type:  types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{ID: "usr_1"},
	})

	info := &grpc.UnaryServerInfo{FullMethod: "/auth.v1.AuthService/LoginUser"}
	_, _ = interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})

	record := handler.only(t)
	if record.Message != info.FullMethod {
		t.Errorf("expected message %q, got %q", info.FullMethod, record.Message)
	}

	m := recordAttrs(record)
	assertAttr(t, m, "type", "canonical-log-line")
	assertAttr(t, m, "grpc_method", info.FullMethod)
	assertAttr(t, m, "grpc_code", "OK")
	assertAttr(t, m, "request_id", "req_1")
	assertAttr(t, m, "auth_type", "user")
	assertAttr(t, m, "user_id", "usr_1")
	if _, ok := m["error"]; ok {
		t.Error("expected no error attribute on a successful call")
	}
}

func TestCanonicalLogInterceptor_EmittedRecordForFailedCall(t *testing.T) {
	t.Parallel()
	handler, interceptor := newCapturingInterceptor()

	info := &grpc.UnaryServerInfo{FullMethod: "/core.v1.CoreService/GetTransaction"}
	_, _ = interceptor(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.NotFound, "transaction not found")
	})

	m := recordAttrs(handler.only(t))
	assertAttr(t, m, "grpc_code", "NotFound")
	if _, ok := m["error"]; !ok {
		t.Error("expected an error attribute when the handler fails")
	}
}

func TestCanonicalLogInterceptor_RecordsAreIndependentPerCall(t *testing.T) {
	t.Parallel()
	handler, interceptor := newCapturingInterceptor()

	info := &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}
	_, _ = interceptor(appctx.WithRequestID(context.Background(), "req_1"), nil, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	_, _ = interceptor(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.PermissionDenied, "denied")
	})

	records := handler.all()
	if len(records) != 2 {
		t.Fatalf("expected 2 canonical log lines, got %d", len(records))
	}

	first, second := recordAttrs(records[0]), recordAttrs(records[1])
	assertAttr(t, first, "request_id", "req_1")
	assertAttr(t, first, "grpc_code", "OK")
	if _, ok := second["request_id"]; ok {
		t.Error("expected the second record to carry no request_id")
	}
	assertAttr(t, second, "grpc_code", "PermissionDenied")
}
