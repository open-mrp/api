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

// UpdateCustomerRequest is the request to partially update a customer.
type UpdateCustomerRequest struct {
	// The ID of the customer to update.
	CustomerID string `path:"id" validate:"required"`
	// The customer name.
	Name *string `json:"name,omitempty" validate:"omitempty,max=255"`
	// The customer number.
	Number *string `json:"number,omitempty" validate:"omitempty,max=255"`
	// A note about the customer.
	Note *string `json:"note,omitempty" nullable:"true"`
	// The status code.
	StatusCode *constants.AccountStatusCode `json:"status_code,omitempty"`
	// The customer email address. Send null to clear.
	Email *string `json:"email,omitempty" validate:"omitempty,max=255" nullable:"true"`
	// The customer phone number. Send null to clear.
	Phone *string `json:"phone,omitempty" validate:"omitempty,max=255" nullable:"true"`
	// The customer website URL. Send null to clear.
	URL *string `json:"url,omitempty" validate:"omitempty,max=255" nullable:"true"`
	// Whether the customer is EDI enabled.
	IsEdiEnabled *bool `json:"is_edi_enabled,omitempty"`
	// The commission policy for this customer.
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy,omitempty" nullable:"false"`
	// The freight policy for this customer.
	FreightPolicy *constants.FreightPolicy `json:"freight_policy,omitempty" nullable:"false"`
	// The default carrier ID.
	DefaultCarrierID *string `json:"default_carrier_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The default service level ID.
	DefaultServiceLevelID *string `json:"default_service_level_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The default payment term ID.
	DefaultPaymentTermID *string `json:"default_payment_term_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The default shipping term ID.
	DefaultShippingTermID *string `json:"default_shipping_term_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The default priority code.
	DefaultPriorityCode *constants.PriorityCode `json:"default_priority_code,omitempty"`
	// The default sales rep user ID.
	DefaultSalesRepUserID *string `json:"default_sales_rep_user_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The bill-to address ID.
	BillToAddressID *string `json:"bill_to_address_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The ship-to address ID.
	ShipToAddressID *string `json:"ship_to_address_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The customer price group IDs. When provided, replaces all existing price groups.
	CustomerPriceGroupIDs *[]string `json:"customer_price_group_ids,omitempty"`
	// The customer type group ID.
	CustomerTypeGroupID *string `json:"customer_type_group_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The carrier billing type.
	CarrierBillingType *constants.CarrierBillingType `json:"carrier_billing_type,omitempty" nullable:"false"`
	// The carrier billing account number.
	CarrierBillingAccount *string `json:"carrier_billing_account,omitempty" nullable:"true" validate:"omitempty,max=255"`
}

var sampleUpdateCustomerName = "Acme Corp Updated"
var sampleUpdateCustomerNote = "Updated account notes"
var sampleUpdateCustomerDefaultCarrierID = apiresource.SampleCarrierID
var sampleUpdateCustomerFreightPolicy = constants.FreightPolicyBilled
var sampleUpdateCustomerRequest = &UpdateCustomerRequest{
	Name:             &sampleUpdateCustomerName,
	Note:             &sampleUpdateCustomerNote,
	DefaultCarrierID: &sampleUpdateCustomerDefaultCarrierID,
	FreightPolicy:    &sampleUpdateCustomerFreightPolicy,
}

func (*UpdateCustomerRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateCustomerRequest)
}

type UpdateCustomerEndpoint struct{}

func (e *UpdateCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateCustomerRequest, *apiresource.Customer] {
	return &apiendpoint.APIEndpoint[*UpdateCustomerRequest, *apiresource.Customer]{
		Title:             "Update Customer",
		Description:       "Partially updates a customer account. When a Stripe integration is active, customer changes are synced to Stripe.",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/customers/{id}",
		Request:           &UpdateCustomerRequest{},
		Response:          &apiresource.Customer{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
			return svc.(CustomerSvc).UpdateCustomer
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
