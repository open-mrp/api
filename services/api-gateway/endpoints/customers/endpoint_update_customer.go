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
	"github.com/augno/api/shared/field"
)

// Request to partially update a customer.
type UpdateCustomerRequest struct {
	// Customer ID.
	CustomerID string `path:"id" validate:"required"`
	// Customer name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Customer number.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Note.
	Note field.Clearable[string] `json:"note,omitzero"`
	// Account status code.
	StatusCode field.Optional[constants.AccountStatusCode] `json:"status,omitzero"`
	// Email address. Send null to clear.
	Email field.Clearable[string] `json:"email,omitzero" validate:"omitempty,max=255"`
	// Phone number. Send null to clear.
	Phone field.Clearable[string] `json:"phone,omitzero" validate:"omitempty,max=255"`
	// Website URL. Send null to clear.
	URL field.Clearable[string] `json:"url,omitzero" validate:"omitempty,max=255"`
	// EDI status.
	EDIStatus field.Optional[constants.EDIStatus] `json:"edi_status,omitzero"`
	// Commission policy.
	CommissionPolicy field.Optional[constants.CommissionPolicy] `json:"commission_policy,omitzero"`
	// Freight policy.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero"`
	// Default carrier ID.
	DefaultCarrierID field.Optional[string] `json:"default_carrier_id,omitzero" validate:"omitempty"`
	// Default service level ID.
	DefaultServiceLevelID field.Clearable[string] `json:"default_service_level_id,omitzero" validate:"omitempty"`
	// Default payment term ID.
	DefaultPaymentTermID field.Optional[string] `json:"default_payment_term_id,omitzero" validate:"omitempty"`
	// Default shipping term ID.
	DefaultShippingTermID field.Optional[string] `json:"default_shipping_term_id,omitzero" validate:"omitempty"`
	// Default priority code.
	DefaultPriorityCode field.Optional[constants.PriorityCode] `json:"default_priority,omitzero"`
	// The ID of the account user to assign as the default sales rep.
	DefaultSalesRepID field.Clearable[string] `json:"default_sales_rep_id,omitzero" validate:"omitempty"`
	// Bill-to address ID.
	BillToAddressID field.Clearable[string] `json:"bill_to_address_id,omitzero" validate:"omitempty"`
	// Ship-to address ID.
	ShipToAddressID field.Clearable[string] `json:"ship_to_address_id,omitzero" validate:"omitempty"`
	// Price group IDs. Replaces all existing price groups when provided.
	CustomerPriceGroupIDs field.Optional[[]string] `json:"customer_price_group_ids,omitzero"`
	// Customer type group ID.
	CustomerTypeGroupID field.Optional[string] `json:"customer_type_group_id,omitzero" validate:"omitempty"`
	// Carrier billing type.
	CarrierBillingType field.Optional[constants.CarrierBillingType] `json:"carrier_billing_type,omitzero"`
	// Carrier billing account number.
	CarrierBillingAccount field.Clearable[string] `json:"carrier_billing_account,omitzero" validate:"omitempty,max=255"`
	// Credit limit. Send null to clear.
	CreditLimit field.Clearable[apirequest.QuantityInput] `json:"credit_limit,omitzero"`
}

var sampleUpdateCustomerName = "Acme Corp Updated"
var sampleUpdateCustomerNote = "Updated account notes"
var sampleUpdateCustomerDefaultCarrierID = apiresource.SampleCarrierID
var sampleUpdateCustomerFreightPolicy = constants.FreightPolicyBilled
var sampleUpdateCustomerRequest = &UpdateCustomerRequest{
	Name:             field.Some(sampleUpdateCustomerName),
	Note:             field.Set(sampleUpdateCustomerNote),
	DefaultCarrierID: field.Some(sampleUpdateCustomerDefaultCarrierID),
	FreightPolicy:    field.Some(sampleUpdateCustomerFreightPolicy),
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
