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
	// Refresh token cookie.
	RefreshToken string `cookie:"__Secure-augno.refresh-token" validate:"required"` // #nosec G117 - Struct field, not a hardcoded credential
}

var sampleRefreshTokenRequest = &RefreshTokenRequest{
	RefreshToken: apiresource.SampleRefreshTokenToken,
}

func (*RefreshTokenRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRefreshTokenRequest)
}

type RefreshTokenEndpoint struct{}

func (e *RefreshTokenEndpoint) Materialize() *apiendpoint.APIEndpoint[*RefreshTokenRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*RefreshTokenRequest, *apiresource.EmptyResource]{
		Title:             "Refresh Token",
		Description:       "Refreshes an access token using a refresh token, setting a new access token in a cookie.",
		Method:            http.MethodPut,
		Route:             "/v1/auth/access-tokens",
		ContentType:       "application/json",
		Request:           &RefreshTokenRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RefreshTokenRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AuthSvc).RefreshToken
		},
	}
}
