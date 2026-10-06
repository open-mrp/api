package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

// What was invoiced over a period, as one figure and day by day, alongside a comparison period when one was asked for.
//
// Invoiced sales are priced from each line's order price and cost at the moment of reading, in each item's base unit. `overall` is always the sum of `periods`.
type AnalyzeSalesSummaryResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_sales_summary_response"`
	// The whole period as one figure.
	Overall *SalesTotals `json:"overall" validate:"required"`
	// The same figures by the caller's local day, oldest first. Days with no invoiced sales are omitted.
	Periods *List[SalesTotals] `json:"periods" validate:"required"`
	// The comparison period as one figure; null when no comparison period was requested.
	Comparison *SalesTotals `json:"comparison"`
	// The comparison period by local day; null when no comparison period was requested.
	ComparisonPeriods *List[SalesTotals] `json:"comparison_periods"`
}

// Invoiced sales for one window.
type SalesTotals struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_totals"`
	// First day of the period, as midnight UTC of the caller's local day; null on a whole-window figure.
	PeriodStart *time.Time `json:"period_start"`
	// Revenue invoiced, priced from each line's order price. `value` is exact, unrounded.
	Revenue *ComputedQuantity `json:"revenue" validate:"required"`
	// Cost of goods for what was invoiced, from each line's order cost; a line with no recorded cost counts as zero.
	//
	// Null when the caller may not see cost (sales reps).
	//
	// Null unless the caller holds `costs:read`; customer and supplier portal users never see it.
	Cost *ComputedQuantity `json:"cost" sensitive:"cost"`
	// Quantity invoiced, normalized to each item category's base unit so unlike units can be added.
	QuantityInvoiced *ComputedQuantity `json:"quantity_invoiced" validate:"required"`
	// Number of distinct invoices behind these totals.
	InvoiceCount int64 `json:"invoice_count"`
	// Number of invoiced lines behind these totals.
	LineCount int64 `json:"line_count"`
}

// Invoiced sales for one slice of the order book.
type SalesBreakdown struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_breakdown"`
	// Identifier of the slice: an item, customer, customer group, product line, sales rep (account user), or discount.
	Key string `json:"key"`
	// Display name for the slice. For an item this is its SKU.
	Label string `json:"label"`
	// The item's description; null unless grouped by product.
	Description *string `json:"description"`
	// Abbreviation of the item's base unit, which `quantity_invoiced` is counted in; null unless grouped by product.
	UnitAbbreviation *string `json:"unit_abbreviation"`
	// The slice's figures for the current period.
	Totals *SalesTotals `json:"totals" validate:"required"`
	// The same slice's figures for the comparison period; null when no comparison period was requested.
	ComparisonTotals *SalesTotals `json:"comparison_totals"`
}

// One invoice's invoiced sales.
type SalesInvoice struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_invoice"`
	// Unique identifier of the invoice.
	ID string `json:"id" validate:"required"`
	// The invoice number.
	Number string `json:"number" validate:"required"`
	// The buying customer's account ID.
	CustomerID string `json:"customer_id" validate:"required"`
	// The buying customer's name.
	CustomerName string `json:"customer_name"`
	// When the invoice was raised.
	InvoicedAt time.Time `json:"invoiced_at" validate:"required"`
	// Number of distinct items on the invoice's matching lines.
	ItemCount int64 `json:"item_count"`
	// Revenue invoiced on the matching lines. `value` is exact, unrounded.
	Revenue *ComputedQuantity `json:"revenue" validate:"required"`
}

var sampleSalesDay = time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)

var SampleSalesTotals = &SalesTotals{
	Object: constants.ObjectTypeSalesTotals,
	Revenue: &ComputedQuantity{
		Object:       constants.ObjectTypeComputedQuantity,
		Value:        "16200.5",
		DisplayValue: "16,200.50",
	},
	Cost: &ComputedQuantity{
		Object:       constants.ObjectTypeComputedQuantity,
		Value:        "12800.125",
		DisplayValue: "12,800.13",
	},
	QuantityInvoiced: &ComputedQuantity{
		Object:       constants.ObjectTypeComputedQuantity,
		Value:        "1200",
		DisplayValue: "1,200",
	},
	InvoiceCount: 42,
	LineCount:    118,
}

var SampleSalesDailyTotals = func() *SalesTotals {
	t := *SampleSalesTotals
	t.PeriodStart = &sampleSalesDay
	return &t
}()

var SampleSalesBreakdown = &SalesBreakdown{
	Object:           constants.ObjectTypeSalesBreakdown,
	Key:              SampleCustomerID,
	Label:            SampleCustomerName,
	Totals:           SampleSalesTotals,
	ComparisonTotals: SampleSalesTotals,
}

var SampleAnalyzeSalesSummaryResponse = &AnalyzeSalesSummaryResponse{
	Object:            constants.ObjectTypeAnalyzeSalesSummaryResponse,
	Overall:           SampleSalesTotals,
	Periods:           NewList([]SalesTotals{*SampleSalesDailyTotals}, PageInfo{}),
	Comparison:        SampleSalesTotals,
	ComparisonPeriods: NewList([]SalesTotals{*SampleSalesDailyTotals}, PageInfo{}),
}

var SampleSalesInvoice = &SalesInvoice{
	Object:       constants.ObjectTypeSalesInvoice,
	ID:           SampleInvoiceID,
	Number:       "10042",
	CustomerID:   SampleCustomerID,
	CustomerName: SampleCustomerName,
	InvoicedAt:   sampleSalesDay,
	ItemCount:    3,
	Revenue:      SampleSalesTotals.Revenue,
}

func (*SalesTotals) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSalesTotals)
}

func (*SalesBreakdown) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSalesBreakdown)
}

func (*AnalyzeSalesSummaryResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAnalyzeSalesSummaryResponse)
}

func (*SalesInvoice) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSalesInvoice)
}
