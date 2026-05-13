package customerproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve product line access for a customer.
type RetrieveCustomerProductLineAccessRequest struct {
	// Customer ID.
	CustomerID string `path:"customer_id" validate:"required"`
}

type RetrieveCustomerProductLineAccessEndpoint struct{}

func (e *RetrieveCustomerProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveCustomerProductLineAccessRequest, *apiresource.CustomerProductLineAccess] {
	return &apiendpoint.APIEndpoint[*RetrieveCustomerProductLineAccessRequest, *apiresource.CustomerProductLineAccess]{
		Title:             "Retrieve Customer Product Line Access",
		Description:       "Returns the product line access for a customer.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/customers/{customer_id}",
		Request:           &RetrieveCustomerProductLineAccessRequest{},
		Response:          &apiresource.CustomerProductLineAccess{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveCustomerProductLineAccessRequest) (*apiresource.CustomerProductLineAccess, *apierror.APIError) {
			return svc.(CustomerProductLineAccessSvc).GetCustomerProductLineAccess
		},
	}
}
