package userep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to upload a user profile photo.
type UploadUserPhotoRequest struct {
	// User ID.
	UserID string `path:"id" validate:"required"`
	// Raw image bytes.
	RawBody []byte `rawbody:"true"`
	// Content type of the image (e.g. image/png).
	ContentType string `header:"Content-Type"`
}

// Uploads a profile photo for a user.
type UploadUserPhotoEndpoint struct{}

func (e *UploadUserPhotoEndpoint) Materialize() *apiendpoint.APIEndpoint[*UploadUserPhotoRequest, *apiresource.UserPhotoUploadResult] {
	return (&apiendpoint.APIEndpoint[*UploadUserPhotoRequest, *apiresource.UserPhotoUploadResult]{
		Title:             "Upload User Photo",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/identity/users/{id}/photo",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestBodyParsing: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UploadUserPhotoRequest) (*apiresource.UserPhotoUploadResult, *apierror.APIError) {
			return svc.(UserSvc).UploadUserPhoto
		},
	})
}
