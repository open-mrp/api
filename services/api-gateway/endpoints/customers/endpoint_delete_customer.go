package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteCustomerRequest is the request to delete a single customer.
type DeleteCustomerRequest struct {
	// The ID of the customer to delete.
	CustomerID string `path:"id" validate:"required"`
}

type DeleteCustomerEndpoint struct{}

func (e *DeleteCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteCustomerRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteCustomerRequest, *apiresource.EmptyResource]{
		Title:             "Delete Customer",
		Description:       "Deletes a customer and its associated account relations, addresses, and account users.",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/sales/customers/{id}",
		Request:           &DeleteCustomerRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(CustomerSvc).DeleteCustomer
		},
	}
}
