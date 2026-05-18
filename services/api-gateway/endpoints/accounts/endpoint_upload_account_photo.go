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
	ContentType string `header:"Content-Type"`
}

// Uploads an account logo. Send as raw binary body.
type UploadAccountPhotoEndpoint struct{}

func (e *UploadAccountPhotoEndpoint) Materialize() *apiendpoint.APIEndpoint[*UploadAccountPhotoRequest, *apiresource.AccountPhotoUploadResult] {
	return (&apiendpoint.APIEndpoint[*UploadAccountPhotoRequest, *apiresource.AccountPhotoUploadResult]{
		Title:             "Upload Account Photo",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/identity/accounts/{id}/photo",
		Request:           &UploadAccountPhotoRequest{},
		Response:          &apiresource.AccountPhotoUploadResult{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestBodyParsing: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UploadAccountPhotoRequest) (*apiresource.AccountPhotoUploadResult, *apierror.APIError) {
			return svc.(AccountSvc).UploadAccountPhoto
		},
	}).WithDocSource(e)
}
