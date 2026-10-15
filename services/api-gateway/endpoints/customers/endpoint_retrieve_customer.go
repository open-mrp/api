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
)

// Request to retrieve a customer by ID.
type RetrieveCustomerRequest struct {
	// Customer ID.
	CustomerID string `path:"id" validate:"required"`
}

// Retrieves a customer.
//
// A customer portal user retrieves only their own customer; any other is reported as not found.
type RetrieveCustomerEndpoint struct{}

func (e *RetrieveCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveCustomerRequest, *apiresource.Customer] {
	return (&apiendpoint.APIEndpoint[*RetrieveCustomerRequest, *apiresource.Customer]{
		Title:             "Retrieve Customer",
		Method:            http.MethodGet,
		Route:             "/v1/sales/customers/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ObjectType:        constants.ObjectTypeCustomer,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
			return svc.(CustomerSvc).GetCustomer
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCustomer,
			Fields:     customerIncludeFields,
		}),
	})
}
