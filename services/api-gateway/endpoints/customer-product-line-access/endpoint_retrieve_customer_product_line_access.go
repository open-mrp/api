package customerproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve product line access for a customer.
type RetrieveCustomerProductLineAccessRequest struct {
	// Customer ID.
	CustomerID string `path:"customer_id" validate:"required"`
}

// Returns a customer's direct product line access record.
//
// A customer with no direct grants has no record and returns a not-found error; product lines the customer reaches through its type group or pricing groups are not reported here.
type RetrieveCustomerProductLineAccessEndpoint struct{}

func (e *RetrieveCustomerProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveCustomerProductLineAccessRequest, *apiresource.CustomerProductLineAccess] {
	return (&apiendpoint.APIEndpoint[*RetrieveCustomerProductLineAccessRequest, *apiresource.CustomerProductLineAccess]{
		Title:             "Retrieve Customer Product Line Access",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/customers/{customer_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeCustomerProductLineAccess,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductLineAccess, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveCustomerProductLineAccessRequest) (*apiresource.CustomerProductLineAccess, *apierror.APIError) {
			return svc.(CustomerProductLineAccessSvc).GetCustomerProductLineAccess
		},
	})
}
