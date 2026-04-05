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

// CreateCustomerRequest is the request to create a new customer.
type CreateCustomerRequest struct {
	// The display name of the customer.
	Name string `json:"name" validate:"required,max=255"`
	// The customer number (auto-generated if omitted).
	Number *string `json:"number,omitempty" validate:"omitempty,max=255"`
	// A note about the customer.
	Note *string `json:"note,omitempty"`
	// The customer email address.
	Email *string `json:"email,omitempty" validate:"omitempty,max=255"`
	// The customer phone number.
	Phone *string `json:"phone,omitempty" validate:"omitempty,max=255"`
	// The customer website URL.
	URL *string `json:"url,omitempty" validate:"omitempty,max=255"`
	// The account status code.
	StatusCode *constants.AccountStatusCode `json:"status_code,omitempty"`
	// Whether the customer is EDI enabled.
	IsEdiEnabled *bool `json:"is_edi_enabled,omitempty"`
	// The commission policy for this customer.
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy,omitempty" nullable:"false"`
	// The freight policy for this customer.
	FreightPolicy *constants.FreightPolicy `json:"freight_policy,omitempty" nullable:"false"`
	// The default carrier ID.
	DefaultCarrierID *string `json:"default_carrier_id,omitempty"`
	// The default service level ID.
	DefaultServiceLevelID *string `json:"default_service_level_id,omitempty"`
	// The default payment term ID.
	DefaultPaymentTermID *string `json:"default_payment_term_id,omitempty"`
	// The default shipping term ID.
	DefaultShippingTermID *string `json:"default_shipping_term_id,omitempty"`
	// The default priority code.
	DefaultPriorityCode *constants.PriorityCode `json:"default_priority_code,omitempty"`
	// The default sales rep user ID.
	DefaultSalesRepUserID *string `json:"default_sales_rep_user_id,omitempty"`
	// The bill-to address ID.
	BillToAddressID *string `json:"bill_to_address_id,omitempty"`
	// The ship-to address ID.
	ShipToAddressID *string `json:"ship_to_address_id,omitempty"`
	// The customer price group IDs.
	CustomerPriceGroupIDs []string `json:"customer_price_group_ids,omitempty"`
	// The customer type group ID.
	CustomerTypeGroupID *string `json:"customer_type_group_id,omitempty"`
	// The carrier billing type.
	CarrierBillingType *constants.CarrierBillingType `json:"carrier_billing_type,omitempty"`
	// The carrier billing account number.
	CarrierBillingAccount *string `json:"carrier_billing_account,omitempty" validate:"omitempty,max=255"`
}

var sampleCreateCustomerNote = "Key enterprise account"
var sampleCreateCustomerDefaultCarrierID = apiresource.SampleCarrierID
var sampleCreateCustomerDefaultPaymentTermID = apiresource.SamplePaymentTermID
var sampleCreateCustomerRequest = &CreateCustomerRequest{
	Name:                 apiresource.SampleCustomerName,
	Note:                 &sampleCreateCustomerNote,
	DefaultCarrierID:     &sampleCreateCustomerDefaultCarrierID,
	DefaultPaymentTermID: &sampleCreateCustomerDefaultPaymentTermID,
}

func (*CreateCustomerRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateCustomerRequest)
}

type CreateCustomerEndpoint struct{}

func (e *CreateCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateCustomerRequest, *apiresource.Customer] {
	return &apiendpoint.APIEndpoint[*CreateCustomerRequest, *apiresource.Customer]{
		Title:             "Create Customer",
		Description:       "Creates a new customer account, auto-generating a customer number if one is not provided.",
		Method:            http.MethodPost,
		Route:             "/v1/sales/customers",
		ContentType:       "application/json",
		Request:           &CreateCustomerRequest{},
		Response:          &apiresource.Customer{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
			return svc.(CustomerSvc).CreateCustomer
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
