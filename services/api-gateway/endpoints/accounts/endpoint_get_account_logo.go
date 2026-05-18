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
	// Account ID.
	AccountID string `path:"id" validate:"required"`
}

// Returns a presigned URL for the account's logo. Expires after one hour.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAccountLogoURLRequest) (*apiresource.AccountLogoURL, *apierror.APIError) {
			return svc.(AccountSvc).GetAccountLogoURL
		},
	})
}
