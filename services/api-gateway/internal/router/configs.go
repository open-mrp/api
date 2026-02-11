package router

import (
	"io"
	"log"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/shared/constants"
)

type BaseConfig struct {
	PlatformMode constants.PlatformMode
	LogPrefix    string
	LogFlags     int
	LogWriter    io.Writer
	AuthClient   *grpcclient.AuthServiceClient
	CoreClient   *grpcclient.CoreServiceClient
	// PaymentClient       *grpcclient.PaymentServiceClient
	PlatformClient      *grpcclient.PlatformServiceClient
	RequestLogPublisher domain.RequestLogPublisher
}

type MainRouterConfig struct {
	BaseConfig
}

type AuthRouterConfig struct {
	BaseConfig
}

func NewMainRouter(baseCfg BaseConfig) *router {
	r := NewRouter()
	r.InitEndpointGroups(MainRouterConfig{BaseConfig: baseCfg})
	return r
}

func NewAuthRouter(baseCfg BaseConfig) *router {
	r := NewRouter()
	r.InitAuthEndpointGroups(AuthRouterConfig{BaseConfig: baseCfg})
	return r
}

func BuildBaseConfig(
	platformMode constants.PlatformMode,
	logPrefix string,
	authClient *grpcclient.AuthServiceClient,
	coreClient *grpcclient.CoreServiceClient,
	// paymentClient *grpcclient.PaymentServiceClient,
	platformClient *grpcclient.PlatformServiceClient,
	reqLogPublisher domain.RequestLogPublisher,
	logWriter io.Writer,
) BaseConfig {
	return BaseConfig{
		PlatformMode: platformMode,
		LogPrefix:    logPrefix,
		LogFlags:     log.LstdFlags,
		LogWriter:    logWriter,
		AuthClient:   authClient,
		CoreClient:   coreClient,
		// PaymentClient:       paymentClient,
		PlatformClient:      platformClient,
		RequestLogPublisher: reqLogPublisher,
	}
}
