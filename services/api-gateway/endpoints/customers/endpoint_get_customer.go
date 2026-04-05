package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetCustomerRequest is the request to retrieve a single customer by ID.
type GetCustomerRequest struct {
	// The ID of the customer to retrieve.
	CustomerID string `path:"id" validate:"required"`
}

type GetCustomerEndpoint struct{}

func (e *GetCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetCustomerRequest, *apiresource.Customer] {
	return &apiendpoint.APIEndpoint[*GetCustomerRequest, *apiresource.Customer]{
		Title:             "Get Customer",
		Description:       "Returns a single customer by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/customers/{id}",
		ContentType:       "application/json",
		Request:           &GetCustomerRequest{},
		Response:          &apiresource.Customer{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
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
			},
		}),
	}
}
