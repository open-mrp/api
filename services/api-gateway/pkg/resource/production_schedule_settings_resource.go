package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// The planning assumptions a production schedule is solved against.
//
// Every value here was a hardcoded constant in the scheduling script this feature replaced. The resource is always fully populated: an account that has never saved settings gets the solver's own defaults, so a caller never has to know which values would otherwise be assumed. `settings_status` says which of the two it is looking at.
type ProductionScheduleSettings struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_schedule_settings"`
	// The department that sets the pace of the factory. Every machine in it is planned, and every step downstream of it responds. A machine that must sit out is excluded through its resource setting rather than by leaving it unselected.
	ConstraintDepartment *Entity `json:"constraint_department"`

	// How many weeks a generated plan covers.
	PlanningHorizonWeeks int32 `json:"planning_horizon_weeks"`
	// How many leading weeks become a commitment when a version is published.
	FrozenWeeks int32 `json:"frozen_weeks"`
	// Day a planning week starts, where 0 is Sunday.
	WeekStartDay int32 `json:"week_start_day"`

	// Months of order history the demand baseline is drawn from.
	DemandWindowMonths int32 `json:"demand_window_months"`
	// Months of history the forecast is fitted to.
	ForecastHistoryMonths int32 `json:"forecast_history_months"`
	// Months the forecast projects forward.
	ForecastMonths int32 `json:"forecast_months"`
	// How demand is derived.
	DemandBasis constants.ScheduleDemandBasis `json:"demand_basis" validate:"required"`
	// Z-score applied to forecast variability.
	ForecastZ float64 `json:"forecast_z"`

	// Typical changeover duration, used to calibrate the changeover model.
	ChangeoverAvgMinutes float64 `json:"changeover_avg_minutes"`
	// Shortest plausible changeover.
	ChangeoverMinMinutes float64 `json:"changeover_min_minutes"`
	// Longest plausible changeover.
	ChangeoverMaxMinutes float64 `json:"changeover_max_minutes"`
	// Hourly labor rate charged to a changeover.
	ChangeoverLaborRate float64 `json:"changeover_labor_rate"`

	// Annual cost of holding stock, as a share of item value.
	HoldingRatePct float64 `json:"holding_rate_pct"`
	// Z-score for the service level safety stock targets.
	ServiceLevelZ float64 `json:"service_level_z"`
	// Weeks between finishing at the constraint and being sellable.
	FinishLeadTimeWeeks float64 `json:"finish_lead_time_weeks"`
	// Default weeks of lead time at the constraint when an item has no measurement.
	DefaultConstraintLeadTimeWeeks float64 `json:"default_constraint_lead_time_weeks"`
	// Ceiling on how far ahead any item is built.
	MaxWeeksSupply float64 `json:"max_weeks_supply"`
	// How many steps downstream department work is derived for.
	MaxFlowDepth int32 `json:"max_flow_depth"`

	// Shifts worked per day.
	ShiftsPerDay int32 `json:"shifts_per_day"`
	// Hours in a shift.
	HoursPerShift float64 `json:"hours_per_shift"`
	// Days worked per week.
	WorkDaysPerWeek int32 `json:"work_days_per_week"`
	// Weeks worked per year.
	WeeksPerYear int32 `json:"weeks_per_year"`
	// Share of machine time a plan may fill. The remainder absorbs changeovers, which are not scheduled as explicit blocks.
	CapacityHeadroomPct float64 `json:"capacity_headroom_pct"`
	// Units in a default production lot.
	DefaultLotUnits float64 `json:"default_lot_units"`

	// Whether schedules are generated on a timer.
	CadenceStatus constants.ActivationStatus `json:"cadence_status" validate:"required"`
	// Cron expression driving the generation cadence.
	GenerationCron *string `json:"generation_cron"`
	// Timezone the cadence is interpreted in.
	GenerationTimezone string `json:"generation_timezone" validate:"required"`
	// Whether a generated version is published automatically.
	AutoPublishStatus constants.ActivationStatus `json:"auto_publish_status" validate:"required"`
	// When the cadence last fired.
	LastGeneratedAt *time.Time `json:"last_generated_at"`

	// Whether these are the merchant's saved values or the solver's defaults.
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

const SampleProductionScheduleResourceSettingID = "pnscrrsd_0192e8a6bf7c8d2e"

// A per-resource override of the account's planning assumptions.
//
// This is where a machine is marked as the planning constraint, and where a department or step declares how many weeks after the constraint its work actually starts.
type ProductionScheduleResourceSetting struct {
	// Resource setting ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_schedule_resource_setting"`
	// What kind of resource this overrides.
	ScopeType constants.ScheduleResourceScope `json:"scope_type" validate:"required"`
	// The machine, department or production step this overrides.
	Scope *Entity `json:"scope" validate:"required"`
	// Whether this resource takes part in planning. Machines are selected by department, so this takes one out rather than opting one in — for a machine down for a rebuild that should not be planned against.
	ParticipationStatus constants.ParticipationStatus `json:"participation_status" validate:"required"`
	// Weeks of lead time at this resource.
	LeadTimeWeeks *float64 `json:"lead_time_weeks"`
	// Weeks after the constraint campaign this resource's work starts.
	LeadTimeOffsetWeeks float64 `json:"lead_time_offset_weeks"`
}

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
