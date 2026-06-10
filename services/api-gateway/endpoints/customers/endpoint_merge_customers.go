package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to merge source customers into a target customer.
type MergeCustomersRequest struct {
	// Target customer ID.
	CustomerID string `path:"id" validate:"required"`
	// Source customer IDs.
	SourceCustomerIDs []string `json:"source_customer_ids" validate:"required"`
}

var sampleMergeCustomersRequest = &MergeCustomersRequest{
	SourceCustomerIDs: []string{apiresource.SampleCustomerID},
}

func (*MergeCustomersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleMergeCustomersRequest)
}

// Merges one or more source customers into a target customer, reassigning all associated records and deleting the source accounts.
type MergeCustomersEndpoint struct{}

func (e *MergeCustomersEndpoint) Materialize() *apiendpoint.APIEndpoint[*MergeCustomersRequest, *apiresource.Customer] {
	return (&apiendpoint.APIEndpoint[*MergeCustomersRequest, *apiresource.Customer]{
		Title:             "Merge Customers",
		Method:            http.MethodPost,
		Route:             "/v1/sales/customers/{id}/actions/merge",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeCustomer,
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
			},
		}),
	})
}
