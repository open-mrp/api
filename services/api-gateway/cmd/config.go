package main

import (
	"fmt"
	"strconv"

	constants "github.com/augno/api/shared/constants"
)

type config struct {
	PlatformMode   constants.PlatformMode
	Port           int
	Version        string
	AuthServiceURL string
	RabbitMQURI    string
}

func loadConfig(getenv func(string) string) (*config, error) {
	platform := getenv("PLATFORM")
	if platform == "" {
		return nil, fmt.Errorf("required environment variable PLATFORM is not set")
	}

	platformMode := constants.PlatformMode(platform)
	if !platformMode.IsValid() {
		return nil, fmt.Errorf("PLATFORM must be a valid platform mode, got: %s", platform)
	}

	portStr := getenv("PORT")
	if portStr == "" {
		return nil, fmt.Errorf("required environment variable PORT is not set")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("PORT must be a valid integer, got: %s", portStr)
	}

	rabbitMQURI := getenv("RABBITMQ_URI")
	if rabbitMQURI == "" {
		return nil, fmt.Errorf("required environment variable RABBITMQ_URI is not set")
	}

	authServiceURL := getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		return nil, fmt.Errorf("required environment variable AUTH_SERVICE_URL is not set")
	}

	config := config{
		PlatformMode:   platformMode,
		Port:           port,
		Version:        "2.0.0",
		AuthServiceURL: authServiceURL,
		RabbitMQURI:    rabbitMQURI,
	}

	return &config, nil
}
