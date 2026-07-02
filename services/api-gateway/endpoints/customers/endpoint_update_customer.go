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
	// Account status code, controlling whether the customer can transact.
	//
	// - `normal`: standard active account with no restrictions.
	// - `preferred`: active account flagged as preferred.
	// - `hold_shipment`: orders can be placed, but shipments are held.
	// - `hold_all`: all activity is on hold.
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
	// - `billed_freight`: freight is billed to the customer, unless overridden on the order.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero"`
	// ID of the default carrier for this customer's shipments.
	DefaultCarrierID field.Optional[string] `json:"default_carrier_id,omitzero" validate:"omitempty"`
	// ID of the default carrier service level.
	DefaultServiceLevelID field.Clearable[string] `json:"default_service_level_id,omitzero" validate:"omitempty"`
	// Default payment term ID.
	DefaultPaymentTermID field.Optional[string] `json:"default_payment_term_id,omitzero" validate:"omitempty"`
	// Default shipping term ID.
	DefaultShippingTermID field.Optional[string] `json:"default_shipping_term_id,omitzero" validate:"omitempty"`
	// Priority applied to new orders for this customer.
	DefaultPriorityCode field.Optional[constants.PriorityCode] `json:"default_priority,omitzero"`
	// The ID of the account user to assign as the default sales rep.
	DefaultSalesRepID field.Clearable[string] `json:"default_sales_rep_id,omitzero" validate:"omitempty"`
	// ID of an existing address to use as the default billing address.
	BillToAddressID field.Clearable[string] `json:"bill_to_address_id,omitzero" validate:"omitempty"`
	// ID of an existing address to use as the default shipping address.
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
var sampleUpdateCustomerRequest = &UpdateCustomerRequest{
	Name:             field.Some(sampleUpdateCustomerName),
	Note:             field.Set(sampleUpdateCustomerNote),
	DefaultCarrierID: field.Some(sampleUpdateCustomerDefaultCarrierID),
	FreightPolicy:    field.Some(sampleUpdateCustomerFreightPolicy),
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
