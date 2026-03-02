package main

import (
	"cmp"
	"fmt"
	"strconv"

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
	envStripeWebhookSecret    = "STRIPE_WEBHOOK_SECRET" // #nosec G101 - Env var name, not a credential
	envStripeAPIKey           = "STRIPE_SECRET_KEY"
	envCoreServiceURL         = "CORE_SERVICE_URL"
	envNotificationServiceURL = "NOTIFICATION_SERVICE_URL"
	envCursorHMACKey          = "CURSOR_HMAC_KEY"        // #nosec G101 - Env var name, not a credential
	envStripePublishableKey   = "STRIPE_PUBLISHABLE_KEY" // #nosec G101 - Env var name, not a credential
	envFrontendURL            = "FRONTEND_URL"
)

type config struct {
	Port                   int
	DBURL                  string
	RabbitMQURI            string
	StripeWebhookSecret    string
	StripeAPIKey           string
	StripePublishableKey   string
	CoreServiceURL         string
	NotificationServiceURL string
	CursorHMACKey          []byte
	FrontendURL            string
}

func (c *config) withDefaults(getenv func(string) string) *config {
	if c == nil {
		c = &config{}
	}

	port := defaultPort
	if p, err := strconv.Atoi(env.GetEnv(envPort, getenv)); err == nil {
		port = p
	}

	return &config{
		Port:                   port,
		DBURL:                  env.GetEnv(envDBURL, getenv),
		RabbitMQURI:            cmp.Or(env.GetEnv(envRabbitMQURI, getenv), defaultRabbitMQURI),
		StripeWebhookSecret:    env.GetEnv(envStripeWebhookSecret, getenv),
		StripeAPIKey:           env.GetEnv(envStripeAPIKey, getenv),
		StripePublishableKey:   env.GetEnv(envStripePublishableKey, getenv),
		CoreServiceURL:         env.GetEnv(envCoreServiceURL, getenv),
		NotificationServiceURL: env.GetEnv(envNotificationServiceURL, getenv),
		CursorHMACKey:          []byte(env.GetEnv(envCursorHMACKey, getenv)),
		FrontendURL:            env.GetEnv(envFrontendURL, getenv),
	}
}

func (c *config) validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.DBURL == "" {
		return fmt.Errorf("the provided database URI is empty")
	}
	if c.StripeWebhookSecret == "" {
		return fmt.Errorf("STRIPE_WEBHOOK_SECRET is required")
	}
	if c.StripeAPIKey == "" {
		return fmt.Errorf("STRIPE_SECRET_KEY is required")
	}
	if c.StripePublishableKey == "" {
		return fmt.Errorf("STRIPE_PUBLISHABLE_KEY is required")
	}
	if c.CoreServiceURL == "" {
		return fmt.Errorf("CORE_SERVICE_URL is required")
	}
	if c.NotificationServiceURL == "" {
		return fmt.Errorf("NOTIFICATION_SERVICE_URL is required")
	}
	if len(c.CursorHMACKey) == 0 {
		return fmt.Errorf("CURSOR_HMAC_KEY is required")
	}
	if c.FrontendURL == "" {
		return fmt.Errorf("FRONTEND_URL is required")
	}
	return nil
}
