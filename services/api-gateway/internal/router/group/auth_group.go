package httpgroup

import (
	authep "github.com/augno/api/services/api-gateway/endpoints/auth"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
)

type AuthEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AuthEndpointGroupConfig struct {
	AuthClient *grpcclient.AuthServiceClient
}

func (*AuthEndpointGroup) Materialize(config AuthEndpointGroupConfig) *AuthEndpointGroup {
	if config.AuthClient == nil {
		return nil
	}

	authController := authep.NewAuthSvc(authep.AuthSvcConfig{
		AuthClient: config.AuthClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:       "Authentication and Token Management",
		Description: "Handles user authentication and token lifecycle operations, including login, registration, password management, and token refresh.",
	}

	loginEndpoint := (&authep.LoginEndpoint{}).Materialize().WithService(inner, authController)
	registerEndpoint := (&authep.RegisterEndpoint{}).Materialize().WithService(inner, authController)
	refreshTokenEndpoint := (&authep.RefreshTokenEndpoint{}).Materialize().WithService(inner, authController)
	requestPasswordResetEndpoint := (&authep.RequestPasswordResetEndpoint{}).Materialize().WithService(inner, authController)
	resetPasswordEndpoint := (&authep.ResetPasswordEndpoint{}).Materialize().WithService(inner, authController)
	revokeRefreshTokenEndpoint := (&authep.RevokeRefreshTokenEndpoint{}).Materialize().WithService(inner, authController)
	updatePasswordEndpoint := (&authep.UpdatePasswordEndpoint{}).MaterializeWithMiddleware(config.AuthClient).WithService(inner, authController)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		loginEndpoint,
		registerEndpoint,
		refreshTokenEndpoint,
		requestPasswordResetEndpoint,
		resetPasswordEndpoint,
		revokeRefreshTokenEndpoint,
		updatePasswordEndpoint,
	}

	return &AuthEndpointGroup{inner}
}
