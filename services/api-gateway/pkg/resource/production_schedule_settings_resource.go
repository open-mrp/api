package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// The planning assumptions a production schedule is solved against.
//
// The whole set is always returned. An account that has never saved settings reads back the values the solver would apply anyway, so a caller never has to know which assumptions are in play; `settings_status` says whether the values were saved on the account or are those defaults.
type ProductionScheduleSettings struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_schedule_settings"`
	// The department that sets the pace of the factory, and the one campaigns are planned onto.
	//
	// Every machine in the department is planned, and the work of downstream departments is derived from what those machines are scheduled to run. A machine that must sit out is taken out through its own resource setting rather than by leaving it unselected. Generation is refused until a constraint department is chosen.
	ConstraintDepartment *Entity `json:"constraint_department"`

	// How many weeks a generated plan covers.
	PlanningHorizonWeeks int32 `json:"planning_horizon_weeks"`
	// How many leading weeks of the horizon become a commitment when a version is published.
	//
	// Nothing is frozen while a version is still a draft. Once published, changing a campaign inside the frozen window requires a reason and is recorded against the plan. Cannot be longer than the planning horizon.
	FrozenWeeks int32 `json:"frozen_weeks"`
	// Day a planning week starts, where 0 is Sunday.
	WeekStartDay int32 `json:"week_start_day"`

	// Months of production history the solver measures run rates, changeover behavior and lead times from.
	DemandWindowMonths int32 `json:"demand_window_months"`
	// Months of order history the demand baseline is drawn from.
	ForecastHistoryMonths int32 `json:"forecast_history_months"`
	// Months the forecast projects forward.
	//
	// Only applies to the `seasonal_ema` basis. A projection of anything other than twelve months is scaled to an annual rate, so the plan always reasons about a year of demand.
	ForecastMonths int32 `json:"forecast_months"`
	// How the demand a plan is solved against is derived from history.
	//
	// - `trailing_12`: the last twelve complete months of orders, spread evenly across the coming year.
	// - `seasonal_ema`: a seasonally adjusted, exponentially smoothed projection that weights recent months more heavily. Falls back to the trailing baseline for an item with no history.
	//
	// Demand overrides are applied on top of whichever baseline is chosen.
	DemandBasis constants.ScheduleDemandBasis `json:"demand_basis" validate:"required"`
	// Z-score used for the confidence interval around the seasonal demand forecast.
	//
	// The plan is solved against the central forecast, so this widens or narrows that interval without changing what gets scheduled.
	ForecastZ float64 `json:"forecast_z"`

	// Typical changeover duration.
	//
	// Changeover time is modelled as rising with the number of new inputs a product introduces, between the minimum and maximum below. The slope is calibrated from production history so the model reproduces this average across the transitions actually observed, which is why the value belongs at the changeover time the floor typically reports rather than at a worst case.
	ChangeoverAvgMinutes float64 `json:"changeover_avg_minutes"`
	// Shortest plausible changeover, and the floor of the changeover model.
	ChangeoverMinMinutes float64 `json:"changeover_min_minutes"`
	// Longest plausible changeover, and the ceiling of the changeover model.
	ChangeoverMaxMinutes float64 `json:"changeover_max_minutes"`
	// Hourly labor rate charged to a changeover.
	//
	// This is a dedicated technician rate rather than an allocated production rate, because one person works a single machine through a changeover. Together with the typical changeover duration it prices the setup cost that decides economic campaign sizes. The constraint department's own labor rate takes precedence when it has one, leaving this as the fallback.
	ChangeoverLaborRate float64 `json:"changeover_labor_rate"`

	// Annual cost of holding stock, as a share of item value.
	//
	// Weighed against the cost of a changeover when campaigns are sized: a higher rate favors shorter, more frequent runs.
	HoldingRatePct float64 `json:"holding_rate_pct"`
	// Z-score behind the safety stock targets.
	//
	// A higher value buys more cover against demand variability at both the constraint and the finished goods stage, at the cost of carrying more stock.
	ServiceLevelZ float64 `json:"service_level_z"`
	// Weeks between coming off the constraint and being sellable.
	//
	// Added to the constraint's own lead time when reorder points are set, so a plan replenishes early enough for a decision made today to become sellable stock.
	FinishLeadTimeWeeks float64 `json:"finish_lead_time_weeks"`
	// Weeks of lead time to assume at the constraint for an item with no measured history.
	//
	// An item's own lead time, measured from production history, is used instead whenever one can be observed.
	DefaultConstraintLeadTimeWeeks float64 `json:"default_constraint_lead_time_weeks"`
	// Ceiling on how far ahead any item is built.
	//
	// An item is only rebuilt once its projected stock falls below the lower of its reorder point and this many weeks of demand, so a slow mover whose statistical reorder point covers months of demand is not topped up ahead of items that are actually short.
	MaxWeeksSupply float64 `json:"max_weeks_supply"`
	// How many steps down the production flow a constraint item is traced to the finished goods it becomes.
	//
	// Demand, stock and lot conventions are pooled onto the constraint item from every finished good the trace reaches, so anything further down the flow than this contributes nothing to the plan. The limit is also what stops a routing that loops back on itself from being traced forever.
	MaxFlowDepth int32 `json:"max_flow_depth"`

	// Shifts worked per day.
	ShiftsPerDay int32 `json:"shifts_per_day"`
	// Hours in a shift.
	HoursPerShift float64 `json:"hours_per_shift"`
	// Days worked per week.
	WorkDaysPerWeek int32 `json:"work_days_per_week"`
	// Weeks worked per year.
	WeeksPerYear int32 `json:"weeks_per_year"`
	// Share of machine time a plan may fill.
	//
	// Shifts, hours and work days give a machine's raw weekly hours; this trims them to what may actually be planned. The remainder absorbs changeovers, which are not scheduled as explicit blocks, so a value of 1 produces a plan that leaves no time to set anything up.
	CapacityHeadroomPct float64 `json:"capacity_headroom_pct"`
	// Units in a default production lot.
	//
	// The last resort in the lot-size chain: a lot set on the item, on its product line, or on the finished goods an intermediate item becomes all take precedence.
	DefaultLotUnits float64 `json:"default_lot_units"`

	// Calendar days between an order being issued and it being due to ship.
	//
	// The last resort in the ship-by chain: a lead time set on the customer, or on the customer's account group, takes precedence. Zero means same-day shipping.
	DefaultCustomerLeadTimeDays int32 `json:"default_customer_lead_time_days"`
	// The account-wide operating calendars: the days the plant tenders freight, and the days a customer's dock accepts it.
	//
	// Behind the per-address and per-customer links and ahead of a plain Monday-to-Friday week. Null on both means every ship-by date is resolved against weekdays alone.
	ShipCalendarID    *string `json:"ship_calendar_id"`
	ReceiveCalendarID *string `json:"receive_calendar_id"`
	// How a SKU is produced when neither it nor its product line says.
	//
	// - `make_to_stock`: built to the forecast, holding a safety stock against its variability.
	// - `make_to_order`: built only against orders already on the book, holding no buffer.
	DefaultFulfillmentPolicy constants.FulfillmentPolicy `json:"default_fulfillment_policy" validate:"required"`

	// Whether schedules are generated automatically on a recurring cadence.
	//
	// While active, each due tick queues a new schedule version; a generation cron expression is required for the cadence to be saved.
	CadenceStatus constants.ActivationStatus `json:"cadence_status" validate:"required"`
	// Standard cron expression driving the generation cadence.
	GenerationCron *string `json:"generation_cron"`
	// Timezone the cadence is interpreted in.
	//
	// Decides when "every Wednesday at 6am" actually happens. A timezone the platform does not recognize falls back to UTC.
	GenerationTimezone string `json:"generation_timezone" validate:"required"`
	// Whether a version produced by the cadence is published automatically.
	//
	// While active, a cadence run publishes as soon as it solves, committing its frozen weeks without anyone reviewing the plan. Otherwise the run leaves a draft for a planner to publish by hand. Versions generated on request are never published automatically.
	AutoPublishStatus constants.ActivationStatus `json:"auto_publish_status" validate:"required"`
	// When the cadence last fired.
	//
	// Stamped when a run is queued rather than when the plan finishes solving, and the next due time is measured from it.
	LastGeneratedAt *time.Time `json:"last_generated_at"`

	// Whether the values returned were saved on the account or are the defaults applied when nothing has been saved.
	SettingsStatus constants.SettingsStatus `json:"settings_status" validate:"required"`

	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleGenerationCron = "0 6 * * 1"

var SampleProductionScheduleSettings = &ProductionScheduleSettings{
	Object:                         constants.ObjectTypeProductionScheduleSettings,
	ConstraintDepartment:           NewEntity(SampleDepartmentID, constants.ObjectTypeDepartment, new(SampleDepartmentName), nil),
	PlanningHorizonWeeks:           13,
	FrozenWeeks:                    1,
	WeekStartDay:                   1,
	DemandWindowMonths:             12,
	ForecastHistoryMonths:          24,
	ForecastMonths:                 12,
	DemandBasis:                    constants.ScheduleDemandBasisTrailing12,
	ForecastZ:                      1.645,
	ChangeoverAvgMinutes:           30,
	ChangeoverMinMinutes:           15,
	ChangeoverMaxMinutes:           90,
	ChangeoverLaborRate:            20,
	HoldingRatePct:                 0.25,
	ServiceLevelZ:                  1.645,
	FinishLeadTimeWeeks:            6,
	DefaultConstraintLeadTimeWeeks: 1.3,
	MaxWeeksSupply:                 12,
	MaxFlowDepth:                   10,
	ShiftsPerDay:                   2,
	HoursPerShift:                  7,
	WorkDaysPerWeek:                5,
	WeeksPerYear:                   52,
	CapacityHeadroomPct:            0.9,
	DefaultLotUnits:                60,
	DefaultCustomerLeadTimeDays:    30,
	DefaultFulfillmentPolicy:       constants.FulfillmentPolicyMakeToStock,
	CadenceStatus:                  constants.ActivationStatusActive,
	GenerationCron:                 &sampleGenerationCron,
	GenerationTimezone:             "America/New_York",
	AutoPublishStatus:              constants.ActivationStatusInactive,
	LastGeneratedAt:                timeutil.TimestampToTimePtr(sampleCreatedAtTimestamp),
	SettingsStatus:                 constants.SettingsStatusStored,
	CreatedAt:                      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:                      timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductionScheduleSettings) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionScheduleSettings)
}

const SampleProductionScheduleResourceSettingID = "pnscrrsd_hegthjeksw87"
const SampleProductionScheduleItemSettingID = "pnscitsd_kw83nzq4mrt6"

// A planning override for one machine, department or production step.
//
// The account's settings apply to every resource; an override changes how one of them is treated — taking a machine out of the plan, or declaring how many weeks a downstream step's work starts after the step that feeds it. A resource has at most one override, and a resource without one is planned on the account settings alone.
type ProductionScheduleResourceSetting struct {
	// Resource setting ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_schedule_resource_setting"`
	// What kind of resource this override applies to.
	ScopeType constants.ScheduleResourceScope `json:"scope_type" validate:"required"`
	// The machine, department or production step this overrides.
	Scope *Entity `json:"scope" validate:"required"`
	// Whether this resource takes part in planning.
	//
	// Machines are chosen by naming the constraint department, so an override is how one is taken out — a machine down for a rebuild — rather than how one is opted in. A machine with no override is planned.
	ParticipationStatus constants.ParticipationStatus `json:"participation_status" validate:"required"`
	// Weeks of lead time at this resource.
	LeadTimeWeeks *float64 `json:"lead_time_weeks"`
	// How many weeks after the step feeding it this resource's work starts.
	//
	// Read when downstream department work is derived from the constraint plan, so it is the production-step override that shifts a plan: without an offset every step lands in the same week as the step feeding it, and the offsets along a chain of steps add up. A schedule is planned in whole weeks, so a fractional offset is truncated.
	LeadTimeOffsetWeeks float64 `json:"lead_time_offset_weeks"`
}

// Planning overrides for one item, on top of the account-wide assumptions.
type ProductionScheduleItemSetting struct {
	// Item setting ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_schedule_item_setting"`
	// The item these overrides apply to.
	Item *Entity `json:"item" validate:"required"`
	// Whether this item takes part in planning.
	//
	// An excluded item is left out of the plan entirely: no campaigns, no policy, no capacity.
	ParticipationStatus constants.ParticipationStatus `json:"participation_status" validate:"required"`
	// Units in one production lot for this item, overriding the lot its product line would supply.
	LotMultipleUnits *float64 `json:"lot_multiple_units"`
	// How this item is produced.
	//
	// - `make_to_stock`: built to the forecast, holding a safety stock against its variability.
	// - `make_to_order`: built only against orders already on the book, holding no buffer.
	//
	// Null inherits from the item's product line, then from the account default.
	FulfillmentPolicy *constants.FulfillmentPolicy `json:"fulfillment_policy"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// The engine's advice on how one SKU should be produced, with the measurements behind it.
type FulfillmentRecommendation struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=fulfillment_recommendation"`
	// The item the advice is about.
	Item *Entity `json:"item" validate:"required"`
	// SKU of that item.
	SKU string `json:"sku"`
	// The product line it sells under, when it sells under one.
	ProductLine *Entity `json:"product_line"`
	// How the item is planned today.
	CurrentPolicy constants.FulfillmentPolicy `json:"current_policy" validate:"required"`
	// How the engine thinks it should be planned.
	RecommendedPolicy constants.FulfillmentPolicy `json:"recommended_policy" validate:"required"`
	// Whether adopting the recommendation would change anything.
	Changes bool `json:"changes"`
	// The rule that decided.
	//
	// - `lead_time_infeasible`: customers are promised less time than production needs, so the stock has to exist before the order does. Checked first, because producing to order is not possible rather than not preferred.
	// - `no_recent_demand`: nothing has sold for long enough that a buffer is dead stock.
	// - `single_customer`: effectively one customer buys it, and that customer is served to order.
	// - `lumpy_demand`: demand arrives rarely and in wildly different sizes, which is the shape a safety stock sizes worst.
	// - `slow_moving_high_value`: expensive units, few sold — the buffer costs more than the service it buys.
	// - `steady_demand`: regular enough to forecast, which is what stocking is for.
	Reason constants.FulfillmentRecommendationReason `json:"reason" validate:"required"`
	// Months observed divided by months with demand: 1 means it sells every month, 3 means once a quarter on average.
	//
	// Measured on monthly buckets, which cannot distinguish two orders in one month from one.
	AverageDemandInterval float64 `json:"average_demand_interval"`
	// Squared coefficient of variation over the months that had demand, measuring how uneven the quantities are.
	CoefficientOfVariation float64 `json:"coefficient_of_variation"`
	// The largest customer's share of this item's demand, as a percentage.
	TopCustomerSharePct float64 `json:"top_customer_share_pct"`
	// Name of that customer.
	TopCustomerName *string `json:"top_customer_name"`
	// Calendar days customers are promised on average, weighted by how much each buys.
	DemandWeightedLeadTimeDays float64 `json:"demand_weighted_lead_time_days"`
	// Annual cost of goods for this item: demand times unit cost.
	AnnualCOGS float64 `json:"annual_cogs"`
	// Months since anything last sold, capped at the observation window.
	MonthsSinceLastSale int32 `json:"months_since_last_sale"`
	// Percentage of demand from customers whose own stated policy disagrees with the recommendation.
	//
	// A policy is resolved per SKU, so an item sold to both a stocking distributor and a contract customer gets one answer either way. A high share here is the signal that the single answer is uncomfortable.
	MixedStreamSharePct float64 `json:"mixed_stream_share_pct"`
}

var sampleTopCustomerName = "Contract Co"

var SampleFulfillmentRecommendation = &FulfillmentRecommendation{
	Object:                     constants.ObjectTypeFulfillmentRecommendation,
	Item:                       NewEntity(SampleItemID, constants.ObjectTypeItem, nil, &sampleScheduleSettingsSKU),
	SKU:                        sampleScheduleSettingsSKU,
	ProductLine:                NewEntity(SampleProductLineID, constants.ObjectTypeProductLine, nil, nil),
	CurrentPolicy:              constants.FulfillmentPolicyMakeToStock,
	RecommendedPolicy:          constants.FulfillmentPolicyMakeToOrder,
	Changes:                    true,
	Reason:                     constants.FulfillmentRecommendationReasonLumpyDemand,
	AverageDemandInterval:      3,
	CoefficientOfVariation:     1.8,
	TopCustomerSharePct:        95,
	TopCustomerName:            &sampleTopCustomerName,
	DemandWeightedLeadTimeDays: 90,
	AnnualCOGS:                 4000,
	MonthsSinceLastSale:        1,
	MixedStreamSharePct:        95,
}

func (*FulfillmentRecommendation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleFulfillmentRecommendation)
}

var sampleItemSettingLotMultiple = 60.0
var sampleItemSettingPolicy = constants.FulfillmentPolicyMakeToOrder

var SampleProductionScheduleItemSetting = &ProductionScheduleItemSetting{
	ID:                  SampleProductionScheduleItemSettingID,
	Object:              constants.ObjectTypeProductionScheduleItemSetting,
	Item:                NewEntity(SampleItemID, constants.ObjectTypeItem, nil, &sampleScheduleSettingsSKU),
	ParticipationStatus: constants.ParticipationStatusIncluded,
	LotMultipleUnits:    &sampleItemSettingLotMultiple,
	FulfillmentPolicy:   &sampleItemSettingPolicy,
	CreatedAt:           timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:           timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductionScheduleItemSetting) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionScheduleItemSetting)
}

var sampleScheduleSettingsSKU = "LKN-001"

var SampleProductionScheduleResourceSetting = &ProductionScheduleResourceSetting{
	ID:                  SampleProductionScheduleResourceSettingID,
	Object:              constants.ObjectTypeProductionScheduleResourceSetting,
	ScopeType:           constants.ScheduleResourceScopeDepartment,
	Scope:               NewEntity(SampleDepartmentID, constants.ObjectTypeDepartment, new(SampleDepartmentName), nil),
	ParticipationStatus: constants.ParticipationStatusIncluded,
	LeadTimeWeeks:       new(0.5),
	LeadTimeOffsetWeeks: 1,
}

func (*ProductionScheduleResourceSetting) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionScheduleResourceSetting)
}
