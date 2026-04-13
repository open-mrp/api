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
	// Filter by whether the customer is a parent account.
	IsParentAccount *bool `query:"is_parent_account"`
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

type ListCustomersEndpoint struct{}

func (e *ListCustomersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCustomersRequest, *apiresource.List[apiresource.CustomerSummary]] {
	return &apiendpoint.APIEndpoint[*ListCustomersRequest, *apiresource.List[apiresource.CustomerSummary]]{
		Title:             "List Customers",
		Description:       "Returns a paginated list of customers for the current account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/customers",
		Request:           &ListCustomersRequest{},
		Response:          &apiresource.List[apiresource.CustomerSummary]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCustomersRequest) (*apiresource.List[apiresource.CustomerSummary], *apierror.APIError) {
			return svc.(CustomerSvc).ListCustomers
		},
	}
}
