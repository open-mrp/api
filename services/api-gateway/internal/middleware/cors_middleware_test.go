package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	cors := CORSMiddleware()(handler)

	t.Run("Normal request with Origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://localhost:4200")
		rr := httptest.NewRecorder()

		cors.ServeHTTP(rr, req)

		if rr.Header().Get("Access-Control-Allow-Origin") != "http://localhost:4200" {
			t.Errorf("expected Access-Control-Allow-Origin to be http://localhost:4200, got %v", rr.Header().Get("Access-Control-Allow-Origin"))
		}
		if rr.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Errorf("expected Access-Control-Allow-Credentials to be true, got %v", rr.Header().Get("Access-Control-Allow-Credentials"))
		}
		if rr.Header().Get("Vary") != "Origin" {
			t.Errorf("expected Vary to be Origin, got %v", rr.Header().Get("Vary"))
		}
	})

	t.Run("Preflight request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "http://localhost:4200")
		rr := httptest.NewRecorder()

		cors.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status OK, got %v", rr.Code)
		}
		if rr.Header().Get("Access-Control-Allow-Origin") != "http://localhost:4200" {
			t.Errorf("expected Access-Control-Allow-Origin to be http://localhost:4200, got %v", rr.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("Request without Origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		cors.ServeHTTP(rr, req)

		if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("expected Access-Control-Allow-Origin to be *, got %v", rr.Header().Get("Access-Control-Allow-Origin"))
		}
	})
}
