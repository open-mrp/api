package userep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UploadUserPhotoRequest is the request to upload a user profile photo.
type UploadUserPhotoRequest struct {
	// The ID of the user.
	UserID string `path:"id" validate:"required"`
	// The raw image bytes.
	RawBody []byte `rawbody:"true"`
	// The content type of the image (e.g. image/png).
	ContentType string `header:"Content-Type"`
}

type UploadUserPhotoEndpoint struct{}

func (e *UploadUserPhotoEndpoint) Materialize() *apiendpoint.APIEndpoint[*UploadUserPhotoRequest, *apiresource.UserPhotoUploadResult] {
	return &apiendpoint.APIEndpoint[*UploadUserPhotoRequest, *apiresource.UserPhotoUploadResult]{
		Title:             "Upload User Photo",
		Description:       "Uploads a profile photo for a user.",
		Method:            http.MethodPut,
		Route:             "/v1/identity/users/{id}/photo",
		Request:           &UploadUserPhotoRequest{},
		Response:          &apiresource.UserPhotoUploadResult{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestBodyParsing: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UploadUserPhotoRequest) (*apiresource.UserPhotoUploadResult, *apierror.APIError) {
			return svc.(UserSvc).UploadUserPhoto
		},
	}
}
