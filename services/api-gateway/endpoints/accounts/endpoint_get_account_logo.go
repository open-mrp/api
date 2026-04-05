package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetAccountLogoURLRequest is the request to get a presigned logo URL.
type GetAccountLogoURLRequest struct {
	// The ID of the account.
	AccountID string `path:"id" validate:"required"`
}

type GetAccountLogoURLEndpoint struct{}

func (e *GetAccountLogoURLEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAccountLogoURLRequest, *apiresource.AccountLogoURL] {
	return &apiendpoint.APIEndpoint[*GetAccountLogoURLRequest, *apiresource.AccountLogoURL]{
		Title:             "Get Account Logo URL",
		Description:       "Returns a presigned URL for the account's logo image. The URL expires after one hour.",
		Method:            http.MethodGet,
		Route:             "/v1/identity/accounts/{id}/logo",
		ContentType:       "application/json",
		Request:           &GetAccountLogoURLRequest{},
		Response:          &apiresource.AccountLogoURL{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAccountLogoURLRequest) (*apiresource.AccountLogoURL, *apierror.APIError) {
			return svc.(AccountSvc).GetAccountLogoURL
		},
	}
}
