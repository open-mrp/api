package accountpriceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetAccountPriceRequest is the request to retrieve a single account price.
type GetAccountPriceRequest struct {
	// The ID of the account price to retrieve.
	AccountPriceID string `path:"id" validate:"required"`
}

type GetAccountPriceEndpoint struct{}

func (e *GetAccountPriceEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAccountPriceRequest, *apiresource.AccountPrice] {
	return &apiendpoint.APIEndpoint[*GetAccountPriceRequest, *apiresource.AccountPrice]{
		Title:             "Get Account Price",
		Description:       "Returns a single account price by its ID.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/account-prices/{id}",
		ContentType:       "application/json",
		Request:           &GetAccountPriceRequest{},
		Response:          &apiresource.AccountPrice{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError) {
			return svc.(AccountPriceSvc).GetAccountPrice
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountPrice,
			Fields:     []string{"recipient_account", "product_line", "categories", "attributes"},
		}),
	}
}
