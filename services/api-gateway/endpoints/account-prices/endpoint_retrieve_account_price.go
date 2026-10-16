package accountpriceep

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

// Request to retrieve an account price.
type RetrieveAccountPriceRequest struct {
	// Account price ID.
	AccountPriceID string `path:"id" validate:"required"`
}

// Returns an account price by ID.
//
// A customer or supplier portal user can only retrieve a price whose recipient is their own account or its parent; any other price is reported as not found.
type RetrieveAccountPriceEndpoint struct{}

func (e *RetrieveAccountPriceEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAccountPriceRequest, *apiresource.AccountPrice] {
	return (&apiendpoint.APIEndpoint[*RetrieveAccountPriceRequest, *apiresource.AccountPrice]{
		Title:             "Retrieve Account Price",
		Method:            http.MethodGet,
		Route:             "/v1/sales/account-prices/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeAccountPrice,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError) {
			return svc.(AccountPriceSvc).GetAccountPrice
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountPrice,
			Fields:     []string{"recipient_account", "product_line", "categories", "attributes"},
		}),
	})
}
