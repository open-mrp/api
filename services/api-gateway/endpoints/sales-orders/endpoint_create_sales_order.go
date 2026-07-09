package salesorderep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create a sales order.
type CreateSalesOrderRequest struct {
	// ID of the customer account the order is for.
	BuyerAccountID string `json:"buyer_account_id" validate:"required"`
	// The customer's own purchase order number, for cross-referencing.
	//
	// Must be unique among your orders for this customer.
	CustomerPurchaseOrderNumber field.Optional[string] `json:"customer_purchase_order_number,omitzero" validate:"omitempty,max=255"`
	// Order note.
	Note field.Optional[string] `json:"note,omitzero"`
	// Carrier ID.
	CarrierID field.Optional[string] `json:"carrier_id,omitzero" validate:"omitempty"`
	// Service level ID.
	ServiceLevelID field.Optional[string] `json:"service_level_id,omitzero" validate:"omitempty"`
	// Who is billed for freight.
	//
	// - `sender`: the sender pays for shipping.
	// - `third_party`: a third party pays for shipping, using the carrier billing account number.
	CarrierBillingType field.Optional[constants.CarrierBillingType] `json:"carrier_billing_type,omitzero" validate:"omitempty"`
	// Carrier billing account number.
	CarrierBillingAccountNumber field.Optional[string] `json:"carrier_billing_account_number,omitzero" validate:"omitempty,max=255"`
	// Fulfillment priority used to rank the order on the shop floor.
	PriorityCode string `json:"priority_code" validate:"required,max=255"`
	// Sales rep ID.
	//
	// When omitted, a rep is assigned automatically: the customer's default sales rep first, then the sales territory matching the ship-to postal code, then the ship-to state.
	SalesRepID field.Optional[string] `json:"sales_rep_id,omitzero" validate:"omitempty"`
	// Shipping term ID.
	ShippingTermID field.Optional[string] `json:"shipping_term_id,omitzero" validate:"omitempty"`
	// Payment term ID.
	PaymentTermID field.Optional[string] `json:"payment_term_id,omitzero" validate:"omitempty"`
	// Order discount ID.
	//
	// When supplied, a discount line is added to the order automatically.
	OrderDiscountID field.Optional[string] `json:"order_discount_id,omitzero" validate:"omitempty"`
	// Promised delivery date.
	PromisedAt field.Optional[time.Time] `json:"promised_at,omitzero"`
	// Bill-to address ID.
	//
	// Must reference an existing address on the order's owner or buyer account.
	BillToAddressID string `json:"bill_to_address_id" validate:"required"`
	// Ship-to address ID.
	//
	// Must reference an existing address on the order's owner or buyer account.
	ShipToAddressID string `json:"ship_to_address_id" validate:"required"`
	// Order lines to create.
	Lines []CreateSalesOrderLineInput `json:"lines" validate:"required,min=1,dive"`
	// Account users who should receive order acknowledgement emails.
	AcknowledgementEmailContacts []SalesOrderEmailContactInput `json:"acknowledgement_email_contacts,omitzero"`
	// Account users who should receive invoice emails.
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

// SalesOrderEmailContactInput represents an account user subscribed to a sales-order email notification type.
type SalesOrderEmailContactInput struct {
	// Account user ID to receive the notification.
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
	PriorityCode:                string(constants.PriorityCodeNormal),
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
// The order number is assigned automatically, and a sales rep is auto-assigned when none is provided. A shipping line is always added to the order, plus a discount line when an order discount is supplied.
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
