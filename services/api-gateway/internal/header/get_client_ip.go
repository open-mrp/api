package header

import (
	"net"
	"net/http"
	"strings"
)

// GetClientIP returns the client IP address from the request.
// If the request is forwarded, the first IP address in the X-Forwarded-For header is returned.
// If the request is not forwarded, the remote address is returned.
// If the remote address is not a valid IP address, the host is returned.
func GetClientIP(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	clientIP := net.ParseIP(host)

	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		forwarded = strings.TrimSpace(forwarded)

		if commaIndex := strings.Index(forwarded, ","); commaIndex != -1 {
			forwarded = strings.TrimSpace(forwarded[:commaIndex])
		}

		if forwardedIP := net.ParseIP(forwarded); forwardedIP != nil {
			clientIP = forwardedIP
		}
	}

	return clientIP
}
