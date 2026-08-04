package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a presigned logo URL.
type GetAccountLogoURLRequest struct {
	// ID of the account whose logo is being fetched.
	AccountID string `path:"id" validate:"required"`
}

// Returns a presigned download URL for the account's logo.
//
// The URL expires one hour after it is generated, so fetch the logo promptly rather than caching it. The response carries no URL when the account has never uploaded a logo or the stored image is no longer available.
type GetAccountLogoURLEndpoint struct{}

func (e *GetAccountLogoURLEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAccountLogoURLRequest, *apiresource.AccountLogoURL] {
	return (&apiendpoint.APIEndpoint[*GetAccountLogoURLRequest, *apiresource.AccountLogoURL]{
		Title:             "Get Account Logo URL",
		Method:            http.MethodGet,
		Route:             "/v1/identity/accounts/{id}/logo",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAccountLogoURLRequest) (*apiresource.AccountLogoURL, *apierror.APIError) {
			return svc.(AccountSvc).GetAccountLogoURL
		},
	})
}
