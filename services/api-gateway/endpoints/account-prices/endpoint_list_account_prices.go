package accountpriceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListAccountPricesRequest is the request to list account prices with optional filters.
type ListAccountPricesRequest struct {
	apiresource.PaginationRequest
	// Filter by recipient account ID.
	RecipientAccountID *string `query:"recipient_account_id"`
}

type ListAccountPricesEndpoint struct{}

func (e *ListAccountPricesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountPricesRequest, *apiresource.List[apiresource.AccountPrice]] {
	return &apiendpoint.APIEndpoint[*ListAccountPricesRequest, *apiresource.List[apiresource.AccountPrice]]{
		Title:             "List Account Prices",
		Description:       "Returns a paginated list of account prices for the current account.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/account-prices",
		Request:           &ListAccountPricesRequest{},
		Response:          &apiresource.List[apiresource.AccountPrice]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountPricesRequest) (*apiresource.List[apiresource.AccountPrice], *apierror.APIError) {
			return svc.(AccountPriceSvc).ListAccountPrices
		},
	}
}
