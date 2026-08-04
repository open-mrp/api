package userep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request for a temporary link to a user's profile photo.
type GetUserPhotoURLRequest struct {
	// User ID.
	UserID string `path:"id" validate:"required"`
}

// Returns a temporary link that can be used to fetch the user's profile photo image.
//
// The link expires one hour after it is issued, and no link is returned for a user who has never uploaded a photo. Users may always fetch their own photo; fetching another user's photo requires read access to team users.
type GetUserPhotoURLEndpoint struct{}

func (e *GetUserPhotoURLEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetUserPhotoURLRequest, *apiresource.UserPhotoURL] {
	return (&apiendpoint.APIEndpoint[*GetUserPhotoURLRequest, *apiresource.UserPhotoURL]{
		Title:             "Get User Photo URL",
		Method:            http.MethodGet,
		Route:             "/v1/identity/users/{id}/photo",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetUserPhotoURLRequest) (*apiresource.UserPhotoURL, *apierror.APIError) {
			return svc.(UserSvc).GetUserPhotoURL
		},
	})
}
