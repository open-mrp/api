package userep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/imageupload"
)

// Request to upload a user profile photo.
type UploadUserPhotoRequest struct {
	// User ID.
	UserID string `path:"id" validate:"required"`
	// The image itself, sent as the raw request body rather than as JSON or a multipart form.
	//
	// Must be a PNG, JPEG, GIF, or WebP image of at most 10 MB. Its format is read from the image itself, whatever `Content-Type` the request names.
	RawBody []byte `rawbody:"true"`
}

// Uploads a profile photo for a user.
//
// The photo replaces any existing one, and the user's `image_url` is repointed at an internal path rather than a fetchable image URL. Because the stored image is not publicly readable, use Get User Photo URL to obtain a temporary link for displaying it.
//
// Users may always upload their own photo. Uploading one for another user requires permission to update team users, and that user must belong to the account you are acting in. A body that is empty or not a PNG, JPEG, GIF, or WebP image is refused with a 400, and one over 10 MB with a 413; nothing is stored either way.
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
		SelfPathParam: "id",
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestBodyParsing: true,
			MaxRawBodyBytes:        imageupload.MaxBytes,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UploadUserPhotoRequest) (*apiresource.UserPhotoUploadResult, *apierror.APIError) {
			return svc.(UserSvc).UploadUserPhoto
		},
	})
}
