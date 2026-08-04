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
)

// Request to create product line access for a customer.
type CreateCustomerProductLineAccessRequest struct {
	// ID of the customer to grant product line access to.
	CustomerID string `json:"customer_id" validate:"required"`
	// IDs of the product lines the customer can access.
	//
	// Must contain at least one ID, and each one must be a product line your account owns; the shared system product lines cannot be granted.
	ProductLineIDs []string `json:"product_line_ids" validate:"required"`
}

var sampleCreateCustomerProductLineAccessRequest = &CreateCustomerProductLineAccessRequest{
	CustomerID:     apiresource.SampleCustomerID,
	ProductLineIDs: []string{apiresource.SampleProductLineID},
}

func (*CreateCustomerProductLineAccessRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateCustomerProductLineAccessRequest)
}

// Grants a customer direct access to a set of product lines.
//
// The customer can then browse and order those product lines, on top of anything it already reaches through its type group or pricing groups.
//
// Each customer can have at most one access record; creating one for a customer that already has one returns a conflict error. Use Update Customer Product Line Access to change an existing record.
type CreateCustomerProductLineAccessEndpoint struct{}

func (e *CreateCustomerProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateCustomerProductLineAccessRequest, *apiresource.CustomerProductLineAccess] {
	return (&apiendpoint.APIEndpoint[*CreateCustomerProductLineAccessRequest, *apiresource.CustomerProductLineAccess]{
		Title:             "Create Customer Product Line Access",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/customers",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeCustomerProductLineAccess,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductLineAccess, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateCustomerProductLineAccessRequest) (*apiresource.CustomerProductLineAccess, *apierror.APIError) {
			return svc.(CustomerProductLineAccessSvc).CreateCustomerProductLineAccess
		},
		LocationFunc: func(resp *apiresource.CustomerProductLineAccess) string {
			return "/v1/sales/product-line-access/customers/" + resp.Customer.ID
		},
	})
}
