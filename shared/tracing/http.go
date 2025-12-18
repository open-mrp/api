package tracing

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	attrHTTPMethod     = "http.method"
	attrHTTPRoute      = "http.route"
	attrHTTPPath       = "http.url.path"
	attrHTTPStatusCode = "http.status_code"
	attrHTTPQuery      = "http.url.query"
	attrServerAddress  = "server.address"
	attrUserAgent      = "user_agent.original"
)

// WrapGatewayHandler instruments the gateway HTTP handler.
// Root span is created by otelhttp; we enrich it with route, path, and status.
func WrapGatewayHandler(handler http.Handler, opts ...otelhttp.Option) http.Handler {
	options := append([]otelhttp.Option{
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return fmt.Sprintf("HTTP %s %s", r.Method, routeOrPath(r))
		}),
		otelhttp.WithPublicEndpointFn(func(*http.Request) bool { return true }),
		otelhttp.WithServerName("api-gateway"),
	}, opts...)

	instrumented := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		handler.ServeHTTP(rw, r)

		span := trace.SpanFromContext(r.Context())
		if !span.IsRecording() {
			return
		}

		route := routeOrPath(r)
		attrs := []attribute.KeyValue{
			attribute.String(attrHTTPMethod, r.Method),
			attribute.String(attrHTTPRoute, route),
			attribute.String(attrHTTPPath, r.URL.Path),
			attribute.Int(attrHTTPStatusCode, rw.statusCode),
		}
		if host := r.Host; host != "" {
			attrs = append(attrs, attribute.String(attrServerAddress, host))
		}
		if rawQuery := r.URL.RawQuery; rawQuery != "" {
			attrs = append(attrs, attribute.String(attrHTTPQuery, rawQuery))
		}
		if ua := r.UserAgent(); ua != "" {
			attrs = append(attrs, attribute.String(attrUserAgent, ua))
		}

		span.SetAttributes(attrs...)
	})

	return otelhttp.NewHandler(instrumented, "", options...)
}

func routeOrPath(r *http.Request) string {
	if routePattern := getRoutePattern(r); routePattern != "" {
		return routePattern
	}
	return r.URL.Path
}

// getRoutePattern extracts the route pattern from request context if available
func getRoutePattern(r *http.Request) string {
	// Try to get route pattern from context (set by router)
	// The api-gateway uses "route_pattern" as the context key
	if v := r.Context().Value("route_pattern"); v != nil {
		if pattern, ok := v.(string); ok {
			return pattern
		}
	}
	return ""
}

// statusRecorder records the HTTP status code for tracing
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	if sr.statusCode == 0 {
		sr.statusCode = http.StatusOK
	}
	return sr.ResponseWriter.Write(b)
}
