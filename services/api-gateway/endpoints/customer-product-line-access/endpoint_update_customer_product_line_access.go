package customerproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update product line access for a customer.
type UpdateCustomerProductLineAccessRequest struct {
	// Customer ID.
	CustomerID string `path:"customer_id" validate:"required"`
	// The full set of product line IDs the customer has direct access to.
	//
	// Replaces the customer's existing direct grants, and each ID must be a product line your account owns.
	//
	// The list has to name at least one product line: omitting the field or sending an empty list leaves the customer with no record at all, which the request rejects as not found without changing anything. Use Delete Customer Product Line Access to revoke direct access entirely.
	ProductLineIDs field.Optional[[]string] `json:"product_line_ids,omitzero"`
}

var sampleUpdateCustomerProductLineAccessRequest = &UpdateCustomerProductLineAccessRequest{
	ProductLineIDs: field.Some([]string{apiresource.SampleProductLineID}),
}

func (*UpdateCustomerProductLineAccessRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateCustomerProductLineAccessRequest)
}

// Replaces a customer's direct product line access with the provided set.
//
// This is a full replacement, not a merge: product lines omitted from the request lose access. The customer must already have a direct access record; create one with Create Customer Product Line Access first.
type UpdateCustomerProductLineAccessEndpoint struct{}

func (e *UpdateCustomerProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateCustomerProductLineAccessRequest, *apiresource.CustomerProductLineAccess] {
	return (&apiendpoint.APIEndpoint[*UpdateCustomerProductLineAccessRequest, *apiresource.CustomerProductLineAccess]{
		Title:             "Update Customer Product Line Access",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/customers/{customer_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeCustomerProductLineAccess,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductLineAccess, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateCustomerProductLineAccessRequest) (*apiresource.CustomerProductLineAccess, *apierror.APIError) {
			return svc.(CustomerProductLineAccessSvc).UpdateCustomerProductLineAccess
		},
	})
}
