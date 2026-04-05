package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetAccountRequest is the request to retrieve a full account by ID.
type GetAccountRequest struct {
	// The ID of the account to retrieve.
	AccountID string `path:"id" validate:"required"`
}

type GetAccountEndpoint struct{}

func (e *GetAccountEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAccountRequest, *apiresource.Account] {
	return &apiendpoint.APIEndpoint[*GetAccountRequest, *apiresource.Account]{
		Title:             "Get Account",
		Description:       "Returns an account by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/identity/accounts/{id}",
		ContentType:       "application/json",
		Request:           &GetAccountRequest{},
		Response:          &apiresource.Account{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAccountRequest) (*apiresource.Account, *apierror.APIError) {
			return svc.(AccountSvc).GetAccount
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccount,
			Fields:     []string{"branding", "portal"},
		}),
	}
}
