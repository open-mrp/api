package router

import (
	"log"
	"net/http"

	"github.com/augno/api/services/api-gateway/internal/middleware"
	httpgroup "github.com/augno/api/services/api-gateway/internal/router/group"
)

func (r *router) InitEndpointGroups(config MainRouterConfig) {
	registry := NewRegistry()

	// Main endpoints backed by the Auth gRPC service.
	if config.AuthClient == nil {
		panic("Main router: Auth client is a nil pointer")
	}

	// Setup middleware
	middlewareLogger := log.New(config.LogWriter, config.LogPrefix, config.LogFlags)
	requestLogSaver := middleware.NewRequestLogSaver(config.RequestLogPublisher)
	loggingMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.LoggingMiddleware(middlewareLogger, next, requestLogSaver, r)
	}
	authMiddlewareConfig := middleware.AuthMiddlewareConfig{
		AuthClient: config.AuthClient,
	}
	idempotencyMiddlewareConfig := middleware.IdempotencyMiddlewareConfig{
		PlatformClient: config.PlatformClient,
	}

	// Middlewares
	r.AddMiddleware(middleware.TracingMiddleware())
	r.AddMiddleware(middleware.PlatformMiddleware(config.PlatformMode))
	r.AddMiddleware(loggingMiddleware)
	r.AddMiddleware(middleware.CORSMiddleware())
	r.AddMiddleware(middleware.RateLimitMiddleware())
	r.AddMiddleware(middleware.AuthMiddleware(authMiddlewareConfig))
	r.AddMiddleware(middleware.VersionMiddleware())
	r.AddMiddleware(middleware.IdempotencyMiddleware(idempotencyMiddlewareConfig))
	r.AddMiddleware(middleware.RecoverMiddleware())

	// Healthz
	healthGroup := (&httpgroup.HealthEndpointGroup{}).Materialize(httpgroup.HealthEndpointGroupConfig{})
	if healthGroup != nil {
		registry.RegisterGroup(healthGroup.APIEndpointGroup)
	}

	registry.RegisterEndpoints(r)
}

func (r *router) InitAuthEndpointGroups(config AuthRouterConfig) {
	registry := NewRegistry()

	// Auth endpoints backed by the Auth gRPC service.
	if config.AuthClient == nil {
		panic("Auth router: Auth client is a nil pointer")
	}

	middlewareLogger := log.New(config.LogWriter, config.LogPrefix, config.LogFlags)
	requestLogSaver := middleware.NewRequestLogSaver(config.RequestLogPublisher)
	loggingMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.LoggingMiddleware(middlewareLogger, next, requestLogSaver, r)
	}
	idempotencyMiddlewareConfig := middleware.IdempotencyMiddlewareConfig{
		PlatformClient: config.PlatformClient,
	}

	// Middlewares
	r.AddMiddleware(middleware.TracingMiddleware())
	r.AddMiddleware(middleware.PlatformMiddleware(config.PlatformMode))
	r.AddMiddleware(loggingMiddleware)
	r.AddMiddleware(middleware.CORSMiddleware())
	r.AddMiddleware(middleware.RateLimitMiddleware())
	r.AddMiddleware(middleware.AuthSecurityMiddleware())
	r.AddMiddleware(middleware.VersionMiddleware())
	r.AddMiddleware(middleware.IdempotencyMiddleware(idempotencyMiddlewareConfig))
	r.AddMiddleware(middleware.RecoverMiddleware())

	// Auth
	authGroup := (&httpgroup.AuthEndpointGroup{}).Materialize(httpgroup.AuthEndpointGroupConfig{
		AuthClient: config.AuthClient,
	})
	if authGroup != nil {
		registry.RegisterGroup(authGroup.APIEndpointGroup)
	}

	registry.RegisterEndpoints(r)
}
