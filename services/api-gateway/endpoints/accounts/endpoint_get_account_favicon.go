package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a presigned favicon URL.
type GetAccountFaviconURLRequest struct {
	// Account ID.
	AccountID string `path:"id" validate:"required"`
}

// Returns a presigned download URL for the account's customer-portal favicon.
//
// The URL expires one hour after it is generated, so fetch the favicon promptly rather than caching it.
type GetAccountFaviconURLEndpoint struct{}

func (e *GetAccountFaviconURLEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAccountFaviconURLRequest, *apiresource.AccountFaviconURL] {
	return (&apiendpoint.APIEndpoint[*GetAccountFaviconURLRequest, *apiresource.AccountFaviconURL]{
		Title:             "Get Account Favicon URL",
		Method:            http.MethodGet,
		Route:             "/v1/identity/accounts/{id}/favicon",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAccountFaviconURLRequest) (*apiresource.AccountFaviconURL, *apierror.APIError) {
			return svc.(AccountSvc).GetAccountFaviconURL
		},
	})
}
