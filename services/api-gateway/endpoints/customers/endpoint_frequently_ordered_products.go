package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetFrequentlyOrderedProductsRequest is the request to get a customer's frequently ordered products.
type GetFrequentlyOrderedProductsRequest struct {
	// The ID of the customer.
	CustomerID string `path:"id" validate:"required"`
}

type GetFrequentlyOrderedProductsEndpoint struct{}

func (e *GetFrequentlyOrderedProductsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetFrequentlyOrderedProductsRequest, *apiresource.List[apiresource.FrequentlyOrderedProduct]] {
	return &apiendpoint.APIEndpoint[*GetFrequentlyOrderedProductsRequest, *apiresource.List[apiresource.FrequentlyOrderedProduct]]{
		Title:             "Get Frequently Ordered Products",
		Description:       "Returns the most frequently ordered products for a customer based on historical sales order data.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/customers/{id}/frequently-ordered-products",
		Request:           &GetFrequentlyOrderedProductsRequest{},
		Response:          &apiresource.List[apiresource.FrequentlyOrderedProduct]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetFrequentlyOrderedProductsRequest) (*apiresource.List[apiresource.FrequentlyOrderedProduct], *apierror.APIError) {
			return svc.(CustomerSvc).GetFrequentlyOrderedProducts
		},
	}
}
