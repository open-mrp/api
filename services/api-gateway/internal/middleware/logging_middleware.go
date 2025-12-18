package middleware

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"time"

	apicontext "github.com/augno/api/services/api-gateway/internal/context"
	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/services/api-gateway/internal/header"
	"github.com/augno/api/shared/id"
)

type RouteMatcher interface {
	GetRoutes() []any
}

func findRouteTemplate(router any, method, path string) string {
	routeMatcher, ok := router.(RouteMatcher)
	if !ok {
		return ""
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

		if routePath == path {
			return routePath
		}

		if pattern, ok := routeMap["PathPattern"].(*regexp.Regexp); ok && pattern != nil && pattern.MatchString(path) {
			return routePath
		}
	}

	return ""
}

func LoggingMiddleware(logger *log.Logger, next http.HandlerFunc, saver *asyncRequestLogSaver, router any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now().UTC()

		length := id.IDLength19
		requestID, err := id.GenID(id.RequestIDPrefix, &length)
		if err != nil {
			panic(err)
		}

		userAgent := r.UserAgent()
		idempotencyKey := r.Header.Get("Idempotency-Key")
		clientIP := header.GetClientIP(r)

		normalizedRoute := r.URL.Path
		if router != nil {
			if routeTemplate := findRouteTemplate(router, r.Method, r.URL.Path); routeTemplate != "" {
				normalizedRoute = routeTemplate
			}
		}

		requestLog := &domain.RequestLog{
			ID:              requestID,
			Method:          r.Method,
			Host:            r.Host,
			Path:            r.URL.Path,
			NormalizedRoute: normalizedRoute,
			UserAgent:       userAgent,
			IdempotencyKey:  idempotencyKey,
			ClientIP:        clientIP,
			ClientIPString: func() string {
				if len(clientIP) == 0 {
					return ""
				}
				return clientIP.String()
			}(),
			OccurredAt: start,
		}

		ctx := context.WithValue(r.Context(), apicontext.RequestLogKey, requestLog)
		r = r.WithContext(ctx)

		lrw := newLoggingResponseWriter(w)

		defer func() {
			latency := int64(time.Since(start).Microseconds())
			requestLog.StatusCode = lrw.statusCode
			requestLog.Latency = latency

			if saver != nil {
				if err := saver.Save(ctx, requestLog); err != nil {
					logger.Printf("Error saving request log: %v", err)
				}
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

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK, false}
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
	return lrw.ResponseWriter.Write(data)
}
