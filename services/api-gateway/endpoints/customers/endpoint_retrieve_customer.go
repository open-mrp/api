package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a customer by ID.
type RetrieveCustomerRequest struct {
	// Customer ID.
	CustomerID string `path:"id" validate:"required"`
}

type RetrieveCustomerEndpoint struct{}

func (e *RetrieveCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveCustomerRequest, *apiresource.Customer] {
	return &apiendpoint.APIEndpoint[*RetrieveCustomerRequest, *apiresource.Customer]{
		Title:             "Retrieve Customer",
		Description:       "Returns a customer by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/customers/{id}",
		ContentType:       "application/json",
		Request:           &RetrieveCustomerRequest{},
		Response:          &apiresource.Customer{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
			return svc.(CustomerSvc).GetCustomer
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCustomer,
			Fields: []string{
				"bill_to_address",
				"ship_to_address",
				"type",
				"parent_account",
				"freight_preferences.carrier",
				"freight_preferences.service_level",
				"defaults.payment_term",
				"defaults.shipping_term",
				"defaults.sales_rep",
				"defaults.priority",
				"contact_info",
				"freight_preferences",
				"defaults",
				"notification_preferences",
				"price_groups",
				"child_accounts",
				"credit_limit",
			},
		}),
	}
}
