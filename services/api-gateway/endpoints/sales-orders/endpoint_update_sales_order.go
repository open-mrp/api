package salesorderep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/timeutil"
)

// Request to update a sales order.
type UpdateSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// ID of the customer the order is for. Changes only while the order is a `draft`.
	//
	// Existing lines keep their prices; they are not repriced for the new customer.
	CustomerID field.Optional[string] `json:"customer_id,omitzero"`
	// The customer's own purchase order number. Send `null` to clear it.
	CustomerPurchaseOrderNumber field.Clearable[string] `json:"customer_purchase_order_number,omitzero" validate:"omitempty,max=255"`
	// Your team's instructions for the order. Send `null` to clear them.
	Instructions field.Clearable[string] `json:"instructions,omitzero" validate:"omitempty,max=5000"`
	// The address to bill: a saved address by `address_id`, or the fields inline. Replaces the order's copy.
	BillTo field.Optional[apirequest.AddressSnapshotInput] `json:"bill_to,omitzero"`
	// The address to ship to: a saved address by `address_id`, or the fields inline. Replaces the order's copy.
	ShipTo field.Optional[apirequest.AddressSnapshotInput] `json:"ship_to,omitzero"`
	// ID of the site the order ships from.
	SiteID field.Optional[string] `json:"site_id,omitzero"`
	// How the order ships. Only the fields you send change.
	Freight field.Optional[UpdateSalesOrderFreightInput] `json:"freight,omitzero"`
	// ID of your account user credited with the order. Send `null` to clear it.
	SalesRepID field.Clearable[string] `json:"sales_rep_id,omitzero"`
	// ID of the order's payment term.
	PaymentTermID field.Optional[string] `json:"payment_term_id,omitzero"`
	// ID of the order's shipping term.
	ShippingTermID field.Optional[string] `json:"shipping_term_id,omitzero"`
	// ID of the order discount recorded on the order. Send `null` to clear it.
	//
	// Changing it doesn't add, reprice or remove the order's discount line.
	OrderDiscountID field.Clearable[string] `json:"order_discount_id,omitzero"`
	// Whether an order acknowledgment was sent. Set `sent` when you acknowledged the order outside OpenMRP, or `not_sent` to reset it.
	AcknowledgmentStatus field.Optional[constants.AcknowledgmentStatus] `json:"acknowledgment_status,omitzero"`
	// When the order is due to ship. Only the fields you send change; clear one basis to switch to another.
	Commitment field.Optional[UpdateSalesOrderCommitmentInput] `json:"commitment,omitzero"`
	// Who receives the order's documents by email. Replaces the order's recipients; `[]` clears them.
	EmailRecipients field.Optional[[]apirequest.EmailRecipientInput] `json:"email_recipients,omitzero"`
	// Replaces the order's acknowledgment recipients alone.
	AcknowledgementEmailContacts field.Optional[[]SalesOrderEmailContactInput] `json:"acknowledgement_email_contacts,omitzero" visibility:"internal"`
	// Replaces the order's invoice recipients alone.
	InvoiceEmailContacts field.Optional[[]SalesOrderEmailContactInput] `json:"invoice_email_contacts,omitzero" visibility:"internal"`
	// Key-value pairs to store on the order, merged into the ones it already has.
	//
	// Keys left out are kept, so `{}` changes nothing. Setting a key to `null` removes it; an empty string is stored as a value. Sending `metadata: null` removes every key. An object holds at most 50 keys. Keys are up to 40 characters and cannot contain `[` or `]`; values are up to 500 characters.
	Metadata field.Clearable[map[string]*string] `json:"metadata,omitzero" validate:"omitempty,max=50,dive,keys,min=1,max=40,excludesall=[],endkeys,omitempty,max=500"`
}

// How an order ships. Only the fields you send change.
type UpdateSalesOrderFreightInput struct {
	// ID of the carrier.
	CarrierID field.Optional[string] `json:"carrier_id,omitzero"`
	// ID of a service level of the carrier. Send `null` to clear it.
	ServiceLevelID field.Clearable[string] `json:"service_level_id,omitzero"`
	// Who the carrier bills. Send `null` to clear it.
	CarrierBilling field.Clearable[CarrierBillingInput] `json:"carrier_billing,omitzero"`
}

// When an order is due to ship. Only the fields you send change; at most one of `promised_on`, `lead_time_override` and `ship_by_override_on` can be set.
type UpdateSalesOrderCommitmentInput struct {
	// The date delivery is promised to the customer. Send `null` to clear it.
	PromisedOn field.Clearable[timeutil.Date] `json:"promised_on,omitzero"`
	// Business days from issue to ship for this order alone, in the `day` unit. Send `null` to clear it.
	LeadTimeOverride field.Clearable[apirequest.QuantityInput] `json:"lead_time_override,omitzero"`
	// The exact date the order is due to ship. Send `null` to clear it.
	ShipByOverrideOn field.Clearable[timeutil.Date] `json:"ship_by_override_on,omitzero"`
	// Whether the customer can accept the order late: `soft` or `hard`. Send `null` to clear it.
	Firmness field.Clearable[constants.CommitmentFirmness] `json:"firmness,omitzero"`
}

var sampleUpdateSalesOrderRequest = &UpdateSalesOrderRequest{
	Instructions: field.Set("Ship with the replacement parts."),
	Freight:      field.Some(UpdateSalesOrderFreightInput{CarrierID: field.Some(apiresource.SampleCarrierID)}),
	ShipTo:       field.Some(apirequest.AddressSnapshotInput{AddressID: field.Some(apiresource.SampleAddressID)}),
}

func (*UpdateSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSalesOrderRequest)
}

// Updates a sales order. Only the fields you send change.
//
// Changing the carrier, service level or ship-to never reprices the freight line: estimate freight again and update the shipping line yourself. Change the status with the issue, unissue and cancel actions.
type UpdateSalesOrderEndpoint struct{}

func (e *UpdateSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSalesOrderRequest, *apiresource.SalesOrder] {
	return (&apiendpoint.APIEndpoint[*UpdateSalesOrderRequest, *apiresource.SalesOrder]{
		Title:             "Update Sales Order",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeSalesOrder,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
			return svc.(SalesOrderSvc).UpdateSalesOrder
		},
		IncludeConfig: salesOrderIncludeConfig(),
	})
}
