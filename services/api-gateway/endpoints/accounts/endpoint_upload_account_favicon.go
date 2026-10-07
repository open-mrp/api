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

// Request to upload a customer-portal favicon.
type UploadAccountFaviconRequest struct {
	// ID of the account the favicon belongs to.
	AccountID string `path:"id" validate:"required"`
	// Raw image bytes.
	//
	// Must be a PNG, JPEG, GIF, WebP, or ICO image of at most 10 MB. Its format is read from the image itself, whatever `Content-Type` the request names.
	RawBody []byte `rawbody:"true"`
}

// Uploads a customer-portal favicon.
//
// Send the image as the raw request body, not as multipart form data. Use a small square PNG (e.g. 32x32 or 64x64) for the best result in browser tabs. The uploaded image replaces any existing favicon and is shown on the account's customer portal. You can only upload a favicon for the account you are acting in.
//
// A body that is empty or not a PNG, JPEG, GIF, WebP, or ICO image is refused with a 400, and one over 10 MB with a 413; the existing favicon is kept either way.
type UploadAccountFaviconEndpoint struct{}

func (e *UploadAccountFaviconEndpoint) Materialize() *apiendpoint.APIEndpoint[*UploadAccountFaviconRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*UploadAccountFaviconRequest, *apiresource.EmptyResource]{
		Title:               "Upload Account Favicon",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/identity/accounts/{id}/favicon",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate}},
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestBodyParsing: true,
			MaxRawBodyBytes:        imageupload.MaxBytes,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UploadAccountFaviconRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountSvc).UploadAccountFavicon
		},
	})
}
