package customerep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list customers.
type ListCustomersRequest struct {
	object.ListRequest
	apirequest.CreatedRange
	// Case-insensitive substring match on `name` or `number`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return customers with any of these statuses: `active` or `archived`.
	Statuses []constants.ArchiveStatus `query:"statuses"`
	// Only return customers with any of these standings: `normal`, `hold_shipment` or `hold_all`.
	Standings []constants.CustomerStanding `query:"standings"`
	// Only return customers in any of these account groups.
	GroupIDs []string `query:"group_ids"`
	// Only return customers in any of these pricing groups.
	PricingGroupIDs []string `query:"pricing_group_ids" visibility:"internal"`
	// Only return customers assigned any of these price lists directly.
	PriceListIDs []string `query:"price_list_ids" validate:"omitempty,max=100"`
	// Only return customers whose default sales rep is one of these account users.
	SalesRepIDs []string `query:"sales_rep_ids"`
	// Only return customers with any of these default shipping terms.
	ShippingTermIDs []string `query:"shipping_term_ids"`
	// Only return customers with any of these default payment terms.
	PaymentTermIDs []string `query:"payment_term_ids"`
	// Only return customers with any of these commission policies set on the customer itself.
	CommissionPolicies []constants.CommissionPolicy `query:"commission_policies"`
	// Only return customers with any of these freight policies set on the customer itself.
	FreightPolicies []constants.FreightPolicy `query:"freight_policies"`
	// Only return customers with any of these default carriers.
	CarrierIDs []string `query:"carrier_ids"`
	// Only return customers with any of these default service levels.
	ServiceLevelIDs []string `query:"service_level_ids"`
	// Only return customers with any of these relationship types: `standalone`, `parent` or `child`.
	RelationshipTypes []constants.CustomerRelationshipType `query:"relationship_types"`
	// Only return customers with at least one address in this city.
	City *string `query:"city"`
	// Only return customers with at least one address in this state.
	State *string `query:"state"`
	// Only return customers with at least one address in this postal code.
	PostalCode *string `query:"postal_code"`
}

// Returns a list of customers.
//
// Sorted by creation date, newest first.
type ListCustomersEndpoint struct{}

func (e *ListCustomersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCustomersRequest, *apiresource.List[apiresource.Customer]] {
	return (&apiendpoint.APIEndpoint[*ListCustomersRequest, *apiresource.List[apiresource.Customer]]{
		Title:             "List Customers",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/customers",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeCustomer,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCustomersRequest) (*apiresource.List[apiresource.Customer], *apierror.APIError) {
			return svc.(CustomerSvc).ListCustomers
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCustomer,
			Fields:     customerIncludeFields,
		}),
	})
}
