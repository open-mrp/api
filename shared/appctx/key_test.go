package appctx

import (
	"context"
	"net/url"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

// Mirrors the private key types other packages declare (e.g. agent-service's internal/llm ctxKey):
// same underlying string, different named type.
type foreignCtxKey string

type foreignNoTraceKey struct{}

type keyIsolationCase struct {
	name  string
	key   string
	value any
	with  func(context.Context) context.Context
	get   func(context.Context) bool
}

func keyIsolationCases() []keyIsolationCase {
	accountID := "acct_1"
	identity := &types.Identity{Type: types.IdentityActorTypeUser, Actor: &types.IdentityActor{ID: "usr_1", AccountID: &accountID}}
	requestLog := &RequestLog{ID: "rql_1"}
	requestURL := &url.URL{Path: "/v1/orders"}
	meta := &IdempotencyResponseMetadata{}
	httpMeta := &HTTPResponseMetadata{}

	return []keyIsolationCase{
		{
			name:  "identity",
			key:   "identity",
			value: identity,
			with:  func(ctx context.Context) context.Context { return WithIdentity(ctx, identity) },
			get:   func(ctx context.Context) bool { _, ok := GetIdentityFromContext(ctx); return ok },
		},
		{
			name:  "request_log",
			key:   "request_log",
			value: requestLog,
			with:  func(ctx context.Context) context.Context { return WithRequestLog(ctx, requestLog) },
			get:   func(ctx context.Context) bool { _, ok := GetRequestLog(ctx); return ok },
		},
		{
			name:  "request_id",
			key:   "request_id",
			value: "req_1",
			with:  func(ctx context.Context) context.Context { return WithRequestID(ctx, "req_1") },
			get:   func(ctx context.Context) bool { _, ok := GetRequestID(ctx); return ok },
		},
		{
			name:  "request_url",
			key:   "request_url",
			value: requestURL,
			with:  func(ctx context.Context) context.Context { return WithRequestURL(ctx, requestURL) },
			get:   func(ctx context.Context) bool { _, ok := GetRequestURL(ctx); return ok },
		},
		{
			name:  "route_pattern",
			key:   "route_pattern",
			value: "/v1/orders/{order_id}",
			with:  func(ctx context.Context) context.Context { return WithRoutePattern(ctx, "/v1/orders/{order_id}") },
			get:   func(ctx context.Context) bool { _, ok := GetRoutePattern(ctx); return ok },
		},
		{
			name:  "allowed_methods",
			key:   "allowed_methods",
			value: []string{"GET"},
			with:  func(ctx context.Context) context.Context { return WithAllowedMethods(ctx, []string{"GET"}) },
			get:   func(ctx context.Context) bool { _, ok := GetAllowedMethods(ctx); return ok },
		},
		{
			name:  "path_params",
			key:   "path_params",
			value: map[string]string{"order_id": "or_1"},
			with: func(ctx context.Context) context.Context {
				return WithPathParams(ctx, map[string]string{"order_id": "or_1"})
			},
			get: func(ctx context.Context) bool { _, ok := GetPathParams(ctx); return ok },
		},
		{
			name:  "idempotency_key",
			key:   "idempotency_key",
			value: "ik_1",
			with:  func(ctx context.Context) context.Context { return WithIdempotencyKey(ctx, "ik_1") },
			get:   func(ctx context.Context) bool { _, ok := GetIdempotencyKey(ctx); return ok },
		},
		{
			name:  "idempotency_key_id",
			key:   "idempotency_key_id",
			value: "idk_1",
			with:  func(ctx context.Context) context.Context { return WithIdempotencyKeyID(ctx, "idk_1") },
			get:   func(ctx context.Context) bool { _, ok := GetIdempotencyKeyID(ctx); return ok },
		},
		{
			name:  "idempotency_response_metadata",
			key:   "idempotency_response_metadata",
			value: meta,
			with:  func(ctx context.Context) context.Context { return WithIdempotencyResponseMetadata(ctx, meta) },
			get:   func(ctx context.Context) bool { _, ok := GetIdempotencyResponseMetadata(ctx); return ok },
		},
		{
			name:  "http_response_metadata",
			key:   "http_response_metadata",
			value: httpMeta,
			with: func(ctx context.Context) context.Context {
				next, _ := WithHTTPResponseMetadata(ctx)
				return next
			},
			get: func(ctx context.Context) bool { _, ok := GetHTTPResponseMetadata(ctx); return ok },
		},
		{
			name:  "platform",
			key:   "platform",
			value: constants.PlatformModeProduction,
			with:  func(ctx context.Context) context.Context { return WithPlatform(ctx, constants.PlatformModeProduction) },
			get:   func(ctx context.Context) bool { _, ok := GetPlatformFromContext(ctx); return ok },
		},
		{
			name:  "api_version",
			key:   "api_version",
			value: version.APIVersion{},
			with:  func(ctx context.Context) context.Context { return WithAPIVersion(ctx, version.APIVersion{}) },
			get:   func(ctx context.Context) bool { _, ok := GetAPIVersionFromContext(ctx); return ok },
		},
		{
			name:  "handler",
			key:   "handler",
			value: "auth.Login",
			with:  func(ctx context.Context) context.Context { return WithHandler(ctx, "auth.Login") },
			get:   func(ctx context.Context) bool { _, ok := GetHandler(ctx); return ok },
		},
		{
			name:  "external_host",
			key:   "external_host",
			value: "api.example.com",
			with:  func(ctx context.Context) context.Context { return WithExternalHost(ctx, "api.example.com") },
			get:   func(ctx context.Context) bool { _, ok := GetExternalHost(ctx); return ok },
		},
		{
			name:  "propagated_client_ip",
			key:   "propagated_client_ip",
			value: "198.51.100.7",
			with:  func(ctx context.Context) context.Context { return WithPropagatedClientIP(ctx, "198.51.100.7") },
			get:   func(ctx context.Context) bool { _, ok := GetPropagatedClientIP(ctx); return ok },
		},
	}
}

// Collision resistance is the reason contextKey is an unexported named type: a foreign key whose
// underlying string is identical must be invisible to this package's getters.
func TestContextKeys_ForeignKeyIsNotReadable(t *testing.T) {
	t.Parallel()
	for _, tt := range keyIsolationCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.WithValue(context.Background(), foreignCtxKey(tt.key), tt.value)
			if tt.get(ctx) {
				t.Errorf("value stored under foreignCtxKey(%q) leaked into the appctx getter", tt.key)
			}
		})
	}
}

func TestContextKeys_AppctxValueIsNotReadableByForeignKey(t *testing.T) {
	t.Parallel()
	for _, tt := range keyIsolationCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := tt.with(context.Background())
			if got := ctx.Value(foreignCtxKey(tt.key)); got != nil {
				t.Errorf("appctx value leaked to foreignCtxKey(%q): %+v", tt.key, got)
			}
			if got := ctx.Value(tt.key); got != nil {
				t.Errorf("appctx value leaked to the raw string key %q: %+v", tt.key, got)
			}
		})
	}
}

func TestNoTraceKey_IsIsolatedFromForeignStructKeys(t *testing.T) {
	t.Parallel()
	suppressed := context.WithValue(context.Background(), foreignNoTraceKey{}, true)
	if !ShouldTrace(suppressed) {
		t.Error("a foreign struct key suppressed tracing")
	}

	ctx := WithNoTrace(context.Background())
	if got := ctx.Value(foreignNoTraceKey{}); got != nil {
		t.Errorf("no-trace flag leaked to a foreign struct key: %+v", got)
	}
}

// The no-trace flag is the one key not built on contextKey, so nothing above pins it to a private
// type: demoting it to a raw string would leave every getter test green while any package storing
// a same-named string value silently suppressed tracing.
// rawStringKey launders a string into an any so a test can store a value under a bare string context key. Writing one literally is a mistake the vet check rightly flags, but a caller elsewhere in the process doing exactly that is the collision these tests assert appctx is immune to.
func rawStringKey(s string) any { return s }

func TestNoTraceKey_IsNotAStringKey(t *testing.T) {
	t.Parallel()
	names := []string{"no_trace", "noTrace", "no-trace", "trace"}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := WithNoTrace(context.Background()).Value(name); got != nil {
				t.Errorf("no-trace flag readable through the raw string key %q: %+v", name, got)
			}
			if got := WithNoTrace(context.Background()).Value(foreignCtxKey(name)); got != nil {
				t.Errorf("no-trace flag readable through foreignCtxKey(%q): %+v", name, got)
			}
			if !ShouldTrace(context.WithValue(context.Background(), rawStringKey(name), true)) {
				t.Errorf("a value stored under the raw string key %q suppressed tracing", name)
			}
		})
	}
}
