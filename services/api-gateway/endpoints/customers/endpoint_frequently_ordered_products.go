package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get frequently ordered products for a customer.
type GetFrequentlyOrderedProductsRequest struct {
	// Customer ID.
	CustomerID string `path:"id" validate:"required"`
}

// Returns the products a customer orders most often, based on historical sales order data.
//
// Returns up to 12 items, each paired with the unit the customer orders it in most often and ranked by the number of order lines placed in that unit. Only products of type `sale` that are visible in the customer portal are counted, so lines for services, shipping, tax, credits, and returns are ignored, and hiding a product from the portal removes its lines from the ranking.
type GetFrequentlyOrderedProductsEndpoint struct{}

func (e *GetFrequentlyOrderedProductsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetFrequentlyOrderedProductsRequest, *apiresource.List[apiresource.FrequentlyOrderedProduct]] {
	return (&apiendpoint.APIEndpoint[*GetFrequentlyOrderedProductsRequest, *apiresource.List[apiresource.FrequentlyOrderedProduct]]{
		Title:               "Get Frequently Ordered Products",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/sales/customers/{id}/frequently-ordered-products",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}, {Domain: types.PermissionDomainItems, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetFrequentlyOrderedProductsRequest) (*apiresource.List[apiresource.FrequentlyOrderedProduct], *apierror.APIError) {
			return svc.(CustomerSvc).GetFrequentlyOrderedProducts
		},
	})
}
