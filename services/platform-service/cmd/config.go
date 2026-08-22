package main

import (
	"cmp"
	"fmt"
	"strconv"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/env"
)

var (
	defaultPort        = contracts.GRPCPort
	defaultRabbitMQURI = "amqp://guest:guest@rabbitmq:5672/" // #nosec G101 - Default dev URI, not a production credential
)

const (
	envPort          = "PORT"
	envDBURL         = "DB_URL"
	envRabbitMQURI   = "RABBITMQ_URI"
	envCursorHMACKey = "CURSOR_HMAC_KEY"
	envPlatformMode  = "PLATFORM"
)

// config represents the configuration for the platform service.
type config struct {
	// Port (optional; default: 9092) specifies the port on which the gRPC server will listen.
	Port int

	// DBURL (required) is the database connection URI.
	DBURL string

	// RabbitMQURI (optional; default: "amqp://guest:guest@rabbitmq:5672/") is the RabbitMQ connection URI.
	RabbitMQURI string

	// CursorHMACKey (required) is the HMAC key used to sign and verify pagination cursors.
	CursorHMACKey []byte

	// PlatformMode (optional; default: "production") determines the platform mode.
	PlatformMode constants.PlatformMode
}

// withDefaults sets the default values for the configuration.
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
		Port:          port,
		DBURL:         env.GetEnv(envDBURL, getenv),
		RabbitMQURI:   cmp.Or(env.GetEnv(envRabbitMQURI, getenv), defaultRabbitMQURI),
		CursorHMACKey: []byte(env.GetEnv(envCursorHMACKey, getenv)),
		PlatformMode:  platformMode,
	}
}

// validate validates the configuration.
func (c *config) validate() error {
	if c == nil {
		return fmt.Errorf("platform-service: config is nil")
	}
	if !c.PlatformMode.IsValid() {
		return fmt.Errorf("platform-service: the provided platform mode is invalid: %s", c.PlatformMode)
	}
	if c.DBURL == "" {
		return fmt.Errorf("platform-service: the provided database URI is empty")
	}
	if len(c.CursorHMACKey) == 0 {
		return fmt.Errorf("platform-service: CURSOR_HMAC_KEY is required")
	}
	return nil
}
