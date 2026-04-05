package main

import (
	"cmp"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/env"
)

var (
	defaultPort        = contracts.GRPCPort
	defaultRabbitMQURI = "amqp://guest:guest@rabbitmq:5672/" // #nosec G101 - Default dev URI, not a production credential
)

const (
	envPort                       = "PORT"
	envDBURL                      = "DB_URL"
	envRabbitMQURI                = "RABBITMQ_URI"
	envPlatformMode               = "PLATFORM"
	envCursorHMACKey              = "CURSOR_HMAC_KEY"               // #nosec G101 - Env var name, not a credential
	envGoogleMapsAPIKey           = "GOOGLE_MAPS_API_KEY"           // #nosec G101 -- env var name, not a credential
	envIntegrationEncryptionKey   = "INTEGRATION_ENCRYPTION_KEY"    // #nosec G101 - Env var name, not a credential
	envIntegrationEncryptionKeyID = "INTEGRATION_ENCRYPTION_KEY_ID" // #nosec G101 - Env var name, not a credential
	envAWSRegion                  = "AWS_REGION"
	envAccountPhotosBucket        = "ACCOUNT_PHOTOS_BUCKET"
	envUserPhotosBucket           = "USER_PHOTOS_BUCKET"
	envShippingLabelsBucket       = "SHIPPING_LABELS_BUCKET"
	envFrontendURL                = "FRONTEND_URL"
)

// config represents the configuration for the core service.
type config struct {
	// Port (optional; default: 9092) specifies the port on which the gRPC server will listen.
	Port int

	// DBURL (required) is the database connection URI.
	DBURL string

	// RabbitMQURI (optional; default: "amqp://guest:guest@rabbitmq:5672/") is the RabbitMQ connection URI.
	RabbitMQURI string

	// PlatformMode (optional; default: "production") determines the platform mode.
	PlatformMode constants.PlatformMode

	// CursorHMACKey (required) is the key used to HMAC-sign pagination cursors.
	CursorHMACKey []byte

	// GoogleMapsAPIKey (required) is the Google Maps API key for address validation.
	GoogleMapsAPIKey string

	// IntegrationEncryptionKey (required) is the hex-encoded AES-256 key for encrypting integration credentials.
	IntegrationEncryptionKey string

	// IntegrationEncryptionKeyID (required) identifies which encryption key version was used.
	IntegrationEncryptionKeyID string

	// AWSRegion (required) is the AWS region for S3 operations.
	AWSRegion string

	// AccountPhotosBucket (required) is the S3 bucket for account logos.
	AccountPhotosBucket string

	// UserPhotosBucket (required) is the S3 bucket for user profile photos.
	UserPhotosBucket string

	// ShippingLabelsBucket (required) is the S3 bucket for shipping label assets.
	ShippingLabelsBucket string

	// FrontendURL (required) is the base URL of the frontend application, used for checkout return URLs.
	FrontendURL string
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
		Port:                       port,
		DBURL:                      env.GetEnv(envDBURL, getenv),
		RabbitMQURI:                cmp.Or(env.GetEnv(envRabbitMQURI, getenv), defaultRabbitMQURI),
		PlatformMode:               platformMode,
		CursorHMACKey:              []byte(env.GetEnv(envCursorHMACKey, getenv)),
		GoogleMapsAPIKey:           env.GetEnv(envGoogleMapsAPIKey, getenv),
		IntegrationEncryptionKey:   env.GetEnv(envIntegrationEncryptionKey, getenv),
		IntegrationEncryptionKeyID: env.GetEnv(envIntegrationEncryptionKeyID, getenv),
		AWSRegion:                  env.GetEnv(envAWSRegion, getenv),
		AccountPhotosBucket:        env.GetEnv(envAccountPhotosBucket, getenv),
		UserPhotosBucket:           env.GetEnv(envUserPhotosBucket, getenv),
		ShippingLabelsBucket:       env.GetEnv(envShippingLabelsBucket, getenv),
		FrontendURL:                env.GetEnv(envFrontendURL, getenv),
	}
}

// validate validates the configuration.
func (c *config) validate() error {
	if c == nil {
		return fmt.Errorf("core-service: config is nil")
	}
	if !c.PlatformMode.IsValid() {
		return fmt.Errorf("core-service: the provided platform mode is invalid: %s", c.PlatformMode)
	}
	if c.DBURL == "" {
		return fmt.Errorf("core-service: the provided database URI is empty")
	}
	if len(c.CursorHMACKey) == 0 {
		return fmt.Errorf("core-service: CURSOR_HMAC_KEY is required")
	}
	if c.IntegrationEncryptionKey == "" {
		return fmt.Errorf("core-service: INTEGRATION_ENCRYPTION_KEY is required")
	}
	decodedKey, err := hex.DecodeString(c.IntegrationEncryptionKey)
	if err != nil {
		return fmt.Errorf("core-service: INTEGRATION_ENCRYPTION_KEY must be valid hex: %w", err)
	}
	if len(decodedKey) != 32 {
		return fmt.Errorf("core-service: INTEGRATION_ENCRYPTION_KEY must decode to 32 bytes (AES-256)")
	}
	if c.IntegrationEncryptionKeyID == "" {
		return fmt.Errorf("core-service: INTEGRATION_ENCRYPTION_KEY_ID is required")
	}
	if c.FrontendURL == "" {
		return fmt.Errorf("core-service: FRONTEND_URL is required")
	}

	if !c.PlatformMode.IsTest() {
		if c.GoogleMapsAPIKey == "" {
			return fmt.Errorf("core-service: GOOGLE_MAPS_API_KEY is required")
		}
		if c.AWSRegion == "" {
			return fmt.Errorf("core-service: AWS_REGION is required")
		}
		if c.AccountPhotosBucket == "" {
			return fmt.Errorf("core-service: ACCOUNT_PHOTOS_BUCKET is required")
		}
		if c.UserPhotosBucket == "" {
			return fmt.Errorf("core-service: USER_PHOTOS_BUCKET is required")
		}
		if c.ShippingLabelsBucket == "" {
			return fmt.Errorf("core-service: SHIPPING_LABELS_BUCKET is required")
		}
	}
	return nil
}
