package httpgroup

import (
	"fmt"

	authep "github.com/open-mrp/api/services/api-gateway/endpoints/auth"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	"github.com/open-mrp/api/services/api-gateway/internal/middleware"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
)

type AuthEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AuthEndpointGroupConfig struct {
	// AuthClient (required) is the auth-service gRPC client.
	AuthClient *grpcclient.AuthServiceClient

	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *AuthEndpointGroupConfig) validate() error {
	if c.AuthClient == nil {
		return fmt.Errorf("auth endpoint group: auth client is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("auth endpoint group: core client is required")
	}
	return nil
}

func (*AuthEndpointGroup) Materialize(config *AuthEndpointGroupConfig) *AuthEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	authController := authep.NewAuthSvc(&authep.AuthSvcConfig{
		AuthClient: config.AuthClient.Client,
		CoreClient: config.CoreClient.Client,
	})

	authMw := middleware.AuthMiddleware(&middleware.AuthMiddlewareConfig{
		AuthClient: config.AuthClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:       "Authentication",
		Description: "User authentication and token lifecycle operations, including login, registration, password management, and token refresh.",
	}

	loginEndpoint := apiendpoint.From(&authep.LoginEndpoint{}).WithService(inner, authController)
	registerEndpoint := apiendpoint.From(&authep.RegisterEndpoint{}).WithService(inner, authController)
	magicLoginEndpoint := apiendpoint.From(&authep.MagicLoginEndpoint{}).WithService(inner, authController)
	refreshTokenEndpoint := apiendpoint.From(&authep.RefreshTokenEndpoint{}).WithService(inner, authController)
	requestPasswordResetEndpoint := apiendpoint.From(&authep.RequestPasswordResetEndpoint{}).WithService(inner, authController)
	resetPasswordEndpoint := apiendpoint.From(&authep.ResetPasswordEndpoint{}).WithService(inner, authController)
	revokeRefreshTokenEndpoint := apiendpoint.From(&authep.RevokeRefreshTokenEndpoint{}).WithMiddleware(authMw).WithService(inner, authController)
	updatePasswordEndpoint := apiendpoint.From(&authep.UpdatePasswordEndpoint{}).WithMiddleware(authMw).WithService(inner, authController)
	updateScannerPasswordEndpoint := apiendpoint.From(&authep.UpdateScannerPasswordEndpoint{}).WithMiddleware(authMw).WithService(inner, authController)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		loginEndpoint,
		registerEndpoint,
		magicLoginEndpoint,
		refreshTokenEndpoint,
		requestPasswordResetEndpoint,
		resetPasswordEndpoint,
		revokeRefreshTokenEndpoint,
		updatePasswordEndpoint,
		updateScannerPasswordEndpoint,
	}

	return &AuthEndpointGroup{inner}
}
