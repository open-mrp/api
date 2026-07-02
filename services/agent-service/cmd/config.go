package main

import (
	"cmp"
	"fmt"
	"strconv"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/env"
)

var (
	defaultPort        = contracts.GRPCPort
	defaultRabbitMQURI = "amqp://guest:guest@rabbitmq:5672/" // #nosec G101 - Default dev URI, not a production credential
)

const (
	envPort                   = "PORT"
	envDBURL                  = "DB_URL"
	envRabbitMQURI            = "RABBITMQ_URI"
	envCoreServiceURL         = "CORE_SERVICE_URL"
	envNotificationServiceURL = "NOTIFICATION_SERVICE_URL"
	envCursorHMACKey          = "CURSOR_HMAC_KEY"   // #nosec G101 - Env var name, not a credential
	envStripeAPIKey           = "STRIPE_SECRET_KEY" // #nosec G101 - Env var name, not a credential
	envAgentArtifactsBucket   = "AGENT_ARTIFACTS_BUCKET"
	envAgentInboundMailBucket = "AGENT_INBOUND_EMAIL_BUCKET"
	envBillingServiceURL      = "BILLING_SERVICE_URL"
	envPlatformMode           = "PLATFORM"
	envGatewayInternalURL     = "API_GATEWAY_INTERNAL_URL"
	envInternalSvcToken       = "INTERNAL_SERVICE_TOKEN" // #nosec G101 - Env var name, not a credential
)

// config represents the configuration for the agent service.
type config struct {
	// Port (optional; default: 9092) specifies the port on which the gRPC server will listen.
	Port int

	// DBURL (required) is the database connection URI.
	DBURL string

	// RabbitMQURI (optional; default: "amqp://guest:guest@rabbitmq:5672/") is the RabbitMQ connection URI.
	RabbitMQURI string

	// PlatformMode (optional; default: "production") determines the platform mode.
	PlatformMode constants.PlatformMode

	// CoreServiceURL (required) is the core service address for gRPC.
	CoreServiceURL string

	// NotificationServiceURL (optional) is the notification-service address for gRPC, used by the
	// agent's email reply/draft tools. When empty, those tools are unavailable.
	NotificationServiceURL string

	// CursorHMACKey (required) is the HMAC key used to sign and verify pagination cursors.
	CursorHMACKey []byte

	// StripeSecretKey (required) is the API key for the Stripe AI Gateway.
	// Not enforced when PlatformMode is "test".
	StripeSecretKey string

	// AgentArtifactsBucket (optional; default: "") is the S3 bucket for storing agent artifacts.
	AgentArtifactsBucket string

	// AgentInboundMailBucket (optional; default: "") is the S3 bucket for inbound agent emails.
	AgentInboundMailBucket string

	// BillingServiceURL (required) is the billing service address for gRPC.
	BillingServiceURL string

	// GatewayInternalURL (optional; default: "") is the api-gateway internal listener base URL used by generated endpoint-tools. When empty (together with InternalServiceToken), endpoint-tools are not wired.
	GatewayInternalURL string

	// InternalServiceToken (optional; default: "") is the shared secret presented to the gateway internal listener.
	InternalServiceToken string
}

func (c *config) withDefaults(getenv func(string) string) *config {
	if c == nil {
		c = &config{}
	}

	port := defaultPort
	if p, err := strconv.Atoi(env.GetEnv(envPort, getenv)); err == nil {
		port = p
	}

	platformMode := constants.PlatformModeProduction
	if p := env.GetEnv(envPlatformMode, getenv); p != "" {
		platformMode = constants.PlatformMode(p)
	}

	return &config{
		Port:                   port,
		DBURL:                  env.GetEnv(envDBURL, getenv),
		RabbitMQURI:            cmp.Or(env.GetEnv(envRabbitMQURI, getenv), defaultRabbitMQURI),
		PlatformMode:           platformMode,
		CoreServiceURL:         env.GetEnv(envCoreServiceURL, getenv),
		NotificationServiceURL: env.GetEnv(envNotificationServiceURL, getenv),
		CursorHMACKey:          []byte(env.GetEnv(envCursorHMACKey, getenv)),
		StripeSecretKey:        env.GetEnv(envStripeAPIKey, getenv),
		AgentArtifactsBucket:   env.GetEnv(envAgentArtifactsBucket, getenv),
		AgentInboundMailBucket: env.GetEnv(envAgentInboundMailBucket, getenv),
		BillingServiceURL:      env.GetEnv(envBillingServiceURL, getenv),
		GatewayInternalURL:     env.GetEnv(envGatewayInternalURL, getenv),
		InternalServiceToken:   env.GetEnv(envInternalSvcToken, getenv),
	}
}

func (c *config) validate() error {
	if c == nil {
		return fmt.Errorf("agent-service: config is nil")
	}
	if c.DBURL == "" {
		return fmt.Errorf("agent-service: the provided database URI is empty")
	}
	if c.CoreServiceURL == "" {
		return fmt.Errorf("agent-service: CORE_SERVICE_URL is required")
	}
	if len(c.CursorHMACKey) == 0 {
		return fmt.Errorf("agent-service: CURSOR_HMAC_KEY is required")
	}
	if !c.PlatformMode.IsValid() {
		return fmt.Errorf("agent-service: the provided platform mode is invalid: %s", c.PlatformMode)
	}
	if !c.PlatformMode.IsTest() && c.StripeSecretKey == "" {
		return fmt.Errorf("agent-service: STRIPE_SECRET_KEY is required")
	}
	if c.BillingServiceURL == "" {
		return fmt.Errorf("agent-service: BILLING_SERVICE_URL is required")
	}
	return nil
}
