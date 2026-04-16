package middleware

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/augno/api/services/api-gateway/internal/header"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var loggingMiddlewareTracer = tracing.GetTracer("api-gateway.logging_middleware")

// excludedRoutes are normalized route patterns that should be marked as non-public
// regardless of the endpoint's Public flag. These are automation or internal-only
// routes that should not appear in the customer-facing request log listing.
var excludedRoutes = map[string]struct{}{
	"/v1/identity/me":                             {},
	"/v1/tenancy/me/tenancy":                      {},
	"/v1/auth/refresh-tokens":                     {},
	"/v1/core/request-logs":                       {},
	"/v1/auth/api-keys/actions/fetch-doc-api-key": {},
}

type RouteMatcher interface {
	GetRoutes() []any
}

type routeMatch struct {
	template string
	isPublic bool
}

func findRouteTemplate(router any, method, path string) routeMatch {
	routeMatcher, ok := router.(RouteMatcher)
	if !ok {
		return routeMatch{}
	}

	routes := routeMatcher.GetRoutes()
	for _, routeInterface := range routes {
		routeMap, ok := routeInterface.(map[string]any)
		if !ok {
			continue
		}

		routeMethod, ok := routeMap["Method"].(string)
		if !ok || routeMethod != method {
			continue
		}

		routePath, ok := routeMap["Path"].(string)
		if !ok {
			continue
		}

		matched := false
		if routePath == path {
			matched = true
		} else if pattern, ok := routeMap["PathPattern"].(*regexp.Regexp); ok && pattern != nil && pattern.MatchString(path) {
			matched = true
		}

		if matched {
			isPublic, _ := routeMap["Public"].(bool)
			return routeMatch{template: routePath, isPublic: isPublic}
		}
	}

	return routeMatch{}
}

func LoggingMiddleware(logger *log.Logger, next http.HandlerFunc, saver saver, router any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now().UTC()

		length := id.IDLength19
		requestID, err := id.GenID(id.RequestIDPrefix, &length)
		if err != nil {
			panic(err)
		}

		userAgent := r.UserAgent()
		clientIP := header.GetClientIP(r)

		normalizedRoute := r.URL.Path
		publicEndpoint := true
		if router != nil {
			if match := findRouteTemplate(router, r.Method, r.URL.Path); match.template != "" {
				normalizedRoute = match.template
				publicEndpoint = match.isPublic
			}
		}
		if _, excluded := excludedRoutes[normalizedRoute]; excluded {
			publicEndpoint = false
		}

		// Capture API version from header for logging
		var apiVersion *string
		if versionStr := r.Header.Get(header.VersionHeader); versionStr != "" {
			apiVersion = &versionStr
		}

		referrer := r.Referer()

		requestLog := &appctx.RequestLog{
			ID:              requestID,
			Method:          r.Method,
			Host:            r.Host,
			Path:            r.URL.Path,
			NormalizedRoute: normalizedRoute,
			UserAgent:       &userAgent,
			ClientIP:        clientIP,
			ClientIPString: func() *string {
				if len(clientIP) == 0 {
					return nil
				}
				s := clientIP.String()
				return &s
			}(),
			Referrer:       &referrer,
			OccurredAt:     start,
			APIVersion:     apiVersion,
			PublicEndpoint: publicEndpoint,
		}

		span := trace.SpanFromContext(r.Context())
		if span.SpanContext().IsValid() {
			span.SetAttributes(attribute.String("request.id", requestID))
			traceID := span.SpanContext().TraceID().String()
			requestLog.TraceID = &traceID
		}

		ctx := appctx.WithRequestLog(r.Context(), requestLog)
		r = r.WithContext(ctx)

		lrw := newLoggingResponseWriter(w)

		defer func() {
			if r.URL.Path == "/healthz" || r.Method == http.MethodOptions {
				return
			}

			latency := int64(time.Since(start).Microseconds())
			requestLog.StatusCode = lrw.statusCode
			requestLog.LatencyUs = latency

			if !requestLog.ShieldResponseBody && len(lrw.body) > 0 {
				if lrw.bodyFull {
					s := string(lrw.body)
					requestLog.ResponseJSON = &s
				} else {
					s := fmt.Sprintf(`{"_truncated":true,"_original_size_exceeded":%d}`, maxResponseLogSize)
					requestLog.ResponseJSON = &s
				}
			}

			if saver != nil && !requestLog.SkipSave {
				_, span := loggingMiddlewareTracer.Start(ctx, "middleware.logging.save_request_log")
				if err := saver.Save(ctx, requestLog); err != nil {
					logger.Printf("Error saving request log: %v", err)
				}
				span.End()
			}

			milliseconds := latency / 1000
			microseconds := latency % 1000
			logger.Printf(
				"[%s] %s %s %s %d %d.%03dms",
				requestID,
				r.Method,
				r.URL.Path,
				r.Proto,
				lrw.statusCode,
				milliseconds,
				microseconds,
			)
		}()

		next.ServeHTTP(lrw, r)
	}
}

const maxResponseLogSize = 256 << 10 // 256 KB

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
	body       []byte
	bodyFull   bool // true when the full response fit within maxResponseLogSize
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	if !lrw.written {
		lrw.statusCode = code
		lrw.ResponseWriter.WriteHeader(code)
		lrw.written = true
	}
}

func (lrw *loggingResponseWriter) Write(data []byte) (int, error) {
	if !lrw.written {
		lrw.WriteHeader(http.StatusOK)
	}
	if len(lrw.body) < maxResponseLogSize {
		remaining := maxResponseLogSize - len(lrw.body)
		if len(data) <= remaining {
			lrw.body = append(lrw.body, data...)
			lrw.bodyFull = true
		} else {
			lrw.body = append(lrw.body, data[:remaining]...)
			lrw.bodyFull = false
		}
	} else {
		lrw.bodyFull = false
	}
	return lrw.ResponseWriter.Write(data)
}
