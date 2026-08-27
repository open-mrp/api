package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

// What was invoiced over a window, as one figure and sliced every way the order book is organized.
//
// Every slice is derived from the same invoiced lines as `overall`, so a drilldown always sums back to the headline. The one exception is noted on `by_product_line`.
type AnalyzeSalesBreakdownResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_sales_breakdown_response"`
	// The whole window as one figure.
	Overall *SalesTotals `json:"overall" validate:"required"`
	// The same figures broken into periods, by the date each invoice was raised.
	Periods *List[SalesTotals] `json:"periods" validate:"required"`
	// The same window by item, highest revenue first.
	ByItem *List[SalesBreakdown] `json:"by_item" validate:"required"`
	// The same window by customer, highest revenue first.
	ByCustomer *List[SalesBreakdown] `json:"by_customer" validate:"required"`
	// The same window by customer group, highest revenue first.
	ByCustomerGroup *List[SalesBreakdown] `json:"by_customer_group" validate:"required"`
	// The same window by product line, highest revenue first.
	ByProductLine *List[SalesBreakdown] `json:"by_product_line" validate:"required"`
	// The same window by sales rep, highest revenue first.
	BySalesRep *List[SalesBreakdown] `json:"by_sales_rep" validate:"required"`
	// The same window by the order-level discount applied, highest revenue first.
	//
	// Orders carrying no discount are grouped under a single slice with an empty `key`, so the untouched majority is visible rather than dropped.
	ByDiscount *List[SalesBreakdown] `json:"by_discount" validate:"required"`
}

// Invoiced sales for one window.
type SalesTotals struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_totals"`
	// First day of the period; absent on the overall figure.
	PeriodStart *time.Time `json:"period_start"`
	// Revenue invoiced, priced from each line's order price.
	Revenue *ComputedQuantity `json:"revenue" validate:"required"`
	// Cost of goods for what was invoiced.
	//
	// Null when the caller did not ask for cost, and null for lines that captured no cost — a zero here would read as "free to make" rather than "not recorded".
	Cost *ComputedQuantity `json:"cost"`
	// Quantity invoiced, normalized to each item category's base unit so unlike units can be added.
	QuantityInvoiced *ComputedQuantity `json:"quantity_invoiced" validate:"required"`
	// Number of distinct invoices behind these totals.
	InvoiceCount int32 `json:"invoice_count"`
	// Number of invoiced lines behind these totals.
	LineCount int32 `json:"line_count"`
}

// Invoiced sales for one slice of the order book.
type SalesBreakdown struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_breakdown"`
	// Identifier of the slice — an item, customer, customer group, product line, sales rep, or discount. Empty when the dimension is unset on the lines in it.
	Key string `json:"key"`
	// Display name for the slice.
	Label string `json:"label"`
	// The sales figures for it, on the same shape as the overall window.
	Totals *SalesTotals `json:"totals" validate:"required"`
}

var SampleSalesTotals = &SalesTotals{
	Object:           constants.ObjectTypeSalesTotals,
	Revenue:          SampleComputedQuantity,
	Cost:             SampleComputedQuantity,
	QuantityInvoiced: SampleComputedQuantity,
	InvoiceCount:     42,
	LineCount:        118,
}

var SampleSalesBreakdown = &SalesBreakdown{
	Object: constants.ObjectTypeSalesBreakdown,
	Key:    SampleCustomerID,
	Label:  SampleCustomerName,
	Totals: SampleSalesTotals,
}

var SampleAnalyzeSalesBreakdownResponse = &AnalyzeSalesBreakdownResponse{
	Object:          constants.ObjectTypeAnalyzeSalesBreakdownResponse,
	Overall:         SampleSalesTotals,
	Periods:         NewList([]SalesTotals{*SampleSalesTotals}, PageInfo{}),
	ByItem:          NewList([]SalesBreakdown{*SampleSalesBreakdown}, PageInfo{}),
	ByCustomer:      NewList([]SalesBreakdown{*SampleSalesBreakdown}, PageInfo{}),
	ByCustomerGroup: NewList([]SalesBreakdown{*SampleSalesBreakdown}, PageInfo{}),
	ByProductLine:   NewList([]SalesBreakdown{*SampleSalesBreakdown}, PageInfo{}),
	BySalesRep:      NewList([]SalesBreakdown{*SampleSalesBreakdown}, PageInfo{}),
	ByDiscount:      NewList([]SalesBreakdown{*SampleSalesBreakdown}, PageInfo{}),
}

func (*SalesTotals) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSalesTotals)
}

func (*SalesBreakdown) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSalesBreakdown)
}

func (*AnalyzeSalesBreakdownResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAnalyzeSalesBreakdownResponse)
}
