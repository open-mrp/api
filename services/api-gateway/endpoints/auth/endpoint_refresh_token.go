package authep

import (
	"context"
	"net/http"
	"sync"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
)

// Represents a request to refresh an access token
type RefreshTokenRequest struct {
	// The refresh token (can be provided via Authorization header with Bearer or Basic scheme, or via refresh token cookie)
	RefreshToken string `header:"Authorization" cookie:"__Secure-augno.refresh-token" validate:"required"`
}

var sampleRefreshTokenRequest = &RefreshTokenRequest{
	RefreshToken: apiresource.SampleRefreshTokenToken,
}

func (*RefreshTokenRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRefreshTokenRequest)
}

type RefreshTokenEndpoint struct {
	apiendpoint.APIEndpoint[*RefreshTokenRequest, *apiresource.EmptyResource]

	group    *apiendpoint.APIEndpointGroup
	service  AuthCtrl
	platform constants.PlatformMode
	bindOnce sync.Once
	handler  http.HandlerFunc
}

const refreshTokenEndpointDescription = `This endpoint is utilized to refresh an access token using a refresh token.
Once completed, a new access token is set in a cookie. Learn more about authentication and authorization in our 
[documentation](https://docs.augno.com/guides/authentication).
`

func (e *RefreshTokenEndpoint) Materialize() apiendpoint.APIEndpointer {
	e.APIEndpoint = apiendpoint.APIEndpoint[*RefreshTokenRequest, *apiresource.EmptyResource]{
		Title:             "Refresh Token",
		Description:       refreshTokenEndpointDescription,
		Method:            http.MethodPost,
		Route:             "/v1/auth/access-tokens",
		ContentType:       "application/json",
		Request:           &RefreshTokenRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		IsPublic:          true,
		Handler: func(ctrl any) apiendpoint.HandlerFunc[
			*RefreshTokenRequest, *apiresource.EmptyResource,
		] {
			return apiendpoint.HandlerFunc[
				*RefreshTokenRequest, *apiresource.EmptyResource,
			](func(ctx context.Context, req *RefreshTokenRequest) (*apiresource.EmptyResource, *contracts.APIError) {
				return ctrl.(AuthCtrl).RefreshToken(ctx, req)
			})
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
	return e
}

func (e *RefreshTokenEndpoint) GetHandler() http.HandlerFunc {
	e.bindOnce.Do(func() {
		be := apiendpoint.Bind(e.APIEndpoint, e.service)
		e.handler = httptransport.ConvertToHTTPHandler(be)
	})
	return e.handler
}

func (e *RefreshTokenEndpoint) WithGroup(g *apiendpoint.APIEndpointGroup, service AuthCtrl, platform constants.PlatformMode) *RefreshTokenEndpoint {
	e.group = g
	e.service = service
	e.platform = platform
	return e
}
