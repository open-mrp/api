package middleware

import (
	"net/http"

	"github.com/open-mrp/apikit/tracing"
)

// TracingMiddleware adds OpenTelemetry tracing to the request. It skips tracing for health check endpoints to save resources.
func TracingMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		instrumented := tracing.WrapHandler(next)

		return func(w http.ResponseWriter, r *http.Request) {
			instrumented.ServeHTTP(w, r)
		}
	}
}
