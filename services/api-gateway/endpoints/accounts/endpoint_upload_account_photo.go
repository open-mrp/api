package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UploadAccountPhotoRequest is the request to upload an account logo.
type UploadAccountPhotoRequest struct {
	// The ID of the account.
	AccountID string `path:"id" validate:"required"`
	// The raw image bytes.
	RawBody []byte `rawbody:"true"`
	// The content type of the image (e.g. image/png).
	ContentType string `header:"Content-Type"`
}

type UploadAccountPhotoEndpoint struct{}

func (e *UploadAccountPhotoEndpoint) Materialize() *apiendpoint.APIEndpoint[*UploadAccountPhotoRequest, *apiresource.AccountPhotoUploadResult] {
	return &apiendpoint.APIEndpoint[*UploadAccountPhotoRequest, *apiresource.AccountPhotoUploadResult]{
		Title:             "Upload Account Photo",
		Description:       "Uploads a logo image for an account as a raw binary body.",
		Method:            http.MethodPut,
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
	}
}
