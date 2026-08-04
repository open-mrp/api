package userep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to upload a user profile photo.
type UploadUserPhotoRequest struct {
	// User ID.
	UserID string `path:"id" validate:"required"`
	// The image itself, sent as the raw request body rather than as JSON or a multipart form.
	RawBody []byte `rawbody:"true"`
	// MIME type of the image (e.g. `image/png`).
	ContentType string `header:"Content-Type"`
}

// Uploads a profile photo for a user.
//
// The photo replaces any existing one, and the user's `image_url` is repointed at an internal path rather than a fetchable image URL. Because the stored image is not publicly readable, use Get User Photo URL to obtain a temporary link for displaying it.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainTeamUsers, Action: types.ActionUpdate},
		},
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestBodyParsing: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UploadUserPhotoRequest) (*apiresource.UserPhotoUploadResult, *apierror.APIError) {
			return svc.(UserSvc).UploadUserPhoto
		},
	})
}
