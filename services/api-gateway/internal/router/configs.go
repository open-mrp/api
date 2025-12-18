package router

import (
	"log"

	"io"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/shared/constants"
)

// BaseConfig contains the common configuration fields shared by both router types.
type BaseConfig struct {
	PlatformMode        constants.PlatformMode
	LogPrefix           string
	LogFlags            int
	LogWriter           io.Writer
	AuthClient          *grpcclient.AuthServiceClient
	RequestLogPublisher domain.RequestLogPublisher
}

type MainRouterConfig struct {
	BaseConfig
}

type AuthRouterConfig struct {
	BaseConfig
}

// NewMainRouter creates and initializes a main router with the given configuration.
func NewMainRouter(baseCfg BaseConfig) *router {
	r := NewRouter()
	r.InitEndpointGroups(MainRouterConfig{BaseConfig: baseCfg})
	return r
}

// NewAuthRouter creates and initializes an auth router with the given configuration.
func NewAuthRouter(baseCfg BaseConfig) *router {
	r := NewRouter()
	r.InitAuthEndpointGroups(AuthRouterConfig{BaseConfig: baseCfg})
	return r
}

// BuildBaseConfig constructs a BaseConfig from the given parameters.
func BuildBaseConfig(
	platformMode constants.PlatformMode,
	logPrefix string,
	authClient *grpcclient.AuthServiceClient,
	reqLogPublisher domain.RequestLogPublisher,
	logWriter io.Writer,
) BaseConfig {
	return BaseConfig{
		PlatformMode:        platformMode,
		LogPrefix:           logPrefix,
		LogFlags:            log.LstdFlags,
		LogWriter:           logWriter,
		AuthClient:          authClient,
		RequestLogPublisher: reqLogPublisher,
	}
}
