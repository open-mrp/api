package salesorderep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to create a sales order.
type CreateSalesOrderRequest struct {
	// ID of the customer account the order is for.
	BuyerAccountID string `json:"buyer_account_id" validate:"required"`
	// The customer's own purchase order number, for cross-referencing.
	//
	// Must be unique among your orders for this customer.
	CustomerPurchaseOrderNumber field.Optional[string] `json:"customer_purchase_order_number,omitzero" validate:"omitempty,max=255"`
	// Free-form note about the order.
	Note field.Optional[string] `json:"note,omitzero"`
	// ID of the carrier that will ship the order.
	//
	// Falls back to the customer's default carrier; the order is rejected when neither is available.
	CarrierID field.Optional[string] `json:"carrier_id,omitzero" validate:"omitempty"`
	// ID of the carrier service level the order ships on.
	//
	// Falls back to the customer's default service level, but only when `carrier_id` is also omitted — supplying a carrier without a service level leaves the service level unset.
	ServiceLevelID field.Optional[string] `json:"service_level_id,omitzero" validate:"omitempty"`
	// Who is billed for freight.
	//
	// - `sender`: the sender pays for shipping.
	// - `third_party`: a third party pays for shipping, using the carrier billing account number.
	CarrierBillingType field.Optional[constants.CarrierBillingType] `json:"carrier_billing_type,omitzero" validate:"omitempty"`
	// Carrier billing account number charged when `carrier_billing_type` is `third_party`.
	CarrierBillingAccountNumber field.Optional[string] `json:"carrier_billing_account_number,omitzero" validate:"omitempty,max=255"`
	// Fulfillment priority used to rank the order on the shop floor.
	PriorityCode constants.PriorityCode `json:"priority_code" validate:"required"`
	// ID of the account user to credit as the order's sales rep.
	//
	// When omitted, a rep is assigned automatically: the customer's default sales rep first, then the sales territory matching the ship-to postal code, then the ship-to state. No rep is assigned when the customer is commission-exempt or every ordered product belongs to a commission-exempt product line.
	SalesRepID field.Optional[string] `json:"sales_rep_id,omitzero" validate:"omitempty"`
	// ID of the shipping terms for the order.
	//
	// Falls back to the customer's default shipping term; the order is rejected when neither is available.
	ShippingTermID field.Optional[string] `json:"shipping_term_id,omitzero" validate:"omitempty"`
	// ID of the payment terms for the order.
	//
	// Falls back to the customer's default payment term; the order is rejected when neither is available.
	PaymentTermID field.Optional[string] `json:"payment_term_id,omitzero" validate:"omitempty"`
	// The order-level discount to apply, given as either its ID or its unique code.
	//
	// The discount is realized as an extra negative-priced line on the order rather than as a separate total.
	OrderDiscountID field.Optional[string] `json:"order_discount_id,omitzero" validate:"omitempty"`
	// Date delivery is promised to the customer.
	//
	// The order's ship-by date is worked back from this: the goods have to reach the customer on a day they receive, so transit and both operating calendars are subtracted from it. Mutually exclusive with lead_time_override_days and ship_by_override_date.
	PromisedAt field.Optional[time.Time] `json:"promised_at,omitzero"`
	// Days between this order being issued and it being due to ship, replacing the customer's standing lead time for this order alone.
	//
	// Already a ship lead time, so no carrier transit is subtracted from it. Mutually exclusive with promised_at and ship_by_override_date.
	LeadTimeOverrideDays field.Optional[int32] `json:"lead_time_override_days,omitzero" validate:"omitempty,gte=0,lte=3650"`
	// The exact date the order is due to ship, bypassing transit and the customer's receiving days.
	//
	// Still moved back to the nearest earlier day the plant ships on, since a date nobody can ship on is not a deadline. Mutually exclusive with promised_at and lead_time_override_days.
	ShipByOverrideDate field.Optional[time.Time] `json:"ship_by_override_date,omitzero"`
	// Bill-to address ID.
	//
	// Must reference an existing address on the order's owner or buyer account.
	BillToAddressID string `json:"bill_to_address_id" validate:"required"`
	// Ship-to address ID.
	//
	// Must reference an existing address on the order's owner or buyer account.
	ShipToAddressID string `json:"ship_to_address_id" validate:"required"`
	// The line items to put on the order.
	//
	// The freight line, and the discount line when `order_discount_id` is supplied, are added on top of these automatically.
	Lines []CreateSalesOrderLineInput `json:"lines" validate:"required,min=1,dive"`
	// Users who should receive order acknowledgement emails for this order.
	//
	// Each must be a user on the customer's account.
	AcknowledgementEmailContacts []SalesOrderEmailContactInput `json:"acknowledgement_email_contacts,omitzero"`
	// Users who should receive invoice emails for this order.
	//
	// Each must be a user on the customer's account.
	InvoiceEmailContacts []SalesOrderEmailContactInput `json:"invoice_email_contacts,omitzero"`
}

// Line item input for a create sales order request.
//
// The item, unit cost, and (unless an internal user supplies a `unit_price` override) the unit price are resolved server-side from the product. The quantity unit must belong to the product's unit group.
type CreateSalesOrderLineInput struct {
	// ID of the product being ordered.
	ProductID string `json:"product_id" validate:"required"`
	// Quantity ordered.
	Quantity apirequest.QuantityInput `json:"quantity" validate:"required"`
	// SKU recorded on the line.
	//
	// Defaults to the product's SKU when omitted.
	ProductSKU field.Optional[string] `json:"product_sku,omitzero" validate:"omitempty,max=255"`
	// Description recorded on the line.
	//
	// Defaults to the product's description when omitted.
	ProductDescription field.Optional[string] `json:"product_description,omitzero"`
	// Unit price override.
	//
	// Honored only for internal users; for customer accounts it is ignored and the price is calculated server-side.
	UnitPrice field.Optional[apirequest.RateInput] `json:"unit_price,omitzero"`
}

// A user subscribed to one of a sales order's email notifications.
type SalesOrderEmailContactInput struct {
	// ID of the account user who should receive the notification.
	AccountUserID string `json:"account_user_id" validate:"required"`
}

var sampleCreateSONote = "Rush order for trade show"
var sampleCreateSOCarrierID = apiresource.SampleCarrierID
var sampleCreateSOServiceLevelID = apiresource.SampleServiceLevelID
var sampleCreateSOCustomerPONumber = "PO-88231"
var sampleCreateSOCarrierBillingAccount = "123456789"
var sampleCreateSalesOrderRequest = &CreateSalesOrderRequest{
	BuyerAccountID:              apiresource.SampleCustomerID,
	CustomerPurchaseOrderNumber: field.Some(sampleCreateSOCustomerPONumber),
	Note:                        field.Some(sampleCreateSONote),
	CarrierID:                   field.Some(sampleCreateSOCarrierID),
	ServiceLevelID:              field.Some(sampleCreateSOServiceLevelID),
	CarrierBillingType:          field.Some(constants.CarrierBillingTypeSender),
	CarrierBillingAccountNumber: field.Some(sampleCreateSOCarrierBillingAccount),
	PriorityCode:                constants.PriorityCodeNormal,
	SalesRepID:                  field.Some(apiresource.SampleAccountUserID),
	ShippingTermID:              field.Some(apiresource.SampleShippingTermID),
	PaymentTermID:               field.Some(apiresource.SamplePaymentTermID),
	OrderDiscountID:             field.Some(apiresource.SampleOrderDiscountID),
	PromisedAt:                  field.Some(time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC)),
	BillToAddressID:             apiresource.SampleAddressID,
	ShipToAddressID:             apiresource.SampleAddressID,
	Lines: []CreateSalesOrderLineInput{
		{
			ProductID: apiresource.SampleProductID,
			Quantity: apirequest.QuantityInput{
				Value:  "10",
				UnitID: apiresource.SampleUnitID,
			},
		},
	},
	AcknowledgementEmailContacts: []SalesOrderEmailContactInput{
		{AccountUserID: apiresource.SampleAccountUserID},
	},
	InvoiceEmailContacts: []SalesOrderEmailContactInput{
		{AccountUserID: apiresource.SampleAccountUserID},
	},
}

func (*CreateSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSalesOrderRequest)
}

// Creates a sales order in `estimate` status.
//
// The order number is assigned automatically, and a sales rep is auto-assigned when none is provided. Line prices and costs are resolved server-side from each product. A shipping line carrying the estimated freight charge is added to the order, plus a negative-priced discount line when an order discount is supplied. The order is not committed for fulfillment until it is issued.
type CreateSalesOrderEndpoint struct{}

func (e *CreateSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSalesOrderRequest, *apiresource.SalesOrder] {
	return (&apiendpoint.APIEndpoint[*CreateSalesOrderRequest, *apiresource.SalesOrder]{
		Title:               "Create Sales Order",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/sales/sales-orders",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		Preview:             true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionCreate}},
		ObjectType:          constants.ObjectTypeSalesOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
			return svc.(SalesOrderSvc).CreateSalesOrder
		},
		LocationFunc: func(resp *apiresource.SalesOrder) string {
			return "/v1/sales/sales-orders/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSalesOrder,
			Fields:     []string{"customer", "sales_rep", "bill_to_address", "ship_to_address", "freight", "payment_term", "shipping_term", "order_discount", "totals", "contacts", "related.pick", "related.production_run", "related.shipments", "related.invoices", "lines", "lines.product", "lines.quantity_ordered", "lines.quantity_ordered.unit", "lines.unit_price", "lines.unit_price.numerator_unit", "lines.unit_price.denominator_unit", "lines.unit_cost", "lines.unit_cost.numerator_unit", "lines.unit_cost.denominator_unit", "lines.totals"},
		}),
	})
}
