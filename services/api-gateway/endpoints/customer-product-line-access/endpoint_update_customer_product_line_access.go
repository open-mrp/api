package customerproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateCustomerProductLineAccessRequest is the request to update product line access for a customer.
type UpdateCustomerProductLineAccessRequest struct {
	// The ID of the customer.
	CustomerID string `path:"customer_id" validate:"required"`
	// The IDs of the product lines to grant access to.
	ProductLineIDs *[]string `json:"product_line_ids,omitempty" nullable:"false"`
}

var sampleUpdateCustomerProductLineAccessRequest = &UpdateCustomerProductLineAccessRequest{
	ProductLineIDs: &[]string{apiresource.SampleProductLineID},
}

func (*UpdateCustomerProductLineAccessRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateCustomerProductLineAccessRequest)
}

type UpdateCustomerProductLineAccessEndpoint struct{}

func (e *UpdateCustomerProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateCustomerProductLineAccessRequest, *apiresource.CustomerProductLineAccess] {
	return &apiendpoint.APIEndpoint[*UpdateCustomerProductLineAccessRequest, *apiresource.CustomerProductLineAccess]{
		Title:             "Update Customer Product Line Access",
		Description:       "Replaces all product line access for a customer.",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/customers/{customer_id}",
		Request:           &UpdateCustomerProductLineAccessRequest{},
		Response:          &apiresource.CustomerProductLineAccess{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateCustomerProductLineAccessRequest) (*apiresource.CustomerProductLineAccess, *apierror.APIError) {
			return svc.(CustomerProductLineAccessSvc).UpdateCustomerProductLineAccess
		},
	}
}
