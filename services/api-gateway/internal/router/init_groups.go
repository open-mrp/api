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
	saver := middleware.NewRequestLogSaver(config.RequestLogPublisher)
	asyncSaver := middleware.NewAsyncRequestLogSaver(1000, saver)
	loggingMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.LoggingMiddleware(middlewareLogger, next, asyncSaver, r)
	}
	authMiddlewareConfig := middleware.AuthMiddlewareConfig{
		AuthClient: config.AuthClient,
	}

	// Middlewares
	r.AddMiddleware(middleware.TracingMiddleware())
	r.AddMiddleware(middleware.PlatformMiddleware(config.PlatformMode))
	r.AddMiddleware(loggingMiddleware)
	r.AddMiddleware(middleware.CORSMiddleware())
	r.AddMiddleware(middleware.RateLimitMiddleware())
	r.AddMiddleware(middleware.AuthMiddleware(authMiddlewareConfig))
	r.AddMiddleware(middleware.RecoverMiddleware())

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
	saver := middleware.NewRequestLogSaver(config.RequestLogPublisher)
	asyncSaver := middleware.NewAsyncRequestLogSaver(1000, saver)
	loggingMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.LoggingMiddleware(middlewareLogger, next, asyncSaver, r)
	}

	// Middlewares
	r.AddMiddleware(middleware.TracingMiddleware())
	r.AddMiddleware(middleware.PlatformMiddleware(config.PlatformMode))
	r.AddMiddleware(loggingMiddleware)
	r.AddMiddleware(middleware.CORSMiddleware())
	r.AddMiddleware(middleware.RateLimitMiddleware())
	r.AddMiddleware(middleware.AuthSecurityMiddleware())
	r.AddMiddleware(middleware.RecoverMiddleware())

	authGroup := (&httpgroup.AuthEndpointGroup{}).Materialize(httpgroup.AuthEndpointGroupConfig{
		PlatformMode: config.PlatformMode,
		AuthClient:   config.AuthClient,
	})
	if authGroup != nil {
		registry.RegisterGroup(authGroup.APIEndpointGroup)
	}

	registry.RegisterEndpoints(r)
}
