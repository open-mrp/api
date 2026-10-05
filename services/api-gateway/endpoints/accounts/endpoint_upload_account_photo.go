package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/imageupload"
)

// Request to upload an account logo.
type UploadAccountPhotoRequest struct {
	// ID of the account the logo belongs to.
	AccountID string `path:"id" validate:"required"`
	// Raw image bytes.
	//
	// Must be a PNG, JPEG, GIF, or WebP image of at most 10 MB. Its format is read from the image itself, whatever `Content-Type` the request names.
	RawBody []byte `rawbody:"true"`
}

// Uploads an account logo.
//
// Send the image as the raw request body, not as multipart form data. The uploaded image replaces any existing logo and can be retrieved via the Get Account Logo URL endpoint. You can only upload a logo for the account you are acting in.
//
// A body that is empty or not a PNG, JPEG, GIF, or WebP image is refused with a 400, and one over 10 MB with a 413; the existing logo is kept either way.
type UploadAccountPhotoEndpoint struct{}

func (e *UploadAccountPhotoEndpoint) Materialize() *apiendpoint.APIEndpoint[*UploadAccountPhotoRequest, *apiresource.AccountPhotoUploadResult] {
	return (&apiendpoint.APIEndpoint[*UploadAccountPhotoRequest, *apiresource.AccountPhotoUploadResult]{
		Title:               "Upload Account Logo",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/identity/accounts/{id}/photo",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate}},
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestBodyParsing: true,
			MaxRawBodyBytes:        imageupload.MaxBytes,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UploadAccountPhotoRequest) (*apiresource.AccountPhotoUploadResult, *apierror.APIError) {
			return svc.(AccountSvc).UploadAccountPhoto
		},
	})
}
