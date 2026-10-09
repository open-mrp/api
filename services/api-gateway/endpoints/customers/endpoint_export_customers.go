package customerep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Filters which customers land in the exported file: the customer list's filters, which select the same customers here.
type ExportCustomersRequest struct {
	// Free-text search matched against each customer's name, number, notes and email.
	Query field.Optional[string] `json:"q,omitzero" validate:"omitempty,max=500"`
	// Filter by customer type group IDs (the account group of type `type_group` returned in the customer's `type` field).
	CustomerGroupIDs []string `json:"customer_group_ids,omitzero"`
	// Filter to customers that belong to any of these pricing groups.
	PricingGroupIDs []string `json:"pricing_group_ids,omitzero"`
	// Filter to customers whose default sales rep is one of these account users.
	SalesRepIDs []string `json:"sales_rep_ids,omitzero"`
	// Filter by the customer's account standing.
	StatusCodes []constants.AccountStatusCode `json:"status_codes,omitzero"`
	// Filter by default shipping term IDs.
	ShippingTermIDs []string `json:"shipping_term_ids,omitzero"`
	// Filter by default payment term IDs.
	PaymentTermIDs []string `json:"payment_term_ids,omitzero"`
	// Filter by the commission policy set on the customer itself.
	//
	// Policies inherited from the customer's type group or price groups are not considered here.
	CommissionPolicyCodes []constants.CommissionPolicy `json:"commission_status_codes,omitzero"`
	// Filter by the freight policy set on the customer itself.
	//
	// Policies inherited from the customer's type group or price groups are not considered here.
	FreightPolicyCodes []constants.FreightPolicy `json:"freight_status_codes,omitzero"`
	// Filter by default carrier IDs.
	CarrierIDs []string `json:"carrier_ids,omitzero"`
	// Filter by default service level IDs.
	ServiceLevelIDs []string `json:"service_level_ids,omitzero"`
	// Filter by whether the customer has child accounts.
	ParentAccountStatus field.Optional[constants.CustomerParentAccountStatus] `json:"parent_account_status,omitzero"`
	// Filter to customers with any address in this city (exact match).
	//
	// When combined with `state` or `postal_code`, a single address must match all provided values.
	City field.Optional[string] `json:"city,omitzero"`
	// Filter to customers with any address in this state (exact match).
	State field.Optional[string] `json:"state,omitzero"`
	// Filter to customers with any address in this postal code (exact match).
	PostalCode field.Optional[string] `json:"postal_code,omitzero"`
	// Filter to customers created at or after this timestamp (inclusive).
	StartDate field.Optional[time.Time] `json:"starts_at,omitzero"`
	// Filter to customers created at or before this timestamp (inclusive).
	EndDate field.Optional[time.Time] `json:"ends_at,omitzero"`
}

var sampleExportCustomersRequest = &ExportCustomersRequest{
	CustomerGroupIDs: []string{apiresource.SampleAccountGroupID},
	StatusCodes:      []constants.AccountStatusCode{constants.AccountStatusCodeNormal},
}

func (*ExportCustomersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportCustomersRequest)
}

// Starts an export of every customer the filters select and returns the job that tracks it.
//
// The file has one row per customer, newest first, with its defaults, default addresses and contacts. An export matching more than 50,000 customers fails with a request to narrow the filters.
type ExportCustomersEndpoint struct{}

func (e *ExportCustomersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportCustomersRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportCustomersRequest, *apiresource.Job]{
		Title:               "Export Customers",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/sales/customers/actions/export",
		SuccessStatusCode:   http.StatusAccepted,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeJob,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportCustomersRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(CustomerSvc).ExportCustomers
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
