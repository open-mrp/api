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
	defaultPort         = contracts.GRPCPort
	defaultPlatformMode = constants.PlatformModeProduction
	defaultRabbitMQURI  = "amqp://guest:guest@rabbitmq:5672/" // #nosec G101 - Default dev URI, not a production credential
)

const (
	envPort         = "PORT"
	envPlatformMode = "PLATFORM"
	envDBURL        = "DB_URL"
	envAWSRegion    = "AWS_REGION"
	envRabbitMQURI  = "RABBITMQ_URI"
)

// config represents the configuration for the notification service.
type config struct {
	// Port (optional; default: 9092) specifies the port on which the gRPC server will listen.
	Port int

	// PlatformMode (optional; default: "production") determines the platform mode.
	PlatformMode constants.PlatformMode

	// DBURL (required) is the database connection URI.
	DBURL string

	// AWSRegion (required) is the AWS region for SES email delivery.
	AWSRegion string

	// RabbitMQURI (optional; default: "amqp://guest:guest@rabbitmq:5672/") is the RabbitMQ connection URI.
	RabbitMQURI string
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
		Port:         port,
		PlatformMode: platformMode,
		DBURL:        env.GetEnv(envDBURL, getenv),
		AWSRegion:    env.GetEnv(envAWSRegion, getenv),
		RabbitMQURI:  cmp.Or(env.GetEnv(envRabbitMQURI, getenv), defaultRabbitMQURI),
	}
}

// validate validates the configuration.
func (c *config) validate() error {
	if c == nil {
		return fmt.Errorf("notification-service: config is nil")
	}
	if c.DBURL == "" {
		return fmt.Errorf("notification-service: the provided database URI is empty")
	}
	if !c.PlatformMode.IsValid() {
		return fmt.Errorf("notification-service: the provided platform mode is invalid: %s", c.PlatformMode)
	}
	if !c.PlatformMode.IsTest() && c.AWSRegion == "" {
		return fmt.Errorf("notification-service: the provided AWS region is empty")
	}
	return nil
}
