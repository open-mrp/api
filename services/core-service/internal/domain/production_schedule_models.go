package domain

import (
	"encoding/json"
	"time"

	"github.com/augno/api/services/core-service/internal/scheduling"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
)

// ProductionScheduleItemSetting is a merchant override for one item's planning.
type ProductionScheduleItemSetting struct {
	ItemID           string
	IsExcluded       bool
	LotMultipleUnits float64
	// FulfillmentPolicyCode is an explicit per-item override; empty means the item inherits from its product line, then the account default.
	FulfillmentPolicyCode string
}

// LoadSolverInputParams is everything needed to assemble a solve.
type LoadSolverInputParams struct {
	AccountID    string
	PlanningAsOf time.Time
	Settings     scheduling.Settings

	DemandWindowMonths    int
	ForecastHistoryMonths int
	ForecastMonths        int
	DemandBasisCode       string
	ForecastZ             float64

	// ConstraintDepartmentID is the department that sets the pace of the factory. Every machine in it is planned; everything downstream responds.
	ConstraintDepartmentID string

	ItemSettings map[string]ProductionScheduleItemSetting

	// DefaultFulfillmentPolicy is the last resort in the policy chain. Empty means make-to-stock.
	DefaultFulfillmentPolicy string

	// PinnedCampaigns are hand-edited campaigns already on the plan a regenerate is keeping; the solver plans around them.
	PinnedCampaigns []scheduling.PinnedCampaign
}

// GetConstraintBatchMeasurementsParams scopes the batch-history read to one account, one demand window and the constraint machines.
type GetConstraintBatchMeasurementsParams struct {
	AccountID   string
	WindowStart time.Time
	WindowEnd   time.Time
	MachineIDs  []string
	// ConstraintDepartmentID scopes measurements to batches whose production step belongs to the constraint department, so scans from other stages recorded against a constraint machine do not enter the plan.
	ConstraintDepartmentID string
}

// ConstraintBatchRow is one historical batch as read from the database: the measurement the solver consumes plus the raw scan metadata the input assembly needs alongside it.
type ConstraintBatchRow struct {
	Measurement scheduling.BatchMeasurement
	// QuantityUnitID is the unit the batch was scanned in; nil when the batch carries no quantity unit.
	QuantityUnitID *string
	// QuantityUnitRatio is the scan unit's ratio to its unit group's base unit (e.g. 2 for a pair counted in eaches); 0 when the batch carries no quantity unit.
	QuantityUnitRatio float64
	// ProductionStepID mirrors the raw column: nil when the batch has no step. Kept distinct from the mapped measurement so presence is not conflated with an empty string.
	ProductionStepID *string
}

// StepConsumptionRow is one input item a production step consumes.
type StepConsumptionRow struct {
	ProductionStepID string
	ItemID           string
}

// GetSeedBatchesParams bounds the genealogy seeds to the demand window, matching the batch-measurement window.
type GetSeedBatchesParams struct {
	AccountID   string
	ItemIDs     []string
	WindowStart time.Time
	WindowEnd   time.Time
}

// SeedBatchRow is one scanned batch a genealogy walk can start from.
type SeedBatchRow struct {
	BatchID string
	ItemID  string
}

// BatchFlowChildRow is one immediate downstream batch in the genealogy.
type BatchFlowChildRow struct {
	ParentBatchID string
	BatchID       string
	ItemID        string
}

// SellableProductRow is one sellable product carried by an item.
type SellableProductRow struct {
	ProductID string
	ItemID    string
	SKU       string
	// ProductLineID is nil when the product sells under no line.
	ProductLineID *string
}

// GetPooledOrderDemandParams scopes the order-demand read to one account, one history window and a set of products.
type GetPooledOrderDemandParams struct {
	AccountID   string
	WindowStart time.Time
	WindowEnd   time.Time
	ProductIDs  []string
}

// OpenOrderRequirementRow is one issued order line's outstanding quantity and the date it is due to ship.
//
// This is demand the plan owes rather than demand it forecasts. ShipByDate is nil for orders issued before commitments were tracked.
type OpenOrderRequirementRow struct {
	SalesOrderID     string
	SalesOrderNumber string
	SalesOrderLineID string
	ProductID        string
	ShipByDate       *time.Time
	OutstandingQty   float64
}

// CustomerDemandRow is one product's sold quantity to one customer in one calendar month.
type CustomerDemandRow struct {
	ProductID      string
	BuyerAccountID string
	Year           int
	Month          int
	Quantity       float64
}

// ProductionScheduleLineOrder is one campaign's contribution to one order.
type ProductionScheduleLineOrder struct {
	ID                       string
	ProductionScheduleLineID string
	SalesOrderID             string
	SalesOrderNumber         string
	SalesOrderLineID         string
	AllocatedQuantity        float64

	// Denormalized from the line so a caller can read what is being built without a second round trip.
	ItemID     string
	SKU        string
	WeekIndex  int32
	MachineID  string
	ShipByDate *time.Time
}

// CreateLineOrderParams is one link to write.
type CreateLineOrderParams struct {
	ProductionScheduleLineID string
	SalesOrderID             string
	SalesOrderLineID         string
	AllocatedQuantity        float64
}

// AnalyzeDeliveryPerformanceParams scopes a delivery measurement to one account, window and slice of the order book.
type AnalyzeDeliveryPerformanceParams struct {
	AccountID   string
	StartDate   time.Time
	EndDate     time.Time
	Granularity string

	DeliveryFilters
}

// DeliveryFilters narrows a delivery measurement to part of the order book. Every filter is empty-means-all, and they combine with AND.
type DeliveryFilters struct {
	CustomerIDs      []string
	CustomerGroupIDs []string
	ProductLineIDs   []string
	SalesRepIDs      []string
}

// DeliveryPerformanceResult is the whole delivery picture for one window.
type DeliveryPerformanceResult struct {
	Overall scheduling.DeliveryPerformance
	Periods []scheduling.DeliveryPerformance
	Backlog []scheduling.BacklogBucket
	// Lateness bands every miss by how far it missed by. An average cannot tell "everything slips a day" from "four orders are two months late", and those are opposite problems.
	Lateness []scheduling.LatenessBucket

	// The same window sliced four ways. Each is ordered worst-first, so the row that needs a conversation is the first one.
	ByCustomer         []scheduling.DeliveryBreakdown
	ByCustomerGroup    []scheduling.DeliveryBreakdown
	ByProductLine      []scheduling.DeliveryBreakdown
	ByCommitmentSource []scheduling.DeliveryBreakdown

	// UncommittedOrderCount is issued orders in the window carrying no ship-by date, excluded from every rate above. Reported so the exclusion is visible rather than silent.
	UncommittedOrderCount int
}

// ScheduleOrderCoverage is one order a version does not fully build in time, with the campaigns earmarked for the part it does.
type ScheduleOrderCoverage struct {
	SalesOrderID     string
	SalesOrderNumber string
	ItemID           string
	SKU              string
	UnitsAtRisk      float64
	DueWeek          int
	ReasonCode       string
	ShipByDate       *time.Time
	CoveringLines    []ScheduleOrderCoverageLine
}

// ScheduleOrderCoverageLine is one campaign earmarked for an order.
type ScheduleOrderCoverageLine struct {
	ProductionScheduleLineID string
	WeekIndex                int32
	MachineID                string
	AllocatedQuantity        float64
}

// PromiseDateQuote is the earliest date the published plan could ship a quantity.
type PromiseDateQuote struct {
	ItemID   string
	Quantity float64
	// EarliestShipDate and EarliestWeekIndex are nil when the published horizon cannot supply the quantity at all.
	EarliestShipDate  *time.Time
	EarliestWeekIndex *int
	IsPromisable      bool

	ProductionScheduleID      string
	ProductionScheduleVersion int32
}

// FulfillmentRecommendation is the engine's advice for one item, with the measurements behind it.
type FulfillmentRecommendation struct {
	scheduling.Recommendation
	// ProductLineID is the line the item sells under, empty when it sells under none.
	ProductLineID string
	// MixedStreamShare is the percentage of this item's demand coming from customers whose own policy disagrees with the recommendation. A high share means the single-policy-per-SKU model is straining on this item.
	MixedStreamShare float64
}

// CustomerFulfillmentProfile is how one customer buys: the time they allow and the policy they state, each already resolved through its own chain.
type CustomerFulfillmentProfile struct {
	CustomerAccountID string
	CustomerName      string
	// LeadTimeDays is resolved customer -> account group -> account, the same chain an order's ship-by is stamped from.
	LeadTimeDays int
	// FulfillmentPolicyCode is empty when neither the customer nor its group states one.
	FulfillmentPolicyCode string
}

// PooledMonthlyDemandRow is one product's sold quantity for one calendar month.
type PooledMonthlyDemandRow struct {
	ProductID string
	Year      int
	Month     int
	Quantity  float64
}

// ProductLineItemRow maps one product line to one item sold under it.
type ProductLineItemRow struct {
	ProductLineID string
	ItemID        string
}

// ItemProductLineRow maps one item to the product line it sells under.
type ItemProductLineRow struct {
	ItemID        string
	ProductLineID string
}

// ProductionScheduleSettingsRow is the merchant's stored planning assumptions as one raw row, before code defaults are merged in by the service.
type ProductionScheduleSettingsRow struct {
	PlanningHorizonWeeks           int
	FrozenWeeks                    int
	WeekStartDay                   int
	ShiftsPerDay                   int
	HoursPerShift                  float64
	WorkDaysPerWeek                int
	WeeksPerYear                   int
	CapacityHeadroomPct            float64
	DefaultLotUnits                float64
	ChangeoverAvgMinutes           float64
	ChangeoverMinMinutes           float64
	ChangeoverMaxMinutes           float64
	ChangeoverLaborRate            float64
	HoldingRatePct                 float64
	ServiceLevelZ                  float64
	FinishLeadTimeWeeks            float64
	DefaultConstraintLeadTimeWeeks float64
	MaxWeeksSupply                 float64
	MaxFlowDepth                   int

	DefaultCustomerLeadTimeDays  int
	DefaultFulfillmentPolicyCode string
	RecommendationThresholds     scheduling.RecommendationThresholds

	DemandWindowMonths    int
	ForecastHistoryMonths int
	ForecastMonths        int
	DemandBasisCode       string
	ForecastZ             float64
	// ConstraintDepartmentID is empty when no constraint department is configured.
	ConstraintDepartmentID string
}

// EffectiveScheduleSettings is the merchant's planning assumptions with code defaults already applied, so callers never have to handle "not configured".
type EffectiveScheduleSettings struct {
	Settings               scheduling.Settings
	DemandWindowMonths     int
	ForecastHistoryMonths  int
	ForecastMonths         int
	DemandBasisCode        string
	ForecastZ              float64
	ConstraintDepartmentID string
	ItemSettings           map[string]ProductionScheduleItemSetting
	// DefaultFulfillmentPolicy is the account-wide fallback for how a SKU is produced.
	DefaultFulfillmentPolicy string
	// RecommendationThresholds are the cut points the make-to-order recommendation is drawn against.
	RecommendationThresholds scheduling.RecommendationThresholds
	// DefaultCustomerLeadTimeDays is the last resort in a customer's ship-by chain.
	DefaultCustomerLeadTimeDays int
}

// PreviewProductionScheduleParams drives the internal solve-only endpoint.
type PreviewProductionScheduleParams struct {
	AccountID    string
	PlanningAsOf time.Time
	// HorizonWeeks and DemandBasis override the saved settings for this preview only.
	HorizonWeeks int
	DemandBasis  string
}

// Production schedule statuses. These alias the shared enum so the vocabulary has a single source of truth.
const (
	ScheduleStatusDraft      = string(constants.ProductionScheduleStatusDraft)
	ScheduleStatusGenerating = string(constants.ProductionScheduleStatusGenerating)
	ScheduleStatusPublished  = string(constants.ProductionScheduleStatusPublished)
	ScheduleStatusSuperseded = string(constants.ProductionScheduleStatusSuperseded)
	ScheduleStatusArchived   = string(constants.ProductionScheduleStatusArchived)
	ScheduleStatusFailed     = string(constants.ProductionScheduleStatusFailed)
)

// Schedule line statuses, aliasing the shared enum.
const (
	ScheduleLineStatusPlanned    = string(constants.ProductionScheduleLineStatusPlanned)
	ScheduleLineStatusReleased   = string(constants.ProductionScheduleLineStatusReleased)
	ScheduleLineStatusInProgress = string(constants.ProductionScheduleLineStatusInProgress)
	ScheduleLineStatusComplete   = string(constants.ProductionScheduleLineStatusComplete)
	ScheduleLineStatusCancelled  = string(constants.ProductionScheduleLineStatusCancelled)
)

// How a schedule came to exist, aliasing the shared enum.
const (
	ScheduleSourceManual    = string(constants.ScheduleGenerationSourceManual)
	ScheduleSourceScheduled = string(constants.ScheduleGenerationSourceScheduled)
)

// Why a line exists, aliasing the shared enum.
const (
	ScheduleLineSourceSolver = string(constants.ScheduleLineSourceSolver)
	ScheduleLineSourceManual = string(constants.ScheduleLineSourceManual)
)

// ProductionSchedule is one generated version of the plan.
type ProductionSchedule struct {
	ID        string
	AccountID string
	Version   int32

	StatusCode string  `audit:"status_code"`
	Name       *string `audit:"name"`

	PlanningAsOf      time.Time
	HorizonStartDate  time.Time
	HorizonEndDate    time.Time
	HorizonWeeks      int32
	FrozenWeeks       int32
	FrozenThroughDate *time.Time

	DemandBasisCode      string
	GenerationSourceCode string
	SolverVersion        string

	SettingsSnapshot json.RawMessage
	Diagnostics      json.RawMessage
	ErrorMessage     *string

	FrozenLineCount       int32
	FrozenPlannedQuantity float64

	GeneratedByID  *string
	PublishedByID  *string
	PublishedAt    *time.Time
	SupersededByID *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProductionScheduleLine is one planned campaign.
type ProductionScheduleLine struct {
	ID                   string
	ProductionScheduleID string

	WeekIndex     int32     `audit:"week_index"`
	WeekStartDate time.Time `audit:"week_start_date"`

	MachineID        string `audit:"machine_id"`
	ProductionStepID *string
	DepartmentID     *string
	ItemID           string `audit:"item_id"`

	PlannedQuantity float64 `audit:"planned_quantity"`
	PlannedUnitID   *string `audit:"planned_unit_id"`
	// PlannedUnitAbbreviation is joined for display: every quantity on this line is counted in it, and a bare 360 on a plan grid cannot say whether it means pairs or eaches.
	PlannedUnitAbbreviation  *string
	PlannedLots              int32 `audit:"planned_lots"`
	PlannedLotUnits          float64
	PlannedRunHours          float64 `audit:"planned_run_hours"`
	PlannedChangeoverMinutes float64
	SequenceIndex            int32 `audit:"sequence_index"`

	ProjectedOnHandBefore float64
	ProjectedOnHandAfter  float64

	StatusCode      string  `audit:"status_code"`
	SourceCode      string  `audit:"source_code"`
	ReasonCode      *string `audit:"reason_code"`
	IsFrozen        bool    `audit:"is_frozen"`
	ProductionRunID *string
	// Progress, measured from the run this campaign was released as. Zero until the week is released; a released campaign is complete when every batch it issued has been scanned.
	ReleasedBatchCount int64
	ScannedBatchCount  int64
	ScannedQuantity    float64

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProductionScheduleItemPolicy is the per-item "why" behind the lines, snapshotted so a historical plan stays explainable after costs and demand move.
type ProductionScheduleItemPolicy struct {
	ID                   string
	ProductionScheduleID string
	ItemID               string
	SKU                  string

	ProductionStepID *string
	PrimaryMachineID *string
	// UnitID is what every quantity in this policy is counted in; the abbreviation is joined for display.
	UnitID           *string
	UnitAbbreviation *string

	AnnualDemand   float64
	WeeklyDemand   float64
	SecondsPerUnit float64
	UnitCost       float64

	SetupCost   float64
	HoldingCost float64
	EOQUnits    float64

	ConstraintLeadTimeWeeks float64
	FinishLeadTimeWeeks     float64

	SigmaWeeklyPooled     float64
	SigmaDownstreamSum    float64
	SafetyStockPrimary    float64
	SafetyStockDownstream float64

	ReorderPoint  float64
	OrderUpTo     float64
	OnHandEchelon float64
	// The greige stage on its own, and how much the stage holds on average and at peak. The echelon figure above drives the build decision; these describe the store.
	OnHandGreige           float64
	AverageGreigeInventory float64
	MaxGreigeInventory     float64
	// ProjectedOnHand is the echelon position at the end of each horizon week. Weeks with no campaign are the ones this explains: stock draining toward the trigger.
	ProjectedOnHand []float64
	WeeksOfCover    float64
	AnnualRunHours  float64

	ABCClass           *string
	WasEOQCapped       bool
	WasCapacityStarved bool

	// The policy this SKU was solved under, the rule that decided it, and the split between what the order book already owed and what the forecast projected.
	FulfillmentPolicyCode string
	PolicySourceCode      string
	FirmDemandUnits       float64
	ForecastDemandUnits   float64

	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListProductionSchedulesParams struct {
	AccountID   string
	Cursor      *string
	Limit       int32
	Query       *string
	StatusCodes []string
}

type ListProductionSchedulesResult struct {
	Schedules []*ProductionSchedule
	PageInfo  pagination.PageInfo
}

type GetProductionScheduleParams struct {
	AccountID  string
	ScheduleID string
}

type ListProductionScheduleLinesParams struct {
	AccountID  string
	ScheduleID string
	MachineIDs []string
	WeekIndex  *int32
}

// GenerateProductionScheduleParams drives a solve-and-persist.
type GenerateProductionScheduleParams struct {
	AccountID    string
	PlanningAsOf time.Time
	HorizonWeeks int
	DemandBasis  string
	Name         *string
	SourceCode   string
}

// How a regenerate treats the hand edits already on a draft, aliasing the shared enum.
const (
	ScheduleMergeModePreserveManual = string(constants.ScheduleMergeModePreserveManual)
	ScheduleMergeModeReplaceAll     = string(constants.ScheduleMergeModeReplaceAll)
)

// What a regenerate would do to one campaign, aliasing the shared enum.
const (
	ScheduleDiffAdded     = string(constants.ScheduleDiffChangeAdded)
	ScheduleDiffRemoved   = string(constants.ScheduleDiffChangeRemoved)
	ScheduleDiffChanged   = string(constants.ScheduleDiffChangeChanged)
	ScheduleDiffUnchanged = string(constants.ScheduleDiffChangeUnchanged)
)

// RegenerateProductionScheduleParams drives a re-solve of an existing draft.
type RegenerateProductionScheduleParams struct {
	ScheduleID string
	// MergeMode decides what happens to hand-edited campaigns. Empty defaults to preserving them: silently discarding a planner's work is never the safe default.
	MergeMode string
	// PlanningAsOf, HorizonWeeks and DemandBasis override the stored version's own values. Empty or zero reuses what the version was generated with, so a plain regenerate answers "what would the solver say now" rather than "what would it say about a different question".
	PlanningAsOf *time.Time
	HorizonWeeks int
	DemandBasis  string
}

// ScheduleDiffLine is one campaign as the current plan and a fresh solve each see it.
type ScheduleDiffLine struct {
	ChangeCode string
	ItemID     string
	SKU        string
	MachineID  string
	WeekIndex  int32
	// CurrentQuantity and ProposedQuantity are zero on the side that does not have the campaign at all; ChangeCode says which side that is.
	CurrentQuantity  float64
	ProposedQuantity float64
	// CurrentIsManual marks a campaign a person created or edited. These are what preserve_manual keeps and what replace_all destroys.
	CurrentIsManual bool
}

// ScheduleRegeneratePreview is what a regenerate would do, without doing it.
type ScheduleRegeneratePreview struct {
	ScheduleID    string
	SolverVersion string
	PlanningAsOf  time.Time
	Lines         []ScheduleDiffLine
	AddedCount    int32
	RemovedCount  int32
	ChangedCount  int32
	// ManualLineCount is how many hand-edited campaigns the version currently holds, and DiscardedManualCount how many of them replace_all would destroy. The two are shown together so the cost of the destructive mode is a number rather than a warning.
	ManualLineCount      int32
	DiscardedManualCount int32
}

// ProductionScheduleFinishedPolicy is one finished SKU's own inventory target, snapshotted per version.
//
// The greige policy pools every finished good a constraint item feeds into one echelon figure, which is the right basis for deciding whether to build. These rows are what that pooling hides: per-SKU demand, per-SKU variability, and a buffer sized against the finishing lead time rather than the constraint's.
type ProductionScheduleFinishedPolicy struct {
	ID                   string
	AccountID            string
	ProductionScheduleID string

	ItemID        string
	SKU           string
	GreigeItemID  string
	GreigeSKU     string
	ProductLineID *string

	AnnualDemand float64
	WeeklyDemand float64
	SigmaWeekly  float64

	SafetyStock  float64
	ReorderPoint float64
	OnHand       float64
	WeeksOfCover float64

	CreatedAt time.Time
	UpdatedAt time.Time
}
