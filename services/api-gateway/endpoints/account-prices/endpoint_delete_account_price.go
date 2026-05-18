package accountpriceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete an account price.
type DeleteAccountPriceRequest struct {
	// Account price ID.
	AccountPriceID string `path:"id" validate:"required"`
}

// Deletes an account price. Associated category constraints, attribute constraints, and the rate record are also removed.
type DeleteAccountPriceEndpoint struct{}

func (e *DeleteAccountPriceEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAccountPriceRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteAccountPriceRequest, *apiresource.EmptyResource]{
		Title:             "Delete Account Price",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/account-prices/{id}",
		ContentType:       "application/json",
		Request:           &DeleteAccountPriceRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAccountPriceRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountPriceSvc).DeleteAccountPrice
		},
	}).WithDocSource(e)
}
