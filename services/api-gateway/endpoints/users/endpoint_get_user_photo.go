package userep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request for a user's presigned photo URL.
type GetUserPhotoURLRequest struct {
	// User ID.
	UserID string `path:"id" validate:"required"`
}

// Returns a presigned URL for the user's profile photo.
//
// The URL expires one hour after it is issued.
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
