package customerproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to update product line access for a customer.
type UpdateCustomerProductLineAccessRequest struct {
	// Customer ID.
	CustomerID string `path:"customer_id" validate:"required"`
	// Product line IDs to grant access to.
	ProductLineIDs *[]string `json:"product_line_ids,omitempty" nullable:"false"`
}

var sampleUpdateCustomerProductLineAccessRequest = &UpdateCustomerProductLineAccessRequest{
	ProductLineIDs: &[]string{apiresource.SampleProductLineID},
}

func (*UpdateCustomerProductLineAccessRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateCustomerProductLineAccessRequest)
}

// Replaces all product line access for a customer.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateCustomerProductLineAccessRequest) (*apiresource.CustomerProductLineAccess, *apierror.APIError) {
			return svc.(CustomerProductLineAccessSvc).UpdateCustomerProductLineAccess
		},
	})
}
