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

// --- Weeks of Sales Analytics ---

type AnalyzeWeeksOfSalesParams struct {
	AccountID     string
	PeriodInWeeks int32
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
}

type OeeDepartment struct {
	DepartmentID          string
	DepartmentName        string
	GoodUnits             float64
	WasteUnits            float64
	SecondsUnits          float64
	EstimatedRuntimeHours float64
}
