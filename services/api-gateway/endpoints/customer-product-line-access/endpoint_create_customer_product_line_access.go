package customerproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreateCustomerProductLineAccessRequest is the request to create product line access for a customer.
type CreateCustomerProductLineAccessRequest struct {
	// The ID of the customer.
	CustomerID string `json:"customer_id" validate:"required"`
	// The IDs of the product lines to grant access to.
	ProductLineIDs []string `json:"product_line_ids" validate:"required"`
}

var sampleCreateCustomerProductLineAccessRequest = &CreateCustomerProductLineAccessRequest{
	CustomerID:     apiresource.SampleCustomerID,
	ProductLineIDs: []string{apiresource.SampleProductLineID},
}

func (*CreateCustomerProductLineAccessRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateCustomerProductLineAccessRequest)
}

type CreateCustomerProductLineAccessEndpoint struct{}

func (e *CreateCustomerProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateCustomerProductLineAccessRequest, *apiresource.CustomerProductLineAccess] {
	return &apiendpoint.APIEndpoint[*CreateCustomerProductLineAccessRequest, *apiresource.CustomerProductLineAccess]{
		Title:             "Create Customer Product Line Access",
		Description:       "Creates product line access for a customer.",
		Method:            http.MethodPost,
		Route:             "/v1/sales/product-line-access/customers",
		Request:           &CreateCustomerProductLineAccessRequest{},
		Response:          &apiresource.CustomerProductLineAccess{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateCustomerProductLineAccessRequest) (*apiresource.CustomerProductLineAccess, *apierror.APIError) {
			return svc.(CustomerProductLineAccessSvc).CreateCustomerProductLineAccess
		},
	}
}
