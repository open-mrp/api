package domain

import "time"

type AnalyzeScheduleAttainmentParams struct {
	AccountID     string
	StartDate     time.Time
	EndDate       time.Time
	GroupBy       string
	MachineIDs    []string
	DepartmentIDs []string
}

// AttainmentBucket is one row of the breakdown.
//
// Both ratios are reported because they answer different questions and either alone is misleading: attainment caps every SKU at what was asked for, so over-building one easy item cannot paper over a total miss on another; output ratio does not cap, so it is the only one that shows over-production.
type AttainmentBucket struct {
	// Key identifies the bucket within the chosen grouping — an ISO week start, a machine id, a department id or an item id.
	Key   string
	Label string

	WeekStartDate *time.Time

	PlannedQuantity float64
	ActualQuantity  float64
	// MatchedQuantity is SUM(LEAST(actual, planned)) — the attainment numerator.
	MatchedQuantity float64
	WasteQuantity   float64
	// UnplannedQuantity is production with no matching planned line. It is surfaced rather than discarded: it is the schedule-breaker number.
	UnplannedQuantity float64

	PlannedRunHours float64
	PlannedLines    int64
	BatchCount      int64

	// Nil when the denominator is zero — a week nobody planned has no attainment, which is not the same as 0%.
	AttainmentPct  *float64
	OutputRatioPct *float64
}

// FrozenAdherence measures how well a published commitment survived contact with the week it covered.
type FrozenAdherence struct {
	ScheduleID      string
	Version         int32
	FrozenLineCount int64
	// FrozenPlannedQuantity is the quantity captured at publish, never recomputed, so the denominator cannot drift as lines are added later.
	FrozenPlannedQuantity float64

	DeviatedLines int64
	AddedLines    int64
	AbsDeltaUnits float64
	// OffPlanLines counts campaigns the floor ran inside the frozen window on a scheduled machine that the frozen plan never called for. Working around a commitment breaks it as surely as editing it does, so it scores the same way.
	OffPlanLines    int64
	OffPlanQuantity float64
	LineAdherence   *float64
	UnitsAdherence  *float64
	FrozenThroughAt *time.Time
}

// SelectAttainmentBaselinesParams scopes the baseline read to an account and analysis window.
type SelectAttainmentBaselinesParams struct {
	AccountID   string
	WindowStart time.Time
	WindowEnd   time.Time
}

// AttainmentBaselineRow is one published schedule version whose horizon overlaps the window. Rows arrive newest-publish-first.
type AttainmentBaselineRow struct {
	ScheduleID       string
	Version          int32
	HorizonStartDate time.Time
	HorizonEndDate   time.Time
	// PublishedAt is nil for a version that was never published — a draft was never a commitment.
	PublishedAt       *time.Time
	FrozenThroughDate *time.Time
	FrozenLineCount   int32
	// FrozenPlannedQuantity is the quantity captured at publish, pre-converted from its decimal column.
	FrozenPlannedQuantity float64
}

// SumPlannedByWeekParams scopes the planned read to one baseline version within the window.
type SumPlannedByWeekParams struct {
	AccountID            string
	ProductionScheduleID string
	WindowStart          time.Time
	WindowEnd            time.Time
}

// AttainmentPlannedRow is planned quantity and run hours per (week, machine, item) for one baseline version.
type AttainmentPlannedRow struct {
	WeekStartDate   time.Time
	MachineID       string
	ItemID          string
	DepartmentID    *string
	PlannedQuantity float64
	PlannedRunHours float64
	LineCount       int64
}

// SumActualsByWeekParams scopes the actuals read to an account and scan window.
type SumActualsByWeekParams struct {
	AccountID   string
	WindowStart time.Time
	WindowEnd   time.Time
}

// AttainmentActualRow is what was actually produced per (week, machine, item), bucketed to the Monday of the scan week.
type AttainmentActualRow struct {
	WeekStartDate  time.Time
	MachineID      *string
	ItemID         string
	DepartmentID   *string
	ActualQuantity float64
	WasteQuantity  float64
	BatchCount     int64
}

// AttainmentDeviationRow counts frozen-week changes for one baseline version.
type AttainmentDeviationRow struct {
	ProductionScheduleID string
	DeviationCount       int64
	AddedCount           float64
	AbsDeltaQuantity     float64
}

// AttainmentLabelRow resolves a type id to the name the UI should show for it.
type AttainmentLabelRow struct {
	ID    string
	Label string
}

type ScheduleAttainmentResult struct {
	StartDate time.Time
	EndDate   time.Time
	GroupBy   string

	// Baselines names the published versions the measurement was taken against, so a number can always be traced back to the plan that produced it.
	BaselineScheduleIDs []string

	Buckets []AttainmentBucket
	Totals  AttainmentBucket

	// ScheduledMachineCount is how many machines the plan asked for over the window. Every figure above covers those machines only, so this is what says how wide the measurement was.
	ScheduledMachineCount int64

	FrozenAdherence []FrozenAdherence
	// HasBaseline is false when nothing was ever published over the window. Every ratio is nil in that case, and the caller should say "no plan" rather than "0%".
	HasBaseline bool
}
