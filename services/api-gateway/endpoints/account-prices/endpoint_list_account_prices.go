package accountpriceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list account prices.
type ListAccountPricesRequest struct {
	apiresource.PaginationRequest
	// Filters results to prices whose recipient is this customer account.
	//
	// A child account also matches the prices recorded against its parent, since those price its orders too.
	RecipientAccountID *string `query:"recipient_account_id"`
}

// Returns a paginated list of account prices, newest first.
//
// The search term matches the recipient customer's name or their customer number. Customer portal users always see only the prices that apply to their own account, whatever `recipient_account_id` is set to.
type ListAccountPricesEndpoint struct{}

func (e *ListAccountPricesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountPricesRequest, *apiresource.List[apiresource.AccountPrice]] {
	return (&apiendpoint.APIEndpoint[*ListAccountPricesRequest, *apiresource.List[apiresource.AccountPrice]]{
		Title:             "List Account Prices",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-prices",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountPrice,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountPricesRequest) (*apiresource.List[apiresource.AccountPrice], *apierror.APIError) {
			return svc.(AccountPriceSvc).ListAccountPrices
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountPrice,
			Fields:     []string{"recipient_account", "product_line", "categories", "attributes"},
		}),
	})
}
