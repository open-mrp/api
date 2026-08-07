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

// Request to partially update a customer.
type UpdateCustomerRequest struct {
	// Customer ID.
	CustomerID string `path:"id" validate:"required"`
	// The customer's business name, as shown throughout the app and on documents.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Human-readable customer number used to identify the account, distinct from the `id`.
	//
	// Must be unique within your account.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Free-form note about the customer.
	Note field.Clearable[string] `json:"note,omitzero"`
	// The customer's account standing.
	//
	// - `normal`: standard account with no restrictions.
	// - `preferred`: account flagged for prioritized handling.
	// - `hold_shipment`: the customer's shipments should be held, typically over a credit problem, while orders can still be placed.
	// - `hold_all`: all activity for the customer should be held.
	StatusCode field.Optional[constants.AccountStatusCode] `json:"status,omitzero"`
	// Email address.
	Email field.Clearable[string] `json:"email,omitzero" validate:"omitempty,max=255"`
	// Phone number.
	Phone field.Clearable[string] `json:"phone,omitzero" validate:"omitempty,max=255"`
	// Website URL.
	URL field.Clearable[string] `json:"url,omitzero" validate:"omitempty,max=255"`
	// Whether EDI (Electronic Data Interchange) is enabled for exchanging orders and documents with this customer.
	EDIStatus field.Optional[constants.EDIStatus] `json:"edi_status,omitzero"`
	// How sales commission applies to this customer's orders.
	//
	// - `commission_exempt`: this customer's orders are exempt from sales commission.
	// - `commission_applied`: sales commission is calculated on this customer's orders.
	CommissionPolicy field.Optional[constants.CommissionPolicy] `json:"commission_policy,omitzero"`
	// Whether this customer is billed for freight on their orders.
	//
	// - `free_freight`: the customer is not billed for freight.
	// - `billed_freight`: freight is billed to the customer.
	//
	// Freight is also waived when the customer's type group, one of its price groups, or a product line the ordered products belong to is `free_freight`.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero"`
	// Calendar days between an order being issued and it being due to ship.
	//
	// Sets each order's `ship_by_date` when it is issued. Leave unset to inherit the customer's account group lead time, then the account default.
	LeadTimeDays field.Clearable[int32] `json:"lead_time_days,omitzero" validate:"omitempty,gte=0,lte=3650"`
	// ID of the carrier used on this customer's orders when the order does not specify one.
	DefaultCarrierID field.Optional[string] `json:"default_carrier_id,omitzero" validate:"omitempty"`
	// ID of the carrier service level used when an order takes its carrier from this customer's default.
	DefaultServiceLevelID field.Clearable[string] `json:"default_service_level_id,omitzero" validate:"omitempty"`
	// ID of the payment term used on this customer's orders when the order does not specify one.
	DefaultPaymentTermID field.Optional[string] `json:"default_payment_term_id,omitzero" validate:"omitempty"`
	// ID of the shipping term used on this customer's orders when the order does not specify one.
	DefaultShippingTermID field.Optional[string] `json:"default_shipping_term_id,omitzero" validate:"omitempty"`
	// Priority used to pre-fill new orders for this customer.
	DefaultPriorityCode field.Optional[constants.PriorityCode] `json:"default_priority,omitzero"`
	// The ID of the account user to credit as the sales rep on this customer's orders.
	//
	// Must be an account user on your own account.
	DefaultSalesRepID field.Clearable[string] `json:"default_sales_rep_id,omitzero" validate:"omitempty"`
	// ID of an existing address to use as the default billing address.
	//
	// The address is linked to the customer's account if it is not already.
	BillToAddressID field.Clearable[string] `json:"bill_to_address_id,omitzero" validate:"omitempty"`
	// ID of an existing address to use as the default shipping address.
	//
	// The address is linked to the customer's account if it is not already.
	ShipToAddressID field.Clearable[string] `json:"ship_to_address_id,omitzero" validate:"omitempty"`
	// IDs of the account groups of type `pricing_group` to assign to this customer, used to apply pricing rules.
	//
	// When provided, replaces the customer's full set of existing price groups.
	CustomerPriceGroupIDs field.Optional[[]string] `json:"customer_price_group_ids,omitzero"`
	// ID of the account group of type `type_group` that categorizes this customer (for example "Distributors").
	CustomerTypeGroupID field.Optional[string] `json:"customer_type_group_id,omitzero" validate:"omitempty"`
	// Who pays the carrier for shipments.
	//
	// - `sender`: the shipper (you) pays the carrier.
	// - `third_party`: a third party is billed, using `carrier_billing_account`.
	CarrierBillingType field.Optional[constants.CarrierBillingType] `json:"carrier_billing_type,omitzero"`
	// Carrier billing account number charged when `carrier_billing_type` is `third_party`.
	CarrierBillingAccount field.Clearable[string] `json:"carrier_billing_account,omitzero" validate:"omitempty,max=255"`
	// Maximum credit extended to this customer.
	CreditLimit field.Clearable[apirequest.QuantityInput] `json:"credit_limit,omitzero"`
}

var sampleUpdateCustomerName = "Acme Corp Updated"
var sampleUpdateCustomerNote = "Updated account notes"
var sampleUpdateCustomerDefaultCarrierID = apiresource.SampleCarrierID
var sampleUpdateCustomerFreightPolicy = constants.FreightPolicyBilled
var sampleUpdateCustomerEmail = "orders@acme.com"
var sampleUpdateCustomerPhone = "555-123-4567"
var sampleUpdateCustomerURL = "https://acme.com"
var sampleUpdateCustomerCarrierBillingAccount = "123456789"
var sampleUpdateCustomerRequest = &UpdateCustomerRequest{
	Name:                  field.Some(sampleUpdateCustomerName),
	Number:                field.Some(apiresource.SampleCustomerNumber),
	Note:                  field.Set(sampleUpdateCustomerNote),
	StatusCode:            field.Some(constants.AccountStatusCodeNormal),
	Email:                 field.Set(sampleUpdateCustomerEmail),
	Phone:                 field.Set(sampleUpdateCustomerPhone),
	URL:                   field.Set(sampleUpdateCustomerURL),
	EDIStatus:             field.Some(constants.EDIStatusDisabled),
	CommissionPolicy:      field.Some(constants.CommissionPolicyApplied),
	FreightPolicy:         field.Some(sampleUpdateCustomerFreightPolicy),
	DefaultCarrierID:      field.Some(sampleUpdateCustomerDefaultCarrierID),
	DefaultServiceLevelID: field.Set(apiresource.SampleServiceLevelID),
	DefaultPaymentTermID:  field.Some(apiresource.SamplePaymentTermID),
	DefaultShippingTermID: field.Some(apiresource.SampleShippingTermID),
	DefaultPriorityCode:   field.Some(constants.PriorityCodeNormal),
	DefaultSalesRepID:     field.Set(apiresource.SampleAccountUserID),
	BillToAddressID:       field.Set(apiresource.SampleAddressID),
	ShipToAddressID:       field.Set(apiresource.SampleAddressID),
	CustomerPriceGroupIDs: field.Some([]string{apiresource.SampleAccountGroupID}),
	CustomerTypeGroupID:   field.Some(apiresource.SampleAccountGroupID),
	CarrierBillingType:    field.Some(constants.CarrierBillingTypeSender),
	CarrierBillingAccount: field.Set(sampleUpdateCustomerCarrierBillingAccount),
	CreditLimit: field.Set(apirequest.QuantityInput{
		Value:  "10000.00",
		UnitID: apiresource.SampleUnitID,
	}),
}

func (*UpdateCustomerRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateCustomerRequest)
}

// Partially updates a customer account.
//
// Only the fields provided in the request are changed. Nullable fields can be set to `null` to clear their current value.
type UpdateCustomerEndpoint struct{}

func (e *UpdateCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateCustomerRequest, *apiresource.Customer] {
	return (&apiendpoint.APIEndpoint[*UpdateCustomerRequest, *apiresource.Customer]{
		Title:               "Update Customer",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/sales/customers/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeCustomer,
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
