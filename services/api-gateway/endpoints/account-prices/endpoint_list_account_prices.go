package accountpriceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list account prices.
type ListAccountPricesRequest struct {
	apiresource.PaginationRequest
	// Recipient account ID filter.
	RecipientAccountID *string `query:"recipient_account_id"`
}

// Returns a paginated list of account prices for the current account.
type ListAccountPricesEndpoint struct{}

func (e *ListAccountPricesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountPricesRequest, *apiresource.List[apiresource.AccountPrice]] {
	return (&apiendpoint.APIEndpoint[*ListAccountPricesRequest, *apiresource.List[apiresource.AccountPrice]]{
		Title:             "List Account Prices",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-prices",
		Request:           &ListAccountPricesRequest{},
		Response:          &apiresource.List[apiresource.AccountPrice]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountPricesRequest) (*apiresource.List[apiresource.AccountPrice], *apierror.APIError) {
			return svc.(AccountPriceSvc).ListAccountPrices
		},
	}).WithDocSource(e)
}
