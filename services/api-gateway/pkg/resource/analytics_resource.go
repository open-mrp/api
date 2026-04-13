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

// SchemaExample returns an example of AnalyzeSalesResponse for documentation.
func (*AnalyzeSalesResponse) SchemaExample() any {
	return nil
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
	// The product type ID.
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
	Department *BasicInfo `json:"department"`
	// The category information.
	Category BasicInfo `json:"category" validate:"required"`
	// The total costs.
	TotalCosts CostBreakdown `json:"total_costs" validate:"required"`
	// The productive costs.
	ProductiveCosts CostBreakdown `json:"productive_costs" validate:"required"`
	// The waste costs.
	WasteCosts CostBreakdown `json:"waste_costs" validate:"required"`
	// The seconds costs.
	SecondsCosts CostBreakdown `json:"seconds_costs" validate:"required"`
}

// BasicInfo represents a simple ID and name pair.
type BasicInfo struct {
	// The unique identifier.
	ID string `json:"id" validate:"required"`
	// The display name.
	Name string `json:"name" validate:"required"`
}

// CostBreakdown represents a detailed cost breakdown with sub-quantities.
type CostBreakdown struct {
	// The total amount.
	Total BaseQuantity `json:"total" validate:"required"`
	// The labor amount.
	Labor BaseQuantity `json:"labor" validate:"required"`
	// The materials amount.
	Materials BaseQuantity `json:"materials" validate:"required"`
	// The overhead amount.
	Overhead BaseQuantity `json:"overhead" validate:"required"`
	// The time amount.
	Time BaseQuantity `json:"time" validate:"required"`
	// The quantity amount.
	Quantity BaseQuantity `json:"quantity" validate:"required"`
}

// BaseQuantity represents a quantity with its unit of measure.
type BaseQuantity struct {
	// The measured value.
	Measure float64 `json:"measure" validate:"required"`
	// The unit information.
	Unit BaseQuantityUnit `json:"unit" validate:"required"`
}

// BaseQuantityUnit represents the unit of a base quantity.
type BaseQuantityUnit struct {
	// The unit name.
	Name string `json:"name" validate:"required"`
	// The unit abbreviation.
	Abbreviation string `json:"abbreviation" validate:"required"`
	// The unit type.
	Type string `json:"type" validate:"required"`
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
	// The product type ID.
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
	QuantityInInventory BaseQuantity `json:"quantity_in_inventory" validate:"required"`
	// The order point quantity.
	OrderPoint *BaseQuantity `json:"order_point"`
	// The lead time.
	LeadTime *BaseQuantity `json:"lead_time"`
	// The quantity in demand.
	QuantityInDemand BaseQuantity `json:"quantity_in_demand" validate:"required"`
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
	Location *BasicInfo `json:"location"`
	// The lot information.
	Lot *AnalyticsLot `json:"lot"`
	// The owner account.
	OwnerAccount BasicInfo `json:"owner_account" validate:"required"`
	// The holder account.
	HolderAccount BasicInfo `json:"holder_account" validate:"required"`
	// The remaining quantity.
	RemainingQuantity BaseQuantity `json:"remaining_quantity" validate:"required"`
	// The weighted average unit cost.
	WeightedAverageUnitCost AnalyticsRate `json:"weighted_average_unit_cost" validate:"required"`
	// The inventory value.
	InventoryValue *BaseQuantity `json:"inventory_value"`
	// The date of the oldest receipt.
	OldestReceiptAt *time.Time `json:"oldest_receipt_at"`
	// The date of the newest receipt.
	NewestReceiptAt *time.Time `json:"newest_receipt_at"`
}

// AnalyticsItem represents a lightweight item reference.
type AnalyticsItem struct {
	// The item ID.
	ID string `json:"id" validate:"required"`
	// The item SKU.
	Sku string `json:"sku" validate:"required"`
	// The item description.
	Description *string `json:"description"`
}

// AnalyticsLot represents a lot for analytics.
type AnalyticsLot struct {
	// The lot ID.
	ID string `json:"id" validate:"required"`
	// The lot number.
	Number string `json:"number" validate:"required"`
}

// AnalyticsRate represents a rate with numerator and denominator quantities.
type AnalyticsRate struct {
	// The numerator quantity.
	Numerator BaseQuantity `json:"numerator" validate:"required"`
	// The denominator quantity.
	Denominator BaseQuantity `json:"denominator" validate:"required"`
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
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// The demand forecast rows.
	Data []DemandForecastRow `json:"data" validate:"required"`
	// The fraction of the current month elapsed.
	CurrentMonthFraction float64 `json:"current_month_fraction" validate:"required"`
}

// DemandForecastRow represents a single item's demand forecast data.
type DemandForecastRow struct {
	// The item ID.
	ItemID string `json:"item_id" validate:"required"`
	// The product line ID.
	ProductLineID *string `json:"product_line_id"`
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
	Date time.Time `json:"date" validate:"required"`
	// The demand value.
	Demand float64 `json:"demand" validate:"required"`
}

// DemandForecastForecastPoint represents a forecasted data point with confidence bounds.
type DemandForecastForecastPoint struct {
	// The date.
	Date time.Time `json:"date" validate:"required"`
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
	Date time.Time `json:"date" validate:"required"`
	// The revenue value.
	Revenue float64 `json:"revenue" validate:"required"`
}

// AnalyzeOeeResponse represents the response from the analyze OEE endpoint.
type AnalyzeOeeResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_oee_response"`
	// The OEE data by department.
	Departments []OeeDepartment `json:"departments" validate:"required"`
}

// OeeDepartment represents OEE metrics for a single department.
type OeeDepartment struct {
	// The department ID.
	DepartmentID string `json:"department_id" validate:"required"`
	// The department name.
	DepartmentName string `json:"department_name" validate:"required"`
	// The number of good units produced.
	GoodUnits float64 `json:"good_units" validate:"required"`
	// The number of waste units.
	WasteUnits float64 `json:"waste_units" validate:"required"`
	// The number of seconds units.
	SecondsUnits float64 `json:"seconds_units" validate:"required"`
	// The estimated runtime in hours.
	EstimatedRuntimeHours float64 `json:"estimated_runtime_hours" validate:"required"`
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
	ProductLine BasicInfo `json:"product_line" validate:"required"`
	// The on-hand quantity.
	QuantityOnHand BaseQuantity `json:"quantity_on_hand" validate:"required"`
	// The average weekly sales quantity.
	AverageSalesQuantity BaseQuantity `json:"average_sales_quantity" validate:"required"`
	// The number of weeks of inventory on hand.
	WeeksOfSales float64 `json:"weeks_of_sales" validate:"required"`
}
