package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a customer.
type DeleteCustomerRequest struct {
	// Customer ID.
	CustomerID string `path:"id" validate:"required"`
}

// Deletes a customer.
//
// Fails with a conflict error if any sales orders still reference the customer; delete or reassign those orders, or merge the customer into another first.
type DeleteCustomerEndpoint struct{}

func (e *DeleteCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteCustomerRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteCustomerRequest, *apiresource.EmptyResource]{
		Title:             "Delete Customer",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/sales/customers/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(CustomerSvc).DeleteCustomer
		},
	})
}
