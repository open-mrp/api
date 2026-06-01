package customerep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list customers.
type ListCustomersRequest struct {
	apiresource.PaginationRequest
	// Filter by customer group IDs.
	CustomerGroupIDs []string `query:"customer_group_ids"`
	// Filter by pricing group IDs.
	PricingGroupIDs []string `query:"pricing_group_ids"`
	// Filter by sales rep IDs.
	SalesRepIDs []string `query:"sales_rep_ids"`
	// Filter by status codes.
	StatusCodes []constants.AccountStatusCode `query:"status_codes"`
	// Filter by shipping term IDs.
	ShippingTermIDs []string `query:"shipping_term_ids"`
	// Filter by payment term IDs.
	PaymentTermIDs []string `query:"payment_term_ids"`
	// Filter by commission status codes.
	CommissionPolicyCodes []constants.CommissionPolicy `query:"commission_status_codes"`
	// Filter by freight status codes.
	FreightPolicyCodes []constants.FreightPolicy `query:"freight_status_codes"`
	// Filter by carrier IDs.
	CarrierIDs []string `query:"carrier_ids"`
	// Filter by service level IDs.
	ServiceLevelIDs []string `query:"service_level_ids"`
	// Filter by whether the customer has child accounts.
	ParentAccountStatus *constants.CustomerParentAccountStatus `query:"parent_account_status"`
	// Filter by city.
	City *string `query:"city"`
	// Filter by state.
	State *string `query:"state"`
	// Filter by postal code.
	PostalCode *string `query:"postal_code"`
	// Filter by start date (created after).
	StartDate *time.Time `query:"start_date"`
	// Filter by end date (created before).
	EndDate *time.Time `query:"end_date"`
}

// Returns a paginated list of customers for the current account.
type ListCustomersEndpoint struct{}

func (e *ListCustomersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCustomersRequest, *apiresource.List[apiresource.Customer]] {
	return (&apiendpoint.APIEndpoint[*ListCustomersRequest, *apiresource.List[apiresource.Customer]]{
		Title:             "List Customers",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/customers",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeCustomer,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCustomersRequest) (*apiresource.List[apiresource.Customer], *apierror.APIError) {
			return svc.(CustomerSvc).ListCustomers
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
	})
}
