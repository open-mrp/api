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
)

// Request to partially update a customer.
type UpdateCustomerRequest struct {
	// Customer ID.
	CustomerID string `path:"id" validate:"required"`
	// Customer name.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Customer number.
	Number *string `json:"number,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Note.
	Note *string `json:"note,omitempty" nullable:"true"`
	// Account status code.
	StatusCode *constants.AccountStatusCode `json:"status,omitempty" nullable:"false"`
	// Email address. Send null to clear.
	Email *string `json:"email,omitempty" validate:"omitempty,max=255" nullable:"true"`
	// Phone number. Send null to clear.
	Phone *string `json:"phone,omitempty" validate:"omitempty,max=255" nullable:"true"`
	// Website URL. Send null to clear.
	URL *string `json:"url,omitempty" validate:"omitempty,max=255" nullable:"true"`
	// Whether EDI is enabled.
	IsEdiEnabled *bool `json:"is_edi_enabled,omitempty" nullable:"false"`
	// Commission policy.
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy,omitempty" nullable:"false"`
	// Freight policy.
	FreightPolicy *constants.FreightPolicy `json:"freight_policy,omitempty" nullable:"false"`
	// Default carrier ID.
	DefaultCarrierID *string `json:"default_carrier_id,omitempty" nullable:"false" validate:"omitempty,max=191"`
	// Default service level ID.
	DefaultServiceLevelID *string `json:"default_service_level_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// Default payment term ID.
	DefaultPaymentTermID *string `json:"default_payment_term_id,omitempty" nullable:"false" validate:"omitempty,max=191"`
	// Default shipping term ID.
	DefaultShippingTermID *string `json:"default_shipping_term_id,omitempty" nullable:"false" validate:"omitempty,max=191"`
	// Default priority code.
	DefaultPriorityCode *constants.PriorityCode `json:"default_priority,omitempty" nullable:"false"`
	// The ID of the account user to assign as the default sales rep.
	DefaultSalesRepUserID *string `json:"default_sales_rep_user_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// Bill-to address ID.
	BillToAddressID *string `json:"bill_to_address_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// Ship-to address ID.
	ShipToAddressID *string `json:"ship_to_address_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// Price group IDs. Replaces all existing price groups when provided.
	CustomerPriceGroupIDs *[]string `json:"customer_price_group_ids,omitempty" nullable:"false"`
	// Customer type group ID.
	CustomerTypeGroupID *string `json:"customer_type_group_id,omitempty" nullable:"false" validate:"omitempty,max=191"`
	// Carrier billing type.
	CarrierBillingType *constants.CarrierBillingType `json:"carrier_billing_type,omitempty" nullable:"false"`
	// Carrier billing account number.
	CarrierBillingAccount *string `json:"carrier_billing_account,omitempty" nullable:"true" validate:"omitempty,max=255"`
	// Credit limit. Send null to clear.
	CreditLimit apirequest.NullableInput[apirequest.QuantityInput] `json:"credit_limit,omitempty"`
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
		ContentType:       "application/json",
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
				"credit_limit",
			},
		}),
	}
}
