package main

import "fmt"

type config struct {
	DBURL       string
	RabbitMQURI string
}

func loadConfig(getenv func(string) string) (config, error) {
	dbURL := getenv("DB_URL")
	if dbURL == "" {
		return config{}, fmt.Errorf("required environment variable DB_URL is not set")
	}

	rabbitMQURI := getenv("RABBITMQ_URI")
	if rabbitMQURI == "" {
		return config{}, fmt.Errorf("required environment variable RABBITMQ_URI is not set")
	}

	return config{
		DBURL:       dbURL,
		RabbitMQURI: rabbitMQURI,
	}, nil
}
