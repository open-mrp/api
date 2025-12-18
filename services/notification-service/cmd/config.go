package main

import "fmt"

type config struct {
	DBURL       string
	AWSRegion   string
	RabbitMQURI string
}

func loadConfig(getenv func(string) string) (config, error) {
	dbURL := getenv("DB_URL")
	if dbURL == "" {
		return config{}, fmt.Errorf("required environment variable DB_URL is not set")
	}

	awsRegion := getenv("AWS_REGION")
	if awsRegion == "" {
		return config{}, fmt.Errorf("required environment variable AWS_REGION is not set")
	}

	awsAccessKeyID := getenv("AWS_ACCESS_KEY_ID")
	if awsAccessKeyID == "" {
		return config{}, fmt.Errorf("required environment variable AWS_ACCESS_KEY_ID is not set")
	}

	awsSecretAccessKey := getenv("AWS_SECRET_ACCESS_KEY")
	if awsSecretAccessKey == "" {
		return config{}, fmt.Errorf("required environment variable AWS_SECRET_ACCESS_KEY is not set")
	}

	rabbitMQURI := getenv("RABBITMQ_URI")
	if rabbitMQURI == "" {
		return config{}, fmt.Errorf("required environment variable RABBITMQ_URI is not set")
	}

	return config{
		DBURL:       dbURL,
		AWSRegion:   awsRegion,
		RabbitMQURI: rabbitMQURI,
	}, nil
}
