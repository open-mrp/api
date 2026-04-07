package httpgroup

import (
	"fmt"

	authep "github.com/augno/api/services/api-gateway/endpoints/auth"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	"github.com/augno/api/services/api-gateway/internal/middleware"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
)

type AuthEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AuthEndpointGroupConfig struct {
	AuthClient *grpcclient.AuthServiceClient
}

func (c *AuthEndpointGroupConfig) validate() error {
	if c.AuthClient == nil {
		return fmt.Errorf("auth endpoint group: auth client is required")
	}
	return nil
}

func (*AuthEndpointGroup) Materialize(config *AuthEndpointGroupConfig) *AuthEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	authController := authep.NewAuthSvc(&authep.AuthSvcConfig{
		AuthClient: config.AuthClient.Client,
	})

	authMw := middleware.AuthMiddleware(&middleware.AuthMiddlewareConfig{
		AuthClient: config.AuthClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:       "Authentication and Token Management",
		Description: "User authentication and token lifecycle operations, including login, registration, password management, and token refresh.",
	}

	loginEndpoint := (&authep.LoginEndpoint{}).Materialize().WithService(inner, authController)
	registerEndpoint := (&authep.RegisterEndpoint{}).Materialize().WithService(inner, authController)
	magicLoginEndpoint := (&authep.MagicLoginEndpoint{}).Materialize().WithService(inner, authController)
	refreshTokenEndpoint := (&authep.RefreshTokenEndpoint{}).Materialize().WithService(inner, authController)
	requestPasswordResetEndpoint := (&authep.RequestPasswordResetEndpoint{}).Materialize().WithService(inner, authController)
	resetPasswordEndpoint := (&authep.ResetPasswordEndpoint{}).Materialize().WithService(inner, authController)
	revokeRefreshTokenEndpoint := (&authep.RevokeRefreshTokenEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, authController)
	updatePasswordEndpoint := (&authep.UpdatePasswordEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, authController)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		loginEndpoint,
		registerEndpoint,
		magicLoginEndpoint,
		refreshTokenEndpoint,
		requestPasswordResetEndpoint,
		resetPasswordEndpoint,
		revokeRefreshTokenEndpoint,
		updatePasswordEndpoint,
	}

	return &AuthEndpointGroup{inner}
}
