package accountep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
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
		Title:             "Retrieve Account",
		Method:            http.MethodGet,
		Route:             "/v1/identity/accounts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccount,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAccountRequest) (*apiresource.Account, *apierror.APIError) {
			return svc.(AccountSvc).GetAccount
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccount,
			Fields:     []string{"branding", "portal", "default_billing_address", "default_shipping_address"},
		}),
	})
}
