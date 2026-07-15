package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to upload a customer-portal favicon.
type UploadAccountFaviconRequest struct {
	// Account ID.
	AccountID string `path:"id" validate:"required"`
	// Raw image bytes.
	RawBody []byte `rawbody:"true"`
	// Content type of the image (e.g., image/png).
	//
	// The image is stored as `image/png` if this header is omitted, so set it to match the actual format you upload.
	ContentType string `header:"Content-Type"`
}

// Uploads a customer-portal favicon.
//
// Send the image as the raw request body, not as multipart form data. Use a small square PNG (e.g. 32x32 or 64x64) for the best result in browser tabs. The uploaded image replaces any existing favicon and is shown on the account's customer portal. You can only upload a favicon for the account you are acting in.
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
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UploadAccountFaviconRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountSvc).UploadAccountFavicon
		},
	})
}
