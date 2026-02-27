package router

import (
	"fmt"
	"io"
	"log"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/shared/constants"
)

type BaseConfig struct {
	PlatformMode        constants.PlatformMode
	LogPrefix           string
	LogFlags            int
	LogWriter           io.Writer
	AuthClient          *grpcclient.AuthServiceClient
	CoreClient          *grpcclient.CoreServiceClient
	BillingClient       *grpcclient.BillingServiceClient
	PlatformClient      *grpcclient.PlatformServiceClient
	RequestLogPublisher domain.RequestLogPublisher
}

type MainRouterConfig struct {
	BaseConfig
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
		RequestLogPublisher: c.RequestLogPublisher,
	}
}

func (c *BaseConfig) validate() error {
	if c.LogWriter == nil {
		return fmt.Errorf("base config: log writer is required")
	}
	if c.AuthClient == nil {
		return fmt.Errorf("base config: auth client is required")
	}
	if c.RequestLogPublisher == nil {
		return fmt.Errorf("base config: request log publisher is required")
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
	reqLogPublisher domain.RequestLogPublisher,
	logWriter io.Writer,
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
		RequestLogPublisher: reqLogPublisher,
	}
}
