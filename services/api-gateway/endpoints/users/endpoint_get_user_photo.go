package userep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetUserPhotoURLRequest is the request to get a presigned photo URL.
type GetUserPhotoURLRequest struct {
	// The ID of the user.
	UserID string `path:"id" validate:"required"`
}

type GetUserPhotoURLEndpoint struct{}

func (e *GetUserPhotoURLEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetUserPhotoURLRequest, *apiresource.UserPhotoURL] {
	return &apiendpoint.APIEndpoint[*GetUserPhotoURLRequest, *apiresource.UserPhotoURL]{
		Title:             "Get User Photo URL",
		Description:       "Returns a presigned URL for the user's profile photo. The URL expires after one hour.",
		Method:            http.MethodGet,
		Route:             "/v1/identity/users/{id}/photo",
		ContentType:       "application/json",
		Request:           &GetUserPhotoURLRequest{},
		Response:          &apiresource.UserPhotoURL{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetUserPhotoURLRequest) (*apiresource.UserPhotoURL, *apierror.APIError) {
			return svc.(UserSvc).GetUserPhotoURL
		},
	}
}
