package main

import (
	"fmt"
	"strconv"

	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/env"
)

var (
	defaultPort = contracts.GRPCPort
)

const (
	envPort  = "PORT"
	envDBURL = "DB_URL"
)

// config represents the configuration for the core service.
type config struct {
	// Port (optional; default: 9092) specifies the port on which the gRPC server will listen.
	Port int

	// DBURL (required) is the database connection URI.
	DBURL string
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
		Port:  port,
		DBURL: env.GetEnv(envDBURL, getenv),
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
