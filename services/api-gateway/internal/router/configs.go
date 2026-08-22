package router

import (
	"fmt"
	"io"
	"log"

	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	"github.com/open-mrp/api/shared/constants"
)

type BaseConfig struct {
	// PlatformMode (optional; default: "" i.e. unset) is the platform mode propagated to endpoint groups and middleware.
	PlatformMode constants.PlatformMode

	// LogPrefix (optional; default: "") is prepended to every router log line.
	LogPrefix string

	// LogFlags (optional; default: log.LstdFlags) are the standard-library log flags for the router logger. The zero value is treated as unset.
	LogFlags int

	// LogWriter (required) receives router log output.
	LogWriter io.Writer

	// AuthClient (required) is the auth-service gRPC client.
	AuthClient *grpcclient.AuthServiceClient

	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient

	// BillingClient (required) is the billing-service gRPC client.
	BillingClient *grpcclient.BillingServiceClient

	// PlatformClient (required) is the platform-service gRPC client.
	PlatformClient *grpcclient.PlatformServiceClient

	// AgentClient (required) is the agent-service gRPC client.
	AgentClient *grpcclient.AgentServiceClient

	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient

	// RequestLogPublisher (required) publishes request logs to the outbox.
	RequestLogPublisher domain.RequestLogPublisher

	// TrustedProxyHops (optional; default: 0, meaning XFF is not trusted) is the number of reverse-proxy hops in front of this service whose X-Forwarded-For entries can be trusted. See header.GetClientIP for details.
	TrustedProxyHops int
}

type MainRouterConfig struct {
	BaseConfig

	// Internal, when true, serves the endpoint groups on the trusted internal listener: AuthMiddleware is replaced with InternalAuthMiddleware so an agent identity supplied by agent-service is honored. This listener must never be exposed behind the public ALB.
	Internal bool

	// InternalServiceToken (required when Internal) is the shared secret gating identity trust on the internal listener.
	InternalServiceToken string
}

type AuthRouterConfig struct {
	BaseConfig
}

// WithDefaults returns a new BaseConfig with zero-value fields replaced by defaults.
func (c *BaseConfig) WithDefaults() *BaseConfig {
	if c == nil {
		c = &BaseConfig{}
	}

	logFlags := c.LogFlags
	if logFlags == 0 {
		logFlags = log.LstdFlags
	}

	return &BaseConfig{
		PlatformMode:        c.PlatformMode,
		LogPrefix:           c.LogPrefix,
		LogFlags:            logFlags,
		LogWriter:           c.LogWriter,
		AuthClient:          c.AuthClient,
		CoreClient:          c.CoreClient,
		BillingClient:       c.BillingClient,
		PlatformClient:      c.PlatformClient,
		AgentClient:         c.AgentClient,
		NotificationClient:  c.NotificationClient,
		RequestLogPublisher: c.RequestLogPublisher,
		TrustedProxyHops:    c.TrustedProxyHops,
	}
}

func (c *BaseConfig) validate() error {
	if c.LogWriter == nil {
		return fmt.Errorf("base config: log writer is required")
	}
	if c.AuthClient == nil {
		return fmt.Errorf("base config: auth client is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("base config: core client is required")
	}
	if c.BillingClient == nil {
		return fmt.Errorf("base config: billing client is required")
	}
	if c.PlatformClient == nil {
		return fmt.Errorf("base config: platform client is required")
	}
	if c.AgentClient == nil {
		return fmt.Errorf("base config: agent client is required")
	}
	if c.NotificationClient == nil {
		return fmt.Errorf("base config: notification client is required")
	}
	if c.RequestLogPublisher == nil {
		return fmt.Errorf("base config: request log publisher is required")
	}
	return nil
}

func (c *MainRouterConfig) validate() error {
	if err := c.BaseConfig.validate(); err != nil {
		return err
	}
	if c.Internal && c.InternalServiceToken == "" {
		return fmt.Errorf("main router config: internal service token is required when Internal is set")
	}
	return nil
}

func NewMainRouter(baseCfg *BaseConfig) *router {
	baseCfg = baseCfg.WithDefaults()
	if err := baseCfg.validate(); err != nil {
		panic(err)
	}

	r := NewRouter()
	r.InitEndpointGroups(MainRouterConfig{BaseConfig: *baseCfg})
	return r
}

// NewInternalRouter builds a router that serves the same endpoint groups as the main router but trusts an agent identity via InternalAuthMiddleware instead of validating user credentials. It must only be served on the internal listener.
func NewInternalRouter(baseCfg *BaseConfig, serviceToken string) *router {
	baseCfg = baseCfg.WithDefaults()
	cfg := MainRouterConfig{BaseConfig: *baseCfg, Internal: true, InternalServiceToken: serviceToken}
	if err := cfg.validate(); err != nil {
		panic(err)
	}

	r := NewRouter()
	r.InitEndpointGroups(cfg)
	return r
}

func NewAuthRouter(baseCfg *BaseConfig) *router {
	baseCfg = baseCfg.WithDefaults()
	if err := baseCfg.validate(); err != nil {
		panic(err)
	}

	r := NewRouter()
	r.InitAuthEndpointGroups(AuthRouterConfig{BaseConfig: *baseCfg})
	return r
}

type WebhookRouterConfig struct {
	BaseConfig
}

func NewWebhookRouter(baseCfg *BaseConfig) *router {
	baseCfg = baseCfg.WithDefaults()
	if err := baseCfg.validate(); err != nil {
		panic(err)
	}

	r := NewRouter()
	r.InitWebhookEndpointGroups(WebhookRouterConfig{BaseConfig: *baseCfg})
	return r
}

func BuildBaseConfig(
	platformMode constants.PlatformMode,
	logPrefix string,
	authClient *grpcclient.AuthServiceClient,
	coreClient *grpcclient.CoreServiceClient,
	billingClient *grpcclient.BillingServiceClient,
	platformClient *grpcclient.PlatformServiceClient,
	agentClient *grpcclient.AgentServiceClient,
	notificationClient *grpcclient.NotificationServiceClient,
	reqLogPublisher domain.RequestLogPublisher,
	logWriter io.Writer,
	trustedProxyHops int,
) *BaseConfig {
	return &BaseConfig{
		PlatformMode:        platformMode,
		LogPrefix:           logPrefix,
		LogFlags:            log.LstdFlags,
		LogWriter:           logWriter,
		AuthClient:          authClient,
		CoreClient:          coreClient,
		BillingClient:       billingClient,
		PlatformClient:      platformClient,
		AgentClient:         agentClient,
		NotificationClient:  notificationClient,
		RequestLogPublisher: reqLogPublisher,
		TrustedProxyHops:    trustedProxyHops,
	}
}
