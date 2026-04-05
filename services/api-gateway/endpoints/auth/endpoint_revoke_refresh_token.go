package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to revoke a refresh token
type RevokeRefreshTokenRequest struct {
	// The refresh token cookie.
	RefreshToken string `cookie:"__Secure-augno.refresh-token" validate:"required"` // #nosec G117 - Struct field, not a hardcoded credential
}

var sampleRevokeRefreshTokenRequest = &RevokeRefreshTokenRequest{
	RefreshToken: apiresource.SampleRefreshTokenToken,
}

func (*RevokeRefreshTokenRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRevokeRefreshTokenRequest)
}

type RevokeRefreshTokenEndpoint struct{}

func (e *RevokeRefreshTokenEndpoint) Materialize() *apiendpoint.APIEndpoint[*RevokeRefreshTokenRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*RevokeRefreshTokenRequest, *apiresource.EmptyResource]{
		Title:             "Revoke Refresh Token",
		Description:       "Revokes a refresh token, making it no longer valid for obtaining new access tokens.",
		Method:            http.MethodDelete,
		Route:             "/v1/auth/refresh-tokens",
		ContentType:       "application/json",
		Request:           &RevokeRefreshTokenRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RevokeRefreshTokenRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AuthSvc).RevokeRefreshToken
		},
	}
}
