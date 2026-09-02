package domain

import "time"

// --- Sales Analytics ---

type AnalyzeSalesParams struct {
	AccountID        string
	StartDate        time.Time
	EndDate          time.Time
	ProductLineIDs   []string
	CustomerIDs      []string
	SalesRepIDs      []string
	CustomerGroupIDs []string
	Query            *string
	IsSalesRep       bool
}

type SalesEntry struct {
	ID                  string
	IssuedAt            *time.Time
	CompletedAt         *time.Time
	FirstShipAt         *time.Time
	PromisedAt          *time.Time
	InvoiceDate         time.Time
	InvoiceID           string
	InvoiceNumber       string
	CustomerPO          *string
	SalesOrderNumber    string
	SalesOrderID        string
	SalesRepID          *string
	SalesRepUsername    *string
	CustomerID          string
	ParentCustomerID    *string
	CustomerName        string
	CustomerNumber      string
	CustomerCreatedAt   time.Time
	CustomerTypeGroupID *string
	CustomerGroupName   *string
	ProductLineID       *string
	ProductTypeCode     string
	ItemID              string
	ProductSku          string
	ProductDescription  *string
	CategoryName        string
	ProductLine         *string
	Unit                string
	QuantityInvoiced    float64
	TotalInvoiced       float64
	TotalCost           float64
	TotalProfit         float64
	UnitPrice           float64
	UnitCost            float64
	UnitProfit          float64
	ShipToState         *string
	ShipToCity          *string
	ShipToPostalCode    *string
	ShipToCountry       *string
	OrderDiscountCode   *string
}

// --- Open Batches ---

type AnalyzeOpenBatchesParams struct {
	AccountID      string
	ItemIDs        []string
	ProductLineIDs []string
}

type OpenBatchEntry struct {
	BatchID             string
	BatchNumber         string
	ItemID              string
	ProductSku          string
	ProductDescription  *string
	ScanningStationName *string
	ScanningStationID   *string
	Quantity            float64
	Unit                string
	CreatedAt           time.Time
}

// --- Production Costs ---

type AnalyzeProductionCostsParams struct {
	AccountID      string
	StartDate      *time.Time
	EndDate        *time.Time
	ItemIDs        []string
	ProductLineIDs []string
	DepartmentIDs  []string
	CategoryIDs    []string
}

type ProductionCostEntry struct {
	ItemID             string
	ProductSku         string
	ProductDescription *string
	ProductLine        *string
	TotalQuantity      float64
	TotalCost          float64
	CostPerUnit        float64
	Unit               string
}

// --- Delivery Analytics ---

type AnalyzeDeliveriesParams struct {
	AccountID              string
	StartDate              time.Time
	EndDate                time.Time
	ProductLineIDs         []string
	CustomerIDs            []string
	CustomerGroupIDs       []string
	SalesRepIDs            []string
	TargetDeliveryTimeDays *int32
	OverridePromisedDates  *bool
}

type DeliveryEntry struct {
	InvoiceNumber string
	IssuedAt      *time.Time
	InvoicedAt    *time.Time
	CompletedAt   *time.Time
	FirstShipAt   *time.Time
	PromisedAt    *time.Time
}

type DeliveryStatistics struct {
	AverageTimeToFirstShipment            *float64
	AverageTimeToCompletion               *float64
	OnTimeDeliveryPercentage              *float64
	OnTimeFirstShipmentPercentage         *float64
	TotalOrders                           int32
	OrdersWithFirstShipment               int32
	OrdersWithCompletion                  int32
	OrdersWithPromiseDate                 int32
	OrdersPartiallyFulfilledInPromiseDate int32
	OrdersCompletedWithinPromiseDate      int32
}

type ChartDataPoint struct {
	X float64
	Y float64
}

type DeliveryChartData struct {
	OnTimeDelivery           []ChartDataPoint
	AverageDeliveryTime      []ChartDataPoint
	AverageFirstShipmentTime []ChartDataPoint
}

type DeliveryAnalyticsResult struct {
	Statistics DeliveryStatistics
	ChartData  DeliveryChartData
}

// --- Manufacturing Analytics ---

type AnalyzeManufacturingParams struct {
	AccountID string
	StartDate time.Time
	EndDate   time.Time
	Type      string
}

type AnalyzeManufacturingBatchParams struct {
	AccountID           string
	StartDate           time.Time
	EndDate             time.Time
	ComparisonStartDate time.Time
	ComparisonEndDate   time.Time
	CustomerIDs         []string
	ProductLineIDs      []string
	CustomerGroupIDs    []string
	ItemIDs             []string
}

type ManufacturingMetrics struct {
	Production      float64
	CostsPerUnit    float64
	Margin          float64
	Quality         float64
	LaborEfficiency float64
}

type ManufacturingBatchResult struct {
	Current    ManufacturingMetrics
	Comparison ManufacturingMetrics
}

// --- Orders Analytics ---

type AnalyzeOrdersParams struct {
	AccountID        string
	SalesRepIDs      []string
	ProductLineIDs   []string
	CustomerIDs      []string
	CustomerGroupIDs []string
	IsSalesRep       bool
}

type OrderEntry struct {
	ID                  string
	IssuedAt            *time.Time
	CompletedAt         *time.Time
	FirstShipAt         *time.Time
	PromisedAt          *time.Time
	CustomerPO          *string
	OrderNumber         string
	OrderID             string
	SalesRepID          *string
	SalesRepUsername    *string
	CustomerID          string
	ParentCustomerID    *string
	CustomerName        string
	CustomerNumber      string
	CustomerCreatedAt   time.Time
	CustomerTypeGroupID *string
	CustomerGroupName   *string
	ProductLineID       *string
	ProductTypeCode     string
	ItemID              string
	ProductSku          string
	ProductDescription  *string
	CategoryName        string
	ProductLine         *string
	QuantityOrdered     float64
	QuantityInvoiced    float64
	QuantityBackOrdered float64
	Unit                string
	UnitCost            float64
	UnitPrice           float64
	UnitProfit          float64
	TotalInvoiced       float64
	TotalCost           float64
	TotalProfit         float64
	TotalOrdered        float64
	TotalBackOrdered    float64
	ShipToState         *string
	ShipToCity          *string
	ShipToZipcode       *string
	ShipToCountry       *string
	OrderDiscountCode   *string
}

// --- Quarterly Orders ---

type AnalyzeQuarterlyOrdersParams struct {
	AccountID        string
	SalesRepIDs      []string
	ItemIDs          []string
	ProductLineIDs   []string
	CustomerIDs      []string
	CustomerGroupIDs []string
}

type QuarterlyData struct {
	Q1    float64
	Q2    float64
	Q3    float64
	Q4    float64
	Total float64
}

type YearlyQuarterlyData struct {
	Year int32
	Data QuarterlyData
}

// --- Material Analytics ---

type AnalyzeMaterialsParams struct {
	AccountID     string
	SalesOrderIDs []string
	SupplierIDs   []string
}

type MaterialBaseQuantity struct {
	Measure          float64
	UnitName         string
	UnitAbbreviation string
	UnitType         string
}

type MaterialUnitGroupUnit struct {
	ID               string
	Name             string
	Abbreviation     string
	ConversionFactor float64
	IsBaseUnit       bool
}

type MaterialUnitGroup struct {
	ID    string
	Name  string
	Units []MaterialUnitGroupUnit
}

type MaterialAnalyticsEntry struct {
	MaterialID          string
	ItemID              string
	Sku                 string
	Description         *string
	QuantityInInventory MaterialBaseQuantity
	OrderPoint          *MaterialBaseQuantity
	LeadTime            *MaterialBaseQuantity
	QuantityInDemand    MaterialBaseQuantity
	UnitGroup           MaterialUnitGroup
	SupplierNames       []string
	SupplierPartNumbers []string
}

// --- Inventory Receipt Analytics ---

type AnalyzeInventoryReceiptsParams struct {
	AccountID   string
	ItemIDs     []string
	LocationIDs []string
	LotIDs      []string
}

type InventoryReceiptEntry struct {
	ItemID                          string
	ProductSku                      string
	ProductDescription              *string
	LocationID                      *string
	LocationName                    *string
	LotID                           *string
	LotNumber                       *string
	OwnerAccountID                  string
	OwnerAccountName                string
	HolderAccountID                 string
	HolderAccountName               string
	RemainingQuantity               float64
	WeightedAverageUnitCost         float64
	InventoryValue                  float64
	OldestReceiptAt                 *time.Time
	NewestReceiptAt                 *time.Time
	Unit                            string
	UnitName                        string
	CostNumeratorUnitAbbreviation   string
	CostNumeratorUnitName           string
	CostDenominatorUnitAbbreviation string
	CostDenominatorUnitName         string
}

// --- New Customers Analytics ---

type GetNewCustomersAnalyticsParams struct {
	AccountID        string
	StartDate        time.Time
	EndDate          time.Time
	CustomerGroupIDs []string
	SalesRepIDs      []string
}

type NewCustomerEntry struct {
	CreatedAt time.Time
}

// --- Demand Forecast ---

type GetDemandForecastParams struct {
	AccountID      string
	ProductLineIDs []string
	ItemIDs        []string
	HistoryMonths  *int32
	ForecastMonths *int32
}

type DemandHistoryPoint struct {
	Date   time.Time
	Demand float64
}

type DemandForecastPoint struct {
	Date       time.Time
	Forecast   float64
	LowerBound float64
	UpperBound float64
}

type RevenueHistoryPoint struct {
	Date    time.Time
	Revenue float64
}

type RevenueForecastPoint struct {
	Date       time.Time
	Forecast   float64
	LowerBound float64
	UpperBound float64
}

type DemandForecastItem struct {
	ItemID              string
	ProductLineID       *string
	ProductSku          string
	ProductDescription  *string
	Unit                string
	Currency            string
	History             []DemandHistoryPoint
	Forecast            []DemandForecastPoint
	RevenueHistory      []RevenueHistoryPoint
	RevenueForecast     []RevenueForecastPoint
	SalesHistory        []RevenueHistoryPoint
	SalesForecast       []RevenueForecastPoint
	CurrentMonthDemand  float64
	CurrentMonthRevenue float64
	CurrentMonthSales   float64
}

type DemandForecastResult struct {
	Items                []DemandForecastItem
	CurrentMonthFraction float64
}

// GetDemandForecastWindowParams bounds the raw monthly demand/revenue reads used to build the demand forecast.
type GetDemandForecastWindowParams struct {
	AccountID string
	StartDate time.Time
	EndDate   time.Time
}

// DemandForecastMonthlyDemandRow is one item-month of order-based demand and revenue.
type DemandForecastMonthlyDemandRow struct {
	ItemID             string
	ProductSku         string
	ProductDescription *string
	ProductLineID      *string
	Unit               string
	Currency           string
	DemandYear         int32
	DemandMonth        int32
	MonthlyDemand      float64
	MonthlyRevenue     float64
}

// DemandForecastMonthlyRevenueRow is one item-month of invoice-based revenue.
type DemandForecastMonthlyRevenueRow struct {
	ItemID         string
	RevenueYear    int32
	RevenueMonth   int32
	MonthlyRevenue float64
}

// --- Weeks of Sales Analytics ---

type AnalyzeWeeksOfSalesParams struct {
	AccountID     string
	PeriodInWeeks int32
}

// SaleProductItemRow links a sale-type product's item to its product line.
type SaleProductItemRow struct {
	ItemID        string
	ProductLineID *string
}

// ProductLineInfoRow is a product line's identifier and display name.
type ProductLineInfoRow struct {
	ID   string
	Name string
}

// GetOrderQuantityByProductLineParams scopes ordered-quantity aggregation to one product line and time window.
type GetOrderQuantityByProductLineParams struct {
	AccountID     string
	ProductLineID string
	StartDate     time.Time
	EndDate       time.Time
}

// OrderQuantityByProductLineRow is the aggregate ordered quantity for a product line within a window.
type OrderQuantityByProductLineRow struct {
	TotalQuantity    float64
	UnitAbbreviation string
	UnitType         string
}

type WeeksOfSalesItem struct {
	ProductLineID                        string
	ProductLineName                      string
	QuantityOnHand                       float64
	QuantityOnHandUnitAbbreviation       string
	QuantityOnHandUnitType               string
	AverageSalesQuantity                 float64
	AverageSalesQuantityUnitAbbreviation string
	AverageSalesQuantityUnitType         string
	WeeksOfSales                         float64
}

type WeeksOfSalesResult struct {
	Items []WeeksOfSalesItem
	Count int64
}

// --- OEE Analytics ---

type AnalyzeOeeParams struct {
	AccountID     string
	StartDate     time.Time
	EndDate       time.Time
	DepartmentIDs []string
	// PlannedTimeHours is the scheduled production time per department for the window. Without it Availability has no denominator, so the OEE ratios are returned nil rather than guessed. Keyed by department ID.
	PlannedTimeHours map[string]float64
}

// OeeDowntimeReason is one reason's contribution to a department's downtime.
type OeeDowntimeReason struct {
	ReasonCode      string
	OeeBucket       string
	DowntimeSeconds float64
	EventCount      int64
}

type OeeDepartment struct {
	DepartmentID          string
	DepartmentName        string
	GoodUnits             float64
	WasteUnits            float64
	SecondsUnits          float64
	EstimatedRuntimeHours float64

	// StandardSecondsEarned is the time the period's output should have taken at each production step's own labor rate: ideal cycle time multiplied by units produced. It is the numerator of Performance.
	StandardSecondsEarned float64

	// Measured downtime, split by the OEE term each reason charges. NotScheduled is removed from the denominator rather than counted as a loss.
	AvailabilityLossSeconds float64
	PerformanceLossSeconds  float64
	QualityLossSeconds      float64
	NotScheduledSeconds     float64
	ChangeoverSeconds       float64
	DowntimeEventCount      int64
	DowntimeBreakdown       []OeeDowntimeReason

	// ScheduledSeconds is planned time net of not-scheduled downtime; RunTimeSeconds is scheduled time net of availability losses.
	ScheduledSeconds float64
	RunTimeSeconds   float64

	// Ratios are nil when their denominator is zero or planned time is unknown. A department with no scheduled time has no OEE, which is not the same as 0% OEE.
	AvailabilityPct *float64
	PerformancePct  *float64
	QualityPct      *float64
	OeePct          *float64

	// HasDowntimeData is false when nothing was logged for this department in the window. Callers must surface that: a department that logs no downtime computes 100% Availability, which reads as an improvement rather than missing data.
	HasDowntimeData bool
	// HasPerformanceAnomaly flags Performance > 1, which always means a stale run rate. The raw value is still reported rather than clamped, so the data-quality problem stays visible.
	HasPerformanceAnomaly bool
}

// GetOeeWindowParams bounds the raw OEE reads for one account and reporting window.
type GetOeeWindowParams struct {
	AccountID string
	StartDate time.Time
	EndDate   time.Time
	// WeekStartDay is the weekday the trend read buckets scans on, 0 = Sunday through 6 = Saturday, matching the account's schedule week. Unused by the window-total OEE reads, which do not bucket by week.
	WeekStartDay int
	// MachineIDs restricts the output reads to production on these machines — the machines the plan scheduled. Empty means every machine, so a caller with no schedule still measures the whole floor.
	MachineIDs []string
}

// OeeDepartmentDataRow is one department's unit counts and standard time earned in the window.
type OeeDepartmentDataRow struct {
	DepartmentID          string
	DepartmentName        string
	GoodUnits             float64
	WasteUnits            float64
	SecondsUnits          float64
	StandardSecondsEarned float64
}

// OeeEstimatedRuntimeRow is one department's estimated runtime in the window.
type OeeEstimatedRuntimeRow struct {
	DepartmentID   string
	RuntimeSeconds float64
}

// OeeDowntimeRow is one department-reason aggregate of logged downtime, clipped to the reporting window.
type OeeDowntimeRow struct {
	DepartmentID    string
	ReasonCode      string
	OeeBucket       string
	DowntimeSeconds int64
	EventCount      int64
}

// AnalyzeOeeTrendParams bounds an OEE trend read: the same window and department filter as AnalyzeOee, bucketed into production weeks.
type AnalyzeOeeTrendParams struct {
	AccountID     string
	StartDate     time.Time
	EndDate       time.Time
	DepartmentIDs []string
}

// OeeTrendPeriod is one production week of OEE, rolled up across the departments that had scheduled time in it.
//
// The roll-up is weighted by seconds rather than averaged across departments: a room that ran an hour must not weigh as heavily as one that ran all week.
type OeeTrendPeriod struct {
	StartsAt time.Time
	// EndsAt is exclusive, and is clipped to the requested window so the first and last weeks of a range report against the part of the week that was actually asked for.
	EndsAt time.Time

	GoodUnits             float64
	WasteUnits            float64
	SecondsUnits          float64
	StandardSecondsEarned float64

	ScheduledSeconds        float64
	RunTimeSeconds          float64
	AvailabilityLossSeconds float64
	NotScheduledSeconds     float64

	AvailabilityPct *float64
	PerformancePct  *float64
	QualityPct      *float64
	OeePct          *float64

	// HasDowntimeData is false when nothing was logged in this week, which makes its Availability an estimate rather than a measurement — the same distinction AnalyzeOee draws per department.
	HasDowntimeData    bool
	DowntimeEventCount int64
}

// OeeTrendDepartmentWeekRow is one department's output in one production week.
type OeeTrendDepartmentWeekRow struct {
	WeekStart             time.Time
	DepartmentID          string
	DepartmentName        string
	GoodUnits             float64
	WasteUnits            float64
	SecondsUnits          float64
	StandardSecondsEarned float64
}

// OeeDowntimeIntervalRow is one department's logged downtime window, unclipped: open events arrive already coalesced to now.
type OeeDowntimeIntervalRow struct {
	DepartmentID string
	OeeBucket    string
	StartedAt    time.Time
	EndedAt      time.Time
}

// DepartmentMachineCountRow is the number of machines in one department, the scheduled-time denominator for OEE.
type DepartmentMachineCountRow struct {
	DepartmentID string
	MachineCount int64
}

// AnalyzeRealizedMarginsParams filters the realized-margin audit.
type AnalyzeRealizedMarginsParams struct {
	StartDate time.Time
	EndDate   time.Time
	// CustomerIDs and CustomerGroupIDs narrow the reported findings, not the peer benchmark: a customer must be compared against everyone who bought the SKU, including those the caller did not ask about.
	CustomerIDs       []string
	CustomerGroupIDs  []string
	ProductLineIDs    []string
	TargetGrossMargin *string
	OutlierTolerance  *string
}

// RealizedMarginFinding is one customer/SKU trading relationship flagged by the audit.
type RealizedMarginFinding struct {
	CustomerID              string
	CustomerGroupID         string
	ItemID                  string
	ProductLineID           string
	UnitAbbreviation        string
	QuantityInvoiced        string
	Revenue                 string
	Cost                    string
	AverageUnitPrice        string
	PeerMedianPrice         *string
	BelowPeerMedianFraction *string
	GrossMargin             *string
	LineCount               int
	Reason                  string
}

// RealizedMarginAnalysis is the rolled-up result: one row per flagged customer and SKU, plus what the sweep covered.
type RealizedMarginAnalysis struct {
	Findings               []RealizedMarginFinding
	LinesAnalyzed          int
	RelationshipsAnalyzed  int
	BelowPeerMedianCount   int
	BelowTargetMarginCount int
	MarginNotAssessedCount int
}

// AnalyzeCustomerPricingParams filters the contracted-pricing audit.
type AnalyzeCustomerPricingParams struct {
	// CustomerIDs and CustomerGroupIDs narrow the reported findings, not the peer benchmark: a price must be compared against every comparable price, including those the caller did not ask about.
	CustomerIDs       []string
	CustomerGroupIDs  []string
	TargetGrossMargin *string
	OutlierTolerance  *string
}

// CustomerPricingFinding is one contracted price flagged by the audit.
type CustomerPricingFinding struct {
	AccountPriceID          string
	CustomerID              string
	ProductLineID           string
	AttributeIDs            []string
	UnitPrice               string
	NumeratorUnitID         string
	NumeratorUnitAbbr       string
	DenominatorUnitID       string
	DenominatorAbbr         string
	PeerMedianPrice         *string
	BelowPeerMedianFraction *string
	GrossMargin             *string
	Origin                  string
	Reason                  string
}

// CustomerPricingAnalysis is the swept result: the flagged prices plus what the sweep covered.
type CustomerPricingAnalysis struct {
	Findings               []CustomerPricingFinding
	PricesAnalyzed         int
	BelowPeerMedianCount   int
	BelowTargetMarginCount int
	MarginNotAssessedCount int
	Notes                  []string
}
