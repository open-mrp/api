package main

import (
	"fmt"
	"strconv"
)

type config struct {
	DBURL        string
	FrontendURL  string
	JWTSecret    string
	Pepper       []byte
	Port         int
	RabbitMQURI  string
	TemplatesDir string
}

func loadConfig(getenv func(string) string) (config, error) {
	dbURL := getenv("DB_URL")
	if dbURL == "" {
		return config{}, fmt.Errorf("required environment variable DB_URL is not set")
	}

	frontendURL := getenv("FRONTEND_URL")
	if frontendURL == "" {
		return config{}, fmt.Errorf("required environment variable FRONTEND_URL is not set")
	}

	jwtSecret := getenv("JWT_SECRET")
	if jwtSecret == "" {
		return config{}, fmt.Errorf("required environment variable JWT_SECRET is not set")
	}

	pepper := getenv("PEPPER")
	if pepper == "" {
		return config{}, fmt.Errorf("required environment variable PEPPER is not set")
	}

	port := 9092
	if portEnv := getenv("PORT"); portEnv != "" {
		var err error
		port, err = strconv.Atoi(portEnv)
		if err != nil {
			return config{}, fmt.Errorf("PORT must be a valid integer, got: %s", portEnv)
		}
	}

	rabbitMQURI := getenv("RABBITMQ_URI")
	if rabbitMQURI == "" {
		return config{}, fmt.Errorf("required environment variable RABBITMQ_URI is not set")
	}

	templatesDir := getenv("TEMPLATES_DIR")
	if templatesDir == "" {
		templatesDir = "services/auth-service/templates"
	}

	return config{
		DBURL:        dbURL,
		JWTSecret:    jwtSecret,
		Pepper:       []byte(pepper),
		Port:         port,
		RabbitMQURI:  rabbitMQURI,
		FrontendURL:  frontendURL,
		TemplatesDir: templatesDir,
	}, nil
}
