package header

import (
	"net"
	"net/http"
	"strings"
)

// GetClientIP returns the client IP address for the request, using the
// X-Forwarded-For chain only to the extent it can be trusted.
//
// trustedProxyHops is the number of reverse-proxy hops in front of this
// service. Each trusted proxy is expected to APPEND the IP of the connection
// it received from to X-Forwarded-For (this is what AWS ALB does). The
// rightmost trustedProxyHops entries in the chain are therefore the ones
// written by trusted infrastructure, and the entry at position
// `len(parts) - trustedProxyHops` is the IP of the original client as
// observed by the outermost trusted proxy.
//
// When trustedProxyHops is 0 (no trusted proxy in front), the X-Forwarded-For
// header is ignored entirely — it is fully attacker-controlled and must not
// be used for any security-sensitive purpose such as rate limiting. RemoteAddr
// is returned instead.
//
// When fewer XFF entries are present than expected, the trusted-proxy chain
// was not fully traversed, and we fall back to RemoteAddr rather than risk
// trusting an attacker-supplied entry.
func GetClientIP(r *http.Request, trustedProxyHops int) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	clientIP := net.ParseIP(host)

	if trustedProxyHops <= 0 {
		return clientIP
	}

	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded == "" {
		return clientIP
	}

	parts := strings.Split(forwarded, ",")
	if trustedProxyHops > len(parts) {
		return clientIP
	}

	candidate := strings.TrimSpace(parts[len(parts)-trustedProxyHops])
	if ip := net.ParseIP(candidate); ip != nil {
		return ip
	}
	return clientIP
}
