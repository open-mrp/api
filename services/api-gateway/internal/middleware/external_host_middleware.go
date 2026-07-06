package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/augno/api/shared/appctx"
)

// ExternalHostMiddleware stores the host the browser addressed in the request context. Requests proxied through the frontend on a customer's custom portal domain arrive with the original host in X-Forwarded-Host; direct requests use the Host header. Cookie scoping uses this to decide between the shared .augno.com domain and a host-only cookie — a spoofed X-Forwarded-Host can only downgrade the caller to a host-only cookie on the spoofed host, so the header does not need to be validated against a domain allowlist.
func ExternalHostMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			host := r.Header.Get("X-Forwarded-Host")
			if host == "" {
				host = r.Host
			}
			// X-Forwarded-Host may carry a comma-separated chain; the first entry is the client-facing host.
			if idx := strings.IndexByte(host, ','); idx >= 0 {
				host = host[:idx]
			}
			host = strings.TrimSpace(host)
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}

			ctx := appctx.WithExternalHost(r.Context(), strings.ToLower(host))
			next(w, r.WithContext(ctx))
		}
	}
}
