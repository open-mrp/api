package main

import (
	"cmp"
	"fmt"
	"strconv"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/crypto"
	"github.com/augno/api/shared/env"
)

var (
	defaultPort              = contracts.GRPCPort
	defaultRabbitMQURI       = "amqp://guest:guest@rabbitmq:5672/" // #nosec G101 - Default dev URI, not a production credential
	defaultCoreServiceURL    = fmt.Sprintf("core-service:%d", contracts.GRPCPort)
	defaultBillingServiceURL = fmt.Sprintf("billing-service:%d", contracts.GRPCPort)
)

const (
	envPort                   = "PORT"
	envDBURL                  = "DB_URL"
	envFrontendURL            = "FRONTEND_URL"
	envJWTSecret              = "JWT_SECRET"
	envPepper                 = "PEPPER"
	envRabbitMQURI            = "RABBITMQ_URI"
	envCoreServiceURL         = "CORE_SERVICE_URL"
	envPlatformServiceURL     = "PLATFORM_SERVICE_URL"
	envDocAPIKeyEncryptionKey = "DOC_API_KEY_ENCRYPTION_KEY" // #nosec G101 - Env var name, not a credential
	envBillingServiceURL      = "BILLING_SERVICE_URL"
	envCursorHMACKey          = "CURSOR_HMAC_KEY" // #nosec G101 - Env var name, not a credential
	envPlatformMode           = "PLATFORM"
)

// config represents the configuration for the auth service.
type config struct {
	// Port (optional; default: 9092) specifies the port on which the gRPC server will listen.
	Port int

	// DBURL (required) is the database connection URI.
	DBURL string

	// FrontendURL (required) is the base URL of the frontend application.
	FrontendURL string

	// JWTSecret (required) is the secret used to sign JWT tokens.
	JWTSecret string // #nosec G117 - Struct field, not a hardcoded credential

	// Pepper (required) is the additional secret mixed into password hashes.
	Pepper []byte

	// RabbitMQURI (optional; default: "amqp://guest:guest@rabbitmq:5672/") is the RabbitMQ connection URI.
	RabbitMQURI string

	// CoreServiceURL (optional; default: "core-service:9092") is the core service address for gRPC.
	CoreServiceURL string

	// PlatformServiceURL (required) is the platform service address for gRPC.
	PlatformServiceURL string

	// DocAPIKeyEncryptionKey (optional) is the 32-byte AES-256 key used to encrypt doc API key secrets.
	DocAPIKeyEncryptionKey []byte

	// BillingServiceURL (optional; default: "billing-service:9092") is the billing service address for gRPC.
	BillingServiceURL string

	// CursorHMACKey (required) is the key used to HMAC-sign pagination cursors.
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

	key, err := crypto.DecodeHexKey256(env.GetEnv(envDocAPIKeyEncryptionKey, getenv))
	if err != nil {
		panic(err)
	}

	platformMode := constants.PlatformModeProduction
	if p := env.GetEnv(envPlatformMode, getenv); p != "" {
		platformMode = constants.PlatformMode(p)
	}

	return &config{
		Port:                   port,
		DBURL:                  env.GetEnv(envDBURL, getenv),
		FrontendURL:            env.GetEnv(envFrontendURL, getenv),
		JWTSecret:              env.GetEnv(envJWTSecret, getenv),
		Pepper:                 []byte(env.GetEnv(envPepper, getenv)),
		RabbitMQURI:            cmp.Or(env.GetEnv(envRabbitMQURI, getenv), defaultRabbitMQURI),
		CoreServiceURL:         cmp.Or(env.GetEnv(envCoreServiceURL, getenv), defaultCoreServiceURL),
		PlatformServiceURL:     env.GetEnv(envPlatformServiceURL, getenv),
		DocAPIKeyEncryptionKey: key,
		BillingServiceURL:      cmp.Or(env.GetEnv(envBillingServiceURL, getenv), defaultBillingServiceURL),
		CursorHMACKey:          []byte(env.GetEnv(envCursorHMACKey, getenv)),
		PlatformMode:           platformMode,
	}
}

// validate validates the configuration.
func (c *config) validate() error {
	if c == nil {
		return fmt.Errorf("auth-service: config is nil")
	}
	if c.DBURL == "" {
		return fmt.Errorf("auth-service: the provided database URI is empty")
	}
	if c.FrontendURL == "" {
		return fmt.Errorf("auth-service: the provided frontend URL is empty")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("auth-service: the provided JWT secret is empty")
	}
	if len(c.Pepper) == 0 {
		return fmt.Errorf("auth-service: the provided pepper is empty")
	}
	if len(c.CursorHMACKey) == 0 {
		return fmt.Errorf("auth-service: CURSOR_HMAC_KEY is required")
	}
	return nil
}
