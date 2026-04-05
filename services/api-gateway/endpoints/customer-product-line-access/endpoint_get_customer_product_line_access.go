package customerproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetCustomerProductLineAccessRequest is the request to retrieve product line access for a single customer.
type GetCustomerProductLineAccessRequest struct {
	// The ID of the customer.
	CustomerID string `path:"customer_id" validate:"required"`
}

type GetCustomerProductLineAccessEndpoint struct{}

func (e *GetCustomerProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetCustomerProductLineAccessRequest, *apiresource.CustomerProductLineAccess] {
	return &apiendpoint.APIEndpoint[*GetCustomerProductLineAccessRequest, *apiresource.CustomerProductLineAccess]{
		Title:             "Get Customer Product Line Access",
		Description:       "Returns the product line access for a customer.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/product-line-access/customers/{customer_id}",
		Request:           &GetCustomerProductLineAccessRequest{},
		Response:          &apiresource.CustomerProductLineAccess{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetCustomerProductLineAccessRequest) (*apiresource.CustomerProductLineAccess, *apierror.APIError) {
			return svc.(CustomerProductLineAccessSvc).GetCustomerProductLineAccess
		},
	}
}
