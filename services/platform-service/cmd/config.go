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
	envPort        = "PORT"
	envDBURL       = "DB_URL"
	envRabbitMQURI = "RABBITMQ_URI"
)

// config represents the configuration for the platform service.
type config struct {
	// Port (optional; default: 9092) specifies the port on which the gRPC server will listen.
	Port int

	// DBURL (required) is the database connection URI.
	DBURL string

	// RabbitMQURI (optional; default: "amqp://guest:guest@rabbitmq:5672/") is the RabbitMQ connection URI.
	RabbitMQURI string
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

	return &config{
		Port:        port,
		DBURL:       env.GetEnv(envDBURL, getenv),
		RabbitMQURI: cmp.Or(env.GetEnv(envRabbitMQURI, getenv), defaultRabbitMQURI),
	}
}

// validate validates the configuration.
func (c *config) validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.DBURL == "" {
		return fmt.Errorf("the provided database URI is empty")
	}
	return nil
}
