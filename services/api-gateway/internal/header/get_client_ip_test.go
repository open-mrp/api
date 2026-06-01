package header

import (
	"net/http/httptest"
	"testing"
)

func TestGetClientIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		remoteAddr       string
		xForwardedFor    string
		trustedProxyHops int
		want             string
	}{
		{
			name:             "no proxy and no XFF returns RemoteAddr",
			remoteAddr:       "203.0.113.10:5555",
			xForwardedFor:    "",
			trustedProxyHops: 0,
			want:             "203.0.113.10",
		},
		{
			name:             "no proxy ignores XFF even when present",
			remoteAddr:       "203.0.113.10:5555",
			xForwardedFor:    "198.51.100.77",
			trustedProxyHops: 0,
			want:             "203.0.113.10",
		},
		{
			name:             "one trusted hop with single XFF entry uses that entry",
			remoteAddr:       "10.0.0.1:443",
			xForwardedFor:    "203.0.113.10",
			trustedProxyHops: 1,
			want:             "203.0.113.10",
		},
		{
			name:             "one trusted hop uses RIGHTMOST XFF entry (not attacker leftmost)",
			remoteAddr:       "10.0.0.1:443",
			xForwardedFor:    "198.51.100.77, 203.0.113.10",
			trustedProxyHops: 1,
			want:             "203.0.113.10",
		},
		{
			name:             "attacker stuffed XFF with junk on the left is ignored",
			remoteAddr:       "10.0.0.1:443",
			xForwardedFor:    "evil, junk, not-an-ip, 203.0.113.10",
			trustedProxyHops: 1,
			want:             "203.0.113.10",
		},
		{
			name:             "two trusted hops peels both off and returns the entry the outer proxy added",
			remoteAddr:       "10.0.0.1:443",
			xForwardedFor:    "198.51.100.77, 203.0.113.10, 10.1.1.1",
			trustedProxyHops: 2,
			want:             "203.0.113.10",
		},
		{
			name:             "hops greater than XFF length falls back to RemoteAddr",
			remoteAddr:       "10.0.0.1:443",
			xForwardedFor:    "203.0.113.10",
			trustedProxyHops: 2,
			want:             "10.0.0.1",
		},
		{
			name:             "trusted hop with invalid candidate falls back to RemoteAddr",
			remoteAddr:       "10.0.0.1:443",
			xForwardedFor:    "not-an-ip",
			trustedProxyHops: 1,
			want:             "10.0.0.1",
		},
		{
			name:             "IPv6 RemoteAddr is parsed correctly",
			remoteAddr:       "[2001:db8::1]:5555",
			xForwardedFor:    "",
			trustedProxyHops: 0,
			want:             "2001:db8::1",
		},
		{
			name:             "RemoteAddr without port is handled",
			remoteAddr:       "203.0.113.10",
			xForwardedFor:    "",
			trustedProxyHops: 0,
			want:             "203.0.113.10",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tc.xForwardedFor)
			}

			got := GetClientIP(req, tc.trustedProxyHops)
			if got == nil {
				t.Fatalf("GetClientIP returned nil; want %s", tc.want)
			}
			if got.String() != tc.want {
				t.Fatalf("GetClientIP = %s; want %s", got.String(), tc.want)
			}
		})
	}
}
