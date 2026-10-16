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

// Request to create a sales order.
type CreateSalesOrderRequest struct {
	// ID of the customer the order is for.
	CustomerID string `json:"customer_id" validate:"required"`
	// The customer's own purchase order number. Unique among the customer's orders.
	CustomerPurchaseOrderNumber field.Optional[string] `json:"customer_purchase_order_number,omitzero" validate:"omitempty,max=255"`
	// The order's currency, as a lowercase ISO 4217 code. Only `usd` is supported; any other code fails with `currency_not_supported`.
	Currency field.Optional[string] `json:"currency,omitzero"`
	// Your team's instructions for the order. Omit to copy the customer's; send `null` for none.
	Instructions field.Clearable[string] `json:"instructions,omitzero" validate:"omitempty,max=5000"`
	// The address to bill: a saved address by `address_id`, or the fields inline. Omit to use the customer's default bill-to address.
	BillTo field.Optional[apirequest.AddressSnapshotInput] `json:"bill_to,omitzero"`
	// The address to ship to: a saved address by `address_id`, or the fields inline. Omit to use the customer's default ship-to address.
	ShipTo field.Optional[apirequest.AddressSnapshotInput] `json:"ship_to,omitzero"`
	// ID of the site the order ships from. Omit to use your default site.
	SiteID field.Optional[string] `json:"site_id,omitzero"`
	// How the order ships. Anything omitted is copied from the customer's defaults.
	Freight field.Optional[CreateSalesOrderFreightInput] `json:"freight,omitzero"`
	// ID of your account user to credit with the order.
	//
	// Omit to assign one: the customer's default sales rep, else the sales territory covering the ship-to address. None is assigned when the customer, or every ordered item's product line, is commission-exempt.
	SalesRepID field.Optional[string] `json:"sales_rep_id,omitzero"`
	// ID of the order's payment term. Omit to use the customer's default.
	PaymentTermID field.Optional[string] `json:"payment_term_id,omitzero"`
	// ID of the order's shipping term. Omit to use the customer's default.
	ShippingTermID field.Optional[string] `json:"shipping_term_id,omitzero"`
	// The order discount to apply, by its ID or its code.
	OrderDiscountID field.Optional[string] `json:"order_discount_id,omitzero"`
	// When the order is due to ship. Omit to use the customer's lead time.
	Commitment field.Optional[CreateSalesOrderCommitmentInput] `json:"commitment,omitzero"`
	// Who receives the order's documents by email. Omit to copy the customer's `notification_recipients`; send `[]` for none.
	//
	// Each recipient must be one of the customer's account users and receives `acknowledgment`, `invoice` or `shipment` documents.
	EmailRecipients field.Optional[[]apirequest.EmailRecipientInput] `json:"email_recipients,omitzero"`
	// The order's lines.
	Lines []CreateSalesOrderLineInput `json:"lines" validate:"required,min=1,dive"`
	// Key-value pairs to store on the order for your own references, such as its ID in another system.
	//
	// Up to 50 keys. Keys are up to 40 characters and cannot contain `[` or `]`; values are strings up to 500 characters, and an empty string is stored as one. A key sent as `null` is not stored.
	Metadata map[string]*string `json:"metadata,omitzero" validate:"omitempty,max=50,dive,keys,min=1,max=40,excludesall=[],endkeys,omitempty,max=500"`
}

// How a new order ships.
type CreateSalesOrderFreightInput struct {
	// ID of the carrier. Omit to use the customer's default carrier; the order is rejected when neither is set.
	CarrierID field.Optional[string] `json:"carrier_id,omitzero"`
	// ID of a service level of the carrier. Omit to use the customer's default when the carrier is the customer's too.
	ServiceLevelID field.Optional[string] `json:"service_level_id,omitzero"`
	// Who the carrier bills. Omit to copy the customer's.
	CarrierBilling field.Optional[CarrierBillingInput] `json:"carrier_billing,omitzero"`
}

// Who a carrier bills for an order's freight.
type CarrierBillingInput struct {
	// `sender`: you pay the carrier. `third_party`: the carrier bills `account_number`.
	Type constants.CarrierBillingType `json:"type" validate:"required"`
	// The carrier account billed when `type` is `third_party`.
	AccountNumber field.Optional[string] `json:"account_number,omitzero" validate:"omitempty,max=255"`
}

// When a new order is due to ship. Give at most one of `promised_on`, `lead_time_override` and `ship_by_override_on`.
type CreateSalesOrderCommitmentInput struct {
	// The date delivery is promised to the customer. The ship-by date is worked back from it through transit and the customer's receiving days.
	PromisedOn field.Optional[timeutil.Date] `json:"promised_on,omitzero"`
	// Business days from issue to ship for this order alone, in the `day` unit.
	LeadTimeOverride field.Optional[apirequest.QuantityInput] `json:"lead_time_override,omitzero"`
	// The exact date the order is due to ship, bypassing transit and the customer's receiving days.
	ShipByOverrideOn field.Optional[timeutil.Date] `json:"ship_by_override_on,omitzero"`
	// Whether the customer can accept the order late: `soft` or `hard`.
	Firmness field.Optional[constants.CommitmentFirmness] `json:"firmness,omitzero"`
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
	UnitPrice field.Optional[apirequest.UnitPriceInput] `json:"unit_price,omitzero"`
	// Key-value pairs to store on the line for your own references, such as its ID in another system.
	//
	// Up to 50 keys. Keys are up to 40 characters and cannot contain `[` or `]`; values are strings up to 500 characters, and an empty string is stored as one. A key sent as `null` is not stored.
	Metadata map[string]*string `json:"metadata,omitzero" validate:"omitempty,max=50,dive,keys,min=1,max=40,excludesall=[],endkeys,omitempty,max=500"`
}

var sampleCreateSalesOrderRequest = &CreateSalesOrderRequest{
	CustomerID:                  apiresource.SampleCustomerID,
	CustomerPurchaseOrderNumber: field.Some("PO-88231"),
	ShipTo:                      field.Some(apirequest.AddressSnapshotInput{AddressID: field.Some(apiresource.SampleAddressID)}),
	Freight: field.Some(CreateSalesOrderFreightInput{
		CarrierID:      field.Some(apiresource.SampleCarrierID),
		ServiceLevelID: field.Some(apiresource.SampleServiceLevelID),
	}),
	Commitment: field.Some(CreateSalesOrderCommitmentInput{PromisedOn: field.Some(timeutil.NewDate(2026, 5, 20))}),
	Lines: []CreateSalesOrderLineInput{
		{
			ProductID: apiresource.SampleProductID,
			Quantity:  apirequest.QuantityInput{Value: "10", UnitID: apiresource.SampleUnitID},
		},
	},
}

func (*CreateSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSalesOrderRequest)
}

// Creates a sales order as a `draft`.
//
// Anything you omit is copied from the customer's `defaults`: addresses, carrier, terms, lead time, sales rep and instructions. Issue it to start fulfillment.
type CreateSalesOrderEndpoint struct{}

func (e *CreateSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSalesOrderRequest, *apiresource.SalesOrder] {
	return (&apiendpoint.APIEndpoint[*CreateSalesOrderRequest, *apiresource.SalesOrder]{
		Title:             "Create Sales Order",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionCreate}}},
		ObjectType:        constants.ObjectTypeSalesOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
			return svc.(SalesOrderSvc).CreateSalesOrder
		},
		LocationFunc: func(resp *apiresource.SalesOrder) string {
			return "/v1/sales/sales-orders/" + resp.ID
		},
		IncludeConfig: salesOrderIncludeConfig(),
	})
}
