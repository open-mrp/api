package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
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
	// The customer's account standing.
	//
	// - `normal`: standard account with no restrictions.
	// - `preferred`: account flagged for prioritized handling.
	// - `hold_shipment`: the customer's shipments should be held, typically over a credit problem, while orders can still be placed.
	// - `hold_all`: all activity for the customer should be held.
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
	// - `billed_freight`: freight is billed to the customer.
	//
	// Freight is also waived when the customer's type group, one of its price groups, or a product line the ordered products belong to is `free_freight`.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero" default:"billed_freight"`
	// Calendar days between an order being issued and it being due to ship.
	//
	// Sets each order's `ship_by_date` when it is issued. Leave unset to inherit the customer's account group lead time, then the account default.
	LeadTimeDays field.Optional[int32] `json:"lead_time_days,omitzero" validate:"omitempty,gte=0,lte=3650"`
	// The operating calendar naming the days this customer's dock accepts freight.
	//
	// Sits in the same chain as lead_time_days: leaving it unset falls through to the customer's group, then the account default, then Monday to Friday. A promised delivery date is never worked back from a day nobody is there to receive on.
	ReceiveCalendarID field.Optional[string] `json:"receive_calendar_id,omitzero" validate:"omitempty"`
	// ID of the carrier used on this customer's orders when the order does not specify one.
	DefaultCarrierID string `json:"default_carrier_id" validate:"required"`
	// ID of the carrier service level used when an order takes its carrier from this customer's default.
	DefaultServiceLevelID field.Optional[string] `json:"default_service_level_id,omitzero" validate:"omitempty"`
	// ID of the payment term used on this customer's orders when the order does not specify one.
	DefaultPaymentTermID string `json:"default_payment_term_id" validate:"required"`
	// ID of the shipping term used on this customer's orders when the order does not specify one.
	DefaultShippingTermID string `json:"default_shipping_term_id" validate:"required"`
	// Priority used to pre-fill new orders for this customer.
	DefaultPriorityCode field.Optional[constants.PriorityCode] `json:"default_priority,omitzero" default:"normal"`
	// The ID of the account user to credit as the sales rep on this customer's orders.
	//
	// Must be an account user on your own account.
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
var sampleCreateCustomerEmail = "orders@acme.com"
var sampleCreateCustomerPhone = "555-123-4567"
var sampleCreateCustomerURL = "https://acme.com"
var sampleCreateCustomerCarrierBillingAccount = "123456789"
var sampleCreateCustomerRequest = &CreateCustomerRequest{
	Name:                  apiresource.SampleCustomerName,
	Number:                field.Some(apiresource.SampleCustomerNumber),
	Note:                  field.Some(sampleCreateCustomerNote),
	Email:                 field.Some(sampleCreateCustomerEmail),
	Phone:                 field.Some(sampleCreateCustomerPhone),
	URL:                   field.Some(sampleCreateCustomerURL),
	StatusCode:            field.Some(constants.AccountStatusCodeNormal),
	EDIStatus:             field.Some(constants.EDIStatusDisabled),
	CommissionPolicy:      field.Some(constants.CommissionPolicyApplied),
	FreightPolicy:         field.Some(constants.FreightPolicyBilled),
	DefaultCarrierID:      apiresource.SampleCarrierID,
	DefaultServiceLevelID: field.Some(apiresource.SampleServiceLevelID),
	DefaultPaymentTermID:  apiresource.SamplePaymentTermID,
	DefaultShippingTermID: apiresource.SampleShippingTermID,
	DefaultPriorityCode:   field.Some(constants.PriorityCodeNormal),
	DefaultSalesRepID:     field.Some(apiresource.SampleAccountUserID),
	CustomerPriceGroupIDs: []string{apiresource.SampleAccountGroupID},
	CustomerTypeGroupID:   apiresource.SampleAccountGroupID,
	CarrierBillingType:    field.Some(constants.CarrierBillingTypeSender),
	CarrierBillingAccount: field.Some(sampleCreateCustomerCarrierBillingAccount),
	CreditLimit: field.Some(apirequest.QuantityInput{
		Value:  "10000.00",
		UnitID: apiresource.SampleUnitID,
	}),
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
		Title:               "Create Customer",
		Method:              http.MethodPost,
		Route:               "/v1/sales/customers",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionCreate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeCustomer,
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
