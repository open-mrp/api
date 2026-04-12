package tenancyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetCurrentUserRequest is the request to retrieve the authenticated user's profile.
type GetCurrentUserRequest struct{}

type GetCurrentUserEndpoint struct{}

func (e *GetCurrentUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetCurrentUserRequest, *apiresource.User] {
	return &apiendpoint.APIEndpoint[*GetCurrentUserRequest, *apiresource.User]{
		Title:             "Get Current User",
		Description:       "Returns the authenticated user's profile information.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/me",
		Request:           &GetCurrentUserRequest{},
		Response:          &apiresource.User{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetCurrentUserRequest) (*apiresource.User, *apierror.APIError) {
			return svc.(TenancySvc).GetCurrentUser
		},
	}
}
