package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augno/api/services/api-gateway/internal/header"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/version"
)

func TestVersionMiddleware_MissingHeader(t *testing.T) {
	t.Parallel()
	middleware := VersionMiddleware()

	handler := middleware(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when version header is missing")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rl := &appctx.RequestLog{}
	req = req.WithContext(appctx.WithRequestLog(req.Context(), rl))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var response apierror.APIErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error.Code != apierror.ErrorCodeAPIVersionRequired {
		t.Errorf("Expected error code %s, got %s", apierror.ErrorCodeAPIVersionRequired, response.Error.Code)
	}
}

func TestVersionMiddleware_InvalidVersion(t *testing.T) {
	t.Parallel()
	middleware := VersionMiddleware()

	handler := middleware(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when version is invalid")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(header.VersionHeader, "invalid-version")
	rl := &appctx.RequestLog{}
	req = req.WithContext(appctx.WithRequestLog(req.Context(), rl))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var response apierror.APIErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error.Code != apierror.ErrorCodeAPIVersionInvalid {
		t.Errorf("Expected error code %s, got %s", apierror.ErrorCodeAPIVersionInvalid, response.Error.Code)
	}
}

func TestVersionMiddleware_InvalidFormat(t *testing.T) {
	t.Parallel()
	middleware := VersionMiddleware()

	handler := middleware(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when version format is invalid")
	})

	// Test invalid format
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(header.VersionHeader, "1.0") // missing codename
	rl := &appctx.RequestLog{}
	req = req.WithContext(appctx.WithRequestLog(req.Context(), rl))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var response apierror.APIErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error.Code != apierror.ErrorCodeAPIVersionInvalid {
		t.Errorf("Expected error code %s, got %s", apierror.ErrorCodeAPIVersionInvalid, response.Error.Code)
	}
}

func TestVersionMiddleware_ValidVersion(t *testing.T) {
	t.Parallel()
	middleware := VersionMiddleware()

	var capturedVersion version.APIVersion
	var capturedOk bool
	handler := middleware(func(w http.ResponseWriter, r *http.Request) {
		capturedVersion, capturedOk = appctx.GetAPIVersionFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(header.VersionHeader, "1.0.forge-preview.1")
	rl := &appctx.RequestLog{}
	req = req.WithContext(appctx.WithRequestLog(req.Context(), rl))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if !capturedOk {
		t.Error("Expected version in context")
	}

	if capturedVersion.String() != "1.0.forge-preview.1" {
		t.Errorf("Expected version %s in context, got %s", "1.0.forge-preview.1", capturedVersion.String())
	}

	if capturedVersion.Codename != "forge" {
		t.Errorf("Expected codename %s, got %s", "forge", capturedVersion.Codename)
	}

	if rr.Header().Get(header.VersionHeader) != "1.0.forge-preview.1" {
		t.Errorf("Expected version %s in response header, got %s", "1.0.forge-preview.1", rr.Header().Get(header.VersionHeader))
	}
}

func TestVersionMiddleware_SkipsHealthz(t *testing.T) {
	t.Parallel()
	middleware := VersionMiddleware()

	handlerCalled := false
	handler := middleware(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("Handler should be called for /healthz")
	}

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}
