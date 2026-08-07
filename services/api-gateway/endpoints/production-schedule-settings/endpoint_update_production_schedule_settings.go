package productionschedulesettingsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to replace the account's planning assumptions.
type UpdateProductionScheduleSettingsRequest struct {
	// ID of the department that sets the pace of the factory, and the one campaigns are planned onto.
	//
	// Every machine in the department is planned, and the work of downstream departments is derived from what those machines are scheduled to run. Sending null, or leaving the field out of a request that otherwise replaces the settings, both leave the account with no constraint department — and generation is refused until one is chosen again.
	ConstraintDepartmentID field.Clearable[string] `json:"constraint_department_id,omitzero"`
	// How many weeks a generated plan covers.
	PlanningHorizonWeeks int32 `json:"planning_horizon_weeks" validate:"required,gte=1,lte=104"`
	// How many leading weeks of the horizon become a commitment when a version is published.
	//
	// Cannot be longer than the planning horizon. Once a version is published, changing a campaign inside the frozen window requires a reason and is recorded against the plan.
	FrozenWeeks int32 `json:"frozen_weeks" validate:"gte=0"`
	// Day a planning week starts, where 0 is Sunday.
	WeekStartDay int32 `json:"week_start_day" validate:"gte=0,lte=6"`
	// Months of production history the solver measures run rates, changeover behavior and lead times from.
	DemandWindowMonths int32 `json:"demand_window_months" validate:"required,gte=1"`
	// Months of order history the demand baseline is drawn from.
	ForecastHistoryMonths int32 `json:"forecast_history_months" validate:"required,gte=1"`
	// Months the forecast projects forward.
	//
	// Only applies to the `seasonal_ema` basis. A projection of anything other than twelve months is scaled to an annual rate, so the plan always reasons about a year of demand.
	ForecastMonths int32 `json:"forecast_months" validate:"required,gte=1"`
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
	// Changeover time is modelled as rising with the number of new inputs a product introduces, between the minimum and maximum below. The slope is calibrated from production history so the model reproduces this average across the transitions actually observed; set it to the changeover time the floor typically reports rather than to a worst case.
	ChangeoverAvgMinutes float64 `json:"changeover_avg_minutes" validate:"gte=0"`
	// Shortest plausible changeover, and the floor of the changeover model.
	//
	// Cannot exceed the maximum.
	ChangeoverMinMinutes float64 `json:"changeover_min_minutes" validate:"gte=0"`
	// Longest plausible changeover, and the ceiling of the changeover model.
	ChangeoverMaxMinutes float64 `json:"changeover_max_minutes" validate:"gte=0"`
	// Hourly labor rate charged to a changeover.
	//
	// This should be a dedicated technician rate rather than an allocated production rate, because one person works a single machine through a changeover. The constraint department's own labor rate takes precedence when it has one, leaving this as the fallback.
	ChangeoverLaborRate float64 `json:"changeover_labor_rate" validate:"gte=0"`
	// Annual cost of holding stock, as a share of item value.
	//
	// Weighed against the cost of a changeover when campaigns are sized: a higher rate favors shorter, more frequent runs.
	HoldingRatePct float64 `json:"holding_rate_pct" validate:"gte=0"`
	// Z-score for service level safety stock targets.
	ServiceLevelZ float64 `json:"service_level_z" validate:"gte=0"`
	// Weeks between coming off the constraint and being sellable.
	FinishLeadTimeWeeks float64 `json:"finish_lead_time_weeks" validate:"gte=0"`
	// Weeks of lead time to assume at the constraint for an item with no measured history.
	//
	// An item's own lead time, measured from production history, is used instead whenever one can be observed.
	DefaultConstraintLeadTimeWeeks float64 `json:"default_constraint_lead_time_weeks" validate:"gte=0"`
	// Ceiling on how far ahead any item is built.
	//
	// An item is only rebuilt once its projected stock falls below the lower of its reorder point and this many weeks of demand, so a slow mover whose statistical reorder point covers months of demand is not topped up ahead of items that are actually short.
	MaxWeeksSupply float64 `json:"max_weeks_supply" validate:"required,gt=0"`
	// How many steps down the production flow a constraint item is traced to the finished goods it becomes.
	//
	// Demand, stock and lot conventions are pooled onto the constraint item from every finished good the trace reaches, so anything further down the flow than this contributes nothing to the plan. The limit is also what stops a routing that loops back on itself from being traced forever.
	MaxFlowDepth int32 `json:"max_flow_depth" validate:"gte=1,lte=50"`
	// Shifts worked per day.
	ShiftsPerDay int32 `json:"shifts_per_day" validate:"required,gte=1"`
	// Hours in a shift.
	HoursPerShift float64 `json:"hours_per_shift" validate:"required,gt=0"`
	// Days worked per week.
	WorkDaysPerWeek int32 `json:"work_days_per_week" validate:"required,gte=1,lte=7"`
	// Weeks worked per year.
	WeeksPerYear int32 `json:"weeks_per_year" validate:"required,gte=1,lte=53"`
	// Share of machine time a plan may fill.
	//
	// Shifts, hours and work days give a machine's raw weekly hours; this trims them to what may actually be planned. The remainder absorbs changeovers, which are not scheduled as explicit blocks, so a value of 1 produces a plan that leaves no time to set anything up.
	CapacityHeadroomPct float64 `json:"capacity_headroom_pct" validate:"required,gt=0,lte=1"`
	// Units in a default production lot.
	//
	// The last resort in the lot-size chain: a lot set on the item, on its product line, or on the finished goods an intermediate item becomes all take precedence.
	DefaultLotUnits float64 `json:"default_lot_units" validate:"required,gt=0"`
	// Calendar days between an order being issued and it being due to ship.
	//
	// The last resort in the ship-by chain: a lead time set on the customer, or on the customer's account group, takes precedence. Zero commits the account to same-day shipping on every order that falls through to it, so this update replaces the whole settings object and omitting the field is not the same as leaving it alone.
	DefaultCustomerLeadTimeDays int32 `json:"default_customer_lead_time_days" validate:"gte=0,lte=3650"`
	// How a SKU is produced when neither it nor its product line says.
	//
	// - `make_to_stock`: built to the forecast, holding a safety stock against its variability.
	// - `make_to_order`: built only against orders already on the book, holding no buffer.
	DefaultFulfillmentPolicy constants.FulfillmentPolicy `json:"default_fulfillment_policy" validate:"required"`
	// Whether schedules are generated automatically on a recurring cadence.
	//
	// While active, each due tick queues a new schedule version.
	CadenceStatus constants.ActivationStatus `json:"cadence_status" validate:"required"`
	// Standard cron expression driving the generation cadence.
	//
	// Must be present and parse as a standard cron expression whenever the cadence is active, otherwise the whole update is rejected.
	GenerationCron field.Clearable[string] `json:"generation_cron,omitzero"`
	// Timezone the cadence is interpreted in.
	//
	// Decides when "every Wednesday at 6am" actually happens. A timezone the platform does not recognize falls back to UTC.
	GenerationTimezone string `json:"generation_timezone" validate:"required"`
	// Whether a version produced by the cadence is published automatically.
	//
	// While active, a cadence run publishes as soon as it solves, committing its frozen weeks without anyone reviewing the plan. Otherwise the run leaves a draft for a planner to publish by hand. Versions generated on request are never published automatically.
	AutoPublishStatus constants.ActivationStatus `json:"auto_publish_status" validate:"required"`
}

var sampleUpdateSettingsRequest = &UpdateProductionScheduleSettingsRequest{
	PlanningHorizonWeeks:        13,
	FrozenWeeks:                 1,
	WeekStartDay:                1,
	DemandWindowMonths:          12,
	ForecastHistoryMonths:       24,
	ForecastMonths:              12,
	DemandBasis:                 constants.ScheduleDemandBasisTrailing12,
	MaxWeeksSupply:              12,
	MaxFlowDepth:                10,
	ShiftsPerDay:                2,
	HoursPerShift:               7,
	WorkDaysPerWeek:             5,
	WeeksPerYear:                52,
	CapacityHeadroomPct:         0.9,
	DefaultLotUnits:             60,
	DefaultCustomerLeadTimeDays: 30,
	DefaultFulfillmentPolicy:    constants.FulfillmentPolicyMakeToStock,
	CadenceStatus:               constants.ActivationStatusInactive,
	GenerationTimezone:          "UTC",
	AutoPublishStatus:           constants.ActivationStatusInactive,
}

func (*UpdateProductionScheduleSettingsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSettingsRequest)
}

// Replaces the planning assumptions production schedules are solved against.
//
// Settings are replaced wholesale rather than patched, because they are read as one coherent set: a horizon that no longer matches the frozen window, or a capacity headroom that no longer matches the shift pattern, would produce a plan nobody intended. Send the full set on every call — a value the request leaves out is never carried over from what was stored.
//
// The set is validated together, so a frozen window longer than the horizon, a minimum changeover above the maximum, or an active cadence with no valid schedule expression is rejected as a whole.
//
// Existing schedule versions are unaffected — each one records the assumptions it was solved under, so changing settings changes future plans only.
type UpdateProductionScheduleSettingsEndpoint struct{}

func (e *UpdateProductionScheduleSettingsEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductionScheduleSettingsRequest, *apiresource.ProductionScheduleSettings] {
	return (&apiendpoint.APIEndpoint[*UpdateProductionScheduleSettingsRequest, *apiresource.ProductionScheduleSettings]{
		Title:             "Update Production Schedule Settings",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedule-settings",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionScheduleSettings,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductionScheduleSettingsRequest) (*apiresource.ProductionScheduleSettings, *apierror.APIError) {
			return svc.(ProductionScheduleSettingsSvc).UpdateSettings
		},
	})
}
