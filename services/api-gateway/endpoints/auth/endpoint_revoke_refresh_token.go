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

// Request to revoke a refresh token
type RevokeRefreshTokenRequest struct {
	// The refresh token (can be provided via Authorization header with Bearer or Basic scheme, or via refresh token cookie)
	RefreshToken string `header:"Authorization" cookie:"__Secure-augno.refresh-token" validate:"required"`
}

var sampleRevokeRefreshTokenRequest = &RevokeRefreshTokenRequest{
	RefreshToken: apiresource.SampleRefreshTokenToken,
}

func (*RevokeRefreshTokenRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRevokeRefreshTokenRequest)
}

type RevokeRefreshTokenEndpoint struct {
	apiendpoint.APIEndpoint[*RevokeRefreshTokenRequest, *apiresource.EmptyResource]

	group    *apiendpoint.APIEndpointGroup
	service  AuthCtrl
	platform constants.PlatformMode
	bindOnce sync.Once
	handler  http.HandlerFunc
}

const revokeRefreshTokenEndpointDescription = `This endpoint is utilized to revoke a refresh token.
Once completed, the refresh token is revoked and is no longer valid for refreshing an access token. Learn more about authentication and authorization in our 
[documentation](https://docs.augno.com/guides/authentication).
`

func (e *RevokeRefreshTokenEndpoint) Materialize() apiendpoint.APIEndpointer {
	e.APIEndpoint = apiendpoint.APIEndpoint[*RevokeRefreshTokenRequest, *apiresource.EmptyResource]{
		Title:             "Revoke Refresh Token",
		Description:       revokeRefreshTokenEndpointDescription,
		Method:            http.MethodDelete,
		Route:             "/v1/auth/refresh-tokens",
		ContentType:       "application/json",
		Request:           &RevokeRefreshTokenRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		IsPublic:          false,
		Handler: func(ctrl any) apiendpoint.HandlerFunc[
			*RevokeRefreshTokenRequest, *apiresource.EmptyResource,
		] {
			return apiendpoint.HandlerFunc[
				*RevokeRefreshTokenRequest, *apiresource.EmptyResource,
			](func(ctx context.Context, req *RevokeRefreshTokenRequest) (*apiresource.EmptyResource, *contracts.APIError) {
				return ctrl.(AuthCtrl).RevokeRefreshToken(ctx, req)
			})
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
	return e
}

func (e *RevokeRefreshTokenEndpoint) GetHandler() http.HandlerFunc {
	e.bindOnce.Do(func() {
		be := apiendpoint.Bind(e.APIEndpoint, e.service)
		e.handler = httptransport.ConvertToHTTPHandler(be)
	})
	return e.handler
}

func (e *RevokeRefreshTokenEndpoint) WithGroup(g *apiendpoint.APIEndpointGroup, service AuthCtrl, platform constants.PlatformMode) *RevokeRefreshTokenEndpoint {
	e.group = g
	e.service = service
	e.platform = platform
	return e
}
