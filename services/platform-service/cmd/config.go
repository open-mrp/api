package main

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/env"
)

var (
	defaultPort        = contracts.GRPCPort
	defaultRabbitMQURI = "amqp://guest:guest@rabbitmq:5672/" // #nosec G101 - Default dev URI, not a production credential
)

const (
	envPort           = "PORT"
	envDBURL          = "DB_URL"
	envRabbitMQURI    = "RABBITMQ_URI"
	envCursorHMACKey  = "CURSOR_HMAC_KEY"
	envPlatformMode   = "PLATFORM"
	envAWSRegion      = "AWS_REGION"
	envPayloadsBucket = "PAYLOADS_BUCKET"

	envStripeSecretKey                = "STRIPE_SECRET_KEY"
	envAccountFollowupEnabled         = "ACCOUNT_FOLLOWUP_ENABLED"
	envAccountFollowupReviewerEmail   = "ACCOUNT_FOLLOWUP_REVIEWER_EMAIL"
	envAccountFollowupReviewBaseURL   = "ACCOUNT_FOLLOWUP_REVIEW_BASE_URL"
	envAccountFollowupModel           = "ACCOUNT_FOLLOWUP_MODEL"
	envAccountFollowupDelay           = "ACCOUNT_FOLLOWUP_DELAY"
	envAccountFollowupPollInterval    = "ACCOUNT_FOLLOWUP_POLL_INTERVAL"
	envAccountFollowupExcludedDomains = "ACCOUNT_FOLLOWUP_EXCLUDED_DOMAINS"
	defaultAccountFollowupReviewer    = "dane@openmrp.ai"
	defaultAccountFollowupReviewURL   = "https://api.openmrp.ai/account-followups/review"
	defaultAccountFollowupModel       = "claude-haiku-4.5"
	defaultAWSRegion                  = "us-east-2"
)

// config represents the configuration for the platform service.
type config struct {
	// Port (optional; default: 9092) specifies the port on which the gRPC server will listen.
	Port int

	// DBURL (required) is the database connection URI.
	DBURL string

	// RabbitMQURI (optional; default: "amqp://guest:guest@rabbitmq:5672/") is the RabbitMQ connection URI.
	RabbitMQURI string

	// CursorHMACKey (required) is the HMAC key used to sign and verify pagination cursors.
	CursorHMACKey []byte

	// PlatformMode (optional; default: "production") determines the platform mode.
	PlatformMode constants.PlatformMode

	// AWSRegion (optional; default: "us-east-2") is the AWS region of the payloads bucket.
	AWSRegion string

	// PayloadsBucket (optional; default: "") is the S3 bucket the api-gateway writes request log bodies
	// to. When empty, only logs whose bodies were stored inline can return them.
	PayloadsBucket string

	// AccountFollowupEnabled (optional; default: false) turns on drafting follow-ups for new registrants. Registrations are recorded either way, so enabling it later still drafts for everyone since.
	AccountFollowupEnabled bool

	// StripeSecretKey (required when AccountFollowupEnabled) authenticates follow-up drafting to the Stripe AI Gateway.
	StripeSecretKey string

	// AccountFollowupReviewerEmail (optional; default: "dane@openmrp.ai") receives every draft for approval.
	AccountFollowupReviewerEmail string

	// AccountFollowupReviewBaseURL (optional; default: "https://api.openmrp.ai/account-followups/review") is the review page linked from each review email.
	AccountFollowupReviewBaseURL string

	// AccountFollowupModel (optional; default: "claude-haiku-4.5") is the gateway model that drafts follow-ups. A short paragraph from a structured summary does not need a larger model.
	AccountFollowupModel string

	// AccountFollowupDelay (optional; default: 24h) is how long after registering a follow-up is drafted.
	AccountFollowupDelay time.Duration

	// AccountFollowupPollInterval (optional; default: 1h) is how often due follow-ups are looked for.
	AccountFollowupPollInterval time.Duration

	// AccountFollowupExcludedDomains (optional; default: the team's and test domains) is a comma-separated list of registrant domains that never get a follow-up.
	AccountFollowupExcludedDomains []string
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

	followupEnabled, _ := strconv.ParseBool(env.GetEnv(envAccountFollowupEnabled, getenv))
	followupDelay, _ := time.ParseDuration(env.GetEnv(envAccountFollowupDelay, getenv))
	followupPollInterval, _ := time.ParseDuration(env.GetEnv(envAccountFollowupPollInterval, getenv))
	var excludedDomains []string
	for _, d := range strings.Split(env.GetEnv(envAccountFollowupExcludedDomains, getenv), ",") {
		if d = strings.TrimSpace(d); d != "" {
			excludedDomains = append(excludedDomains, d)
		}
	}

	return &config{
		Port:           port,
		DBURL:          env.GetEnv(envDBURL, getenv),
		RabbitMQURI:    cmp.Or(env.GetEnv(envRabbitMQURI, getenv), defaultRabbitMQURI),
		CursorHMACKey:  []byte(env.GetEnv(envCursorHMACKey, getenv)),
		PlatformMode:   platformMode,
		AWSRegion:      cmp.Or(env.GetEnv(envAWSRegion, getenv), defaultAWSRegion),
		PayloadsBucket: env.GetEnv(envPayloadsBucket, getenv),

		AccountFollowupEnabled:         followupEnabled,
		StripeSecretKey:                env.GetEnv(envStripeSecretKey, getenv),
		AccountFollowupReviewerEmail:   cmp.Or(env.GetEnv(envAccountFollowupReviewerEmail, getenv), defaultAccountFollowupReviewer),
		AccountFollowupReviewBaseURL:   cmp.Or(env.GetEnv(envAccountFollowupReviewBaseURL, getenv), defaultAccountFollowupReviewURL),
		AccountFollowupModel:           cmp.Or(env.GetEnv(envAccountFollowupModel, getenv), defaultAccountFollowupModel),
		AccountFollowupDelay:           followupDelay,
		AccountFollowupPollInterval:    followupPollInterval,
		AccountFollowupExcludedDomains: excludedDomains,
	}
}

// validate validates the configuration.
func (c *config) validate() error {
	if c == nil {
		return fmt.Errorf("platform-service: config is nil")
	}
	if !c.PlatformMode.IsValid() {
		return fmt.Errorf("platform-service: the provided platform mode is invalid: %s", c.PlatformMode)
	}
	if c.DBURL == "" {
		return fmt.Errorf("platform-service: the provided database URI is empty")
	}
	if len(c.CursorHMACKey) == 0 {
		return fmt.Errorf("platform-service: CURSOR_HMAC_KEY is required")
	}
	if c.AccountFollowupEnabled && c.StripeSecretKey == "" {
		return fmt.Errorf("platform-service: %s is required when %s is set", envStripeSecretKey, envAccountFollowupEnabled)
	}
	return nil
}
