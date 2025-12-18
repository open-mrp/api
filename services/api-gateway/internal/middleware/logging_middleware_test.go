package middleware

import (
	"net"
	"net/http"
	"testing"

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
