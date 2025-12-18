package httpgroup

import (
	authep "github.com/augno/api/services/api-gateway/endpoints/auth"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/shared/constants"
)

type AuthEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AuthEndpointGroupConfig struct {
	PlatformMode constants.PlatformMode
	AuthClient   *grpcclient.AuthServiceClient
}

func (*AuthEndpointGroup) Materialize(config AuthEndpointGroupConfig) *AuthEndpointGroup {
	if config.AuthClient == nil {
		return nil
	}

	authController := authep.NewAuthCtrl(authep.AuthCtrlConfig{
		AuthClient: config.AuthClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:       "Authentication and Token Management",
		Description: "Handles user authentication and token lifecycle operations, including login, registration, password management, and token refresh.",
	}

	loginEndpoint := (&authep.LoginEndpoint{}).Materialize().(*authep.LoginEndpoint)
	loginEndpoint = loginEndpoint.WithGroup(inner, authController, config.PlatformMode)

	registerEndpoint := (&authep.RegisterEndpoint{}).Materialize().(*authep.RegisterEndpoint)
	registerEndpoint = registerEndpoint.WithGroup(inner, authController, config.PlatformMode)

	refreshTokenEndpoint := (&authep.RefreshTokenEndpoint{}).Materialize().(*authep.RefreshTokenEndpoint)
	refreshTokenEndpoint = refreshTokenEndpoint.WithGroup(inner, authController, config.PlatformMode)

	requestPasswordResetEndpoint := (&authep.RequestPasswordResetEndpoint{}).Materialize().(*authep.RequestPasswordResetEndpoint)
	requestPasswordResetEndpoint = requestPasswordResetEndpoint.WithGroup(inner, authController, config.PlatformMode)

	resetPasswordEndpoint := (&authep.ResetPasswordEndpoint{}).Materialize().(*authep.ResetPasswordEndpoint)
	resetPasswordEndpoint = resetPasswordEndpoint.WithGroup(inner, authController, config.PlatformMode)

	revokeRefreshTokenEndpoint := (&authep.RevokeRefreshTokenEndpoint{}).Materialize().(*authep.RevokeRefreshTokenEndpoint)
	revokeRefreshTokenEndpoint = revokeRefreshTokenEndpoint.WithGroup(inner, authController, config.PlatformMode)

	updatePasswordEndpoint := (&authep.UpdatePasswordEndpoint{}).Materialize().(*authep.UpdatePasswordEndpoint)
	updatePasswordEndpoint = updatePasswordEndpoint.WithGroup(inner, authController, config.PlatformMode, config.AuthClient)

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
