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
	defaultPort               = 8081
	defaultAuthServiceURI     = fmt.Sprintf("auth-service:%d", contracts.GRPCPort)
	defaultCoreServiceURI     = fmt.Sprintf("core-service:%d", contracts.GRPCPort)
	defaultBillingServiceURI  = fmt.Sprintf("billing-service:%d", contracts.GRPCPort)
	defaultPlatformServiceURI = fmt.Sprintf("platform-service:%d", contracts.GRPCPort)
	defaultRabbitMQURI        = "amqp://guest:guest@rabbitmq:5672/" // #nosec G101 - Default dev URI, not a production credential
	defaultPlatformMode       = constants.PlatformModeProduction
)

const (
	envPlatformMode       = "PLATFORM"
	envPort               = "PORT"
	envAuthServiceURL     = "AUTH_SERVICE_URL"
	envCoreServiceURL     = "CORE_SERVICE_URL"
	envBillingServiceURL  = "BILLING_SERVICE_URL"
	envPlatformServiceURL = "PLATFORM_SERVICE_URL"
	envDBURL              = "DB_URL"
	envRabbitMQURI        = "RABBITMQ_URI"
	envFrontendURL        = "FRONTEND_URL"
)

// config represents the configuration for the API gateway.
type config struct {
	// PlatformMode (optional; default: "production") specifies the mode the platform is running in.
	PlatformMode constants.PlatformMode

	// Port (optional; default: 8081) specifies the port on which the HTTP server will listen.
	Port int

	// AuthServiceURI (optional; default: "auth-service:9092") is the auth service address for gRPC.
	AuthServiceURI string

	// CoreServiceURI (optional; default: "core-service:9092") is the core service address for gRPC.
	CoreServiceURI string

	// BillingServiceURI (optional; default: "billing-service:9092") is the billing service address for gRPC.
	BillingServiceURI string

	// PlatformServiceURI (optional; default: "platform-service:9092") is the platform service address for gRPC.
	PlatformServiceURI string

	// DBURI (required) is the database connection URI.
	DBURI string

	// RabbitMQURI (optional; default: "amqp://guest:guest@rabbitmq:5672/") is the RabbitMQ connection URI.
	RabbitMQURI string

	// FrontendURL (optional) is the base URL of the frontend application, used to build
	// request log links in error responses. When empty, request_log_url will be null.
	FrontendURL string
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

	return &config{
		PlatformMode:       platformMode,
		Port:               port,
		AuthServiceURI:     cmp.Or(env.GetEnv(envAuthServiceURL, getenv), defaultAuthServiceURI),
		CoreServiceURI:     cmp.Or(env.GetEnv(envCoreServiceURL, getenv), defaultCoreServiceURI),
		BillingServiceURI:  cmp.Or(env.GetEnv(envBillingServiceURL, getenv), defaultBillingServiceURI),
		PlatformServiceURI: cmp.Or(env.GetEnv(envPlatformServiceURL, getenv), defaultPlatformServiceURI),
		DBURI:              env.GetEnv(envDBURL, getenv),
		RabbitMQURI:        cmp.Or(env.GetEnv(envRabbitMQURI, getenv), defaultRabbitMQURI),
		FrontendURL:        env.GetEnv(envFrontendURL, getenv),
	}
}

// validate validates the configuration.
func (c *config) validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.DBURI == "" {
		return fmt.Errorf("the provided database URI is empty")
	}
	if !c.PlatformMode.IsValid() {
		return fmt.Errorf("the provided platform mode is invalid: %s", c.PlatformMode)
	}
	return nil
}
