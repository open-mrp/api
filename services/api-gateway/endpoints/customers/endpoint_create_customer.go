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
	// The customer's business name, as shown throughout the app and on documents.
	Name string `json:"name" validate:"required,max=255"`
	// Human-readable customer number used to identify the account, distinct from the `id`.
	//
	// Must be unique within your account. If omitted, the next sequential number is assigned automatically.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Free-form note about the customer.
	Note field.Optional[string] `json:"note,omitzero"`
	// Email address.
	Email field.Optional[string] `json:"email,omitzero" validate:"omitempty,max=255"`
	// Phone number.
	Phone field.Optional[string] `json:"phone,omitzero" validate:"omitempty,max=255"`
	// Website URL.
	URL field.Optional[string] `json:"url,omitzero" validate:"omitempty,max=255"`
	// Account status code, controlling whether the customer can transact.
	//
	// - `normal`: standard active account with no restrictions.
	// - `preferred`: active account flagged as preferred.
	// - `hold_shipment`: orders can be placed, but shipments are held.
	// - `hold_all`: all activity is on hold.
	StatusCode field.Optional[constants.AccountStatusCode] `json:"status,omitzero" default:"normal"`
	// Whether EDI (Electronic Data Interchange) is enabled for exchanging orders and documents with this customer.
	EDIStatus field.Optional[constants.EDIStatus] `json:"edi_status,omitzero" default:"disabled"`
	// How sales commission applies to this customer's orders.
	//
	// - `commission_exempt`: this customer's orders are exempt from sales commission.
	// - `commission_applied`: sales commission is calculated on this customer's orders.
	CommissionPolicy field.Optional[constants.CommissionPolicy] `json:"commission_policy,omitzero" default:"commission_exempt"`
	// Whether this customer is billed for freight on their orders.
	//
	// - `free_freight`: the customer is not billed for freight.
	// - `billed_freight`: freight is billed to the customer, unless overridden on the order.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero" default:"billed_freight"`
	// ID of the default carrier for this customer's shipments.
	DefaultCarrierID string `json:"default_carrier_id" validate:"required"`
	// ID of the default carrier service level.
	DefaultServiceLevelID field.Optional[string] `json:"default_service_level_id,omitzero" validate:"omitempty"`
	// Default payment term ID.
	DefaultPaymentTermID string `json:"default_payment_term_id" validate:"required"`
	// Default shipping term ID.
	DefaultShippingTermID string `json:"default_shipping_term_id" validate:"required"`
	// Priority applied to new orders for this customer.
	DefaultPriorityCode field.Optional[constants.PriorityCode] `json:"default_priority,omitzero" default:"normal"`
	// The ID of the account user to assign as the default sales rep.
	DefaultSalesRepID field.Optional[string] `json:"default_sales_rep_id,omitzero" validate:"omitempty"`
	// IDs of the account groups of type `pricing_group` to assign to this customer, used to apply pricing rules.
	CustomerPriceGroupIDs []string `json:"customer_price_group_ids,omitzero"`
	// ID of the account group of type `type_group` that categorizes this customer (for example "Distributors").
	CustomerTypeGroupID string `json:"customer_type_group_id" validate:"required"`
	// Who pays the carrier for shipments.
	//
	// - `sender`: the shipper (you) pays the carrier.
	// - `third_party`: a third party is billed, using `carrier_billing_account`.
	CarrierBillingType field.Optional[constants.CarrierBillingType] `json:"carrier_billing_type,omitzero" default:"sender"`
	// Carrier billing account number charged when `carrier_billing_type` is `third_party`.
	CarrierBillingAccount field.Optional[string] `json:"carrier_billing_account,omitzero" validate:"omitempty,max=255"`
	// Maximum credit extended to this customer.
	CreditLimit field.Optional[apirequest.QuantityInput] `json:"credit_limit,omitzero"`
	// Default billing address, created along with the customer.
	BillToAddress apirequest.AddressInput `json:"bill_to_address" validate:"required"`
	// Default shipping address, created along with the customer.
	//
	// If identical to `bill_to_address`, a single shared address record is created.
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

// Creates a customer account with its default addresses, fulfillment settings, and order policies.
//
// If `number` is omitted, the next sequential customer number is assigned automatically.
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
