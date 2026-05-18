package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get frequently ordered products for a customer.
type GetFrequentlyOrderedProductsRequest struct {
	// Customer ID.
	CustomerID string `path:"id" validate:"required"`
}

// Returns the most frequently ordered products for a customer based on historical sales order data.
type GetFrequentlyOrderedProductsEndpoint struct{}

func (e *GetFrequentlyOrderedProductsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetFrequentlyOrderedProductsRequest, *apiresource.List[apiresource.FrequentlyOrderedProduct]] {
	return (&apiendpoint.APIEndpoint[*GetFrequentlyOrderedProductsRequest, *apiresource.List[apiresource.FrequentlyOrderedProduct]]{
		Title:             "Get Frequently Ordered Products",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/customers/{id}/frequently-ordered-products",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetFrequentlyOrderedProductsRequest) (*apiresource.List[apiresource.FrequentlyOrderedProduct], *apierror.APIError) {
			return svc.(CustomerSvc).GetFrequentlyOrderedProducts
		},
	})
}
