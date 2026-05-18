package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve an account by ID.
type RetrieveAccountRequest struct {
	// Account ID.
	AccountID string `path:"id" validate:"required"`
}

// Returns an account by ID.
type RetrieveAccountEndpoint struct{}

func (e *RetrieveAccountEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAccountRequest, *apiresource.Account] {
	return (&apiendpoint.APIEndpoint[*RetrieveAccountRequest, *apiresource.Account]{
		Title:             "Retrieve Account",
		Method:            http.MethodGet,
		Route:             "/v1/identity/accounts/{id}",
		ContentType:       "application/json",
		Request:           &RetrieveAccountRequest{},
		Response:          &apiresource.Account{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAccountRequest) (*apiresource.Account, *apierror.APIError) {
			return svc.(AccountSvc).GetAccount
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccount,
			Fields:     []string{"branding", "portal"},
		}),
	}).WithDocSource(e)
}
