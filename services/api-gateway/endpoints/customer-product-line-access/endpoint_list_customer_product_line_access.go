package customerproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list product line access records grouped by customer.
type ListCustomerProductLineAccessRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of product line access records grouped by customer.
type ListCustomerProductLineAccessEndpoint struct{}

func (e *ListCustomerProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCustomerProductLineAccessRequest, *apiresource.List[apiresource.CustomerProductLineAccess]] {
	return (&apiendpoint.APIEndpoint[*ListCustomerProductLineAccessRequest, *apiresource.List[apiresource.CustomerProductLineAccess]]{
		Title:             "List Customer Product Line Access",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/customers",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCustomerProductLineAccessRequest) (*apiresource.List[apiresource.CustomerProductLineAccess], *apierror.APIError) {
			return svc.(CustomerProductLineAccessSvc).ListCustomerProductLineAccess
		},
	})
}
