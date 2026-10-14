package customerproductlineaccessep

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

// Request to delete all product line access for a customer.
type DeleteCustomerProductLineAccessRequest struct {
	// Customer ID.
	CustomerID string `path:"customer_id" validate:"required"`
}

// Removes a customer's direct product line access record.
//
// Access the customer inherits through its type group or pricing groups is not affected. Deleting a record that was already deleted returns an already-deleted error rather than succeeding silently.
type DeleteCustomerProductLineAccessEndpoint struct{}

func (e *DeleteCustomerProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteCustomerProductLineAccessRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteCustomerProductLineAccessRequest, *object.Deleted]{
		Title:             "Delete Customer Product Line Access",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/customers/{customer_id}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeCustomerProductLineAccess,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductLineAccess, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteCustomerProductLineAccessRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(CustomerProductLineAccessSvc).DeleteCustomerProductLineAccess, constants.ObjectTypeCustomerProductLineAccess, "customer_id")
		},
	})
}
