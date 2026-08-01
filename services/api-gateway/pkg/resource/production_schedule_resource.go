package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleProductionScheduleID = "pnsc_0192a4c17b3e4f8a91c2d0"
const SampleProductionScheduleLineID = "pnscln_0192a4c17b3e4f8a91c2"
const SampleProductionScheduleItemPolicyID = "pnscitpc_0192a4c17b3e4f8a"

// The inventory policy computed for one item.
type SchedulePolicy struct {
	// The item this policy is for.
	Item *Entity `json:"item" validate:"required"`
	// SKU of the item.
	SKU string `json:"sku" validate:"required"`
	// Demand used for planning, annualized.
	AnnualDemand float64 `json:"annual_demand"`
	// Demand used for planning, per week.
	WeeklyDemand float64 `json:"weekly_demand"`
	// How long one unit occupies the constraint.
	SecondsPerUnit float64 `json:"seconds_per_unit"`
	// Standard cost per unit.
	UnitCost float64 `json:"unit_cost"`
	// Cost of one changeover, used as the setup cost in the lot-size calculation.
	SetupCost float64 `json:"setup_cost"`
	// Annual cost of holding one unit.
	HoldingCost float64 `json:"holding_cost"`
	// Economic order quantity.
	EOQUnits float64 `json:"eoq_units"`
	// Observed or default lead time at the constraint.
	ConstraintLeadTimeWeeks float64 `json:"constraint_lead_time_weeks"`
	// Lead time from the constraint to sellable stock.
	FinishLeadTimeWeeks float64 `json:"finish_lead_time_weeks"`
	// Buffer held at the constraint, pooled across the finished goods it feeds.
	SafetyStockPrimary float64 `json:"safety_stock_primary"`
	// Buffer held as finished goods.
	SafetyStockDownstream float64 `json:"safety_stock_downstream"`
	// Stock position at which a campaign is triggered.
	ReorderPoint float64 `json:"reorder_point"`
	// Ceiling on how far ahead this item is built.
	OrderUpTo float64 `json:"order_up_to"`
	// Stock on hand at the constraint plus everything downstream of it.
	OnHandEchelon float64 `json:"on_hand_echelon"`
	// Stock sitting at the constraint stage on its own.
	OnHandGreige float64 `json:"on_hand_greige"`
	// What the constraint stage holds on average: its buffer plus half a campaign.
	AverageGreigeInventory float64 `json:"average_greige_inventory"`
	// What the constraint stage holds at its peak: its buffer plus a whole campaign.
	MaxGreigeInventory float64 `json:"max_greige_inventory"`
	// Weeks of demand the current stock covers.
	WeeksOfCover float64 `json:"weeks_of_cover"`
	// ABC class by share of constraint run hours.
	ABCClass *constants.ABCClass `json:"abc_class"`
	// Constraint hours this item's annual demand consumes.
	AnnualRunHours float64 `json:"annual_run_hours"`
}

// One planned production block: make this item, on this machine, in this week.
type ScheduleCampaign struct {
	// The item to produce.
	Item *Entity `json:"item" validate:"required"`
	// SKU of the item.
	SKU string `json:"sku" validate:"required"`
	// The machine assigned to the campaign.
	Machine *Entity `json:"machine" validate:"required"`
	// Zero-based week offset from the start of the horizon.
	WeekIndex int32 `json:"week_index"`
	// Quantity to produce.
	Units float64 `json:"units"`
	// Whole lots the quantity rounds to.
	Lots int32 `json:"lots"`
	// Constraint hours the campaign consumes.
	RunHours float64 `json:"run_hours"`
}

// An item's projected stock position across the horizon.
type ScheduleProjection struct {
	// The item.
	Item *Entity `json:"item" validate:"required"`
	// Projected stock at the end of each week of the horizon.
	OnHandByWeek []float64 `json:"on_hand_by_week"`
}

// A demand override that changed a number, recorded so the plan can explain itself.
type ScheduleAppliedOverride struct {
	// The override that was applied.
	Override *Entity `json:"override" validate:"required"`
	// The item whose demand changed.
	Item *Entity `json:"item" validate:"required"`
	// The first instant of the month the override applied to.
	MonthStart time.Time `json:"month_starts_at"`
	// Demand before the override.
	Before float64 `json:"before"`
	// Demand after the override.
	After float64 `json:"after"`
	// How the override was expressed.
	Adjustment constants.DemandOverrideAdjustment `json:"adjustment"`
	// Why the override exists.
	Reason *constants.DemandOverrideReason `json:"reason"`
}

// What the solver could not do, and why the plan differs from raw history.
type ScheduleDiagnostics struct {
	// Items whose economic lot size was reduced to fit one machine-week, meaning shorter and more frequent campaigns.
	EOQCappedSKUs []string `json:"eoq_capped_skus"`
	// Items that cannot fit even a single lot into a machine-week and are therefore never scheduled.
	UnschedulableSKUs []string `json:"unschedulable_skus"`
	// Items below their reorder point that never won a slot in the horizon. This is the signal that the plant is short of capacity.
	CapacityStarvedSKUs []string `json:"capacity_starved_skus"`
	// Items with no measured run rate, which cannot be scheduled because their machine time is unknown.
	ItemsWithoutRunRate []string `json:"items_without_run_rate"`
	// Number of items the merchant has excluded from planning.
	ExcludedItemCount int32 `json:"excluded_item_count"`
	// Machines the constraint department contributed to this solve.
	ConstraintMachineCount int32 `json:"constraint_machine_count"`
	// Batches found on those machines in the demand window. Zero means nothing has been scanned there, which is why a plan can be empty even with machines configured.
	MeasuredBatchCount int32 `json:"measured_batch_count"`
	// Machines in the constraint department with no production step. Their campaigns derive no downstream department work.
	MachinesWithoutStep int32 `json:"machines_without_step"`
	// Calibrated changeover minutes per additional input.
	ChangeoverSlopeMinutes float64 `json:"changeover_slope_minutes"`
	// Average inputs a product transition introduces, measured from history.
	AverageInputsAdded float64 `json:"average_inputs_added"`
	// Every demand override that moved a number.
	AppliedOverrides *List[ScheduleAppliedOverride] `json:"applied_overrides"`
}

// A production plan produced by the scheduling solver.
type ProductionSchedulePreview struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_schedule_preview"`
	// Version of the solver that produced this plan.
	SolverVersion string `json:"solver_version" validate:"required"`
	// The instant the plan was calculated against.
	PlanningAsOf time.Time `json:"planning_as_of_at" validate:"required"`
	// The computed inventory policy per item, ordered by constraint run hours descending.
	Policies *List[SchedulePolicy] `json:"policies"`
	// The planned production blocks.
	Campaigns *List[ScheduleCampaign] `json:"campaigns"`
	// Projected stock position per item across the horizon.
	Projections *List[ScheduleProjection] `json:"projections"`
	// What the solver could not do.
	Diagnostics ScheduleDiagnostics `json:"diagnostics"`
}

var SampleProductionSchedulePreview = &ProductionSchedulePreview{
	Object:        constants.ObjectTypeProductionSchedulePreview,
	SolverVersion: "v1",
	PlanningAsOf:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	Policies: NewList([]SchedulePolicy{{
		Item: NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil), SKU: "GREIGE-001", AnnualDemand: 5200, WeeklyDemand: 100,
		SecondsPerUnit: 30, UnitCost: 4, EOQUnits: 720, ReorderPoint: 900, ABCClass: &sampleABCClassA,
	}}, PageInfo{}),
	Campaigns: NewList([]ScheduleCampaign{{
		Item: NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil), SKU: "GREIGE-001",
		Machine:   NewEntity(SampleMachineID, constants.ObjectTypeMachine, nil, nil),
		WeekIndex: 0, Units: 720, Lots: 12, RunHours: 6,
	}}, PageInfo{}),
	Projections: NewList([]ScheduleProjection{{
		Item: NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil), OnHandByWeek: []float64{620, 520},
	}}, PageInfo{}),
	Diagnostics: ScheduleDiagnostics{
		EOQCappedSKUs:       []string{},
		UnschedulableSKUs:   []string{},
		CapacityStarvedSKUs: []string{},
		ItemsWithoutRunRate: []string{},
		AppliedOverrides:    NewList([]ScheduleAppliedOverride{}, PageInfo{}),
	},
}

func (*ProductionSchedulePreview) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionSchedulePreview)
}

// A saved production schedule.
//
// Versions are immutable history: generating again creates a new version, and publishing supersedes the previous one rather than editing it, because attainment is measured against whichever version was live at the time.
type ProductionSchedule struct {
	// Schedule ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_schedule"`
	// Sequential version number within the account.
	Version int32 `json:"version" validate:"required"`
	// Where this version is in its lifecycle.
	Status constants.ProductionScheduleStatus `json:"status" validate:"required"`
	// Optional label for the version.
	Name *string `json:"name"`
	// The instant the plan was calculated against.
	PlanningAsOf time.Time `json:"planning_as_of_at" validate:"required"`
	// First instant of the horizon.
	HorizonStartDate time.Time `json:"horizon_starts_at" validate:"required"`
	// First instant of the last day of the horizon.
	HorizonEndDate time.Time `json:"horizon_ends_at" validate:"required"`
	// Length of the horizon in weeks.
	HorizonWeeks int32 `json:"horizon_weeks" validate:"required"`
	// How many leading weeks freeze on publish.
	FrozenWeeks int32 `json:"frozen_weeks"`
	// Last instant covered by the frozen window, set when the version is published.
	FrozenThroughDate *time.Time `json:"frozen_through_at"`
	// Which demand basis produced the plan.
	DemandBasis constants.ScheduleDemandBasis `json:"demand_basis" validate:"required"`
	// What triggered the generation.
	GenerationSource constants.ScheduleGenerationSource `json:"generation_source" validate:"required"`
	// Version of the solver that produced the plan.
	SolverVersion string `json:"solver_version" validate:"required"`
	// The planning assumptions used, frozen at generation so the plan stays explainable after settings change.
	SettingsSnapshot map[string]any `json:"settings_snapshot"`
	// What the solver could not do, frozen at generation.
	Diagnostics ScheduleDiagnostics `json:"diagnostics"`
	// Why generation failed, when it did.
	ErrorMessage *string `json:"error_message"`
	// Number of lines that were frozen at publish. Captured once and never recomputed, because frozen-week adherence measures against what was committed to.
	FrozenLineCount int32 `json:"frozen_line_count"`
	// Total quantity frozen at publish.
	FrozenPlannedQuantity float64 `json:"frozen_planned_quantity"`
	// The actor that generated this version.
	GeneratedBy *Actor `json:"generated_by"`
	// The actor that published this version.
	PublishedBy *Actor `json:"published_by"`
	// When this version was published.
	PublishedAt *time.Time `json:"published_at"`
	// The version that replaced this one.
	SupersededBy *Entity `json:"superseded_by"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProductionSchedule = &ProductionSchedule{
	ID:               SampleProductionScheduleID,
	Object:           constants.ObjectTypeProductionSchedule,
	Version:          3,
	Status:           constants.ProductionScheduleStatusDraft,
	PlanningAsOf:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	HorizonStartDate: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	HorizonEndDate:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
	HorizonWeeks:     13,
	FrozenWeeks:      1,
	DemandBasis:      constants.ScheduleDemandBasisTrailing12,
	GenerationSource: constants.ScheduleGenerationSourceManual,
	SolverVersion:    "v1",
	SettingsSnapshot: map[string]any{},
	Diagnostics: ScheduleDiagnostics{
		EOQCappedSKUs:       []string{},
		UnschedulableSKUs:   []string{},
		CapacityStarvedSKUs: []string{},
		ItemsWithoutRunRate: []string{},
		AppliedOverrides:    NewList([]ScheduleAppliedOverride{}, PageInfo{}),
	},
	GeneratedBy: SampleActor,
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductionSchedule) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionSchedule)
}

// A saved campaign on a production schedule.
type ProductionScheduleLine struct {
	// Line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_schedule_line"`
	// The schedule version this line belongs to.
	ProductionSchedule *Entity `json:"production_schedule" validate:"required"`
	// Zero-based week offset from the start of the horizon.
	WeekIndex int32 `json:"week_index"`
	// First instant of the week this campaign runs in.
	WeekStartDate time.Time `json:"week_starts_at" validate:"required"`
	// The machine assigned to the campaign.
	Machine *Entity `json:"machine" validate:"required"`
	// The production step the campaign runs at.
	ProductionStep *Entity `json:"production_step"`
	// The department the machine belongs to.
	Department *Entity `json:"department"`
	// The item to produce.
	Item *Entity `json:"item" validate:"required"`
	// Quantity to produce.
	PlannedQuantity float64 `json:"planned_quantity"`
	// Unit the quantity is expressed in.
	PlannedUnit *Entity `json:"planned_unit"`
	// Abbreviation of the unit every quantity on this line is counted in, for display.
	//
	// A campaign of 360 means 360 pairs or 360 eaches depending on this, so the two are never meaningful apart.
	PlannedUnitAbbreviation *string `json:"planned_unit_abbreviation"`
	// Whole lots the quantity rounds to.
	PlannedLots int32 `json:"planned_lots"`
	// Units in one lot, which is the batch size the week is released to the floor in.
	PlannedLotUnits float64 `json:"planned_lot_units"`
	// Constraint hours the campaign consumes.
	PlannedRunHours float64 `json:"planned_run_hours"`
	// Modelled changeover time before the campaign.
	PlannedChangeoverMinutes float64 `json:"planned_changeover_minutes"`
	// Order the campaign runs within its week.
	SequenceIndex int32 `json:"sequence_index"`
	// Projected stock before the campaign lands.
	ProjectedOnHandBefore float64 `json:"projected_on_hand_before"`
	// Projected stock after the campaign lands and the week's demand is drawn down.
	ProjectedOnHandAfter float64 `json:"projected_on_hand_after"`
	// Where the line is in its lifecycle.
	Status constants.ProductionScheduleLineStatus `json:"status" validate:"required"`
	// Whether the solver or a person created the line.
	Source constants.ScheduleLineSource `json:"source" validate:"required"`
	// Why the campaign was scheduled.
	Reason *constants.ScheduleChangeReason `json:"reason"`
	// Whether the line is inside the frozen window and can no longer be changed without recording a deviation.
	FreezeStatus constants.ScheduleFreezeStatus `json:"freeze_status" validate:"required"`
	// The production run created from this line.
	ProductionRun *Entity `json:"production_run"`
	// Batches this campaign issued to the floor when its week was released.
	//
	// Zero until the week is released.
	ReleasedBatchCount int64 `json:"released_batch_count"`
	// Batches of this campaign the floor has scanned.
	ScannedBatchCount int64 `json:"scanned_batch_count"`
	// Quantity scanned so far, in the planned unit.
	//
	// Measured from the run the week was released as, matched on this campaign's item, so a run holding several SKUs credits each campaign with only its own work.
	ScannedQuantity float64 `json:"scanned_quantity"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var samplePlannedUnitAbbreviation = "pr"

var SampleProductionScheduleLine = &ProductionScheduleLine{
	ID:                      SampleProductionScheduleLineID,
	Object:                  constants.ObjectTypeProductionScheduleLine,
	ProductionSchedule:      NewEntity(SampleProductionScheduleID, constants.ObjectTypeProductionSchedule, nil, nil),
	WeekIndex:               0,
	WeekStartDate:           timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	Machine:                 NewEntity(SampleMachineID, constants.ObjectTypeMachine, nil, nil),
	Item:                    NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil),
	PlannedQuantity:         720,
	PlannedLots:             12,
	PlannedLotUnits:         60,
	PlannedUnitAbbreviation: &samplePlannedUnitAbbreviation,
	PlannedRunHours:         6,
	Status:                  constants.ProductionScheduleLineStatusPlanned,
	Source:                  constants.ScheduleLineSourceSolver,
	CreatedAt:               timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:               timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductionScheduleLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionScheduleLine)
}

// The per-item policy behind a schedule version.
//
// Snapshotted at generation rather than recomputed, so a historical plan can still explain itself after costs, demand or settings move.
type ProductionScheduleItemPolicy struct {
	// Policy ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_schedule_item_policy"`
	// The schedule version this policy belongs to.
	ProductionSchedule *Entity `json:"production_schedule" validate:"required"`
	// The item the policy is for.
	Item *Entity `json:"item" validate:"required"`
	// SKU of the item.
	SKU string `json:"sku" validate:"required"`
	// Unit every quantity in this policy is counted in.
	Unit *Entity `json:"unit"`
	// Abbreviation of the unit every quantity in this policy is counted in, for display.
	//
	// A reorder point of 2,508 is uninterpretable without it, so the two are never meaningful apart.
	UnitAbbreviation *string `json:"unit_abbreviation"`
	// The production step the item runs at.
	ProductionStep *Entity `json:"production_step"`
	// The machine the item usually runs on.
	PrimaryMachine *Entity `json:"primary_machine"`
	// Demand used for planning, annualized.
	AnnualDemand float64 `json:"annual_demand"`
	// Demand used for planning, per week.
	WeeklyDemand float64 `json:"weekly_demand"`
	// How long one unit occupies the constraint.
	SecondsPerUnit float64 `json:"seconds_per_unit"`
	// Standard cost per unit.
	UnitCost float64 `json:"unit_cost"`
	// Cost of one changeover.
	SetupCost float64 `json:"setup_cost"`
	// Annual cost of holding one unit.
	HoldingCost float64 `json:"holding_cost"`
	// Economic order quantity.
	EOQUnits float64 `json:"eoq_units"`
	// Observed or default lead time at the constraint.
	ConstraintLeadTimeWeeks float64 `json:"constraint_lead_time_weeks"`
	// Lead time from the constraint to sellable stock.
	FinishLeadTimeWeeks float64 `json:"finish_lead_time_weeks"`
	// Pooled weekly demand variability at the constraint.
	SigmaWeeklyPooled float64 `json:"sigma_weekly_pooled"`
	// Summed weekly variability of the finished goods this item becomes.
	SigmaDownstreamSum float64 `json:"sigma_downstream_sum"`
	// Buffer held at the constraint.
	SafetyStockPrimary float64 `json:"safety_stock_primary"`
	// Buffer held as finished goods.
	SafetyStockDownstream float64 `json:"safety_stock_downstream"`
	// Stock position at which a campaign is triggered.
	ReorderPoint float64 `json:"reorder_point"`
	// Ceiling on how far ahead this item is built.
	OrderUpTo float64 `json:"order_up_to"`
	// Stock at the constraint plus everything downstream of it. This is what the build decision is made against — stock already finished still counts against building more.
	OnHandEchelon float64 `json:"on_hand_echelon"`
	// Stock sitting at the constraint stage on its own, which the echelon figure cannot be decomposed back into once summed.
	OnHandGreige float64 `json:"on_hand_greige"`
	// What the constraint stage holds on average: its buffer, plus half a campaign as one lands and drains.
	AverageGreigeInventory float64 `json:"average_greige_inventory"`
	// What the constraint stage holds at its peak: its buffer plus a whole campaign.
	MaxGreigeInventory float64 `json:"max_greige_inventory"`
	// Weeks of demand the current stock covers.
	WeeksOfCover float64 `json:"weeks_of_cover"`
	// The echelon position at the end of each horizon week, after that week's campaigns land and its demand is drawn down. A run of weeks with no campaign is stock draining toward `reorder_point`; this is what makes that visible rather than looking like the solver did nothing.
	ProjectedOnHand []float64 `json:"projected_on_hand"`
	// Constraint hours this item's annual demand consumes.
	AnnualRunHours float64 `json:"annual_run_hours"`
	// ABC class by share of constraint run hours.
	ABCClass *constants.ABCClass `json:"abc_class"`
	// Limits the solver hit while sizing this item's campaigns. Empty when the policy was applied as calculated.
	Constraints []constants.SchedulePolicyConstraint `json:"constraints"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProductionScheduleItemPolicy = &ProductionScheduleItemPolicy{
	ID:                 SampleProductionScheduleItemPolicyID,
	Object:             constants.ObjectTypeProductionScheduleItemPolicy,
	ProductionSchedule: NewEntity(SampleProductionScheduleID, constants.ObjectTypeProductionSchedule, nil, nil),
	Item:               NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil),
	SKU:                "GREIGE-001",
	AnnualDemand:       5200,
	WeeklyDemand:       100,
	SecondsPerUnit:     30,
	UnitCost:           4,
	EOQUnits:           720,
	ReorderPoint:       900,
	AnnualRunHours:     43.3,
	CreatedAt:          timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:          timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductionScheduleItemPolicy) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionScheduleItemPolicy)
}

var sampleABCClassA = constants.ABCClassA

const SampleProductionScheduleDeviationID = "pnscdw_0192c8e49d5a6b0c13e4"
const SampleScheduleDeviationTypeID = "pnscdwtp_01seedqtychg0"

// A kind of hand change to a plan.
type ScheduleDeviationType struct {
	// Deviation type ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=schedule_deviation_type"`
	// Stable code recorded on a deviation.
	Code constants.ScheduleDeviationType `json:"code" validate:"required"`
	// Display name of the type.
	Name string `json:"name" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleScheduleDeviationType = &ScheduleDeviationType{
	ID:        SampleScheduleDeviationTypeID,
	Object:    constants.ObjectTypeScheduleDeviationType,
	Code:      constants.ScheduleDeviationTypeQuantityChanged,
	Name:      "Quantity Changed",
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ScheduleDeviationType) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleScheduleDeviationType)
}

// One hand change to a production schedule.
//
// The log is append-only: it is what frozen-week adherence is measured from, and a plan edited back into shape has to stay distinguishable from one that was right the first time. `before` and `after` are full snapshots of the line, so a deviation stays readable after the line it describes is deleted.
//
// `is_frozen_week` is recorded when the change is made, from the freeze window as it stood at that moment. It is never re-derived, so a later publish cannot retroactively reclassify a past edit.
type ProductionScheduleDeviation struct {
	// Deviation ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_schedule_deviation"`
	// The schedule version the change was made to.
	ProductionSchedule *Entity `json:"production_schedule" validate:"required"`
	// The line that was changed. Null for a line that was removed.
	Line *Entity `json:"line"`
	// What kind of change this was.
	DeviationType constants.ScheduleDeviationType `json:"deviation_type" validate:"required"`
	// Whether the change fell inside the frozen window when it was made.
	FreezeStatus constants.ScheduleFreezeStatus `json:"freeze_status" validate:"required"`
	// The horizon week the change affected, zero-based.
	WeekIndex *int32 `json:"week_index"`
	// The machine whose campaign changed.
	Machine *Entity `json:"machine"`
	// The item whose campaign changed.
	Item *Entity `json:"item"`
	// Snapshot of the line before the change.
	Before map[string]any `json:"before"`
	// Snapshot of the line after the change.
	After map[string]any `json:"after"`
	// Signed change in planned units.
	DeltaQuantity float64 `json:"delta_quantity"`
	// Signed change in planned run hours.
	DeltaRunHours float64 `json:"delta_run_hours"`
	// Why the change was made. Required for changes inside a frozen week.
	Reason *constants.ScheduleChangeReason `json:"reason"`
	// Free-form explanation of the change.
	ReasonNote *string `json:"reason_note"`
	// The actor that made the change.
	Actor *Actor `json:"actor"`
	// When the change was made.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

var (
	sampleDeviationReasonCode = constants.ScheduleChangeReasonRushOrder
	sampleDeviationReasonNote = "Pulled forward for the Northwind rush order."
	sampleDeviationWeekIndex  = int32(0)
)

var SampleProductionScheduleDeviation = &ProductionScheduleDeviation{
	ID:                 SampleProductionScheduleDeviationID,
	Object:             constants.ObjectTypeProductionScheduleDeviation,
	ProductionSchedule: NewEntity(SampleProductionScheduleID, constants.ObjectTypeProductionSchedule, nil, nil),
	Line:               NewEntity(SampleProductionScheduleLineID, constants.ObjectTypeProductionScheduleLine, nil, nil),
	DeviationType:      constants.ScheduleDeviationTypeQuantityChanged,
	FreezeStatus:       constants.ScheduleFreezeStatusFrozen,
	WeekIndex:          &sampleDeviationWeekIndex,
	Machine:            NewEntity(SampleMachineID, constants.ObjectTypeMachine, nil, nil),
	Item:               NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil),
	Before:             map[string]any{"planned_quantity": 600},
	After:              map[string]any{"planned_quantity": 900},
	DeltaQuantity:      300,
	DeltaRunHours:      5,
	Reason:             &sampleDeviationReasonCode,
	ReasonNote:         &sampleDeviationReasonNote,
	Actor:              SampleActor,
	CreatedAt:          timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*ProductionScheduleDeviation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionScheduleDeviation)
}

const SampleProductionScheduleDerivedLineID = "pnscdl_0192d7f5ae6b7c1d24f5"

// Downstream department work implied by a constraint campaign.
//
// The solver only schedules the constraint; every other department's work follows from it by walking the production-step graph. `explosion_depth` is how many steps downstream this sits — depth 1 waits only on the constraint, depth 3 waits on two intermediate steps — which is what a readiness indicator keys off.
//
// The derived week can fall past the schedule's horizon when a long chain follows a late campaign. That work is still returned rather than dropped, because a department needs to see it coming.
type ProductionScheduleDerivedLine struct {
	// Derived line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_schedule_derived_line"`
	// The schedule version this work was derived from.
	ProductionSchedule *Entity `json:"production_schedule" validate:"required"`
	// The constraint campaign this work follows from.
	SourceLine *Entity `json:"source_line" validate:"required"`
	// The production step that does the work.
	ProductionStep *Entity `json:"production_step" validate:"required"`
	// The department that owns the step.
	Department *Entity `json:"department"`
	// The item being worked on.
	Item *Entity `json:"item" validate:"required"`
	// Horizon week the work falls in, zero-based.
	WeekIndex int32 `json:"week_index"`
	// First instant of that week.
	WeekStartDate time.Time `json:"week_starts_at" validate:"required"`
	// Units implied for this step.
	Quantity float64 `json:"quantity"`
	// The unit the quantity is expressed in.
	PlannedUnit *Entity `json:"planned_unit"`
	// How many steps downstream of the constraint this work sits.
	ExplosionDepth int32 `json:"explosion_depth"`
	// Weeks after the constraint campaign this work starts.
	OffsetWeeks int32 `json:"offset_weeks"`
	// Progress of the derived work.
	Status constants.ProductionScheduleLineStatus `json:"status" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProductionScheduleDerivedLine = &ProductionScheduleDerivedLine{
	ID:                 SampleProductionScheduleDerivedLineID,
	Object:             constants.ObjectTypeProductionScheduleDerivedLine,
	ProductionSchedule: NewEntity(SampleProductionScheduleID, constants.ObjectTypeProductionSchedule, nil, nil),
	SourceLine:         NewEntity(SampleProductionScheduleLineID, constants.ObjectTypeProductionScheduleLine, nil, nil),
	ProductionStep:     NewEntity(SampleProductionStepID, constants.ObjectTypeProductionStep, nil, nil),
	Department:         NewEntity(SampleDepartmentID, constants.ObjectTypeDepartment, nil, nil),
	Item:               NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil),
	WeekIndex:          1,
	WeekStartDate:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	Quantity:           600,
	ExplosionDepth:     1,
	OffsetWeeks:        1,
	Status:             constants.ProductionScheduleLineStatusPlanned,
	CreatedAt:          timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:          timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductionScheduleDerivedLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionScheduleDerivedLine)
}

// One campaign as the current plan and a fresh solve each see it.
type ScheduleDiffLine struct {
	// What the regenerate would do to this campaign.
	Change constants.ScheduleDiffChange `json:"change" validate:"required"`
	// The item the campaign produces.
	Item *Entity `json:"item" validate:"required"`
	// SKU of that item.
	SKU string `json:"sku"`
	// The machine the campaign runs on.
	Machine *Entity `json:"machine"`
	// Zero-based horizon week.
	WeekIndex int32 `json:"week_index"`
	// Units the current plan asks for. Zero when the campaign is being added.
	CurrentQuantity float64 `json:"current_quantity"`
	// Units the fresh solve asks for. Zero when the campaign is being removed.
	ProposedQuantity float64 `json:"proposed_quantity"`
	// Whether the current campaign was created or edited by a person.
	CurrentIsManual bool `json:"current_is_manual"`
}

// What a regenerate would change about a draft, without changing it.
//
// A regenerate that silently discards hand-work is abandoned within two cycles, so the destructive mode states its cost as a number before it runs: `discarded_manual_count` is exactly how many hand-edited campaigns `replace_all` would destroy.
type ProductionScheduleRegeneratePreview struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_schedule_regenerate_preview"`
	// The draft this would act on.
	ProductionSchedule *Entity `json:"production_schedule" validate:"required"`
	// Which solver produced the proposal.
	SolverVersion string `json:"solver_version" validate:"required"`
	// The instant the fresh solve planned from.
	PlanningAsOf time.Time `json:"planning_as_of_at" validate:"required"`
	// Every campaign either plan holds, including the ones both agree on.
	Lines *List[ScheduleDiffLine] `json:"lines"`
	// Campaigns the fresh solve wants that the current plan does not have.
	AddedCount int32 `json:"added_count"`
	// Campaigns the current plan has that the fresh solve does not want.
	RemovedCount int32 `json:"removed_count"`
	// Campaigns both hold, in different quantities.
	ChangedCount int32 `json:"changed_count"`
	// Hand-edited campaigns currently on the draft.
	ManualLineCount int32 `json:"manual_line_count"`
	// Hand-edited campaigns `replace_all` would destroy.
	DiscardedManualCount int32 `json:"discarded_manual_count"`
}

var SampleProductionScheduleRegeneratePreview = &ProductionScheduleRegeneratePreview{
	Object:             constants.ObjectTypeProductionScheduleRegeneratePreview,
	ProductionSchedule: NewEntity(SampleProductionScheduleID, constants.ObjectTypeProductionSchedule, nil, nil),
	SolverVersion:      "1.0.0",
	PlanningAsOf:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	Lines: NewList([]ScheduleDiffLine{{
		Change:           constants.ScheduleDiffChangeChanged,
		Item:             NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil),
		SKU:              "MZ-CREW-BLK-L",
		Machine:          NewEntity(SampleMachineID, constants.ObjectTypeMachine, nil, nil),
		WeekIndex:        2,
		CurrentQuantity:  600,
		ProposedQuantity: 720,
	}}, PageInfo{}),
	AddedCount:   1,
	ChangedCount: 1,
}

func (*ProductionScheduleRegeneratePreview) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionScheduleRegeneratePreview)
}

const SampleProductionScheduleFinishedPolicyID = "pnscfipc_0192e8a6c14d70b3"

// One finished SKU's own inventory target, snapshotted onto a schedule version.
//
// The item policy pools every finished good a constraint item feeds into one echelon figure, which is the right basis for deciding whether to build. These rows are what that pooling hides: this SKU's own demand, its own variability, and a buffer sized against the finishing lead time rather than the constraint's — because finishing, not the constraint, is what replenishes this stock.
//
// The two stages do not overlap. The constraint stage holds its pooled buffer and the finished stage holds these, so together they describe the whole network's stock without counting any of it twice.
type ProductionScheduleFinishedPolicy struct {
	// Finished policy ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_schedule_finished_policy"`
	// The schedule version this was snapshotted onto.
	ProductionSchedule *Entity `json:"production_schedule" validate:"required"`
	// The finished good.
	Item *Entity `json:"item" validate:"required"`
	// SKU of the finished good.
	SKU string `json:"sku" validate:"required"`
	// The constraint item this is made from.
	GreigeItem *Entity `json:"greige_item" validate:"required"`
	// SKU of that constraint item.
	GreigeSKU string `json:"greige_sku" validate:"required"`
	// Product line the finished good belongs to.
	ProductLine *Entity `json:"product_line"`
	// This SKU's own annual demand.
	AnnualDemand float64 `json:"annual_demand"`
	// This SKU's own weekly demand.
	WeeklyDemand float64 `json:"weekly_demand"`
	// This SKU's own weekly demand variability. The constraint buffer pools these as the root of the sum of squares; these targets use them one at a time.
	SigmaWeekly float64 `json:"sigma_weekly"`
	// Buffer held as this finished good, covering the finishing lead time.
	SafetyStock float64 `json:"safety_stock"`
	// Stock position at which this finished good needs replenishing.
	ReorderPoint float64 `json:"reorder_point"`
	// This SKU's own stock, not the echelon it contributes to.
	OnHand float64 `json:"on_hand"`
	// Weeks of demand this SKU's own stock covers.
	WeeksOfCover float64 `json:"weeks_of_cover"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProductionScheduleFinishedPolicy = &ProductionScheduleFinishedPolicy{
	ID:                 SampleProductionScheduleFinishedPolicyID,
	Object:             constants.ObjectTypeProductionScheduleFinishedPolicy,
	ProductionSchedule: NewEntity(SampleProductionScheduleID, constants.ObjectTypeProductionSchedule, nil, nil),
	Item:               NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil),
	SKU:                "MZ-CREW-BLK-L",
	GreigeItem:         NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil),
	GreigeSKU:          "MZ-GREIGE-CREW",
	AnnualDemand:       26000,
	WeeklyDemand:       500,
	SigmaWeekly:        130,
	SafetyStock:        524,
	ReorderPoint:       3524,
	OnHand:             3000,
	WeeksOfCover:       6,
	CreatedAt:          timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:          timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductionScheduleFinishedPolicy) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionScheduleFinishedPolicy)
}

// One batch a release created, or would create: a single lot off one planned campaign.
type ReleaseScheduleBatch struct {
	// The item the batch produces.
	Item *Entity `json:"item" validate:"required"`
	// The item's SKU, as it stood when the plan was generated.
	SKU string `json:"sku" validate:"required"`
	// Units in this lot. The last lot of a campaign is short when the planned quantity is not a whole number of lots.
	Quantity float64 `json:"quantity"`
	// The batch that was created. Null on a preview, where nothing has been written.
	Batch *Entity `json:"batch"`
}

// One planned campaign and the lots it broke into.
type ReleasedScheduleLine struct {
	// The schedule line released.
	Line *Entity `json:"line" validate:"required"`
	// The item to produce.
	Item *Entity `json:"item" validate:"required"`
	// The item's SKU, as it stood when the plan was generated.
	SKU string `json:"sku" validate:"required"`
	// The machine the campaign runs on.
	Machine *Entity `json:"machine" validate:"required"`
	// Total units planned for the campaign.
	PlannedQuantity float64 `json:"planned_quantity"`
	// Units in one lot.
	LotUnits float64 `json:"lot_units"`
	// Abbreviation of the unit the quantity and the lot are counted in.
	//
	// `6 × 60` is not an instruction until it says 6 × 60 of what.
	Unit *string `json:"unit"`
	// How many batches the campaign broke into.
	BatchCount int32 `json:"batch_count"`
	// The individual lots, in run order.
	Batches *List[ReleaseScheduleBatch] `json:"batches"`
}

// The production run created from one week of a schedule.
//
// Each planned campaign becomes one batch per lot, so a 360-unit week at a 60-unit lot arrives on the floor as six batches rather than one instruction to make 360.
type ReleaseScheduleWeekResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_schedule_week_release"`
	// The run now carrying the week's work.
	ProductionRun *ProductionRun `json:"production_run"`
	// Zero-based week offset from the start of the horizon.
	WeekIndex int32 `json:"week_index"`
	// First instant of the released week.
	WeekStartDate time.Time `json:"week_starts_at" validate:"required"`
	// How many campaigns were released.
	ReleasedLineCount int32 `json:"released_line_count"`
	// How many batches were created across all campaigns.
	BatchCount int32 `json:"batch_count"`
	// Total units released.
	TotalQuantity float64 `json:"total_quantity"`
	// The campaigns released, each with its lots.
	Lines *List[ReleasedScheduleLine] `json:"lines"`
}

var sampleReleasedMachineName = "Knitter 3"

var SampleReleaseScheduleWeekResult = &ReleaseScheduleWeekResult{
	Object:            constants.ObjectTypeProductionScheduleWeekRelease,
	ProductionRun:     SampleProductionRun,
	WeekIndex:         0,
	WeekStartDate:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	ReleasedLineCount: 1,
	BatchCount:        6,
	TotalQuantity:     360,
	Lines: NewList([]ReleasedScheduleLine{
		{
			Line:            NewEntity(SampleProductionScheduleLineID, constants.ObjectTypeProductionScheduleLine, nil, nil),
			Item:            NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil),
			SKU:             "MZ-GREIGE-CREW",
			Machine:         NewEntity(SampleMachineID, constants.ObjectTypeMachine, &sampleReleasedMachineName, nil),
			PlannedQuantity: 360,
			LotUnits:        60,
			BatchCount:      6,
			Batches: NewList([]ReleaseScheduleBatch{
				{Item: NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil), SKU: "MZ-GREIGE-CREW", Quantity: 60},
			}, PageInfo{}),
		},
	}, PageInfo{}),
}

func (*ReleaseScheduleWeekResult) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReleaseScheduleWeekResult)
}

// What releasing a week would create, with nothing written.
//
// A release makes a numbered production run and every batch under it, which is real work to undo by hand, so the confirmation is driven by this rather than by a count computed in the browser.
type ReleaseScheduleWeekPreview struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_schedule_week_release_preview"`
	// Zero-based week offset from the start of the horizon.
	WeekIndex int32 `json:"week_index"`
	// First instant of the week.
	WeekStartDate time.Time `json:"week_starts_at" validate:"required"`
	// How many campaigns would be released.
	LineCount int32 `json:"line_count"`
	// How many batches would be created.
	BatchCount int32 `json:"batch_count"`
	// Total units that would be released.
	TotalQuantity float64 `json:"total_quantity"`
	// The campaigns that would be released, each with its lots.
	Lines *List[ReleasedScheduleLine] `json:"lines"`
	// Whether the week can be released.
	IsReleasable bool `json:"is_releasable"`
	// Why the week cannot be released.
	BlockedReason *string `json:"blocked_reason"`
	// The run the week was already released as.
	ExistingProductionRun *Entity `json:"existing_production_run"`
}

var SampleReleaseScheduleWeekPreview = &ReleaseScheduleWeekPreview{
	Object:        constants.ObjectTypeProductionScheduleWeekReleasePreview,
	WeekIndex:     0,
	WeekStartDate: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	LineCount:     1,
	BatchCount:    6,
	TotalQuantity: 360,
	IsReleasable:  true,
	Lines: NewList([]ReleasedScheduleLine{
		{
			Line:            NewEntity(SampleProductionScheduleLineID, constants.ObjectTypeProductionScheduleLine, nil, nil),
			Item:            NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil),
			SKU:             "MZ-GREIGE-CREW",
			Machine:         NewEntity(SampleMachineID, constants.ObjectTypeMachine, &sampleReleasedMachineName, nil),
			PlannedQuantity: 360,
			LotUnits:        60,
			BatchCount:      6,
			Batches: NewList([]ReleaseScheduleBatch{
				{Item: NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil), SKU: "MZ-GREIGE-CREW", Quantity: 60},
			}, PageInfo{}),
		},
	}, PageInfo{}),
}

func (*ReleaseScheduleWeekPreview) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReleaseScheduleWeekPreview)
}
