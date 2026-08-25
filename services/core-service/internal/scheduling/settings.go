package scheduling

// Settings are the planning assumptions, already resolved to effective values by the caller. Every field was a hardcoded constant in the script; the mapping is noted so a parity failure can be traced back to a specific line.
type Settings struct {
	// Horizon
	HorizonWeeks int `json:"horizon_weeks"` // HORIZON_WEEKS = 13
	FrozenWeeks  int `json:"frozen_weeks"`
	// WeekStartDay is the weekday a horizon week begins on, 0 = Sunday through 6 = Saturday.
	WeekStartDay int `json:"week_start_day"`

	// Capacity
	ShiftsPerDay        int     `json:"shifts_per_day"`        // SHIFTS = 2
	HoursPerShift       float64 `json:"hours_per_shift"`       // HOURS_PER_SHIFT = 7
	WorkDaysPerWeek     int     `json:"work_days_per_week"`    // WORK_DAYS_WK = 5
	WeeksPerYear        int     `json:"weeks_per_year"`        // WEEKS_YR = 52
	CapacityHeadroomPct float64 `json:"capacity_headroom_pct"` // the 0.9 headroom reserved for changeover
	DefaultLotUnits     float64 `json:"default_lot_units"`     // AVG_DOFF = 60

	// Changeover
	ChangeoverAvgMinutes float64 `json:"changeover_avg_minutes"` // CO_AVG_MIN = 30
	ChangeoverMinMinutes float64 `json:"changeover_min_minutes"` // CO_MIN_MIN = 15
	ChangeoverMaxMinutes float64 `json:"changeover_max_minutes"` // CO_MAX_MIN = 90
	ChangeoverLaborRate  float64 `json:"changeover_labor_rate"`  // CHANGEOVER_LABOR_RATE = 20

	// Inventory policy
	HoldingRatePct                 float64 `json:"holding_rate_pct"`                   // HOLDING_RATE_PCT = 0.25
	ServiceLevelZ                  float64 `json:"service_level_z"`                    // SERVICE_Z = 1.645
	FinishLeadTimeWeeks            float64 `json:"finish_lead_time_weeks"`             // FINISH_LT_WEEKS = 6
	DefaultConstraintLeadTimeWeeks float64 `json:"default_constraint_lead_time_weeks"` // KNIT_LT_WEEKS_DEFAULT = 1.3
	MaxWeeksSupply                 float64 `json:"max_weeks_supply"`                   // MAX_WEEKS_SUPPLY = 12
	MaxFlowDepth                   int     `json:"max_flow_depth"`                     // MAX_FLOW_DEPTH = 10

	// GreigeBufferEnabled makes stage one hold a physical greige buffer at the constraint rather than letting the pooled safety stock live as finished goods. With it off, a campaign is triggered only when the echelon position (greige plus every finished good downstream of it) falls below the reorder point, so a family sitting on plenty of finished stock in the wrong SKUs is read as fully covered and no greige is knit — which leaves stage two with nothing to rebalance the mix from. With it on, a second trigger fires whenever projected greige-on-hand alone falls below its own safety stock, so the constraint keeps the undifferentiated buffer the postponement model assumes. Off by default so the parity gate reproduces the script, which has no such buffer; the service turns it on.
	GreigeBufferEnabled bool `json:"greige_buffer_enabled"`

	// Fulfillment commitments
	//
	// DefaultCustomerLeadTimeDays is the last fallback in the ship-by chain, behind the customer and its account group. It lives with the planning assumptions because it is what a make-to-order promise is measured against: the date it produces has to be the same date the plan is solved to.
	DefaultCustomerLeadTimeDays int `json:"default_customer_lead_time_days"`
}

// DefaultSettings mirrors the script's constants, with one deliberate exception: there is no growth multiplier. The script's GROWTH_MULT defaulted to 2, silently doubling all demand; that intent is now expressed as a demand override, which carries a reason and an author.
func DefaultSettings() Settings {
	return Settings{
		HorizonWeeks:                   13,
		FrozenWeeks:                    1,
		WeekStartDay:                   1,
		ShiftsPerDay:                   2,
		HoursPerShift:                  7,
		WorkDaysPerWeek:                5,
		WeeksPerYear:                   52,
		CapacityHeadroomPct:            0.9,
		DefaultLotUnits:                60,
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
		DefaultCustomerLeadTimeDays:    30,
	}
}

// MaxFlowDepthOrDefault bounds the genealogy walk so a rework cycle cannot loop forever.
func (s Settings) MaxFlowDepthOrDefault() int {
	if s.MaxFlowDepth > 0 {
		return s.MaxFlowDepth
	}
	return 10
}

// MachineWeeklyCapacityHours is the hours one machine can be planned for in a week.
//
// The headroom factor is not slack: changeovers are not scheduled as explicit blocks, so the reserve is what stops a plan that is 100% run time and therefore impossible. Script: SHIFTS * HOURS_PER_SHIFT * WORK_DAYS_WK * 0.9.
func (s Settings) MachineWeeklyCapacityHours() float64 {
	return float64(s.ShiftsPerDay) * s.HoursPerShift * float64(s.WorkDaysPerWeek) * s.CapacityHeadroomPct
}
