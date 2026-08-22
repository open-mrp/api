package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to merge source customers into a target customer.
type MergeCustomersRequest struct {
	// ID of the target customer that receives the merged records.
	CustomerID string `path:"id" validate:"required"`
	// IDs of the source customers to merge into the target.
	//
	// Sources are deleted after the merge. The list must not contain duplicates or the target customer's ID.
	SourceCustomerIDs []string `json:"source_customer_ids" validate:"required"`
}

var sampleMergeCustomersRequest = &MergeCustomersRequest{
	SourceCustomerIDs: []string{apiresource.SampleCustomerID},
}

func (*MergeCustomersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleMergeCustomersRequest)
}

// Merges one or more source customers into a target customer.
//
// Sales orders, invoices, shipments, deliveries, and other transaction records from the source customers are reassigned to the target; price groups, product line access, addresses, and users are consolidated without duplicates; child accounts of the sources are re-parented to the target; the source customers are then deleted.
//
// The target keeps its own name, number, default addresses, and default settings — none of those are copied over from the sources, and the sources' notification recipients are discarded rather than transferred.
type MergeCustomersEndpoint struct{}

func (e *MergeCustomersEndpoint) Materialize() *apiendpoint.APIEndpoint[*MergeCustomersRequest, *apiresource.Customer] {
	return (&apiendpoint.APIEndpoint[*MergeCustomersRequest, *apiresource.Customer]{
		Title:               "Merge Customers",
		Method:              http.MethodPost,
		Route:               "/v1/sales/customers/{id}/actions/merge",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainCustomers, Action: types.ActionDelete}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeCustomer,
		ServiceHandler: func(svc any) func(ctx context.Context, req *MergeCustomersRequest) (*apiresource.Customer, *apierror.APIError) {
			return svc.(CustomerSvc).MergeCustomers
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
