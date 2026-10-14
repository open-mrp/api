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

// Returns a customer by ID.
//
// A customer or supplier portal retrieves only its own account; any other customer is reported as not found.
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
		Preview:           true,
		ObjectType:        constants.ObjectTypeCustomer,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
			return svc.(CustomerSvc).GetCustomer
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCustomer,
			Fields: []string{
				"bill_to_address", "ship_to_address", "type", "parent_account", "freight_preferences.carrier", "freight_preferences.carrier.service_levels", "freight_preferences.service_level", "defaults.payment_term", "defaults.shipping_term", "defaults.sales_rep", "defaults.sales_rep.user", "defaults.priority", "contact_info", "freight_preferences", "defaults", "notification_preferences", "price_groups", "child_accounts", "credit_limit",
			},
		}),
	})
}
