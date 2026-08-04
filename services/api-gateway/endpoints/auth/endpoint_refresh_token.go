package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to refresh an access token.
type RefreshTokenRequest struct {
	// Refresh token, read from the `__Secure-augno.refresh-token` cookie.
	RefreshToken string `cookie:"__Secure-augno.refresh-token" validate:"required"` // #nosec G117 - Struct field, not a hardcoded credential
}

var sampleRefreshTokenRequest = &RefreshTokenRequest{
	RefreshToken: apiresource.SampleRefreshTokenToken,
}

func (*RefreshTokenRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRefreshTokenRequest)
}

// Issues a new access token from the caller's refresh token, setting it in a cookie.
//
// The refresh token itself is not rotated and keeps its original expiration, so the same cookie can be exchanged repeatedly until it expires or is revoked. A refresh token that has been revoked or has expired fails here and the user must sign in again.
type RefreshTokenEndpoint struct{}

func (e *RefreshTokenEndpoint) Materialize() *apiendpoint.APIEndpoint[*RefreshTokenRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*RefreshTokenRequest, *apiresource.EmptyResource]{
		Title:             "Refresh Access Token",
		Method:            http.MethodPut,
		Route:             "/v1/auth/access-tokens",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RefreshTokenRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AuthSvc).RefreshToken
		},
		Extras: apiendpoint.APIEndpointExtras{
			HideFromRequestLog: true,
		},
	})
}
