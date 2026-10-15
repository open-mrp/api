package customerep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete a customer.
type DeleteCustomerRequest struct {
	// Customer ID.
	CustomerID string `path:"id" validate:"required"`
}

// Deletes a customer.
//
// Only while no order, invoice, shipment or child customer references it. Its own price list is deleted with it. Archive it instead, or merge it into another customer.
type DeleteCustomerEndpoint struct{}

func (e *DeleteCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteCustomerRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteCustomerRequest, *object.Deleted]{
		Title:             "Delete Customer",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/sales/customers/{id}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeCustomer,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteCustomerRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(CustomerSvc).DeleteCustomer, constants.ObjectTypeCustomer, "id")
		},
	})
}
