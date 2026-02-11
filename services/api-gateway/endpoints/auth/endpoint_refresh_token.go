package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Represents a request to refresh an access token
type RefreshTokenRequest struct {
	// The refresh token cookie
	RefreshToken string `cookie:"__Secure-augno.refresh-token" validate:"required"` // #nosec G117 - Struct field, not a hardcoded credential
}

var sampleRefreshTokenRequest = &RefreshTokenRequest{
	RefreshToken: apiresource.SampleRefreshTokenToken,
}

func (*RefreshTokenRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRefreshTokenRequest)
}

type RefreshTokenEndpoint struct{}

const refreshTokenEndpointDescription = `This endpoint is utilized to refresh an access token using a refresh token.
Once completed, a new access token is set in a cookie.`

func (e *RefreshTokenEndpoint) Materialize() *apiendpoint.APIEndpoint[*RefreshTokenRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*RefreshTokenRequest, *apiresource.EmptyResource]{
		Title:             "Refresh Token",
		Description:       refreshTokenEndpointDescription,
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
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
