package domain

import "time"

// ProductionScheduleSettings are the merchant-editable planning assumptions.
//
// Every value here was a hardcoded constant in the original knit-scheduling script. The resource is always returned fully populated: an account that has never saved settings gets code defaults rather than nulls, so a caller never has to know which defaults the solver would have applied.
type ProductionScheduleSettings struct {
	AccountID string

	ConstraintDepartmentID *string `audit:"constraint_department_id"`

	PlanningHorizonWeeks int32 `audit:"planning_horizon_weeks"`
	FrozenWeeks          int32 `audit:"frozen_weeks"`
	WeekStartDay         int32 `audit:"week_start_day"`

	DemandWindowMonths    int32   `audit:"demand_window_months"`
	ForecastHistoryMonths int32   `audit:"forecast_history_months"`
	ForecastMonths        int32   `audit:"forecast_months"`
	DemandBasisCode       string  `audit:"demand_basis_code"`
	ForecastZ             float64 `audit:"forecast_z"`

	ChangeoverAvgMinutes float64 `audit:"changeover_avg_minutes"`
	ChangeoverMinMinutes float64 `audit:"changeover_min_minutes"`
	ChangeoverMaxMinutes float64 `audit:"changeover_max_minutes"`
	ChangeoverLaborRate  float64 `audit:"changeover_labor_rate"`

	HoldingRatePct                 float64 `audit:"holding_rate_pct"`
	ServiceLevelZ                  float64 `audit:"service_level_z"`
	FinishLeadTimeWeeks            float64 `audit:"finish_lead_time_weeks"`
	DefaultConstraintLeadTimeWeeks float64 `audit:"default_constraint_lead_time_weeks"`
	MaxWeeksSupply                 float64 `audit:"max_weeks_supply"`
	MaxFlowDepth                   int32   `audit:"max_flow_depth"`

	ShiftsPerDay        int32   `audit:"shifts_per_day"`
	HoursPerShift       float64 `audit:"hours_per_shift"`
	WorkDaysPerWeek     int32   `audit:"work_days_per_week"`
	WeeksPerYear        int32   `audit:"weeks_per_year"`
	CapacityHeadroomPct float64 `audit:"capacity_headroom_pct"`
	DefaultLotUnits     float64 `audit:"default_lot_units"`

	// DefaultCustomerLeadTimeDays is the last fallback in an order's ship-by chain, behind the customer and its account group.
	DefaultCustomerLeadTimeDays int32 `audit:"default_customer_lead_time_days"`
	// ShipCalendarID and ReceiveCalendarID are the account-wide fallbacks behind the per-customer and per-address links, and the last stop before Monday to Friday. They sit with the planning assumptions because they answer the same question the lead time does — when can this order actually leave.
	ShipCalendarID    *string `audit:"ship_calendar_id"`
	ReceiveCalendarID *string `audit:"receive_calendar_id"`

	// DefaultFulfillmentPolicyCode is how a SKU is produced when neither it nor its product line says.
	DefaultFulfillmentPolicyCode string `audit:"default_fulfillment_policy_code"`

	IsEnabled          bool    `audit:"is_enabled"`
	GenerationCron     *string `audit:"generation_cron"`
	GenerationTimezone string  `audit:"generation_timezone"`
	AutoPublish        bool    `audit:"auto_publish"`
	LastGeneratedAt    *time.Time

	// HasStoredSettings is false when the values are code defaults rather than something the merchant chose.
	HasStoredSettings bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

type UpdateProductionScheduleSettingsParams struct {
	AccountID string
	Settings  ProductionScheduleSettings
}

// ProductionScheduleResourceSetting overrides planning behavior for one machine, department or production step.
type ProductionScheduleResourceSetting struct {
	ID                  string
	AccountID           string
	ScopeCode           string
	ScopeRefID          string
	IsExcluded          bool
	LeadTimeWeeks       *float64
	LeadTimeOffsetWeeks float64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type UpsertResourceSettingParams struct {
	AccountID           string
	ScopeCode           string
	ScopeRefID          string
	IsExcluded          bool
	LeadTimeWeeks       *float64
	LeadTimeOffsetWeeks float64
}

// ProductionScheduleItemPlanningSetting is one item's planning override as the API serves it.
//
// Distinct from ProductionScheduleItemSetting, which is the solver's narrower view of the same row: the solve needs the values, the API needs the identity and timestamps too.
type ProductionScheduleItemPlanningSetting struct {
	ID        string
	AccountID string
	ItemID    string
	SKU       string

	IsExcluded bool
	// LotMultipleUnits overrides the lot this item is made in; nil leaves the lot chain alone.
	LotMultipleUnits *float64
	// FulfillmentPolicyCode overrides how this item is produced; nil falls through to its product line, then the account default.
	FulfillmentPolicyCode *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type UpsertItemSettingParams struct {
	AccountID             string
	ItemID                string
	IsExcluded            bool
	LotMultipleUnits      *float64
	FulfillmentPolicyCode *string
}
