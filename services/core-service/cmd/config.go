package main

import (
	"cmp"
	"encoding/hex"
	"encoding/json"
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
	envAssetCDNBaseURL            = "ASSET_CDN_BASE_URL"
	envUserPhotosBucket           = "USER_PHOTOS_BUCKET"
	envShippingLabelsBucket       = "SHIPPING_LABELS_BUCKET"
	envExportsBucket              = "EXPORTS_BUCKET"
	envFrontendURL                = "FRONTEND_URL"
	envAuthServiceURL             = "AUTH_SERVICE_URL"
	envVercelAPIToken             = "VERCEL_API_TOKEN" // #nosec G101 - Env var name, not a credential
	envVercelProjectID            = "VERCEL_PROJECT_ID"
	envVercelTeamID               = "VERCEL_TEAM_ID"
	envAccountMarketingBlurbs     = "ACCOUNT_MARKETING_BLURBS"
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
	// Not enforced when PlatformMode is "test".
	GoogleMapsAPIKey string

	// IntegrationEncryptionKey (required) is the hex-encoded AES-256 key for encrypting integration credentials.
	IntegrationEncryptionKey string

	// IntegrationEncryptionKeyID (required) identifies which encryption key version was used.
	IntegrationEncryptionKeyID string

	// AWSRegion (required) is the AWS region for S3 operations.
	// Not enforced when PlatformMode is "test".
	AWSRegion string

	// AccountPhotosBucket (required) is the S3 bucket for account logos.
	// Not enforced when PlatformMode is "test".
	AccountPhotosBucket string

	// AssetCDNBaseURL (optional; default: "") is the public CDN origin fronting the account photos bucket
	// (e.g. "https://cdn.openmrp.ai"). When set, branding logo/favicon keys are served as stable, non-expiring
	// "<base>/<key>" URLs instead of presigned ones — favicons ride in long-lived HTML that browsers cache
	// past any signature. Unset (dev/e2e against MinIO) falls back to presigning.
	AssetCDNBaseURL string

	// UserPhotosBucket (required) is the S3 bucket for user profile photos.
	// Not enforced when PlatformMode is "test".
	UserPhotosBucket string

	// ShippingLabelsBucket (required) is the S3 bucket for shipping label assets.
	// Not enforced when PlatformMode is "test".
	ShippingLabelsBucket string

	// ExportsBucket (required) is the S3 bucket rendered exports are uploaded to.
	// Not enforced when PlatformMode is "test".
	ExportsBucket string

	// FrontendURL (required) is the base URL of the frontend application, used for checkout return URLs.
	FrontendURL string

	// AuthServiceURL (required) is the gRPC address of auth-service.
	AuthServiceURL string

	// VercelAPIToken (required in production) authenticates portal custom domain calls to the Vercel API. When empty outside production, the stub portal domain provider is used instead.
	VercelAPIToken string

	// VercelProjectID (required in production) is the Vercel project that serves the dashboard frontend; portal custom domains are attached to it.
	VercelProjectID string

	// VercelTeamID (optional) scopes Vercel API calls to a team; empty for personal-scope tokens.
	VercelTeamID string

	// AccountMarketingBlurbs (optional; default: none) is a JSON object mapping an account id to the
	// marketing sentence that account's customer emails carry in the footer, e.g.
	// {"ac_123":"A family owned company making widgets since 1975."}. An account absent from the map
	// gets no footer panel. Kept in configuration so no customer id or customer copy lives in source.
	AccountMarketingBlurbs string
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
		AssetCDNBaseURL:            env.GetEnv(envAssetCDNBaseURL, getenv),
		UserPhotosBucket:           env.GetEnv(envUserPhotosBucket, getenv),
		ShippingLabelsBucket:       env.GetEnv(envShippingLabelsBucket, getenv),
		ExportsBucket:              env.GetEnv(envExportsBucket, getenv),
		FrontendURL:                env.GetEnv(envFrontendURL, getenv),
		AuthServiceURL:             env.GetEnv(envAuthServiceURL, getenv),
		VercelAPIToken:             env.GetEnv(envVercelAPIToken, getenv),
		VercelProjectID:            env.GetEnv(envVercelProjectID, getenv),
		VercelTeamID:               env.GetEnv(envVercelTeamID, getenv),
		AccountMarketingBlurbs:     env.GetEnv(envAccountMarketingBlurbs, getenv),
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
	if c.AuthServiceURL == "" {
		return fmt.Errorf("core-service: AUTH_SERVICE_URL is required")
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
		if c.ExportsBucket == "" {
			return fmt.Errorf("core-service: EXPORTS_BUCKET is required")
		}
	}

	if c.PlatformMode.IsProduction() {
		if c.VercelAPIToken == "" {
			return fmt.Errorf("core-service: VERCEL_API_TOKEN is required")
		}
		if c.VercelProjectID == "" {
			return fmt.Errorf("core-service: VERCEL_PROJECT_ID is required")
		}
	}
	if _, err := c.marketingBlurbs(); err != nil {
		return err
	}
	return nil
}

// marketingBlurbs decodes AccountMarketingBlurbs. An unset value is not an error — it simply means no
// account carries a footer panel.
func (c *config) marketingBlurbs() (map[string]string, error) {
	if c.AccountMarketingBlurbs == "" {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(c.AccountMarketingBlurbs), &m); err != nil {
		return nil, fmt.Errorf("core-service: ACCOUNT_MARKETING_BLURBS must be a JSON object of account id to blurb: %w", err)
	}
	return m, nil
}
