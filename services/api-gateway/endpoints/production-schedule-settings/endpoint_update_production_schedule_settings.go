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
	// ID of the department that sets the pace of the factory. Every machine in it is planned, and every step downstream responds.
	ConstraintDepartmentID field.Clearable[string] `json:"constraint_department_id,omitzero"`
	// How many weeks a generated plan covers.
	PlanningHorizonWeeks int32 `json:"planning_horizon_weeks" validate:"required,gte=1,lte=104"`
	// How many leading weeks become a commitment when a version is published.
	FrozenWeeks int32 `json:"frozen_weeks" validate:"gte=0"`
	// Day a planning week starts, where 0 is Sunday.
	WeekStartDay int32 `json:"week_start_day" validate:"gte=0,lte=6"`
	// Months of order history the demand baseline is drawn from.
	DemandWindowMonths int32 `json:"demand_window_months" validate:"required,gte=1"`
	// Months of history the forecast is fitted to.
	ForecastHistoryMonths int32 `json:"forecast_history_months" validate:"required,gte=1"`
	// Months the forecast projects forward.
	ForecastMonths int32 `json:"forecast_months" validate:"required,gte=1"`
	// How demand is derived.
	DemandBasis constants.ScheduleDemandBasis `json:"demand_basis" validate:"required"`
	// Z-score applied to forecast variability.
	ForecastZ float64 `json:"forecast_z"`
	// Typical changeover duration.
	ChangeoverAvgMinutes float64 `json:"changeover_avg_minutes" validate:"gte=0"`
	// Shortest plausible changeover.
	ChangeoverMinMinutes float64 `json:"changeover_min_minutes" validate:"gte=0"`
	// Longest plausible changeover.
	ChangeoverMaxMinutes float64 `json:"changeover_max_minutes" validate:"gte=0"`
	// Hourly labour rate charged to a changeover.
	ChangeoverLaborRate float64 `json:"changeover_labor_rate" validate:"gte=0"`
	// Annual cost of holding stock, as a share of item value.
	HoldingRatePct float64 `json:"holding_rate_pct" validate:"gte=0"`
	// Z-score for service level safety stock targets.
	ServiceLevelZ float64 `json:"service_level_z" validate:"gte=0"`
	// Weeks between finishing at the constraint and being sellable.
	FinishLeadTimeWeeks float64 `json:"finish_lead_time_weeks" validate:"gte=0"`
	// Default weeks of lead time at the constraint.
	DefaultConstraintLeadTimeWeeks float64 `json:"default_constraint_lead_time_weeks" validate:"gte=0"`
	// Ceiling on how far ahead any item is built.
	MaxWeeksSupply float64 `json:"max_weeks_supply" validate:"required,gt=0"`
	// How many steps downstream department work is derived for.
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
	CapacityHeadroomPct float64 `json:"capacity_headroom_pct" validate:"required,gt=0,lte=1"`
	// Units in a default production lot.
	DefaultLotUnits float64 `json:"default_lot_units" validate:"required,gt=0"`
	// Whether schedules are generated on a timer.
	CadenceStatus constants.ActivationStatus `json:"cadence_status" validate:"required"`
	// Cron expression driving the generation cadence.
	GenerationCron field.Clearable[string] `json:"generation_cron,omitzero"`
	// Timezone the cadence is interpreted in.
	GenerationTimezone string `json:"generation_timezone" validate:"required"`
	// Whether a generated version is published automatically.
	AutoPublishStatus constants.ActivationStatus `json:"auto_publish_status" validate:"required"`
}

var sampleUpdateSettingsRequest = &UpdateProductionScheduleSettingsRequest{
	PlanningHorizonWeeks:  13,
	FrozenWeeks:           1,
	WeekStartDay:          1,
	DemandWindowMonths:    12,
	ForecastHistoryMonths: 24,
	ForecastMonths:        12,
	DemandBasis:           constants.ScheduleDemandBasisTrailing12,
	MaxWeeksSupply:        12,
	MaxFlowDepth:          10,
	ShiftsPerDay:          2,
	HoursPerShift:         7,
	WorkDaysPerWeek:       5,
	WeeksPerYear:          52,
	CapacityHeadroomPct:   0.9,
	DefaultLotUnits:       60,
	CadenceStatus:         constants.ActivationStatusInactive,
	GenerationTimezone:    "UTC",
	AutoPublishStatus:     constants.ActivationStatusInactive,
}

func (*UpdateProductionScheduleSettingsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSettingsRequest)
}

// Replaces the planning assumptions production schedules are solved against.
//
// Settings are replaced wholesale rather than patched, because they are read as one coherent set: a horizon that no longer matches the frozen window, or a capacity headroom that no longer matches the shift pattern, would produce a plan nobody intended.
//
// Existing schedule versions are unaffected — each one snapshots the assumptions it was solved under, so changing settings changes future plans only.
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
