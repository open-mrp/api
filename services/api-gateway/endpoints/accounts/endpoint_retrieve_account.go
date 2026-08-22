package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve an account by ID.
type RetrieveAccountRequest struct {
	// ID of the account to retrieve.
	AccountID string `path:"id" validate:"required"`
}

// Returns an account by ID.
//
// You can only retrieve the account you are acting in; requesting any other account is rejected.
type RetrieveAccountEndpoint struct{}

func (e *RetrieveAccountEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAccountRequest, *apiresource.Account] {
	return (&apiendpoint.APIEndpoint[*RetrieveAccountRequest, *apiresource.Account]{
		Title:               "Retrieve Account",
		Method:              http.MethodGet,
		Route:               "/v1/identity/accounts/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAccount,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAccountRequest) (*apiresource.Account, *apierror.APIError) {
			return svc.(AccountSvc).GetAccount
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccount,
			Fields:     []string{"branding", "portal"},
		}),
	})
}
