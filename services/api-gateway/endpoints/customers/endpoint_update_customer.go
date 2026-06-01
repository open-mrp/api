package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/patch"
)

// Request to partially update a customer.
type UpdateCustomerRequest struct {
	// Customer ID.
	CustomerID string `path:"id" validate:"required"`
	// Customer name.
	Name *string `json:"name,omitempty" validate:"omitempty,max=255"`
	// Customer number.
	Number *string `json:"number,omitempty" validate:"omitempty,max=255"`
	// Note.
	Note *patch.Field[string] `json:"note,omitempty"`
	// Account status code.
	StatusCode *constants.AccountStatusCode `json:"status,omitempty"`
	// Email address. Send null to clear.
	Email *patch.Field[string] `json:"email,omitempty" validate:"omitempty,max=255"`
	// Phone number. Send null to clear.
	Phone *patch.Field[string] `json:"phone,omitempty" validate:"omitempty,max=255"`
	// Website URL. Send null to clear.
	URL *patch.Field[string] `json:"url,omitempty" validate:"omitempty,max=255"`
	// EDI status.
	EDIStatus *constants.EDIStatus `json:"edi_status,omitempty"`
	// Commission policy.
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy,omitempty"`
	// Freight policy.
	FreightPolicy *constants.FreightPolicy `json:"freight_policy,omitempty"`
	// Default carrier ID.
	DefaultCarrierID *string `json:"default_carrier_id,omitempty" validate:"omitempty"`
	// Default service level ID.
	DefaultServiceLevelID *patch.Field[string] `json:"default_service_level_id,omitempty" validate:"omitempty"`
	// Default payment term ID.
	DefaultPaymentTermID *string `json:"default_payment_term_id,omitempty" validate:"omitempty"`
	// Default shipping term ID.
	DefaultShippingTermID *string `json:"default_shipping_term_id,omitempty" validate:"omitempty"`
	// Default priority code.
	DefaultPriorityCode *constants.PriorityCode `json:"default_priority,omitempty"`
	// The ID of the account user to assign as the default sales rep.
	DefaultSalesRepID *patch.Field[string] `json:"default_sales_rep_id,omitempty" validate:"omitempty"`
	// Bill-to address ID.
	BillToAddressID *patch.Field[string] `json:"bill_to_address_id,omitempty" validate:"omitempty"`
	// Ship-to address ID.
	ShipToAddressID *patch.Field[string] `json:"ship_to_address_id,omitempty" validate:"omitempty"`
	// Price group IDs. Replaces all existing price groups when provided.
	CustomerPriceGroupIDs *[]string `json:"customer_price_group_ids,omitempty"`
	// Customer type group ID.
	CustomerTypeGroupID *string `json:"customer_type_group_id,omitempty" validate:"omitempty"`
	// Carrier billing type.
	CarrierBillingType *constants.CarrierBillingType `json:"carrier_billing_type,omitempty"`
	// Carrier billing account number.
	CarrierBillingAccount *patch.Field[string] `json:"carrier_billing_account,omitempty" validate:"omitempty,max=255"`
	// Credit limit. Send null to clear.
	CreditLimit *patch.Field[apirequest.QuantityInput] `json:"credit_limit,omitempty"`
}

var sampleUpdateCustomerName = "Acme Corp Updated"
var sampleUpdateCustomerNote = "Updated account notes"
var sampleUpdateCustomerDefaultCarrierID = apiresource.SampleCarrierID
var sampleUpdateCustomerFreightPolicy = constants.FreightPolicyBilled
var sampleUpdateCustomerRequest = &UpdateCustomerRequest{
	Name:             &sampleUpdateCustomerName,
	Note:             new(patch.Set(sampleUpdateCustomerNote)),
	DefaultCarrierID: &sampleUpdateCustomerDefaultCarrierID,
	FreightPolicy:    &sampleUpdateCustomerFreightPolicy,
}

func (*UpdateCustomerRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateCustomerRequest)
}

// Partially updates a customer account. When a Stripe integration is active, customer changes are synced to Stripe.
type UpdateCustomerEndpoint struct{}

func (e *UpdateCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateCustomerRequest, *apiresource.Customer] {
	return (&apiendpoint.APIEndpoint[*UpdateCustomerRequest, *apiresource.Customer]{
		Title:             "Update Customer",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/sales/customers/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeCustomer,
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
				"credit_limit",
			},
		}),
	})
}
