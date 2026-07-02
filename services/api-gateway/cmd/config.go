package main

import (
	"cmp"
	"fmt"
	"strconv"

	constants "github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/env"
)

var (
	defaultPort                   = 8081
	defaultInternalPort           = 8091
	defaultAuthServiceURI         = fmt.Sprintf("auth-service:%d", contracts.GRPCPort)
	defaultCoreServiceURI         = fmt.Sprintf("core-service:%d", contracts.GRPCPort)
	defaultBillingServiceURI      = fmt.Sprintf("billing-service:%d", contracts.GRPCPort)
	defaultPlatformServiceURI     = fmt.Sprintf("platform-service:%d", contracts.GRPCPort)
	defaultAgentServiceURI        = fmt.Sprintf("agent-service:%d", contracts.GRPCPort)
	defaultNotificationServiceURI = fmt.Sprintf("notification-service:%d", contracts.GRPCPort)
	defaultRabbitMQURI            = "amqp://guest:guest@rabbitmq:5672/" // #nosec G101 - Default dev URI, not a production credential
	defaultPlatformMode           = constants.PlatformModeProduction
)

const (
	envPlatformMode           = "PLATFORM"
	envPort                   = "PORT"
	envInternalPort           = "INTERNAL_PORT"
	envInternalSvcToken       = "INTERNAL_SERVICE_TOKEN"
	envAuthServiceURL         = "AUTH_SERVICE_URL"
	envCoreServiceURL         = "CORE_SERVICE_URL"
	envBillingServiceURL      = "BILLING_SERVICE_URL"
	envPlatformServiceURL     = "PLATFORM_SERVICE_URL"
	envAgentServiceURL        = "AGENT_SERVICE_URL"
	envNotificationServiceURL = "NOTIFICATION_SERVICE_URL"
	envDBURL                  = "DB_URL"
	envRabbitMQURI            = "RABBITMQ_URI"
	envFrontendURL            = "FRONTEND_URL"
	envTrustedProxyHops       = "TRUSTED_PROXY_HOPS"
)

// config represents the configuration for the API gateway.
type config struct {
	// PlatformMode (optional; default: "production") specifies the mode the platform is running in.
	PlatformMode constants.PlatformMode

	// Port (optional; default: 8081) specifies the port on which the HTTP server will listen.
	Port int

	// InternalPort (optional; default: 8091) is the port for the trusted internal listener that serves agent traffic. Only started when InternalServiceToken is set.
	InternalPort int

	// InternalServiceToken (optional; default: "") is the shared secret that gates the internal listener's identity trust. When empty, the internal listener is not started.
	InternalServiceToken string

	// AuthServiceURI (optional; default: "auth-service:9092") is the auth service address for gRPC.
	AuthServiceURI string

	// CoreServiceURI (optional; default: "core-service:9092") is the core service address for gRPC.
	CoreServiceURI string

	// BillingServiceURI (optional; default: "billing-service:9092") is the billing service address for gRPC.
	BillingServiceURI string

	// PlatformServiceURI (optional; default: "platform-service:9092") is the platform service address for gRPC.
	PlatformServiceURI string

	// AgentServiceURI (optional; default: "agent-service:9092") is the agent service address for gRPC.
	AgentServiceURI string

	// NotificationServiceURI (optional; default: "notification-service:9092") is the notification service address for gRPC.
	NotificationServiceURI string

	// DBURI (required) is the database connection URI.
	DBURI string

	// RabbitMQURI (optional; default: "amqp://guest:guest@rabbitmq:5672/") is the RabbitMQ connection URI.
	RabbitMQURI string

	// FrontendURL (optional; default: "") is the base URL of the frontend application, used to build
	// request log links in error responses. When empty, request_log_url will be null.
	FrontendURL string

	// TrustedProxyHops (optional; default: 0) specifies how many reverse-proxy
	// hops sit in front of this service. Each trusted proxy is expected to
	// append the IP it observed as the TCP source to X-Forwarded-For. When set
	// to 0 the X-Forwarded-For header is ignored entirely. In production behind
	// AWS ALB this should be 1.
	TrustedProxyHops int
}

// withDefaults sets the default values for the configuration.
func (c *config) withDefaults(getenv func(string) string) *config {
	if c == nil {
		c = &config{}
	}

	platform := env.GetEnv(envPlatformMode, getenv)
	platformMode := defaultPlatformMode
	if platform != "" {
		platformMode = constants.PlatformMode(platform)
	}

	port := defaultPort
	if p, err := strconv.Atoi(env.GetEnv(envPort, getenv)); err == nil {
		port = p
	}

	internalPort := defaultInternalPort
	if p, err := strconv.Atoi(env.GetEnv(envInternalPort, getenv)); err == nil {
		internalPort = p
	}

	trustedProxyHops := 0
	if h, err := strconv.Atoi(env.GetEnv(envTrustedProxyHops, getenv)); err == nil {
		trustedProxyHops = h
	}

	return &config{
		PlatformMode:           platformMode,
		Port:                   port,
		InternalPort:           internalPort,
		InternalServiceToken:   env.GetEnv(envInternalSvcToken, getenv),
		AuthServiceURI:         cmp.Or(env.GetEnv(envAuthServiceURL, getenv), defaultAuthServiceURI),
		CoreServiceURI:         cmp.Or(env.GetEnv(envCoreServiceURL, getenv), defaultCoreServiceURI),
		BillingServiceURI:      cmp.Or(env.GetEnv(envBillingServiceURL, getenv), defaultBillingServiceURI),
		PlatformServiceURI:     cmp.Or(env.GetEnv(envPlatformServiceURL, getenv), defaultPlatformServiceURI),
		AgentServiceURI:        cmp.Or(env.GetEnv(envAgentServiceURL, getenv), defaultAgentServiceURI),
		NotificationServiceURI: cmp.Or(env.GetEnv(envNotificationServiceURL, getenv), defaultNotificationServiceURI),
		DBURI:                  env.GetEnv(envDBURL, getenv),
		RabbitMQURI:            cmp.Or(env.GetEnv(envRabbitMQURI, getenv), defaultRabbitMQURI),
		FrontendURL:            env.GetEnv(envFrontendURL, getenv),
		TrustedProxyHops:       trustedProxyHops,
	}
}

// validate validates the configuration.
func (c *config) validate() error {
	if c == nil {
		return fmt.Errorf("api-gateway: config is nil")
	}
	if c.DBURI == "" {
		return fmt.Errorf("api-gateway: the provided database URI is empty")
	}
	if !c.PlatformMode.IsValid() {
		return fmt.Errorf("api-gateway: the provided platform mode is invalid: %s", c.PlatformMode)
	}
	if c.TrustedProxyHops < 0 {
		return fmt.Errorf("api-gateway: trusted proxy hops must be non-negative, got %d", c.TrustedProxyHops)
	}
	return nil
}
