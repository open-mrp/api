package header

import (
	"net"
	"net/http"
	"strings"
)

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
