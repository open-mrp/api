package customerproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create product line access for a customer.
type CreateCustomerProductLineAccessRequest struct {
	// Customer ID.
	CustomerID string `json:"customer_id" validate:"required"`
	// Product line IDs to grant access to.
	ProductLineIDs []string `json:"product_line_ids" validate:"required"`
}

var sampleCreateCustomerProductLineAccessRequest = &CreateCustomerProductLineAccessRequest{
	CustomerID:     apiresource.SampleCustomerID,
	ProductLineIDs: []string{apiresource.SampleProductLineID},
}

func (*CreateCustomerProductLineAccessRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateCustomerProductLineAccessRequest)
}

// Creates product line access for a customer.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateCustomerProductLineAccessRequest) (*apiresource.CustomerProductLineAccess, *apierror.APIError) {
			return svc.(CustomerProductLineAccessSvc).CreateCustomerProductLineAccess
		},
		LocationFunc: func(resp *apiresource.CustomerProductLineAccess) string {
			return "/v1/sales/product-line-access/customers/" + resp.Customer.ID
		},
	})
}
