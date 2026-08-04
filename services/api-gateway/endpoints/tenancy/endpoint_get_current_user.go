package tenancyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve the authenticated user's profile.
type GetCurrentUserRequest struct{}

// Returns the profile of the user the request is authenticated as.
//
// This can be called before an account is selected, such as immediately after authentication. Unlike elsewhere, the `image_url` returned here is a short-lived signed link to the image itself, and it is only produced when the request targets an account; without one, no `image_url` is returned even for a user who has uploaded a photo.
type GetCurrentUserEndpoint struct{}

func (e *GetCurrentUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetCurrentUserRequest, *apiresource.User] {
	return (&apiendpoint.APIEndpoint[*GetCurrentUserRequest, *apiresource.User]{
		Title:             "Get Current User",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/me",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeUser,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetCurrentUserRequest) (*apiresource.User, *apierror.APIError) {
			return svc.(TenancySvc).GetCurrentUser
		},
		Extras: apiendpoint.APIEndpointExtras{
			HideFromRequestLog: true,
		},
	})
}
