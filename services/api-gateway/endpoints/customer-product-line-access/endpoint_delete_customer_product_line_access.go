package customerproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete all product line access for a customer.
type DeleteCustomerProductLineAccessRequest struct {
	// Customer ID.
	CustomerID string `path:"customer_id" validate:"required"`
}

// Removes a customer's direct product line access record.
//
// Access the customer inherits through its type group or pricing groups is not affected. Deleting a record that was already deleted returns an already-deleted error rather than succeeding silently.
type DeleteCustomerProductLineAccessEndpoint struct{}

func (e *DeleteCustomerProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteCustomerProductLineAccessRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteCustomerProductLineAccessRequest, *apiresource.EmptyResource]{
		Title:             "Delete Customer Product Line Access",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/customers/{customer_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductLineAccess, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteCustomerProductLineAccessRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(CustomerProductLineAccessSvc).DeleteCustomerProductLineAccess
		},
	})
}
