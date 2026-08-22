package main

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/env"
)

var (
	defaultPort        = contracts.GRPCPort
	defaultRabbitMQURI = "amqp://guest:guest@rabbitmq:5672/" // #nosec G101 - Default dev URI, not a production credential
)

const (
	envPort                       = "PORT"
	envDBURL                      = "DB_URL"
	envRabbitMQURI                = "RABBITMQ_URI"
	envStripeWebhookSecret        = "STRIPE_WEBHOOK_SECRET" // #nosec G101 - Env var name, not a credential
	envStripeAPIKey               = "STRIPE_SECRET_KEY"
	envCoreServiceURL             = "CORE_SERVICE_URL"
	envNotificationServiceURL     = "NOTIFICATION_SERVICE_URL"
	envCursorHMACKey              = "CURSOR_HMAC_KEY" // #nosec G101 - Env var name, not a credential
	envFrontendURL                = "FRONTEND_URL"
	envStripeWebhookVerboseErrors = "STRIPE_WEBHOOK_VERBOSE_ERRORS"
	envStripePublishableKey       = "STRIPE_PUBLISHABLE_KEY" // #nosec G101 - Env var name, not a credential
	envPlatformMode               = "PLATFORM"
)

// config represents the configuration for the billing service.
type config struct {
	// Port (optional; default: 9092) specifies the port on which the gRPC server will listen.
	Port int

	// DBURL (required) is the database connection URI.
	DBURL string

	// RabbitMQURI (optional; default: "amqp://guest:guest@rabbitmq:5672/") is the RabbitMQ connection URI.
	RabbitMQURI string

	// PlatformMode (optional; default: "production") determines the platform mode.
	PlatformMode constants.PlatformMode

	// StripeWebhookSecret (required) is the Stripe webhook signing secret.
	// Not enforced when PlatformMode is "test".
	StripeWebhookSecret string

	// StripeSecretKey (required) is the Stripe secret API key.
	// Not enforced when PlatformMode is "test".
	StripeSecretKey string

	// CoreServiceURL (required) is the core service address for gRPC.
	CoreServiceURL string

	// NotificationServiceURL (required) is the notification service address for gRPC.
	NotificationServiceURL string

	// CursorHMACKey (required) is the HMAC key used to sign and verify pagination cursors.
	CursorHMACKey []byte

	// FrontendURL (required) is the base URL of the frontend application.
	FrontendURL string

	// StripeWebhookVerboseErrors (optional; default: false) when true, 400 responses for webhook signature
	// failures include the underlying Stripe error in the message (for development).
	StripeWebhookVerboseErrors bool

	// StripePublishableKey (required) is the Stripe publishable key sent to clients
	// for Stripe.js initialization.
	// Not enforced when PlatformMode is "test".
	StripePublishableKey string
}

func (c *config) withDefaults(getenv func(string) string) *config {
	if c == nil {
		c = &config{}
	}

	port := defaultPort
	if p, err := strconv.Atoi(env.GetEnv(envPort, getenv)); err == nil {
		port = p
	}

	v := env.GetEnv(envStripeWebhookVerboseErrors, getenv)
	verboseWebhookErrors := strings.EqualFold(v, "true") || v == "1"

	platformMode := constants.PlatformModeProduction
	if p := env.GetEnv(envPlatformMode, getenv); p != "" {
		platformMode = constants.PlatformMode(p)
	}

	return &config{
		Port:                       port,
		DBURL:                      env.GetEnv(envDBURL, getenv),
		RabbitMQURI:                cmp.Or(env.GetEnv(envRabbitMQURI, getenv), defaultRabbitMQURI),
		PlatformMode:               platformMode,
		StripeWebhookSecret:        env.GetEnv(envStripeWebhookSecret, getenv),
		StripeSecretKey:            env.GetEnv(envStripeAPIKey, getenv),
		CoreServiceURL:             env.GetEnv(envCoreServiceURL, getenv),
		NotificationServiceURL:     env.GetEnv(envNotificationServiceURL, getenv),
		CursorHMACKey:              []byte(env.GetEnv(envCursorHMACKey, getenv)),
		FrontendURL:                env.GetEnv(envFrontendURL, getenv),
		StripeWebhookVerboseErrors: verboseWebhookErrors,
		StripePublishableKey:       env.GetEnv(envStripePublishableKey, getenv),
	}
}

func (c *config) validate() error {
	if c == nil {
		return fmt.Errorf("billing-service: config is nil")
	}
	if !c.PlatformMode.IsValid() {
		return fmt.Errorf("billing-service: the provided platform mode is invalid: %s", c.PlatformMode)
	}
	if c.DBURL == "" {
		return fmt.Errorf("billing-service: the provided database URI is empty")
	}
	if !c.PlatformMode.IsTest() {
		if c.StripeWebhookSecret == "" {
			return fmt.Errorf("billing-service: STRIPE_WEBHOOK_SECRET is required")
		}
		if c.StripeSecretKey == "" {
			return fmt.Errorf("billing-service: STRIPE_SECRET_KEY is required")
		}
		if c.StripePublishableKey == "" {
			return fmt.Errorf("billing-service: STRIPE_PUBLISHABLE_KEY is required")
		}
	}
	if c.CoreServiceURL == "" {
		return fmt.Errorf("billing-service: CORE_SERVICE_URL is required")
	}
	if c.NotificationServiceURL == "" {
		return fmt.Errorf("billing-service: NOTIFICATION_SERVICE_URL is required")
	}
	if len(c.CursorHMACKey) == 0 {
		return fmt.Errorf("billing-service: CURSOR_HMAC_KEY is required")
	}
	if c.FrontendURL == "" {
		return fmt.Errorf("billing-service: FRONTEND_URL is required")
	}
	return nil
}
