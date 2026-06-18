package tracing

import (
	"fmt"
	"net/http"

	"github.com/augno/api/shared/appctx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Legacy and semantic-convention attribute keys attached to every HTTP span. Both the older "http.method" style and the newer semconv keys are set so spans are searchable in trace backends regardless of which attribute version they query on.
const (
	attrHTTPMethod     = "http.method"
	attrHTTPRoute      = "http.route"
	attrHTTPPath       = "http.url.path"
	attrHTTPStatusCode = "http.status_code"
	attrHTTPQuery      = "http.url.query"
	attrServerAddress  = "server.address"
	attrUserAgent      = "user_agent.original"
)

// httpTracerName is the instrumentation scope name used for HTTP spans. It follows the OpenTelemetry convention of naming the tracer after the instrumentation library.
const httpTracerName = "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

// WrapGatewayHandler returns an http.Handler that wraps handler with OpenTelemetry tracing. For each inbound request it:
//
//  1. Skips /healthz and OPTIONS requests (no span created).
//  2. Extracts incoming trace context from HTTP headers (W3C TraceContext + Baggage).
//  3. Starts a server span named "HTTP <METHOD> <route>" (e.g. "HTTP GET /api/v1/users").
//  4. Calls the inner handler with the traced context.
//  5. Records response status, route, host, query string, and user-agent as span attributes using both legacy and semantic-convention keys.
//  6. Ends the span immediately after the response is written — not after downstream middleware completes — so span duration accurately reflects response latency.
func WrapGatewayHandler(handler http.Handler) http.Handler {
	propagator := otel.GetTextMapPropagator()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := otel.Tracer(httpTracerName)
		// Skip tracing for health checks and OPTIONS requests
		if r.URL.Path == "/healthz" || r.Method == http.MethodOptions {
			handler.ServeHTTP(w, r)
			return
		}

		// Extract trace context from incoming headers
		ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// Start span with server kind
		spanName := fmt.Sprintf("HTTP %s %s", r.Method, routeOrPath(r))
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
		)

		// Update request with traced context
		r = r.WithContext(ctx)

		// Record response status
		rw := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the actual handler
		handler.ServeHTTP(rw, r)

		// Set attributes and end span immediately after response is written
		route := routeOrPath(r)
		span.SetAttributes(
			semconv.HTTPRequestMethodKey.String(r.Method),
			semconv.HTTPResponseStatusCode(rw.statusCode),
			semconv.URLPath(r.URL.Path),
			semconv.URLScheme("http"),
			attribute.String(attrHTTPMethod, r.Method),
			attribute.String(attrHTTPRoute, route),
			attribute.String(attrHTTPPath, r.URL.Path),
			attribute.Int(attrHTTPStatusCode, rw.statusCode),
		)
		if host := r.Host; host != "" {
			span.SetAttributes(semconv.ServerAddress(host))
			span.SetAttributes(attribute.String(attrServerAddress, host))
		}
		if rawQuery := r.URL.RawQuery; rawQuery != "" {
			span.SetAttributes(attribute.String(attrHTTPQuery, rawQuery))
		}
		if ua := r.UserAgent(); ua != "" {
			span.SetAttributes(semconv.UserAgentOriginal(ua))
			span.SetAttributes(attribute.String(attrUserAgent, ua))
		}

		span.End()
	})
}

// routeOrPath returns the matched route pattern for the request if available (set by the router via context), otherwise falls back to the raw URL path. Using the route pattern (e.g. "/api/v1/users/{id}") instead of the raw path prevents high-cardinality span names from path parameters.
func routeOrPath(r *http.Request) string {
	if routePattern := getRoutePattern(r); routePattern != "" {
		return routePattern
	}
	return r.URL.Path
}

// getRoutePattern extracts the route pattern stored in the request context by the api-gateway router under the "route_pattern" key. Returns an empty string if no pattern was set (e.g. for unmatched routes or non-gateway handlers).
func getRoutePattern(r *http.Request) string {
	if pattern, ok := appctx.GetRoutePattern(r.Context()); ok {
		return pattern
	}
	return ""
}

// statusRecorder wraps an http.ResponseWriter to capture the HTTP status code written by the handler. The captured code is used to set the http.response.status_code span attribute after the handler returns.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and delegates to the wrapped ResponseWriter.
func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// Write delegates to the wrapped ResponseWriter. If WriteHeader was never called (implicit 200), it records http.StatusOK so the span attribute is always set.
func (sr *statusRecorder) Write(b []byte) (int, error) {
	if sr.statusCode == 0 {
		sr.statusCode = http.StatusOK
	}
	return sr.ResponseWriter.Write(b)
}
