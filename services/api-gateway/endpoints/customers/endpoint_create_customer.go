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

// Request to create a customer.
type CreateCustomerRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Customer number. Auto-generated if omitted.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Note.
	Note field.Optional[string] `json:"note,omitzero"`
	// Email address.
	Email field.Optional[string] `json:"email,omitzero" validate:"omitempty,max=255"`
	// Phone number.
	Phone field.Optional[string] `json:"phone,omitzero" validate:"omitempty,max=255"`
	// Website URL.
	URL field.Optional[string] `json:"url,omitzero" validate:"omitempty,max=255"`
	// Account status code.
	StatusCode field.Optional[constants.AccountStatusCode] `json:"status,omitzero" default:"normal"`
	// EDI status.
	EDIStatus field.Optional[constants.EDIStatus] `json:"edi_status,omitzero" default:"disabled"`
	// Commission policy.
	CommissionPolicy field.Optional[constants.CommissionPolicy] `json:"commission_policy,omitzero" default:"commission_exempt"`
	// Freight policy.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero" default:"billed_freight"`
	// Default carrier ID.
	DefaultCarrierID string `json:"default_carrier_id" validate:"required"`
	// Default service level ID.
	DefaultServiceLevelID field.Optional[string] `json:"default_service_level_id,omitzero" validate:"omitempty"`
	// Default payment term ID.
	DefaultPaymentTermID string `json:"default_payment_term_id" validate:"required"`
	// Default shipping term ID.
	DefaultShippingTermID string `json:"default_shipping_term_id" validate:"required"`
	// Default priority code.
	DefaultPriorityCode field.Optional[constants.PriorityCode] `json:"default_priority,omitzero" default:"normal"`
	// The ID of the account user to assign as the default sales rep.
	DefaultSalesRepID field.Optional[string] `json:"default_sales_rep_id,omitzero" validate:"omitempty"`
	// Price group IDs.
	CustomerPriceGroupIDs []string `json:"customer_price_group_ids,omitzero"`
	// Customer type group ID.
	CustomerTypeGroupID string `json:"customer_type_group_id" validate:"required"`
	// Carrier billing type.
	CarrierBillingType field.Optional[constants.CarrierBillingType] `json:"carrier_billing_type,omitzero" default:"sender"`
	// Carrier billing account number.
	CarrierBillingAccount field.Optional[string] `json:"carrier_billing_account,omitzero" validate:"omitempty,max=255"`
	// Credit limit.
	CreditLimit field.Optional[apirequest.QuantityInput] `json:"credit_limit,omitzero"`
	// Bill-to address.
	BillToAddress apirequest.AddressInput `json:"bill_to_address" validate:"required"`
	// Ship-to address.
	ShipToAddress apirequest.AddressInput `json:"ship_to_address" validate:"required"`
}

var sampleCreateCustomerNote = "Key enterprise account"
var sampleCreateCustomerStreetLine1 = "123 Main St"
var sampleCreateCustomerLocality = "New York"
var sampleCreateCustomerState = "NY"
var sampleCreateCustomerPostalCode = "10001"
var sampleCreateCustomerRequest = &CreateCustomerRequest{
	Name:                  apiresource.SampleCustomerName,
	Note:                  field.Some(sampleCreateCustomerNote),
	DefaultCarrierID:      apiresource.SampleCarrierID,
	DefaultPaymentTermID:  apiresource.SamplePaymentTermID,
	DefaultShippingTermID: apiresource.SampleShippingTermID,
	CustomerTypeGroupID:   apiresource.SampleAccountGroupID,
	BillToAddress: apirequest.AddressInput{
		Name:        apiresource.SampleCustomerName,
		StreetLine1: field.SomePtr(&sampleCreateCustomerStreetLine1),
		Locality:    field.SomePtr(&sampleCreateCustomerLocality),
		State:       field.SomePtr(&sampleCreateCustomerState),
		PostalCode:  field.SomePtr(&sampleCreateCustomerPostalCode),
		Country:     "US",
	},
	ShipToAddress: apirequest.AddressInput{
		Name:        apiresource.SampleCustomerName,
		StreetLine1: field.SomePtr(&sampleCreateCustomerStreetLine1),
		Locality:    field.SomePtr(&sampleCreateCustomerLocality),
		State:       field.SomePtr(&sampleCreateCustomerState),
		PostalCode:  field.SomePtr(&sampleCreateCustomerPostalCode),
		Country:     "US",
	},
}

func (*CreateCustomerRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateCustomerRequest)
}

// Creates a customer account. Auto-generates a customer number if one is not provided.
type CreateCustomerEndpoint struct{}

func (e *CreateCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateCustomerRequest, *apiresource.Customer] {
	return (&apiendpoint.APIEndpoint[*CreateCustomerRequest, *apiresource.Customer]{
		Title:             "Create Customer",
		Method:            http.MethodPost,
		Route:             "/v1/sales/customers",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeCustomer,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
			return svc.(CustomerSvc).CreateCustomer
		},
		LocationFunc: func(resp *apiresource.Customer) string {
			return "/v1/sales/customers/" + resp.ID
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
