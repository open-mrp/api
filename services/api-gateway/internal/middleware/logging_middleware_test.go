package middleware

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/services/api-gateway/internal/header"
)

func TestFetchClientIP(t *testing.T) {
	tests := []struct {
		name         string
		remoteAddr   string
		forwardedFor string
		expectedIP   string
		expectedNil  bool
	}{
		{
			name:        "valid IPv4 address",
			remoteAddr:  "192.168.1.100:8080",
			expectedIP:  "192.168.1.100",
			expectedNil: false,
		},
		{
			name:        "valid IPv6 address",
			remoteAddr:  "[2001:db8::1]:8080",
			expectedIP:  "2001:db8::1",
			expectedNil: false,
		},
		{
			name:        "IPv4 without port",
			remoteAddr:  "192.168.1.100",
			expectedIP:  "192.168.1.100",
			expectedNil: false,
		},
		{
			name:        "IPv6 without port",
			remoteAddr:  "2001:db8::1",
			expectedIP:  "2001:db8::1",
			expectedNil: false,
		},
		{
			name:        "invalid remote address",
			remoteAddr:  "invalid-address",
			expectedIP:  "",
			expectedNil: true,
		},
		{
			name:        "empty remote address",
			remoteAddr:  "",
			expectedIP:  "",
			expectedNil: true,
		},
		{
			name:         "X-Forwarded-For overrides remote address",
			remoteAddr:   "192.168.1.100:8080",
			forwardedFor: "203.0.113.1",
			expectedIP:   "203.0.113.1",
			expectedNil:  false,
		},
		{
			name:         "X-Forwarded-For with multiple IPs (first one)",
			remoteAddr:   "192.168.1.100:8080",
			forwardedFor: "203.0.113.1, 198.51.100.1",
			expectedIP:   "203.0.113.1",
			expectedNil:  false,
		},
		{
			name:         "X-Forwarded-For with IPv6",
			remoteAddr:   "192.168.1.100:8080",
			forwardedFor: "2001:db8::1",
			expectedIP:   "2001:db8::1",
			expectedNil:  false,
		},
		{
			name:         "invalid X-Forwarded-For falls back to remote address",
			remoteAddr:   "192.168.1.100:8080",
			forwardedFor: "invalid-ip",
			expectedIP:   "192.168.1.100",
			expectedNil:  false,
		},
		{
			name:         "empty X-Forwarded-For uses remote address",
			remoteAddr:   "192.168.1.100:8080",
			forwardedFor: "",
			expectedIP:   "192.168.1.100",
			expectedNil:  false,
		},
		{
			name:         "both invalid addresses",
			remoteAddr:   "invalid-address",
			forwardedFor: "invalid-ip",
			expectedIP:   "",
			expectedNil:  true,
		},
		{
			name:         "X-Forwarded-For with spaces",
			remoteAddr:   "192.168.1.100:8080",
			forwardedFor: "  203.0.113.1  ",
			expectedIP:   "203.0.113.1",
			expectedNil:  false,
		},
		{
			name:         "X-Forwarded-For with tabs",
			remoteAddr:   "192.168.1.100:8080",
			forwardedFor: "\t203.0.113.1\t",
			expectedIP:   "203.0.113.1",
			expectedNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock request
			req := &http.Request{
				RemoteAddr: tt.remoteAddr,
				Header:     make(http.Header),
			}

			if tt.forwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.forwardedFor)
			}

			// Call the function
			result := header.GetClientIP(req)

			// Check if we expect nil
			if tt.expectedNil {
				if result != nil {
					t.Errorf("expected nil IP, got %v", result)
				}
				return
			}

			// Check if we got a valid IP
			if result == nil {
				t.Errorf("expected IP %s, got nil", tt.expectedIP)
				return
			}

			// Check the IP string representation
			if result.String() != tt.expectedIP {
				t.Errorf("expected IP %s, got %s", tt.expectedIP, result.String())
			}

			// Verify it's a valid IP by parsing it back
			expectedIP := net.ParseIP(tt.expectedIP)
			if expectedIP == nil {
				t.Errorf("expectedIP %s is not a valid IP", tt.expectedIP)
			} else if !result.Equal(expectedIP) {
				t.Errorf("parsed IPs don't match: expected %v, got %v", expectedIP, result)
			}
		})
	}
}

func TestFetchClientIPEdgeCases(t *testing.T) {
	t.Run("X-Forwarded-For with only spaces", func(t *testing.T) {
		req := &http.Request{
			RemoteAddr: "192.168.1.100:8080",
			Header:     make(http.Header),
		}
		req.Header.Set("X-Forwarded-For", "   ")

		result := header.GetClientIP(req)
		if result == nil {
			t.Error("expected IP from remote address, got nil")
		} else if result.String() != "192.168.1.100" {
			t.Errorf("expected 192.168.1.100, got %s", result.String())
		}
	})

	t.Run("X-Forwarded-For with newlines", func(t *testing.T) {
		req := &http.Request{
			RemoteAddr: "192.168.1.100:8080",
			Header:     make(http.Header),
		}
		req.Header.Set("X-Forwarded-For", "\n203.0.113.1\n")

		result := header.GetClientIP(req)
		if result == nil {
			t.Error("expected IP, got nil")
		} else if result.String() != "203.0.113.1" {
			t.Errorf("expected 203.0.113.1, got %s", result.String())
		}
	})

	t.Run("very long X-Forwarded-For header", func(t *testing.T) {
		req := &http.Request{
			RemoteAddr: "192.168.1.100:8080",
			Header:     make(http.Header),
		}
		// Create a very long forwarded header
		longHeader := "203.0.113.1, 198.51.100.1, 192.0.2.1, 203.0.113.2, 198.51.100.2"
		req.Header.Set("X-Forwarded-For", longHeader)

		result := header.GetClientIP(req)
		if result == nil {
			t.Error("expected IP, got nil")
		} else if result.String() != "203.0.113.1" {
			t.Errorf("expected first IP 203.0.113.1, got %s", result.String())
		}
	})
}

func TestLoggingMiddleware_SkipHealthz(t *testing.T) {
	// Create a mock saver that records if Save was called
	var saved bool
	mockS := &mockSaver{
		saveFunc: func(ctx context.Context, rl *domain.RequestLog) error {
			saved = true
			return nil
		},
	}

	// Create an async saver with the mock saver
	asyncSaver := NewAsyncRequestLogSaver(1, mockS)

	// Create a dummy logger
	logger := log.New(io.Discard, "", 0)

	// Create the middleware
	handler := LoggingMiddleware(logger, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, asyncSaver, nil)

	t.Run("skip /healthz GET", func(t *testing.T) {
		saved = false
		// Create a request to /healthz
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		// Serve the request
		handler.ServeHTTP(w, req)

		// Wait a bit for the async saver to process (though it shouldn't be called)
		time.Sleep(20 * time.Millisecond)

		if saved {
			t.Error("Expected /healthz GET request NOT to be saved, but it was")
		}
	})

	t.Run("skip /healthz POST", func(t *testing.T) {
		saved = false
		// Create a request to /healthz
		req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
		w := httptest.NewRecorder()

		// Serve the request
		handler.ServeHTTP(w, req)

		// Wait a bit for the async saver to process (though it shouldn't be called)
		time.Sleep(20 * time.Millisecond)

		if saved {
			t.Error("Expected /healthz POST request NOT to be saved, but it was")
		}
	})

	t.Run("log other paths", func(t *testing.T) {
		saved = false
		// Create a request to /other
		req := httptest.NewRequest(http.MethodGet, "/other", nil)
		w := httptest.NewRecorder()

		// Serve the request
		handler.ServeHTTP(w, req)

		// Wait for the async saver to process
		deadline := time.Now().Add(100 * time.Millisecond)
		for time.Now().Before(deadline) {
			if saved {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}

		if !saved {
			t.Error("Expected /other request to be saved, but it wasn't")
		}
	})
}

type mockSaver struct {
	saveFunc func(ctx context.Context, rl *domain.RequestLog) error
}

func (m *mockSaver) Save(ctx context.Context, rl *domain.RequestLog) error {
	return m.saveFunc(ctx, rl)
}
