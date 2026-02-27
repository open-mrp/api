package router

import (
	"log"
	"net/http"
	"time"

	"github.com/augno/api/services/api-gateway/internal/middleware"
	httpgroup "github.com/augno/api/services/api-gateway/pkg/group"
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
	authMiddlewareConfig := &middleware.AuthMiddlewareConfig{
		AuthClient: config.AuthClient,
	}
	idempotencyMiddlewareConfig := &middleware.IdempotencyMiddlewareConfig{
		PlatformClient: config.PlatformClient,
	}

	// Middlewares
	r.AddMiddleware(middleware.TracingMiddleware())
	r.AddMiddleware(middleware.PlatformMiddleware(config.PlatformMode))
	r.AddMiddleware(loggingMiddleware)
	r.AddMiddleware(middleware.CORSMiddleware())
	r.AddMiddleware(middleware.RateLimitMiddleware())
	r.AddMiddleware(middleware.AuthMiddleware(authMiddlewareConfig))
	r.AddMiddleware(middleware.SubscriptionMiddleware())
	r.AddMiddleware(middleware.SandboxBillingMiddleware())
	r.AddMiddleware(middleware.VersionMiddleware())
	r.AddMiddleware(middleware.IdempotencyMiddleware(idempotencyMiddlewareConfig))
	r.AddMiddleware(middleware.RecoverMiddleware())

	// Healthz
	healthGroup := (&httpgroup.HealthEndpointGroup{}).Materialize(httpgroup.HealthEndpointGroupConfig{})
	if healthGroup != nil {
		registry.RegisterGroup(healthGroup.APIEndpointGroup)
	}

	// Sandboxes
	sandboxesGroup := (&httpgroup.SandboxesEndpointGroup{}).Materialize(&httpgroup.SandboxesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if sandboxesGroup != nil {
		registry.RegisterGroup(sandboxesGroup.APIEndpointGroup)
	}

	// Billing
	billingGroup := (&httpgroup.BillingEndpointGroup{}).Materialize(&httpgroup.BillingEndpointGroupConfig{
		BillingClient: config.BillingClient,
	})
	if billingGroup != nil {
		registry.RegisterGroup(billingGroup.APIEndpointGroup)
	}

	// Units
	unitsGroup := (&httpgroup.UnitsEndpointGroup{}).Materialize(&httpgroup.UnitsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if unitsGroup != nil {
		registry.RegisterGroup(unitsGroup.APIEndpointGroup)
	}

	// Request Logs
	requestLogsGroup := (&httpgroup.RequestLogsEndpointGroup{}).Materialize(&httpgroup.RequestLogsEndpointGroupConfig{
		PlatformClient: config.PlatformClient,
	})
	if requestLogsGroup != nil {
		registry.RegisterGroup(requestLogsGroup.APIEndpointGroup)
	}

	registry.RegisterEndpoints(r)
}

func (r *router) InitWebhookEndpointGroups(config WebhookRouterConfig) {
	registry := NewRegistry()

	// Setup middleware — minimal chain for webhooks: no auth, no CORS, no idempotency
	middlewareLogger := log.New(config.LogWriter, config.LogPrefix, config.LogFlags)
	requestLogSaver := middleware.NewRequestLogSaver(config.RequestLogPublisher)
	loggingMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.LoggingMiddleware(middlewareLogger, next, requestLogSaver, r)
	}

	r.AddMiddleware(middleware.TracingMiddleware())
	r.AddMiddleware(middleware.PlatformMiddleware(config.PlatformMode))
	r.AddMiddleware(middleware.RateLimitMiddlewareWithConfig(100, time.Second))
	r.AddMiddleware(loggingMiddleware)
	r.AddMiddleware(middleware.RecoverMiddleware())

	// Webhooks
	webhooksGroup := (&httpgroup.WebhooksEndpointGroup{}).Materialize(&httpgroup.WebhooksEndpointGroupConfig{
		BillingClient: config.BillingClient,
	})
	if webhooksGroup != nil {
		registry.RegisterGroup(webhooksGroup.APIEndpointGroup)
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
	idempotencyMiddlewareConfig := &middleware.IdempotencyMiddlewareConfig{
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
	authGroup := (&httpgroup.AuthEndpointGroup{}).Materialize(&httpgroup.AuthEndpointGroupConfig{
		AuthClient: config.AuthClient,
	})
	if authGroup != nil {
		registry.RegisterGroup(authGroup.APIEndpointGroup)
	}

	// API Keys
	apiKeysGroup := (&httpgroup.APIKeysEndpointGroup{}).Materialize(&httpgroup.APIKeysEndpointGroupConfig{
		AuthClient: config.AuthClient,
	})
	if apiKeysGroup != nil {
		registry.RegisterGroup(apiKeysGroup.APIEndpointGroup)
	}

	// Registration Sessions
	registrationSessionsGroup := (&httpgroup.RegistrationSessionsEndpointGroup{}).Materialize(&httpgroup.RegistrationSessionsEndpointGroupConfig{
		AuthClient: config.AuthClient,
	})
	if registrationSessionsGroup != nil {
		registry.RegisterGroup(registrationSessionsGroup.APIEndpointGroup)
	}

	registry.RegisterEndpoints(r)
}
