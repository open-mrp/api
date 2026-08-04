package customerep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list customers.
type ListCustomersRequest struct {
	apiresource.PaginationRequest
	// Filter by customer type group IDs (the account group of type `type_group` returned in the customer's `type` field).
	CustomerGroupIDs []string `query:"customer_group_ids"`
	// Filter to customers that belong to any of these pricing groups.
	PricingGroupIDs []string `query:"pricing_group_ids"`
	// Filter to customers whose default sales rep is one of these account users.
	SalesRepIDs []string `query:"sales_rep_ids"`
	// Filter by the customer's account standing.
	StatusCodes []constants.AccountStatusCode `query:"status_codes"`
	// Filter by default shipping term IDs.
	ShippingTermIDs []string `query:"shipping_term_ids"`
	// Filter by default payment term IDs.
	PaymentTermIDs []string `query:"payment_term_ids"`
	// Filter by the commission policy set on the customer itself.
	//
	// Policies inherited from the customer's type group or price groups are not considered here.
	CommissionPolicyCodes []constants.CommissionPolicy `query:"commission_status_codes"`
	// Filter by the freight policy set on the customer itself.
	//
	// Policies inherited from the customer's type group or price groups are not considered here.
	FreightPolicyCodes []constants.FreightPolicy `query:"freight_status_codes"`
	// Filter by default carrier IDs.
	CarrierIDs []string `query:"carrier_ids"`
	// Filter by default service level IDs.
	ServiceLevelIDs []string `query:"service_level_ids"`
	// Filter by whether the customer has child accounts.
	ParentAccountStatus *constants.CustomerParentAccountStatus `query:"parent_account_status"`
	// Filter to customers with any address in this city (exact match).
	//
	// When combined with `state` or `postal_code`, a single address must match all provided values.
	City *string `query:"city"`
	// Filter to customers with any address in this state (exact match).
	State *string `query:"state"`
	// Filter to customers with any address in this postal code (exact match).
	PostalCode *string `query:"postal_code"`
	// Filter to customers created at or after this timestamp (inclusive).
	StartDate *time.Time `query:"start_date"`
	// Filter to customers created at or before this timestamp (inclusive).
	EndDate *time.Time `query:"end_date"`
}

// Returns a paginated list of customers for the current account.
type ListCustomersEndpoint struct{}

func (e *ListCustomersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCustomersRequest, *apiresource.List[apiresource.Customer]] {
	return (&apiendpoint.APIEndpoint[*ListCustomersRequest, *apiresource.List[apiresource.Customer]]{
		Title:               "List Customers",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/sales/customers",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeCustomer,
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
				"freight_preferences.carrier.service_levels",
				"freight_preferences.service_level",
				"defaults.payment_term",
				"defaults.shipping_term",
				"defaults.sales_rep",
				"defaults.sales_rep.user",
				"defaults.priority",
				"contact_info",
				"freight_preferences",
				"defaults",
				"notification_preferences",
				"price_groups",
				"child_accounts",
				"credit_limit",
				"credit_limit.unit",
			},
		}),
	})
}
