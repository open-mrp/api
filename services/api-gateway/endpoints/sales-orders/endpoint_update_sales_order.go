package salesorderep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update a sales order.
type UpdateSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// The customer's own purchase order number, for cross-referencing.
	CustomerPurchaseOrderNumber field.Clearable[string] `json:"customer_purchase_order_number,omitzero" validate:"omitempty,max=255"`
	// Free-form note about the order.
	Note field.Clearable[string] `json:"note,omitzero"`
	// ID of the carrier that will ship the order.
	CarrierID field.Optional[string] `json:"carrier_id,omitzero" validate:"omitempty"`
	// ID of the carrier service level the order ships on.
	ServiceLevelID field.Clearable[string] `json:"service_level_id,omitzero" validate:"omitempty"`
	// Who is billed for freight.
	//
	// - `sender`: the sender pays for shipping.
	// - `third_party`: a third party pays for shipping, using the carrier billing account number.
	CarrierBillingType field.Clearable[constants.CarrierBillingType] `json:"carrier_billing_type,omitzero" validate:"omitempty"`
	// Carrier billing account number charged when `carrier_billing_type` is `third_party`.
	CarrierBillingAccountNumber field.Clearable[string] `json:"carrier_billing_account_number,omitzero" validate:"omitempty,max=255"`
	// New fulfillment priority for the order.
	PriorityCode field.Optional[string] `json:"priority_code,omitzero" validate:"omitempty,max=255"`
	// ID of the account user to credit as the order's sales rep.
	SalesRepID field.Clearable[string] `json:"sales_rep_id,omitzero" validate:"omitempty"`
	// ID of the shipping terms for the order.
	ShippingTermID field.Optional[string] `json:"shipping_term_id,omitzero" validate:"omitempty"`
	// ID of the payment terms for the order.
	PaymentTermID field.Optional[string] `json:"payment_term_id,omitzero" validate:"omitempty"`
	// ID of the order-level discount recorded on the order.
	//
	// Changing this does not add, reprice, or remove the order's discount line; adjust that line directly.
	OrderDiscountID field.Clearable[string] `json:"order_discount_id,omitzero" validate:"omitempty"`
	// Billing address ID.
	//
	// Re-points the order to an existing address. To change an address's contents, use the update-address endpoint.
	BillingAddressID field.Optional[string] `json:"billing_address_id,omitzero" validate:"omitempty"`
	// Shipping address ID.
	//
	// Re-points the order to an existing address. To change an address's contents, use the update-address endpoint.
	ShippingAddressID field.Optional[string] `json:"shipping_address_id,omitzero" validate:"omitempty"`
	// Acknowledgment status of the order.
	//
	// Set to `sent` to mark the acknowledgement as sent without emailing the customer, or `not_sent` to reset it.
	AcknowledgmentStatus field.Optional[constants.AcknowledgmentStatus] `json:"acknowledgment_status,omitzero" validate:"omitempty"`
	// Date delivery is promised to the customer.
	PromisedAt field.Clearable[time.Time] `json:"promised_at,omitzero"`
	// Days between this order being issued and it being due to ship, replacing the customer's standing lead time for this order alone. Mutually exclusive with promised_at and ship_by_override_date; clear one to switch to another.
	LeadTimeOverrideDays field.Clearable[int32] `json:"lead_time_override_days,omitzero" validate:"omitempty,gte=0,lte=3650"`
	// The exact date the order is due to ship, bypassing transit and the customer's receiving days. Mutually exclusive with promised_at and lead_time_override_days.
	ShipByOverrideDate field.Clearable[time.Time] `json:"ship_by_override_date,omitzero"`
	// Moves the order to a different customer account.
	//
	// Existing lines keep the prices they were created with; they are not re-priced against the new customer.
	CustomerID field.Optional[string] `json:"customer_id,omitzero" validate:"omitempty"`
	// Replaces the acknowledgement email contacts on the order.
	//
	// An empty list clears all contacts; omitting the field leaves existing contacts untouched.
	AcknowledgementEmailContacts field.Optional[[]SalesOrderEmailContactInput] `json:"acknowledgement_email_contacts,omitzero"`
	// Replaces the invoice email contacts on the order.
	//
	// An empty list clears all contacts; omitting the field leaves existing contacts untouched.
	InvoiceEmailContacts field.Optional[[]SalesOrderEmailContactInput] `json:"invoice_email_contacts,omitzero"`
}

var sampleUpdateSONote = "Updated shipping instructions"
var sampleUpdateSOCarrierID = apiresource.SampleCarrierID
var sampleUpdateSOPriorityCode = string(constants.PriorityCodeNormal)
var sampleUpdateSOShippingAddressID = apiresource.SampleAddressID
var sampleUpdateSalesOrderRequest = &UpdateSalesOrderRequest{
	Note:              field.Set(sampleUpdateSONote),
	CarrierID:         field.Some(sampleUpdateSOCarrierID),
	PriorityCode:      field.Some(sampleUpdateSOPriorityCode),
	ShippingAddressID: field.Some(sampleUpdateSOShippingAddressID),
}

func (*UpdateSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSalesOrderRequest)
}

// Partially updates a sales order.
//
// Changing the carrier, service level, or ship-to address propagates to the order's existing shipments, but never re-prices the freight line: request a fresh estimate from the quote-freight endpoint and apply it to the shipping line yourself. Order status is changed through the issue, unissue, close, and reopen actions instead of this endpoint.
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
		Preview:           true,
		ObjectType:        constants.ObjectTypeSalesOrder,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
			return svc.(SalesOrderSvc).UpdateSalesOrder
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSalesOrder,
			Fields:     []string{"customer", "sales_rep", "bill_to_address", "ship_to_address", "freight", "payment_term", "shipping_term", "order_discount", "totals", "contacts", "related.pick", "related.production_run", "related.shipments", "related.invoices", "lines", "lines.product", "lines.quantity_ordered", "lines.quantity_ordered.unit", "lines.unit_price", "lines.unit_price.numerator_unit", "lines.unit_price.denominator_unit", "lines.unit_cost", "lines.unit_cost.numerator_unit", "lines.unit_cost.denominator_unit", "lines.totals"},
		}),
	})
}
