package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

// The money on open sales orders: those issued and not yet completed.
//
// Each counted line is valued at its quantity times its unit price, converted between the two units; nothing is rounded.
type AnalyzeOpenOrdersSummaryResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=open_orders_summary"`
	// Value of the counted lines as ordered. `value` is exact, unrounded.
	Ordered *ComputedQuantity `json:"ordered" validate:"required"`
	// Value of what is ordered on them but not yet invoiced.
	BackOrdered *ComputedQuantity `json:"back_ordered" validate:"required"`
	// Value of what has been invoiced against them so far.
	Invoiced *ComputedQuantity `json:"invoiced" validate:"required"`
}

// One item's quantities across the open sales orders, in the item's base unit.
type OpenOrderProduct struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=open_order_product"`
	// The item ordered.
	Item *AnalyticsItem `json:"item" validate:"required"`
	// The item's base unit, which every quantity here is counted in. Null when the item's category has no base unit.
	Unit *Unit `json:"unit"`
	// Quantity ordered on the open lines.
	QuantityOrdered *ComputedQuantity `json:"quantity_ordered" validate:"required"`
	// Quantity ordered on them but not yet invoiced.
	QuantityBackOrdered *ComputedQuantity `json:"quantity_back_ordered" validate:"required"`
	// Quantity invoiced against them so far.
	QuantityInvoiced *ComputedQuantity `json:"quantity_invoiced" validate:"required"`
}

// A sales order as an analytics row names it.
type AnalyticsSalesOrder struct {
	// Unique identifier of the sales order.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order"`
	// The order number, zero-padded as the dashboard shows it.
	Number string `json:"number" validate:"required"`
}

// A customer as an analytics row names it.
type AnalyticsCustomer struct {
	// Unique identifier of the customer's account.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer"`
	// The customer's name.
	Name string `json:"name"`
	// The number the seller knows the customer by; null when it has none.
	Number *string `json:"number"`
}

// Where an order ships to.
type AnalyticsShipTo struct {
	// State or province; null when the order has no shipping address or it has none.
	State *string `json:"state"`
	// Country; null when the order has no shipping address.
	Country *string `json:"country"`
}

// One open sales order and what its counted lines total.
type OpenOrder struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=open_order"`
	// The sales order.
	Order *AnalyticsSalesOrder `json:"order" validate:"required"`
	// The order's status, which is always `issued` for an open order.
	Status constants.SalesOrderStatusCode `json:"status" validate:"required"`
	// When the order was issued.
	IssuedAt time.Time `json:"issued_at" validate:"required"`
	// The buying customer.
	Customer *AnalyticsCustomer `json:"customer" validate:"required"`
	// Where the order ships to.
	ShipTo *AnalyticsShipTo `json:"ship_to" validate:"required"`
	// Number of the order's sale lines the filters count.
	LineCount int64 `json:"line_count"`
	// Value of those lines as ordered. `value` is exact, unrounded.
	TotalOrdered *ComputedQuantity `json:"total_ordered" validate:"required"`
}

// One sale line of a sales order, as the open-orders report counts it.
type OpenOrderLine struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=open_order_line"`
	// Unique identifier of the sales order line.
	ID string `json:"id" validate:"required"`
	// The item ordered.
	Item *AnalyticsItem `json:"item" validate:"required"`
	// The item's base unit, which the quantities are counted in. Null when the item's category has no base unit.
	Unit *Unit `json:"unit"`
	// The line's price per the base unit of its dimension (per each, per gram), whatever unit it was quoted in. `value` is exact, unrounded.
	UnitPrice *ComputedRate `json:"unit_price" validate:"required"`
	// Quantity ordered but not yet invoiced.
	QuantityBackOrdered *ComputedQuantity `json:"quantity_back_ordered" validate:"required"`
	// Quantity invoiced so far.
	QuantityInvoiced *ComputedQuantity `json:"quantity_invoiced" validate:"required"`
	// Value of the line as ordered. `value` is exact, unrounded.
	TotalOrdered *ComputedQuantity `json:"total_ordered" validate:"required"`
}

var sampleOpenOrderMoney = func(value, display string) *ComputedQuantity {
	return &ComputedQuantity{Object: constants.ObjectTypeComputedQuantity, Value: value, DisplayValue: display}
}

var sampleOpenOrderQuantity = func(value, display string) *ComputedQuantity {
	return &ComputedQuantity{Object: constants.ObjectTypeComputedQuantity, Value: value, DisplayValue: display, Unit: SampleEachUnit}
}

var sampleOpenOrderItem = &AnalyticsItem{
	ID:          SampleItemID,
	Object:      constants.ObjectTypeItem,
	Sku:         SampleItemSKU,
	Description: new("Almond butter, 16 oz"),
}

var SampleAnalyzeOpenOrdersSummaryResponse = &AnalyzeOpenOrdersSummaryResponse{
	Object:      constants.ObjectTypeOpenOrdersSummary,
	Ordered:     sampleOpenOrderMoney("48250.5", "48,250.50"),
	BackOrdered: sampleOpenOrderMoney("30125.25", "30,125.25"),
	Invoiced:    sampleOpenOrderMoney("18125.25", "18,125.25"),
}

var SampleOpenOrderProduct = &OpenOrderProduct{
	Object:              constants.ObjectTypeOpenOrderProduct,
	Item:                sampleOpenOrderItem,
	Unit:                SampleEachUnit,
	QuantityOrdered:     sampleOpenOrderQuantity("1200", "1,200 ea"),
	QuantityBackOrdered: sampleOpenOrderQuantity("840", "840 ea"),
	QuantityInvoiced:    sampleOpenOrderQuantity("360", "360 ea"),
}

var SampleOpenOrder = &OpenOrder{
	Object:       constants.ObjectTypeOpenOrder,
	Order:        &AnalyticsSalesOrder{ID: SampleSalesOrderID, Object: constants.ObjectTypeSalesOrder, Number: "000123"},
	Status:       constants.SalesOrderStatusCodeIssued,
	IssuedAt:     sampleSalesDay,
	Customer:     &AnalyticsCustomer{ID: SampleCustomerID, Object: constants.ObjectTypeCustomer, Name: SampleCustomerName, Number: new("C-1001")},
	ShipTo:       &AnalyticsShipTo{State: new("OR"), Country: new("US")},
	LineCount:    3,
	TotalOrdered: sampleOpenOrderMoney("4825.5", "4,825.50"),
}

var SampleOpenOrderLine = &OpenOrderLine{
	Object: constants.ObjectTypeOpenOrderLine,
	ID:     SampleSalesOrderLineID,
	Item:   sampleOpenOrderItem,
	Unit:   SampleEachUnit,
	UnitPrice: &ComputedRate{
		Object:       constants.ObjectTypeComputedRate,
		Value:        "4.25",
		DisplayValue: FormatRateDisplay("4.25", "$", "ea"),
	},
	QuantityBackOrdered: sampleOpenOrderQuantity("840", "840 ea"),
	QuantityInvoiced:    sampleOpenOrderQuantity("360", "360 ea"),
	TotalOrdered:        sampleOpenOrderMoney("5100", "5,100.00"),
}

func (*AnalyzeOpenOrdersSummaryResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAnalyzeOpenOrdersSummaryResponse)
}

func (*OpenOrderProduct) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOpenOrderProduct)
}

func (*OpenOrder) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOpenOrder)
}

func (*OpenOrderLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOpenOrderLine)
}
