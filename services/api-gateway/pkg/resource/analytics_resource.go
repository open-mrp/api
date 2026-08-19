// ! Note: this will be refactored in the future - okay to leave as is.
package apiresource

import (
	"time"

	"github.com/augno/api/shared/constants"
)

// AnalyzeSalesResponse represents the response from the analyze sales endpoint.
type AnalyzeSalesResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// The sales entry data.
	Data []SalesEntry `json:"data" validate:"required"`
}

// SalesEntry represents a single sales transaction entry for analytics.
type SalesEntry struct {
	// Unique identifier for this entry.
	ID string `json:"id" validate:"required"`
	// The date the order was issued.
	IssuedAt *time.Time `json:"issued_at"`
	// The customer purchase order number.
	CustomerPO *string `json:"customer_po"`
	// The order number.
	OrderNumber string `json:"order_number" validate:"required"`
	// The order ID.
	OrderID string `json:"order_id" validate:"required"`
	// The sales representative ID.
	SalesRepID *string `json:"sales_rep_id"`
	// The sales representative username.
	SalesRepUsername *string `json:"sales_rep_username"`
	// The customer ID.
	CustomerID string `json:"customer_id" validate:"required"`
	// The customer name.
	CustomerName string `json:"customer_name" validate:"required"`
	// The customer number.
	CustomerNumber string `json:"customer_number" validate:"required"`
	// The customer type group ID.
	CustomerTypeGroupID *string `json:"customer_type_group_id"`
	// The customer group name.
	CustomerGroupName *string `json:"customer_group_name"`
	// The parent customer ID.
	ParentCustomerID *string `json:"parent_customer_id"`
	// The date the customer was created.
	CustomerCreatedAt time.Time `json:"customer_created_at" validate:"required"`
	// The product line ID.
	ProductLineID *string `json:"product_line_id"`
	// The product line name.
	ProductLine *string `json:"product_line"`
	// The product ID.
	ProductTypeID string `json:"product_type_id" validate:"required"`
	// The item ID.
	ItemID string `json:"item_id" validate:"required"`
	// The product SKU.
	ProductSku string `json:"product_sku" validate:"required"`
	// The product description.
	ProductDescription *string `json:"product_description"`
	// The category name.
	CategoryName string `json:"category_name" validate:"required"`
	// The quantity invoiced.
	QuantityInvoiced float64 `json:"quantity_invoiced" validate:"required"`
	// The unit of measure.
	Unit string `json:"unit" validate:"required"`
	// The unit cost.
	UnitCost float64 `json:"unit_cost" validate:"required"`
	// The unit price.
	UnitPrice float64 `json:"unit_price" validate:"required"`
	// The unit profit.
	UnitProfit float64 `json:"unit_profit" validate:"required"`
	// The total invoiced amount.
	TotalInvoiced float64 `json:"total_invoiced" validate:"required"`
	// The total cost.
	TotalCost float64 `json:"total_cost" validate:"required"`
	// The total profit.
	TotalProfit float64 `json:"total_profit" validate:"required"`
	// The ship-to city.
	ShipToCity *string `json:"ship_to_city"`
	// The ship-to zipcode.
	ShipToZipcode *string `json:"ship_to_zipcode"`
	// The ship-to state.
	ShipToState *string `json:"ship_to_state"`
	// The ship-to country.
	ShipToCountry *string `json:"ship_to_country"`
	// The order discount code.
	OrderDiscountCode *string `json:"discount_code"`
	// The date the order was completed.
	CompletedAt *time.Time `json:"completed_at"`
	// The date of the first shipment.
	FirstShipAt *time.Time `json:"first_ship_at"`
	// The promised delivery date.
	PromisedAt *time.Time `json:"promised_at"`
	// The invoice ID.
	InvoiceID string `json:"invoice_id" validate:"required"`
	// The invoice number.
	InvoiceNumber string `json:"invoice_number" validate:"required"`
	// The date the invoice was created.
	InvoicedAt time.Time `json:"invoiced_at" validate:"required"`
}

// AnalyzeOpenBatchesResponse represents the response from the analyze open batches endpoint.
// Uses the existing OpenBatchSummary type from batch_resource.go.
type AnalyzeOpenBatchesResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// The open batch summary data.
	Data []OpenBatchSummary `json:"data" validate:"required"`
}

// AnalyzeProductionCostsResponse represents the response from the analyze production costs endpoint.
type AnalyzeProductionCostsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// The production cost data.
	Data []ProductionCostItem `json:"data" validate:"required"`
}

// ProductionCostItem represents an aggregated production cost entry.
type ProductionCostItem struct {
	// The department information.
	Department *Entity `json:"department"`
	// The category information.
	Category *Entity `json:"category" validate:"required"`
	// The total costs.
	TotalCosts CostBreakdown `json:"total_costs" validate:"required"`
	// The productive costs.
	ProductiveCosts CostBreakdown `json:"productive_costs" validate:"required"`
	// The waste costs.
	WasteCosts CostBreakdown `json:"waste_costs" validate:"required"`
	// The seconds costs.
	SecondsCosts CostBreakdown `json:"seconds_costs" validate:"required"`
}

// CostBreakdown represents a detailed cost breakdown with sub-quantities.
type CostBreakdown struct {
	// The total amount.
	Total *Quantity `json:"total" validate:"required"`
	// The labor amount.
	Labor *Quantity `json:"labor" validate:"required"`
	// The materials amount.
	Materials *Quantity `json:"materials" validate:"required"`
	// The overhead amount.
	Overhead *Quantity `json:"overhead" validate:"required"`
	// The time amount.
	Time *Quantity `json:"time" validate:"required"`
	// The quantity amount.
	Quantity *Quantity `json:"quantity" validate:"required"`
}

// AnalyzeDeliveriesResponse represents the response from the analyze deliveries endpoint.
type AnalyzeDeliveriesResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_deliveries_response"`
	// The delivery statistics.
	Statistics DeliveryStatistics `json:"statistics" validate:"required"`
	// The chart data for delivery analytics.
	ChartData DeliveryChartData `json:"chart_data" validate:"required"`
}

// DeliveryStatistics represents delivery performance statistics.
type DeliveryStatistics struct {
	// Average time to first shipment in days.
	AverageTimeToFirstShipment *float64 `json:"average_time_to_first_shipment"`
	// Average time to completion in days.
	AverageTimeToCompletion *float64 `json:"average_time_to_completion"`
	// On-time delivery percentage.
	OnTimeDeliveryPercentage *float64 `json:"on_time_delivery_percentage"`
	// On-time first shipment percentage.
	OnTimeFirstShipmentPercentage *float64 `json:"on_time_first_shipment_percentage"`
	// Total number of orders.
	TotalOrders int64 `json:"total_orders" validate:"required"`
	// Number of orders with first shipment.
	OrdersWithFirstShipment int64 `json:"orders_with_first_shipment" validate:"required"`
	// Number of orders with completion.
	OrdersWithCompletion int64 `json:"orders_with_completion" validate:"required"`
	// Number of orders with a promise date.
	OrdersWithPromiseDate int64 `json:"orders_with_promise_date" validate:"required"`
	// Number of orders partially fulfilled within the promise date.
	OrdersPartiallyFulfilledInPromiseDate int64 `json:"orders_partially_fulfilled_in_promise_date" validate:"required"`
	// Number of orders completed within the promise date.
	OrdersCompletedWithinPromiseDate int64 `json:"orders_completed_within_promise_date" validate:"required"`
}

// DeliveryChartData contains chart data for delivery analytics.
type DeliveryChartData struct {
	// On-time delivery chart data.
	OnTimeDelivery ChartData `json:"on_time_delivery" validate:"required"`
	// Average delivery time chart data.
	AverageDeliveryTime ChartData `json:"average_delivery_time" validate:"required"`
	// Average first shipment time chart data.
	AverageFirstShipmentTime ChartData `json:"average_first_shipment_time" validate:"required"`
}

// ChartData represents data for a chart visualization.
type ChartData struct {
	// The chart name/label.
	Name string `json:"name" validate:"required"`
	// The chart type.
	Type string `json:"type" validate:"required"`
	// The chart data points.
	Data []Coordinate `json:"data" validate:"required"`
}

// Coordinate represents a single data point on a chart.
type Coordinate struct {
	// The x-axis value.
	X float64 `json:"x" validate:"required"`
	// The y-axis value.
	Y float64 `json:"y" validate:"required"`
}

// AnalyzeManufacturingResponse represents the response from the analyze manufacturing endpoint.
type AnalyzeManufacturingResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_manufacturing_response"`
	// The computed manufacturing value.
	Value float64 `json:"value" validate:"required"`
}

// AnalyzeManufacturingBatchResponse represents the response from the analyze manufacturing batch endpoint.
type AnalyzeManufacturingBatchResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_manufacturing_batch_response"`
	// The current period metrics.
	Current ManufacturingMetrics `json:"current" validate:"required"`
	// The comparison period metrics.
	Comparison ManufacturingMetrics `json:"comparison" validate:"required"`
}

// ManufacturingMetrics represents manufacturing performance metrics for a period.
type ManufacturingMetrics struct {
	// The production metric value.
	Production float64 `json:"production" validate:"required"`
	// The costs per unit metric value.
	CostsPerUnit float64 `json:"costs_per_unit" validate:"required"`
	// The margin metric value.
	Margin float64 `json:"margin" validate:"required"`
	// The quality metric value.
	Quality float64 `json:"quality" validate:"required"`
	// The labor efficiency metric value.
	LaborEfficiency float64 `json:"labor_efficiency" validate:"required"`
}

// AnalyzeOrdersResponse represents the response from the analyze orders endpoint.
type AnalyzeOrdersResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// The order entry data.
	Data []OrderEntry `json:"data" validate:"required"`
}

// OrderEntry represents a single order entry for analytics.
type OrderEntry struct {
	// Unique identifier for this entry.
	ID string `json:"id" validate:"required"`
	// The date the order was issued.
	IssuedAt *time.Time `json:"issued_at"`
	// The customer purchase order number.
	CustomerPO *string `json:"customer_po"`
	// The order number.
	OrderNumber string `json:"order_number" validate:"required"`
	// The order ID.
	OrderID string `json:"order_id" validate:"required"`
	// The sales representative ID.
	SalesRepID *string `json:"sales_rep_id"`
	// The sales representative username.
	SalesRepUsername *string `json:"sales_rep_username"`
	// The customer ID.
	CustomerID string `json:"customer_id" validate:"required"`
	// The customer name.
	CustomerName string `json:"customer_name" validate:"required"`
	// The customer number.
	CustomerNumber string `json:"customer_number" validate:"required"`
	// The customer type group ID.
	CustomerTypeGroupID *string `json:"customer_type_group_id"`
	// The customer group name.
	CustomerGroupName *string `json:"customer_group_name"`
	// The parent customer ID.
	ParentCustomerID *string `json:"parent_customer_id"`
	// The date the customer was created.
	CustomerCreatedAt time.Time `json:"customer_created_at" validate:"required"`
	// The product line ID.
	ProductLineID *string `json:"product_line_id"`
	// The product line name.
	ProductLine *string `json:"product_line"`
	// The product ID.
	ProductTypeID string `json:"product_type_id" validate:"required"`
	// The item ID.
	ItemID string `json:"item_id" validate:"required"`
	// The product SKU.
	ProductSku string `json:"product_sku" validate:"required"`
	// The product description.
	ProductDescription *string `json:"product_description"`
	// The category name.
	CategoryName string `json:"category_name" validate:"required"`
	// The quantity invoiced.
	QuantityInvoiced float64 `json:"quantity_invoiced" validate:"required"`
	// The unit of measure.
	Unit string `json:"unit" validate:"required"`
	// The unit cost.
	UnitCost float64 `json:"unit_cost" validate:"required"`
	// The unit price.
	UnitPrice float64 `json:"unit_price" validate:"required"`
	// The unit profit.
	UnitProfit float64 `json:"unit_profit" validate:"required"`
	// The total invoiced amount.
	TotalInvoiced float64 `json:"total_invoiced" validate:"required"`
	// The total cost.
	TotalCost float64 `json:"total_cost" validate:"required"`
	// The total profit.
	TotalProfit float64 `json:"total_profit" validate:"required"`
	// The ship-to city.
	ShipToCity *string `json:"ship_to_city"`
	// The ship-to zipcode.
	ShipToZipcode *string `json:"ship_to_zipcode"`
	// The ship-to state.
	ShipToState *string `json:"ship_to_state"`
	// The ship-to country.
	ShipToCountry *string `json:"ship_to_country"`
	// The order discount code.
	OrderDiscountCode *string `json:"discount_code"`
	// The date the order was completed.
	CompletedAt *time.Time `json:"completed_at"`
	// The date of the first shipment.
	FirstShipAt *time.Time `json:"first_ship_at"`
	// The promised delivery date.
	PromisedAt *time.Time `json:"promised_at"`
	// The quantity ordered.
	QuantityOrdered float64 `json:"quantity_ordered" validate:"required"`
	// The quantity back ordered.
	QuantityBackOrdered float64 `json:"quantity_back_ordered" validate:"required"`
	// The total ordered amount.
	TotalOrdered float64 `json:"total_ordered" validate:"required"`
	// The total back ordered amount.
	TotalBackOrdered float64 `json:"total_back_ordered" validate:"required"`
}

// AnalyzeQuarterlyOrdersResponse represents the response from the analyze quarterly orders endpoint.
type AnalyzeQuarterlyOrdersResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_quarterly_orders_response"`
	// The yearly sales data keyed by year string.
	Data map[string]QuarterlySalesData `json:"data" validate:"required"`
}

// QuarterlySalesData represents sales data broken down by quarter.
type QuarterlySalesData struct {
	// First quarter total.
	Q1 float64 `json:"q1" validate:"required"`
	// Second quarter total.
	Q2 float64 `json:"q2" validate:"required"`
	// Third quarter total.
	Q3 float64 `json:"q3" validate:"required"`
	// Fourth quarter total.
	Q4 float64 `json:"q4" validate:"required"`
	// Annual total.
	Total float64 `json:"total" validate:"required"`
}

// AnalyzeMaterialsResponse represents the response from the analyze materials endpoint.
type AnalyzeMaterialsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// The material analytics data.
	Data []MaterialAnalyticsEntry `json:"data" validate:"required"`
}

// MaterialAnalyticsEntry represents a single material analytics entry.
type MaterialAnalyticsEntry struct {
	// Unique identifier for this entry.
	ID string `json:"id" validate:"required"`
	// The item ID.
	ItemID string `json:"item_id" validate:"required"`
	// The SKU.
	Sku string `json:"sku" validate:"required"`
	// The description.
	Description *string `json:"description"`
	// The quantity in inventory.
	QuantityInInventory *Quantity `json:"quantity_in_inventory" validate:"required"`
	// The order point quantity.
	OrderPoint *Quantity `json:"order_point"`
	// The lead time.
	LeadTime *Quantity `json:"lead_time"`
	// The quantity in demand.
	QuantityInDemand *Quantity `json:"quantity_in_demand" validate:"required"`
	// The unit group.
	UnitGroup AnalyticsUnitGroup `json:"unit_group" validate:"required"`
	// The supplier names.
	SupplierNames []string `json:"supplier_names" validate:"required"`
	// The supplier part numbers.
	SupplierPartNumbers []string `json:"supplier_part_numbers" validate:"required"`
}

// AnalyticsUnitGroup represents a unit group for analytics.
type AnalyticsUnitGroup struct {
	// The unit group ID.
	ID string `json:"id" validate:"required"`
	// The unit group name.
	Name string `json:"name" validate:"required"`
	// The units in the group.
	Units []AnalyticsUnitGroupUnit `json:"units" validate:"required"`
}

// AnalyticsUnitGroupUnit represents a unit within a unit group.
type AnalyticsUnitGroupUnit struct {
	// The unit ID.
	ID string `json:"id" validate:"required"`
	// The unit name.
	Name string `json:"name" validate:"required"`
	// The unit abbreviation.
	Abbreviation string `json:"abbreviation" validate:"required"`
	// The conversion factor.
	ConversionFactor float64 `json:"conversion_factor" validate:"required"`
	// Whether this is the base unit.
	IsBaseUnit bool `json:"is_base_unit" validate:"required"`
}

// AnalyzeInventoryReceiptsResponse represents the response from the analyze inventory receipts endpoint.
type AnalyzeInventoryReceiptsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// The inventory receipt summary data.
	Data []InventoryReceiptSummaryEntry `json:"data" validate:"required"`
}

// InventoryReceiptSummaryEntry represents a summary of inventory receipts.
type InventoryReceiptSummaryEntry struct {
	// The item information.
	Item AnalyticsItem `json:"item" validate:"required"`
	// The location information.
	Location *Entity `json:"location"`
	// The lot information.
	Lot *AnalyticsLot `json:"lot"`
	// The owner account.
	OwnerAccount *Entity `json:"owner_account" validate:"required"`
	// The holder account.
	HolderAccount *Entity `json:"holder_account" validate:"required"`
	// The remaining quantity.
	RemainingQuantity *Quantity `json:"remaining_quantity" validate:"required"`
	// The weighted average unit cost.
	WeightedAverageUnitCost AnalyticsRate `json:"weighted_average_unit_cost" validate:"required"`
	// The inventory value.
	InventoryValue *Quantity `json:"inventory_value"`
	// The date of the oldest receipt.
	OldestReceiptAt *time.Time `json:"oldest_receipt_at"`
	// The date of the newest receipt.
	NewestReceiptAt *time.Time `json:"newest_receipt_at"`
}

// AnalyticsItem represents a lightweight item reference.
type AnalyticsItem struct {
	// The item ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// The item SKU.
	Sku string `json:"sku" validate:"required"`
	// The item description.
	Description *string `json:"description"`
}

// AnalyticsLot represents a lot for analytics.
type AnalyticsLot struct {
	// The lot ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=lot"`
	// The lot number.
	Number string `json:"number" validate:"required"`
}

// AnalyticsRate represents a rate with numerator and denominator quantities.
type AnalyticsRate struct {
	// The numerator quantity.
	Numerator *Quantity `json:"numerator" validate:"required"`
	// The denominator quantity.
	Denominator *Quantity `json:"denominator" validate:"required"`
}

// AnalyzeNewCustomersResponse represents the response from the analyze new customers endpoint.
type AnalyzeNewCustomersResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_new_customers_response"`
	// The new customers data.
	NewCustomers NewCustomersData `json:"new_customers" validate:"required"`
}

// NewCustomersData represents new customer time series data.
type NewCustomersData struct {
	// The label for the data series.
	Label string `json:"label" validate:"required"`
	// The data points.
	Data []DateTimeCoordinate `json:"data" validate:"required"`
}

// DateTimeCoordinate represents a time-value data point.
type DateTimeCoordinate struct {
	// The timestamp.
	X time.Time `json:"x" validate:"required"`
	// The value.
	Y float64 `json:"y" validate:"required"`
}

// AnalyzeDemandForecastResponse represents the response from the demand forecast endpoint.
type AnalyzeDemandForecastResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_demand_forecast_response"`
	// The demand forecast rows.
	Data *List[DemandForecastRow] `json:"data" validate:"required"`
	// The fraction of the current month elapsed.
	CurrentMonthFraction float64 `json:"current_month_fraction" validate:"required"`
}

// DemandForecastRow represents a single item's demand forecast data.
type DemandForecastRow struct {
	// The item.
	Item *Entity `json:"item" validate:"required"`
	// The product line.
	ProductLine *Entity `json:"product_line"`
	// The product SKU.
	ProductSku string `json:"product_sku" validate:"required"`
	// The product description.
	ProductDescription *string `json:"product_description"`
	// The unit of measure.
	Unit string `json:"unit" validate:"required"`
	// The currency.
	Currency string `json:"currency" validate:"required"`
	// The historical demand data points.
	History []DemandForecastPoint `json:"history" validate:"required"`
	// The forecasted demand data points.
	Forecast []DemandForecastForecastPoint `json:"forecast" validate:"required"`
	// The historical revenue data points.
	RevenueHistory []RevenueForecastPoint `json:"revenue_history" validate:"required"`
	// The forecasted revenue data points.
	RevenueForecast []DemandForecastForecastPoint `json:"revenue_forecast" validate:"required"`
	// The historical sales data points.
	SalesHistory []RevenueForecastPoint `json:"sales_history" validate:"required"`
	// The forecasted sales data points.
	SalesForecast []DemandForecastForecastPoint `json:"sales_forecast" validate:"required"`
	// The current month demand.
	CurrentMonthDemand float64 `json:"current_month_demand" validate:"required"`
	// The current month revenue.
	CurrentMonthRevenue float64 `json:"current_month_revenue" validate:"required"`
	// The current month sales.
	CurrentMonthSales float64 `json:"current_month_sales" validate:"required"`
}

// DemandForecastPoint represents a historical demand data point.
type DemandForecastPoint struct {
	// The date.
	Date time.Time `json:"at" validate:"required"`
	// The demand value.
	Demand float64 `json:"demand" validate:"required"`
}

// DemandForecastForecastPoint represents a forecasted data point with confidence bounds.
type DemandForecastForecastPoint struct {
	// The date.
	Date time.Time `json:"at" validate:"required"`
	// The forecast value.
	Forecast float64 `json:"forecast" validate:"required"`
	// The lower confidence bound.
	LowerBound float64 `json:"lower_bound" validate:"required"`
	// The upper confidence bound.
	UpperBound float64 `json:"upper_bound" validate:"required"`
}

// RevenueForecastPoint represents a historical revenue data point.
type RevenueForecastPoint struct {
	// The date.
	Date time.Time `json:"at" validate:"required"`
	// The revenue value.
	Revenue float64 `json:"revenue" validate:"required"`
}

// AnalyzeOeeResponse represents the response from the analyze OEE endpoint.
type AnalyzeOeeResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_oee_response"`
	// The OEE data by department.
	Departments *List[OeeDepartment] `json:"departments" validate:"required"`
}

// OeeDepartment represents OEE metrics for a single department.
type OeeDepartment struct {
	// The department.
	Department *Entity `json:"department" validate:"required"`
	// The number of good units produced.
	GoodUnits float64 `json:"good_units" validate:"required"`
	// The number of waste units.
	WasteUnits float64 `json:"waste_units" validate:"required"`
	// The number of seconds units.
	SecondsUnits float64 `json:"seconds_units" validate:"required"`
	// The time this output should have taken at each production step's own labor rate: ideal cycle time multiplied by the units produced. This is the numerator of Performance.
	StandardSecondsEarned float64 `json:"standard_seconds_earned" validate:"required"`
	// The estimated runtime in hours.
	EstimatedRuntimeHours float64 `json:"estimated_runtime_hours" validate:"required"`
	// Logged downtime charged against availability, in seconds.
	AvailabilityLossSeconds float64 `json:"availability_loss_seconds"`
	// Logged downtime charged against performance, in seconds.
	PerformanceLossSeconds float64 `json:"performance_loss_seconds"`
	// Logged downtime charged against quality, in seconds.
	QualityLossSeconds float64 `json:"quality_loss_seconds"`
	// Time nobody planned to run, removed from the OEE denominator rather than counted as a loss.
	NotScheduledSeconds float64 `json:"not_scheduled_seconds"`
	// Time spent changing over between products, in seconds.
	ChangeoverSeconds float64 `json:"changeover_seconds"`
	// Number of downtime events logged in the period.
	DowntimeEventCount int64 `json:"downtime_event_count"`
	// Downtime split by reason, largest first.
	DowntimeBreakdown *List[OeeDowntimeReason] `json:"downtime_breakdown"`
	// Planned time net of not-scheduled downtime, in seconds.
	ScheduledSeconds float64 `json:"scheduled_seconds"`
	// Scheduled time net of availability losses, in seconds.
	RunTimeSeconds float64 `json:"run_time_seconds"`
	// Run time divided by scheduled time.
	AvailabilityPct *float64 `json:"availability_pct"`
	// Standard seconds earned divided by run time: how fast the department ran against the designed speed of its production steps.
	PerformancePct *float64 `json:"performance_pct"`
	// Good units divided by total units produced.
	QualityPct *float64 `json:"quality_pct"`
	// Availability multiplied by performance multiplied by quality.
	OeePct *float64 `json:"oee_pct"`
	// Whether availability was measured from logged downtime or estimated from runtime. A department with no logged downtime computes as perfectly available, so an estimate is labelled rather than presented as a measurement.
	MeasurementStatus constants.OeeMeasurementStatus `json:"measurement_status" validate:"required"`
	// Data-quality warnings for this grouping. Empty when the numbers can be taken at face value.
	Anomalies []constants.OeeAnomaly `json:"anomalies"`
}

// AnalyzeOeeTrendResponse represents the response from the OEE trend endpoint.
type AnalyzeOeeTrendResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_oee_trend_response"`
	// One entry per production week in the requested window, oldest first.
	Periods *List[OeeTrendPeriod] `json:"periods" validate:"required"`
}

// OeeTrendPeriod represents one production week of OEE, rolled up across the departments that had scheduled time in it. Departments with no scheduled time have no OEE and take no part in the roll-up, so their output is not counted here either.
type OeeTrendPeriod struct {
	// The first instant this period covers. Weeks start on Monday; the first and last periods of a window are clipped to the window itself.
	StartsAt time.Time `json:"starts_at" validate:"required"`
	// The instant this period ends, exclusive.
	EndsAt time.Time `json:"ends_at" validate:"required"`
	// The number of good units produced.
	GoodUnits float64 `json:"good_units" validate:"required"`
	// The number of waste units.
	WasteUnits float64 `json:"waste_units" validate:"required"`
	// The number of seconds units.
	SecondsUnits float64 `json:"seconds_units" validate:"required"`
	// The time this output should have taken at each production step's own labor rate: ideal cycle time multiplied by the units produced.
	StandardSecondsEarned float64 `json:"standard_seconds_earned" validate:"required"`
	// Planned time net of not-scheduled downtime, in seconds.
	ScheduledSeconds float64 `json:"scheduled_seconds" validate:"required"`
	// Scheduled time net of availability losses, in seconds.
	RunTimeSeconds float64 `json:"run_time_seconds" validate:"required"`
	// Logged downtime charged against availability, in seconds.
	AvailabilityLossSeconds float64 `json:"availability_loss_seconds" validate:"required"`
	// Time nobody planned to run, removed from the denominator rather than counted as a loss.
	NotScheduledSeconds float64 `json:"not_scheduled_seconds" validate:"required"`
	// Run time divided by scheduled time.
	AvailabilityPct *float64 `json:"availability_pct"`
	// Standard seconds earned divided by run time.
	PerformancePct *float64 `json:"performance_pct"`
	// Good units divided by total units produced.
	QualityPct *float64 `json:"quality_pct"`
	// Availability multiplied by performance multiplied by quality.
	OeePct *float64 `json:"oee_pct"`
	// Whether availability was measured from logged downtime or estimated from runtime.
	MeasurementStatus constants.OeeMeasurementStatus `json:"measurement_status" validate:"required"`
	// Number of downtime events overlapping this period.
	DowntimeEventCount int64 `json:"downtime_event_count" validate:"required"`
}

// OeeDowntimeReason represents one reason's contribution to a department's downtime.
type OeeDowntimeReason struct {
	// Why the machine stopped.
	Reason constants.MachineDowntimeReasonCode `json:"reason" validate:"required"`
	// Which OEE term this reason charges.
	OeeBucket constants.OeeBucket `json:"oee_bucket" validate:"required"`
	// Downtime attributed to this reason, in seconds.
	DowntimeSeconds float64 `json:"downtime_seconds" validate:"required"`
	// Number of events logged against this reason.
	EventCount int64 `json:"event_count" validate:"required"`
}

// AnalyzeWeeksOfSalesResponse represents the response from the weeks-of-sales analytics endpoint.
type AnalyzeWeeksOfSalesResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_weeks_of_sales_response"`
	// The weeks-of-sales items.
	Data []WeeksOfSalesItem `json:"data" validate:"required"`
	// The total count.
	Count int64 `json:"count" validate:"required"`
}

// WeeksOfSalesItem represents a single product line's weeks-of-sales metrics.
type WeeksOfSalesItem struct {
	// The product line.
	ProductLine *Entity `json:"product_line" validate:"required"`
	// The on-hand quantity.
	QuantityOnHand *Quantity `json:"quantity_on_hand" validate:"required"`
	// The average weekly sales quantity.
	AverageSalesQuantity *Quantity `json:"average_sales_quantity" validate:"required"`
	// The number of weeks of inventory on hand.
	WeeksOfSales float64 `json:"weeks_of_sales" validate:"required"`
}

// One row of a schedule-attainment breakdown.
//
// Both ratios are reported because either alone misleads. `attainment_pct` caps each SKU at what was asked for, so over-building one easy item cannot paper over a total miss on another; `output_ratio_pct` does not cap, so it is the only one that reveals over-production.
type AttainmentBucket struct {
	// Identifies the bucket within the chosen grouping — a week start, machine ID, department ID or item ID.
	Key string `json:"key" validate:"required"`
	// Display label for the bucket.
	Label string `json:"label" validate:"required"`
	// First day of the week, when grouping by week.
	WeekStartDate *time.Time `json:"week_starts_at"`
	// Units the live plan called for.
	PlannedQuantity float64 `json:"planned_quantity"`
	// Units actually produced.
	ActualQuantity float64 `json:"actual_quantity"`
	// Units produced that were planned for, capped per campaign at what was asked.
	MatchedQuantity float64 `json:"matched_quantity"`
	// Units scrapped.
	WasteQuantity float64 `json:"waste_quantity"`
	// Units produced with no matching planned campaign.
	UnplannedQuantity float64 `json:"unplanned_quantity"`
	// Machine hours the plan called for.
	PlannedRunHours float64 `json:"planned_run_hours"`
	// Planned campaigns in this bucket.
	PlannedLines int64 `json:"planned_lines"`
	// Batches scanned in this bucket.
	BatchCount int64 `json:"batch_count"`
	// Share of the plan that was met. Null when nothing was planned.
	AttainmentPct *float64 `json:"attainment_pct"`
	// Output as a share of plan, uncapped. Null when nothing was planned.
	OutputRatioPct *float64 `json:"output_ratio_pct"`
}

// How well a published commitment survived the week it covered.
type FrozenAdherence struct {
	// The published version this measures.
	Schedule *Entity `json:"schedule" validate:"required"`
	// Version number of that schedule.
	Version int32 `json:"version"`
	// Campaigns frozen at publish.
	FrozenLineCount int64 `json:"frozen_line_count"`
	// Units frozen at publish.
	FrozenPlannedQuantity float64 `json:"frozen_planned_quantity"`
	// Frozen campaigns that were changed after publish.
	DeviatedLines int64 `json:"deviated_lines"`
	// Campaigns added into the frozen window after publish.
	AddedLines int64 `json:"added_lines"`
	// Total absolute unit change across frozen-week deviations.
	AbsDeltaUnits float64 `json:"abs_delta_units"`
	// Campaigns the floor ran inside the frozen window that the frozen plan never called for, counted per machine-week-SKU.
	//
	// Working around a commitment breaks it as surely as editing it does, so this scores alongside the hand edits rather than beside them.
	OffPlanLines int64 `json:"off_plan_lines"`
	// Units behind those off-plan campaigns.
	OffPlanQuantity float64 `json:"off_plan_quantity"`
	// Share of frozen campaigns that survived untouched. Null when nothing was frozen.
	LineAdherencePct *float64 `json:"line_adherence_pct"`
	// Share of frozen units that survived untouched. Null when nothing was frozen.
	UnitsAdherencePct *float64 `json:"units_adherence_pct"`
	// Last day of the frozen window.
	FrozenThroughDate *time.Time `json:"frozen_through_at"`
}

// Actual production measured against the plan that was live at the time.
//
// The baseline for each week is the version that was published on or before that week began, so republishing mid-horizon cannot rewrite a week the floor has already worked. `baseline_schedules` names the versions used, so any number here can be traced back to the plan that produced it.
type AnalyzeScheduleAttainmentResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_schedule_attainment_response"`
	// Start of the measured period.
	StartDate time.Time `json:"starts_at" validate:"required"`
	// End of the measured period.
	EndDate time.Time `json:"ends_at" validate:"required"`
	// The dimension the breakdown is grouped by.
	GroupBy constants.AttainmentGroupBy `json:"group_by" validate:"required"`
	// The published versions the measurement was taken against.
	BaselineSchedules *List[Entity] `json:"baseline_schedules"`
	// The breakdown.
	Buckets *List[AttainmentBucket] `json:"buckets"`
	// Every bucket combined.
	Totals AttainmentBucket `json:"totals"`
	// Frozen-week adherence per baseline version.
	FrozenAdherence *List[FrozenAdherence] `json:"frozen_adherence"`
	// Whether the period had a plan to measure against. When `no_baseline`, every ratio is null and the period has no plan rather than a missed one.
	BaselineStatus constants.AttainmentBaselineStatus `json:"baseline_status" validate:"required"`
	// Machines the plan asked for over this window.
	//
	// Every figure in this response covers those machines only. Production scanned onto a machine no published version scheduled is excluded outright, so the score measures the plan that was made rather than the whole plant against it.
	ScheduledMachineCount int64 `json:"scheduled_machine_count"`
}

// How reliably promised delivery dates were met.
type AnalyzeDeliveryPerformanceResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_delivery_performance_response"`
	// The whole window as one figure.
	Overall *DeliveryPerformance `json:"overall" validate:"required"`
	// The same figures broken into periods, by the date each order was due.
	Periods *List[DeliveryPerformance] `json:"periods" validate:"required"`
	// Orders already past their promise and still unshipped, by how late they are.
	Backlog *List[DeliveryBacklogBucket] `json:"backlog" validate:"required"`
	// Every miss in the window banded by how far it missed by, shipped and unshipped alike.
	//
	// The companion to `average_days_late`, which cannot tell "everything slips a day" from "most orders are fine and four are two months late". Those are opposite problems with opposite fixes, and one mean reports them identically.
	Lateness *List[DeliveryLatenessBucket] `json:"lateness" validate:"required"`
	// The same window by customer, worst first.
	ByCustomer *List[DeliveryBreakdown] `json:"by_customer" validate:"required"`
	// The same window by customer group, worst first.
	ByCustomerGroup *List[DeliveryBreakdown] `json:"by_customer_group" validate:"required"`
	// The same window by product line, worst first. An order spanning two lines is counted under both — a late order is late for every line on it — so these counts sum to more than the overall total.
	ByProductLine *List[DeliveryBreakdown] `json:"by_product_line" validate:"required"`
	// The same window by which rule produced each ship-by date: an explicitly promised date, the customer's lead time, their parent's, their group's, or the account default.
	//
	// This is what says how much of the score rests on a default nobody deliberately set. A plant whose on-time rate is carried by `account`-sourced commitments is measuring itself against a number it invented.
	ByCommitmentSource *List[DeliveryBreakdown] `json:"by_commitment_source" validate:"required"`
	// Issued orders in the window carrying no ship-by date, excluded from every rate above.
	//
	// Reported so the exclusion is visible: a delivery score computed over half the order book, silently, is worse than one that says which half. A non-zero count here means orders placed before commitments were tracked still need a ship-by date.
	UncommittedOrderCount int32 `json:"uncommitted_order_count"`
}

// Delivery performance for one slice of the order book.
type DeliveryBreakdown struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=delivery_breakdown"`
	// Identifier of the slice — a customer, customer group, product line, or commitment source. Empty when the dimension is unset on the orders in it.
	Key string `json:"key"`
	// Display name for the slice.
	Label string `json:"label"`
	// The delivery figures for it, on the same shape as the overall window.
	Performance *DeliveryPerformance `json:"performance" validate:"required"`
}

// One band of how far the window's misses missed by.
type DeliveryLatenessBucket struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=delivery_lateness_bucket"`
	// Name of the band.
	Label string `json:"label"`
	// Lower bound of the band in days late.
	MinDaysLate int32 `json:"min_days_late"`
	// Upper bound in days late; `0` means unbounded.
	MaxDaysLate int32 `json:"max_days_late"`
	// Orders in the band, shipped and unshipped.
	OrderCount int32 `json:"order_count"`
	// How many of them have since shipped. The remainder are still owed, and are the same orders `backlog` counts.
	ShippedCount int32 `json:"shipped_count"`
	// Quantity still unpacked across the band's orders.
	Units float64 `json:"units"`
}

// Delivery reliability for one period, or for a whole window.
type DeliveryPerformance struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=delivery_performance"`
	// First day of the period; absent on the overall figure.
	PeriodStart *time.Time `json:"period_start"`
	// Orders whose promised ship date fell in this period.
	//
	// This is the denominator for both rates below — orders that were due, not orders that shipped. Measuring against shipments only would let unshipped late orders disappear from the score.
	CommittedOrderCount int32 `json:"committed_order_count"`
	// How many of them have shipped at all.
	ShippedOrderCount int32 `json:"shipped_order_count"`
	// How many shipped on or before the promised date.
	OnTimeOrderCount int32 `json:"on_time_order_count"`
	// How many shipped on time and complete.
	OnTimeInFullCount int32 `json:"on_time_in_full_count"`
	// How many shipped late, plus those already past their date and still unshipped.
	LateOrderCount int32 `json:"late_order_count"`
	// How many due in this period have not shipped at all.
	//
	// These count against on-time: a promise not yet met is not a promise kept.
	NotYetShippedCount int32 `json:"not_yet_shipped_count"`
	// Share of due orders that shipped on time, as a percentage.
	//
	// Null rather than zero when nothing was due, so a quiet week does not render as total failure.
	OnTimePct *float64 `json:"on_time_pct"`
	// Share of due orders that shipped on time and complete, as a percentage.
	OnTimeInFullPct *float64 `json:"on_time_in_full_pct"`
	// Average days late, over late orders only.
	//
	// Averaging over every order would dilute a real problem into a number that looks fine.
	AverageDaysLate *float64 `json:"average_days_late"`
	// Average days from issue to first shipment, over orders that have shipped.
	AverageLeadTimeDays *float64 `json:"average_lead_time_days"`
	// Average lead time these orders were promised.
	//
	// The gap between this and `average_lead_time_days` is what a lead time is renegotiated on.
	AverageCommittedLeadTimeDays *float64 `json:"average_committed_lead_time_days"`
}

// One age band of orders past their promise and still unshipped.
type DeliveryBacklogBucket struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=delivery_backlog_bucket"`
	// Name of the band.
	Label string `json:"label"`
	// Lower bound of the band in days late.
	MinDaysLate int32 `json:"min_days_late"`
	// Upper bound in days late; `0` means unbounded.
	MaxDaysLate int32 `json:"max_days_late"`
	// Orders in the band.
	OrderCount int32 `json:"order_count"`
	// Quantity still owed across them, which is what remains unpacked rather than what was ordered.
	Units float64 `json:"units"`
}

var sampleOnTimePct = 92.31
var sampleOnTimeInFullPct = 88.46
var sampleAvgDaysLate = 4.5
var sampleAvgLeadTime = 26.0
var sampleAvgCommittedLeadTime = 30.0

var SampleAnalyzeDeliveryPerformanceResponse = &AnalyzeDeliveryPerformanceResponse{
	Object: constants.ObjectTypeAnalyzeDeliveryPerformanceResponse,
	Overall: &DeliveryPerformance{
		Object:                       constants.ObjectTypeDeliveryPerformance,
		CommittedOrderCount:          26,
		ShippedOrderCount:            24,
		OnTimeOrderCount:             24,
		OnTimeInFullCount:            23,
		LateOrderCount:               2,
		NotYetShippedCount:           2,
		OnTimePct:                    &sampleOnTimePct,
		OnTimeInFullPct:              &sampleOnTimeInFullPct,
		AverageDaysLate:              &sampleAvgDaysLate,
		AverageLeadTimeDays:          &sampleAvgLeadTime,
		AverageCommittedLeadTimeDays: &sampleAvgCommittedLeadTime,
	},
	Periods: NewList([]DeliveryPerformance{}, PageInfo{}),
	Backlog: NewList([]DeliveryBacklogBucket{{
		Object:      constants.ObjectTypeDeliveryBacklogBucket,
		Label:       "1_7_days",
		MinDaysLate: 1,
		MaxDaysLate: 7,
		OrderCount:  2,
		Units:       340,
	}}, PageInfo{}),
	Lateness: NewList([]DeliveryLatenessBucket{{
		Object:       constants.ObjectTypeDeliveryLatenessBucket,
		Label:        "1_3_days",
		MinDaysLate:  1,
		MaxDaysLate:  3,
		OrderCount:   2,
		ShippedCount: 0,
		Units:        340,
	}}, PageInfo{}),
	ByCustomer: NewList([]DeliveryBreakdown{{
		Object:      constants.ObjectTypeDeliveryBreakdown,
		Key:         SampleCustomerID,
		Label:       "Northwind Textiles",
		Performance: sampleDeliveryBreakdownPerformance,
	}}, PageInfo{}),
	ByCustomerGroup: NewList([]DeliveryBreakdown{}, PageInfo{}),
	ByProductLine:   NewList([]DeliveryBreakdown{}, PageInfo{}),
	ByCommitmentSource: NewList([]DeliveryBreakdown{{
		Object:      constants.ObjectTypeDeliveryBreakdown,
		Key:         string(constants.LeadTimeSourceCustomer),
		Label:       string(constants.LeadTimeSourceCustomer),
		Performance: sampleDeliveryBreakdownPerformance,
	}}, PageInfo{}),
	UncommittedOrderCount: 0,
}

var sampleDeliveryBreakdownPerformance = &DeliveryPerformance{
	Object:              constants.ObjectTypeDeliveryPerformance,
	CommittedOrderCount: 6,
	ShippedOrderCount:   5,
	OnTimeOrderCount:    4,
	OnTimeInFullCount:   4,
	LateOrderCount:      2,
	NotYetShippedCount:  1,
	OnTimePct:           &sampleOnTimePct,
	OnTimeInFullPct:     &sampleOnTimeInFullPct,
	AverageDaysLate:     &sampleAvgDaysLate,
}

// CustomerPricingFinding is one contracted price flagged by the pricing analysis.
type CustomerPricingFinding struct {
	// Identifier for this finding, stable for the same price and customer across runs.
	//
	// One contracted price produces one finding per customer it reaches, so the price's own ID is not unique across findings.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_pricing_finding"`
	// ID of the contracted price behind this finding, which is where it has to be changed.
	AccountPriceID string `json:"account_price_id" validate:"required"`
	// Why this price was flagged.
	Reason constants.PricingFindingReason `json:"reason" validate:"required"`
	// How the customer comes to receive this price.
	Origin constants.AccountPriceOrigin `json:"origin" validate:"required"`
	// The customer receiving the price.
	Customer *Customer `json:"customer" expandable:"true"`
	// The product line the price applies to.
	ProductLine *ProductLine `json:"product_line" expandable:"true"`
	// The attributes narrowing the price; empty when it covers the whole product line.
	Attributes *List[Attribute] `json:"attributes" expandable:"true"`
	// The contracted price.
	UnitPrice *ComputedRate `json:"unit_price" validate:"required"`
	// Median contracted price across every customer with a price for the same product line, attributes and per-unit basis. Null when no other customer has a comparable price.
	PeerMedianPrice *ComputedRate `json:"peer_median_price"`
	// How far below the peer median this price sits, as a fraction between 0 and 1. Null when there is no peer median.
	BelowPeerMedianFraction *string `json:"below_peer_median_fraction" format:"decimal"`
	// Gross margin at this price, as a fraction between 0 and 1. Null when no comparable cost could be established.
	GrossMargin *string `json:"gross_margin" format:"decimal"`
}

// CustomerPricingSummary reports the shape of the analysis behind the findings.
type CustomerPricingSummary struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_pricing_summary"`
	// Contracted prices examined.
	PricesAnalyzed int `json:"prices_analyzed"`
	// Prices flagged for sitting below the peer median.
	BelowPeerMedianCount int `json:"below_peer_median_count"`
	// Prices flagged for failing the target gross margin.
	BelowTargetMarginCount int `json:"below_target_margin_count"`
	// Prices whose margin could not be checked because no comparable cost was available.
	MarginNotAssessedCount int `json:"margin_not_assessed_count"`
	// Anything the analysis had to leave out, so the result never overstates its own coverage.
	Notes []string `json:"notes"`
}

// AnalyzeCustomerPricingResponse represents the response from the customer pricing analysis.
type AnalyzeCustomerPricingResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_customer_pricing_response"`
	// The flagged prices, worst first.
	Findings *List[CustomerPricingFinding] `json:"findings" validate:"required"`
	// What the analysis covered.
	Summary CustomerPricingSummary `json:"summary" validate:"required"`
}

// RealizedMarginFinding is one customer/SKU trading relationship flagged by the realized margin analysis.
type RealizedMarginFinding struct {
	// Identifier for this finding, stable for the same customer and item across runs.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=realized_margin_finding"`
	// Why this trading relationship was flagged.
	Reason constants.PricingFindingReason `json:"reason" validate:"required"`
	// The customer that was charged.
	Customer *Customer `json:"customer" expandable:"true"`
	// The customer group the customer belongs to.
	CustomerGroup *AccountGroup `json:"customer_group" expandable:"true"`
	// The item that was sold.
	Item *Item `json:"item" expandable:"true"`
	// The product line the item belongs to.
	ProductLine *ProductLine `json:"product_line" expandable:"true"`
	// Quantity invoiced over the window.
	QuantityInvoiced *ComputedQuantity `json:"quantity_invoiced" validate:"required"`
	// Revenue invoiced over the window.
	Revenue *ComputedQuantity `json:"revenue" validate:"required"`
	// Cost of goods for the quantity invoiced.
	Cost *ComputedQuantity `json:"cost" validate:"required"`
	// Revenue divided by quantity: the price actually achieved across the window.
	AverageUnitPrice *ComputedRate `json:"average_unit_price" validate:"required"`
	// Median achieved price for this item across every customer that bought it. Null when no other customer bought it.
	PeerMedianPrice *ComputedRate `json:"peer_median_price"`
	// Number of invoiced lines behind these totals.
	LineCount int `json:"line_count"`
	// How far below the peer median this customer's achieved price sits, as a fraction between 0 and 1. Null when there is no peer median.
	BelowPeerMedianFraction *string `json:"below_peer_median_fraction" format:"decimal"`
	// Realized gross margin, as a fraction between 0 and 1. Null when no cost was captured on the lines.
	GrossMargin *string `json:"gross_margin" format:"decimal"`
}

// RealizedMarginSummary reports the shape of the analysis behind the findings.
type RealizedMarginSummary struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=realized_margin_summary"`
	// Invoiced lines examined.
	LinesAnalyzed int `json:"lines_analyzed"`
	// Customer and SKU pairs examined.
	RelationshipsAnalyzed int `json:"relationships_analyzed"`
	// Relationships flagged for an achieved price below the peer median.
	BelowPeerMedianCount int `json:"below_peer_median_count"`
	// Relationships flagged for failing the target gross margin.
	BelowTargetMarginCount int `json:"below_target_margin_count"`
	// Relationships whose margin could not be checked because no cost was captured.
	MarginNotAssessedCount int `json:"margin_not_assessed_count"`
	// Anything the analysis had to leave out, so the result never overstates its own coverage.
	Notes []string `json:"notes"`
}

// AnalyzeRealizedMarginsResponse represents the response from the realized margin analysis.
type AnalyzeRealizedMarginsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_realized_margins_response"`
	// The flagged relationships, most money at stake first.
	Findings *List[RealizedMarginFinding] `json:"findings" validate:"required"`
	// What the analysis covered.
	Summary RealizedMarginSummary `json:"summary" validate:"required"`
}
