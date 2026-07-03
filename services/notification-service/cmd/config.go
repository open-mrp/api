package main

import (
	"cmp"
	"fmt"
	"strconv"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/env"
)

var (
	defaultPort         = contracts.GRPCPort
	defaultPlatformMode = constants.PlatformModeProduction
	defaultRabbitMQURI  = "amqp://guest:guest@rabbitmq:5672/" // #nosec G101 - Default dev URI, not a production credential
)

const (
	envPort                 = "PORT"
	envPlatformMode         = "PLATFORM"
	envDBURL                = "DB_URL"
	envAWSRegion            = "AWS_REGION"
	envRabbitMQURI          = "RABBITMQ_URI"
	envChatBucket           = "CHAT_BUCKET"
	envInboundEmailBucket   = "INBOUND_EMAIL_BUCKET"
	envInboundEmailQueueURL = "INBOUND_EMAIL_QUEUE_URL"
	envInboundEmailRegion   = "INBOUND_EMAIL_REGION"
	envInboundEmailDomain   = "INBOUND_EMAIL_DOMAIN"
)

// defaultInboundEmailRegion is where SES receives inbound mail. SES email receiving is not offered in
// us-east-2 (the cluster region), so the inbound bucket/queue and their clients target us-east-1.
const defaultInboundEmailRegion = "us-east-1"

// config represents the configuration for the notification service.
type config struct {
	// Port (optional; default: 9092) specifies the port on which the gRPC server will listen.
	Port int

	// PlatformMode (optional; default: "production") determines the platform mode.
	PlatformMode constants.PlatformMode

	// DBURL (required) is the database connection URI.
	DBURL string

	// AWSRegion (required) is the AWS region for SES email delivery.
	// Not enforced when PlatformMode is "test".
	AWSRegion string

	// RabbitMQURI (optional; default: "amqp://guest:guest@rabbitmq:5672/") is the RabbitMQ connection URI.
	RabbitMQURI string

	// ChatBucket (required) is the S3 bucket for chat message attachments. Not enforced when PlatformMode is "test".
	ChatBucket string

	// InboundEmailBucket (optional) is the S3 bucket SES writes raw inbound emails to. When set
	// together with InboundEmailQueueURL, the inbound-email consumer runs; left empty (e.g. local dev
	// without AWS) the consumer is skipped.
	InboundEmailBucket string

	// InboundEmailQueueURL (optional) is the SQS queue URL carrying S3 object-created events for
	// inbound emails. Pairs with InboundEmailBucket to enable the inbound-email consumer.
	InboundEmailQueueURL string

	// InboundEmailRegion (optional; default: "us-east-1") is the region of the inbound-email bucket +
	// SQS queue. Distinct from AWSRegion because SES email receiving is not available in us-east-2.
	InboundEmailRegion string

	// InboundEmailDomain (optional) is the Augno-owned subdomain set up for SES receiving (e.g.
	// "inbound.augno.com"). It lets a customer whose corporate domain can't repoint its apex MX at SES
	// (Google/M365/Barracuda) instead forward their support address to <inbox_id>@<this domain>. When set,
	// inbound routing also matches mail delivered to that per-inbox forwarding address; left empty, only
	// direct-to-domain (customer MX → SES) inboxes are matched.
	InboundEmailDomain string
}

// withDefaults sets the default values for the configuration.
func (c *config) withDefaults(getenv func(string) string) *config {
	if c == nil {
		c = &config{}
	}

	platform := env.GetEnv(envPlatformMode, getenv)
	platformMode := defaultPlatformMode
	if platform != "" {
		platformMode = constants.PlatformMode(platform)
	}

	port := defaultPort
	if p, err := strconv.Atoi(env.GetEnv(envPort, getenv)); err == nil {
		port = p
	}

	return &config{
		Port:                 port,
		PlatformMode:         platformMode,
		DBURL:                env.GetEnv(envDBURL, getenv),
		AWSRegion:            env.GetEnv(envAWSRegion, getenv),
		RabbitMQURI:          cmp.Or(env.GetEnv(envRabbitMQURI, getenv), defaultRabbitMQURI),
		ChatBucket:           env.GetEnv(envChatBucket, getenv),
		InboundEmailBucket:   env.GetEnv(envInboundEmailBucket, getenv),
		InboundEmailQueueURL: env.GetEnv(envInboundEmailQueueURL, getenv),
		InboundEmailRegion:   cmp.Or(env.GetEnv(envInboundEmailRegion, getenv), defaultInboundEmailRegion),
		InboundEmailDomain:   env.GetEnv(envInboundEmailDomain, getenv),
	}
}

// validate validates the configuration.
func (c *config) validate() error {
	if c == nil {
		return fmt.Errorf("notification-service: config is nil")
	}
	if c.DBURL == "" {
		return fmt.Errorf("notification-service: the provided database URI is empty")
	}
	if !c.PlatformMode.IsValid() {
		return fmt.Errorf("notification-service: the provided platform mode is invalid: %s", c.PlatformMode)
	}
	if !c.PlatformMode.IsTest() && c.AWSRegion == "" {
		return fmt.Errorf("notification-service: the provided AWS region is empty")
	}
	if !c.PlatformMode.IsTest() && c.ChatBucket == "" {
		return fmt.Errorf("notification-service: the provided chat bucket is empty")
	}
	return nil
}
