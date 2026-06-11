package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to upload an account logo.
type UploadAccountPhotoRequest struct {
	// Account ID.
	AccountID string `path:"id" validate:"required"`
	// Raw image bytes.
	RawBody []byte `rawbody:"true"`
	// Content type of the image (e.g., image/png).
	//
	// The image is stored as `image/png` if this header is omitted, so set it to match the actual format you upload.
	ContentType string `header:"Content-Type"`
}

// Uploads an account logo.
//
// Send the image as the raw request body, not as multipart form data. The uploaded image replaces any existing logo and can be retrieved via the Get Account Logo URL endpoint. You can only upload a logo for the account you are acting in.
type UploadAccountPhotoEndpoint struct{}

func (e *UploadAccountPhotoEndpoint) Materialize() *apiendpoint.APIEndpoint[*UploadAccountPhotoRequest, *apiresource.AccountPhotoUploadResult] {
	return (&apiendpoint.APIEndpoint[*UploadAccountPhotoRequest, *apiresource.AccountPhotoUploadResult]{
		Title:             "Upload Account Logo",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/identity/accounts/{id}/photo",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestBodyParsing: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UploadAccountPhotoRequest) (*apiresource.AccountPhotoUploadResult, *apierror.APIError) {
			return svc.(AccountSvc).UploadAccountPhoto
		},
	})
}
