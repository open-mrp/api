package customerproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteCustomerProductLineAccessRequest is the request to delete all product line access for a customer.
type DeleteCustomerProductLineAccessRequest struct {
	// The ID of the customer.
	CustomerID string `path:"customer_id" validate:"required"`
}

type DeleteCustomerProductLineAccessEndpoint struct{}

func (e *DeleteCustomerProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteCustomerProductLineAccessRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteCustomerProductLineAccessRequest, *apiresource.EmptyResource]{
		Title:             "Delete Customer Product Line Access",
		Description:       "Removes all product line access for a customer.",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/product-line-access/customers/{customer_id}",
		Request:           &DeleteCustomerProductLineAccessRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteCustomerProductLineAccessRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(CustomerProductLineAccessSvc).DeleteCustomerProductLineAccess
		},
	}
}
