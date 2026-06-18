package middleware

import (
	"net"
	"net/http"

	"github.com/augno/api/services/api-gateway/internal/header"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apierror "github.com/augno/api/shared/errors"
)

// blockedIPs is the hardcoded set of IP addresses that are denied access to every router. Entries should be exact-match IP literals (IPv4 or IPv6).
var blockedIPs = map[string]struct{}{
	"49.43.184.36": {},
}

// IsIPBlocked reports whether the given IP is on the block list.
func IsIPBlocked(ip net.IP) bool {
	if ip == nil {
		return false
	}
	_, blocked := blockedIPs[ip.String()]
	return blocked
}

// IPBlockMiddleware rejects requests whose client IP is on the block list. It runs early in the chain so blocked traffic is not rate-limited, authenticated, or otherwise processed.
func IPBlockMiddleware(trustedProxyHops int) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if IsIPBlocked(header.GetClientIP(r, trustedProxyHops)) {
				httptransport.RespondWithAPIError(r.Context(), w, apierror.NewAuthorizationError("Access denied."))
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}
