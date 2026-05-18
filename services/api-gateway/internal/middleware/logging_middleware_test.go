package middleware

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augno/api/services/api-gateway/internal/header"
	"github.com/augno/api/shared/appctx"
)

type stubSaver struct {
	savedRL *appctx.RequestLog
}

func (s *stubSaver) Save(ctx context.Context, rl *appctx.RequestLog) error {
	s.savedRL = rl
	return nil
}

func TestLoggingMiddleware_CapturesAPIVersion(t *testing.T) {
	t.Parallel()
	logger := log.New(io.Discard, "", 0)
	saver := &stubSaver{}

	handler := LoggingMiddleware(logger, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, saver, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(header.VersionHeader, "1.0.forge-preview.1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if saver.savedRL == nil {
		t.Fatal("Expected request log to be saved")
	}
	if saver.savedRL.APIVersion == nil {
		t.Fatal("Expected APIVersion to be captured, got nil")
	}
	if *saver.savedRL.APIVersion != "1.0.forge-preview.1" {
		t.Errorf("Expected APIVersion '1.0.forge', got '%s'", *saver.savedRL.APIVersion)
	}
}

func TestLoggingMiddleware_CapturesAPIVersionEvenIfInvalid(t *testing.T) {
	t.Parallel()
	logger := log.New(io.Discard, "", 0)
	saver := &stubSaver{}

	handler := LoggingMiddleware(logger, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, saver, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	// Set an invalid version - should still be captured for logging
	req.Header.Set(header.VersionHeader, "invalid-version")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if saver.savedRL == nil {
		t.Fatal("Expected request log to be saved")
	}
	// The key fix: API version should be captured even if invalid
	if saver.savedRL.APIVersion == nil {
		t.Fatal("Expected APIVersion to be captured even for invalid version, got nil")
	}
	if *saver.savedRL.APIVersion != "invalid-version" {
		t.Errorf("Expected APIVersion 'invalid-version', got '%s'", *saver.savedRL.APIVersion)
	}
}

func TestLoggingMiddleware_NoAPIVersionWhenHeaderMissing(t *testing.T) {
	t.Parallel()
	logger := log.New(io.Discard, "", 0)
	saver := &stubSaver{}

	handler := LoggingMiddleware(logger, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, saver, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	// No version header set
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if saver.savedRL == nil {
		t.Fatal("Expected request log to be saved")
	}
	// When header is missing, APIVersion should be nil
	if saver.savedRL.APIVersion != nil {
		t.Errorf("Expected APIVersion to be nil when header is missing, got '%s'", *saver.savedRL.APIVersion)
	}
}

func TestLoggingMiddleware_RequestLogInContext(t *testing.T) {
	t.Parallel()
	logger := log.New(io.Discard, "", 0)
	saver := &stubSaver{}

	var capturedRL *appctx.RequestLog
	handler := LoggingMiddleware(logger, func(w http.ResponseWriter, r *http.Request) {
		rl, ok := appctx.GetRequestLog(r.Context())
		if ok {
			capturedRL = rl
		}
		w.WriteHeader(http.StatusOK)
	}, saver, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(header.VersionHeader, "1.0.forge-preview.1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if capturedRL == nil {
		t.Fatal("Expected request log to be in context")
	}

	if capturedRL.APIVersion == nil {
		t.Fatal("Expected APIVersion in context request log, got nil")
	}

	if *capturedRL.APIVersion != "1.0.forge-preview.1" {
		t.Errorf("Expected APIVersion '1.0.forge' in context, got '%s'", *capturedRL.APIVersion)
	}
}

func TestLoggingMiddleware_RedactsSensitiveResponseFields(t *testing.T) {
	t.Parallel()
	logger := log.New(io.Discard, "", 0)
	saver := &stubSaver{}

	handler := LoggingMiddleware(logger, func(w http.ResponseWriter, r *http.Request) {
		rl, ok := appctx.GetRequestLog(r.Context())
		if !ok {
			t.Fatal("missing request log")
		}
		rl.SensitiveResponseFields = map[string]bool{"api_key_secret": true}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"api_key_secret":"aug_sk_prod_secret","object":"created_api_key","api_key_info":{"id":"apke_123","object":"api_key"}}`))
	}, saver, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if saver.savedRL == nil || saver.savedRL.ResponseJSON == nil {
		t.Fatal("expected response json in saved log")
	}
	if want := `{"api_key_info":{"id":"apke_123","object":"api_key"},"api_key_secret":"****","object":"created_api_key"}`; *saver.savedRL.ResponseJSON != want {
		t.Fatalf("response json = %q want %q", *saver.savedRL.ResponseJSON, want)
	}
}

func TestLoggingMiddleware_RedactFailure_omitsResponseJSON(t *testing.T) {
	t.Parallel()
	logger := log.New(io.Discard, "", 0)
	saver := &stubSaver{}

	handler := LoggingMiddleware(logger, func(w http.ResponseWriter, r *http.Request) {
		rl, ok := appctx.GetRequestLog(r.Context())
		if !ok {
			t.Fatal("missing request log")
		}
		rl.SensitiveResponseFields = map[string]bool{"x": true}
		w.WriteHeader(http.StatusOK)
		// Not valid JSON as a single top-level value → RedactJSON returns nil → omit stored response.
		_, _ = w.Write([]byte(`not-json`))
	}, saver, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if saver.savedRL == nil {
		t.Fatal("expected saved log")
	}
	if saver.savedRL.ResponseJSON != nil {
		t.Fatalf("expected nil ResponseJSON on redact failure, got %q", *saver.savedRL.ResponseJSON)
	}
}

// stubRouteMatcher implements RouteMatcher for testing public endpoint detection.
type stubRouteMatcher struct {
	routes []any
}

func (s *stubRouteMatcher) GetRoutes() []any { return s.routes }

func TestLoggingMiddleware_PublicEndpoint_WithPublicRoute(t *testing.T) {
	t.Parallel()
	logger := log.New(io.Discard, "", 0)
	saver := &stubSaver{}

	router := &stubRouteMatcher{
		routes: []any{
			map[string]any{"Method": "GET", "Path": "/v1/invoices", "PathPattern": nil, "Public": true},
		},
	}

	handler := LoggingMiddleware(logger, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, saver, router)

	req := httptest.NewRequest(http.MethodGet, "/v1/invoices", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if saver.savedRL == nil {
		t.Fatal("Expected request log to be saved")
	}
	if !saver.savedRL.PublicEndpoint {
		t.Error("Expected PublicEndpoint to be true for a public route")
	}
}

func TestLoggingMiddleware_PublicEndpoint_NoRouter(t *testing.T) {
	t.Parallel()
	logger := log.New(io.Discard, "", 0)
	saver := &stubSaver{}

	handler := LoggingMiddleware(logger, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, saver, nil)

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if saver.savedRL == nil {
		t.Fatal("Expected request log to be saved")
	}
	if !saver.savedRL.PublicEndpoint {
		t.Error("Expected PublicEndpoint to default to true when no router is provided")
	}
}
