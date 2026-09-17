package service

import (
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestAggregateOeeDowntime_SplitsByBucket(t *testing.T) {
	t.Parallel()

	rows := []domain.OeeDowntimeRow{
		{DepartmentID: "dp_1", ReasonCode: "breakdown", OeeBucket: domain.OeeBucketAvailability, DowntimeSeconds: 600, EventCount: 2},
		{DepartmentID: "dp_1", ReasonCode: "changeover", OeeBucket: domain.OeeBucketAvailability, DowntimeSeconds: 1800, EventCount: 3},
		{DepartmentID: "dp_1", ReasonCode: "minor_stop", OeeBucket: domain.OeeBucketPerformance, DowntimeSeconds: 120, EventCount: 1},
		{DepartmentID: "dp_1", ReasonCode: "quality_hold", OeeBucket: domain.OeeBucketQuality, DowntimeSeconds: 300, EventCount: 1},
		{DepartmentID: "dp_1", ReasonCode: "no_schedule", OeeBucket: domain.OeeBucketNotScheduled, DowntimeSeconds: 7200, EventCount: 1},
	}

	got := aggregateOeeDowntime(rows)
	totals, ok := got["dp_1"]
	if !ok {
		t.Fatal("expected totals for dp_1")
	}

	if totals.availability != 2400 {
		t.Errorf("availability = %v, want 2400 (breakdown + changeover)", totals.availability)
	}
	if totals.performance != 120 {
		t.Errorf("performance = %v, want 120", totals.performance)
	}
	if totals.quality != 300 {
		t.Errorf("quality = %v, want 300", totals.quality)
	}
	if totals.notScheduled != 7200 {
		t.Errorf("notScheduled = %v, want 7200", totals.notScheduled)
	}
	// Changeover is an availability reason, so it must be counted in that bucket AND reported separately for the changeover KPI.
	if totals.changeover != 1800 {
		t.Errorf("changeover = %v, want 1800", totals.changeover)
	}
	if totals.events != 8 {
		t.Errorf("events = %v, want 8", totals.events)
	}
}

func TestAggregateOeeDowntime_SortsBreakdownByLossDescending(t *testing.T) {
	t.Parallel()

	rows := []domain.OeeDowntimeRow{
		{DepartmentID: "dp_1", ReasonCode: "minor_stop", OeeBucket: domain.OeeBucketPerformance, DowntimeSeconds: 120, EventCount: 1},
		{DepartmentID: "dp_1", ReasonCode: "breakdown", OeeBucket: domain.OeeBucketAvailability, DowntimeSeconds: 600, EventCount: 1},
	}

	totals := aggregateOeeDowntime(rows)["dp_1"]
	if totals.reasons[0].ReasonCode != "breakdown" {
		t.Errorf("first reason = %q, want breakdown (largest loss first)", totals.reasons[0].ReasonCode)
	}
}

func TestAggregateOeeDowntime_ClampsNegativeClip(t *testing.T) {
	t.Parallel()

	rows := []domain.OeeDowntimeRow{
		{DepartmentID: "dp_1", ReasonCode: "breakdown", OeeBucket: domain.OeeBucketAvailability, DowntimeSeconds: -60, EventCount: 1},
	}

	totals := aggregateOeeDowntime(rows)["dp_1"]
	if totals.availability != 0 {
		t.Errorf("availability = %v, want 0; a negative clip must not subtract from a loss total", totals.availability)
	}
}

// Run time is capacity net of not-scheduled and availability downtime. Not-scheduled time is
// removed from Planned Production Time rather than charged as a loss.
func TestComputeOeeRatios_NotScheduledLeavesDenominator(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:               90,
		WasteUnits:              10,
		StandardSecondsEarned:   3600,
		NotScheduledSeconds:     3600, // 1h nobody planned to run
		AvailabilityLossSeconds: 1800, // half an hour of breakdowns inside the planned time
	}

	// 10h capacity, 1h of which nobody scheduled -> 9h Planned Production Time; half an hour
	// of that was lost to availability downtime -> run time is 9h - 30min.
	computeOeeRatios(dept, 10)

	if dept.ScheduledSeconds != 9*3600 {
		t.Errorf("scheduled = %v, want %v (not-scheduled time is removed, not charged)", dept.ScheduledSeconds, 9*3600)
	}
	if dept.RunTimeSeconds != 9*3600-1800 {
		t.Errorf("runTime = %v, want %v", dept.RunTimeSeconds, 9*3600-1800)
	}

	wantAvailability := (9*3600.0 - 1800) / (9 * 3600.0)
	if dept.AvailabilityPct == nil || *dept.AvailabilityPct != wantAvailability {
		t.Errorf("availability = %v, want %v", dept.AvailabilityPct, wantAvailability)
	}
	if dept.QualityPct == nil || *dept.QualityPct != 0.9 {
		t.Errorf("quality = %v, want 0.9", dept.QualityPct)
	}
}

func TestComputeOeeRatios_NilWhenCapacityUnknown(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{GoodUnits: 90, WasteUnits: 10, StandardSecondsEarned: 3600}
	computeOeeRatios(dept, 0)

	// An unscheduled department has no capacity and so no OEE. Reporting 0% would read as a real result.
	if dept.AvailabilityPct != nil {
		t.Errorf("availability = %v, want nil when capacity is unknown", *dept.AvailabilityPct)
	}
	if dept.OeePct != nil {
		t.Errorf("oee = %v, want nil when capacity is unknown", *dept.OeePct)
	}
	// Quality needs no capacity, so it is still measurable.
	if dept.QualityPct == nil {
		t.Error("quality = nil, want 0.9; quality does not depend on capacity")
	}
}

func TestComputeOeeRatios_FlagsPerformanceAnomalyWithoutClamping(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:             100,
		WasteUnits:            0,
		StandardSecondsEarned: 7200, // more standard time earned than the equipment could have run
	}
	// 1h capacity, no downtime -> 3600s run time, but 7200s of standard time earned:
	// impossible at a correct rate, so P > 1 flags a stale/optimistic labor rate, under-logged
	// downtime, or a mis-set capacity.
	computeOeeRatios(dept, 1)

	if dept.PerformancePct == nil {
		t.Fatal("performance = nil, want the raw over-100% value")
	}
	if *dept.PerformancePct != 2 {
		t.Errorf("performance = %v, want 2 (raw, not clamped to 1)", *dept.PerformancePct)
	}
	if !dept.HasPerformanceAnomaly {
		t.Error("hasPerformanceAnomaly = false, want true; P > 1 means earned standard time exceeds run time")
	}
}

func TestComputeOeeRatios_OeeIsProductOfThree(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:               90,
		WasteUnits:              10,
		StandardSecondsEarned:   1800,
		AvailabilityLossSeconds: 1800, // half the hour was lost, so availability < 1
	}
	computeOeeRatios(dept, 1) // 1h capacity

	if dept.AvailabilityPct == nil || dept.PerformancePct == nil || dept.QualityPct == nil || dept.OeePct == nil {
		t.Fatal("expected all four ratios to be set")
	}
	want := *dept.AvailabilityPct * *dept.PerformancePct * *dept.QualityPct
	if *dept.OeePct != want {
		t.Errorf("oee = %v, want %v", *dept.OeePct, want)
	}
}

func TestApplyOeeDowntime_NoDataIsNotZeroDowntime(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{}
	applyOeeDowntime(dept, nil)

	// The flag is what stops the UI reporting a fake availability improvement for a department that simply has not logged anything yet.
	if dept.HasDowntimeData {
		t.Error("hasDowntimeData = true, want false when nothing was logged")
	}
	if dept.DowntimeBreakdown == nil {
		t.Error("downtimeBreakdown = nil, want an empty slice so it serializes as []")
	}
}

// Seconds-grade units are output but not good. Counting them only in the numerator's complement is what keeps a plant producing nothing but irregulars from reporting perfect quality.
func TestComputeOeeRatios_SecondsUnitsCountAgainstQuality(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{GoodUnits: 90, WasteUnits: 5, SecondsUnits: 5}
	computeOeeRatios(dept, 0)

	if dept.QualityPct == nil {
		t.Fatal("quality = nil, want 0.9")
	}
	if *dept.QualityPct != 0.9 {
		t.Errorf("quality = %v, want 0.9 (90 good of 100 produced, seconds included in the total)", *dept.QualityPct)
	}
}

// Performance is time over time. Deriving it from the irregular-unit count, as an earlier version did, silently reported a unit count as a duration.
func TestComputeOeeRatios_PerformanceIgnoresSecondsUnits(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{GoodUnits: 100, SecondsUnits: 7200}
	computeOeeRatios(dept, 1)

	if dept.PerformancePct != nil {
		t.Errorf("performance = %v, want nil; no standard time was earned", *dept.PerformancePct)
	}
}

// Raw shift configuration, no headroom: OEE measures against the time the equipment could
// actually run, and the changeover the headroom reserves is charged back as an availability loss.
func TestMachineWeeklyCapacityHours_RawShiftConfig(t *testing.T) {
	t.Parallel()

	settings := &domain.ProductionScheduleSettings{ShiftsPerDay: 2, HoursPerShift: 8, WorkDaysPerWeek: 5}
	assert.InDelta(t, 80, machineWeeklyCapacityHours(settings), 0.001, "2 shifts x 8h x 5 days = 80h, no headroom")
	assert.Equal(t, 0.0, machineWeeklyCapacityHours(nil), "nil settings yield no capacity")
}

// A window's capacity is one machine-week times the number of weeks it spans, so a partial
// window is measured against the part of a week it covers.
func TestOeeWindowWeeks(t *testing.T) {
	t.Parallel()

	monday := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	assert.InDelta(t, 4, oeeWindowWeeks(monday, monday.AddDate(0, 0, 28)), 0.001, "28 days is four weeks")
	assert.InDelta(t, 1.0/7.0, oeeWindowWeeks(monday, monday.AddDate(0, 0, 1)), 0.001, "one day is a seventh of a week")
	assert.Equal(t, 0.0, oeeWindowWeeks(monday, monday), "an empty window spans no weeks")
}

// An empty filter means every department; a non-empty one keeps only what was asked for.
func TestFilterDeptHours_HonoursTheFilter(t *testing.T) {
	hours := map[string]float64{"dp_knit": 80, "dp_dye": 40}

	assert.Equal(t, hours, filterDeptHours(hours, nil), "no filter keeps everything")

	got := filterDeptHours(hours, map[string]bool{"dp_knit": true})
	assert.Equal(t, map[string]float64{"dp_knit": 80}, got)
}

// The canonical OEE example: an ideal cycle time of one minute a unit, 320 units produced,
// 400 minutes of run time. The department ran at 80% of its designed speed. Run time is
// capacity net of downtime, not a scan span.
func TestComputeOeeRatios_PerformanceIsIdealTimeOverRunTime(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:               320,
		StandardSecondsEarned:   320 * 60,
		AvailabilityLossSeconds: 200 * 60, // 10h capacity minus 200min down leaves 400min run time
	}
	computeOeeRatios(dept, 10) // 10h capacity = 600min

	assert.InDelta(t, 400*60.0, dept.RunTimeSeconds, 0.001)
	if dept.PerformancePct == nil {
		t.Fatal("performance = nil, want 0.8")
	}
	assert.InDelta(t, 0.8, *dept.PerformancePct, 0.0001)
	assert.False(t, dept.HasPerformanceAnomaly)
}

// Minor stops and reduced speed are speed losses, not downtime. Only availability-bucket
// downtime comes out of run time, so a performance-bucket loss stays inside Performance's
// denominator and surfaces there. A machine that ran a full hour but earned only half an hour
// of standard time ran at half speed.
func TestComputeOeeRatios_PerformanceLossStaysInRunTime(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:              100,
		StandardSecondsEarned:  1800,
		PerformanceLossSeconds: 1800, // logged for the Pareto; not subtracted from run time
	}
	computeOeeRatios(dept, 1) // 1h capacity, no availability downtime

	assert.InDelta(t, 3600, dept.RunTimeSeconds, 0.001, "the machine ran the whole planned hour")
	if dept.PerformancePct == nil {
		t.Fatal("performance = nil, want 0.5")
	}
	assert.InDelta(t, 0.5, *dept.PerformancePct, 0.0001, "the half hour of minor stops has to land somewhere, and Performance is where")
	if dept.AvailabilityPct == nil || *dept.AvailabilityPct != 1 {
		t.Errorf("availability = %v, want 1; minor stops are not an availability loss", dept.AvailabilityPct)
	}
}

// Performance divides by run time, so a department with no capacity has no Performance for the
// same reason it has no Availability. Reporting one anyway would mean two departments in the same
// table answering different questions.
func TestComputeOeeRatios_PerformanceNilWithoutCapacity(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{GoodUnits: 100, StandardSecondsEarned: 3600}
	computeOeeRatios(dept, 0)

	if dept.PerformancePct != nil {
		t.Errorf("performance = %v, want nil when capacity is unknown", *dept.PerformancePct)
	}
}

// The department was scheduled but logged downtime for the entire planned time, so there was no
// run time: nothing could have run at any speed. Zero run time is no Performance, not 0%.
func TestComputeOeeRatios_PerformanceNilWhenNeverRan(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:               100,
		StandardSecondsEarned:   1800,
		AvailabilityLossSeconds: 3600, // the whole hour of capacity was lost to downtime
	}
	computeOeeRatios(dept, 1) // 1h capacity

	if dept.AvailabilityPct == nil || *dept.AvailabilityPct != 0 {
		t.Errorf("availability = %v, want 0", dept.AvailabilityPct)
	}
	if dept.PerformancePct != nil {
		t.Errorf("performance = %v, want nil; there was no run time to be fast or slow in", *dept.PerformancePct)
	}
}

// Run time is capacity minus downtime, so it can never exceed Planned Production Time: there is
// no overrun, Availability stays <= 100%, and Performance is a true speed ratio. This is what
// makes Carolon's Performance read at rate instead of over 100% — the scan-span denominator that
// caused that is gone.
func TestComputeOeeRatios_RunTimeCannotExceedCapacity(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:             1000,
		StandardSecondsEarned: 100 * 3600, // 100h of ideal work
	}
	computeOeeRatios(dept, 106) // 106h capacity, no logged downtime

	assert.InDelta(t, 106*3600.0, dept.ScheduledSeconds, 0.001)
	assert.InDelta(t, 106*3600.0, dept.RunTimeSeconds, 0.001, "no downtime means run time equals Planned Production Time")
	assert.InDelta(t, 106*3600.0, dept.OperatingTimeSeconds, 0.001, "operating time is run time: one clock")
	assert.Equal(t, 0.0, dept.OverrunSeconds, "overrun is retired: run time cannot exceed capacity")
	if dept.AvailabilityPct == nil || *dept.AvailabilityPct != 1 {
		t.Errorf("availability = %v, want 1", dept.AvailabilityPct)
	}
	if dept.PerformancePct == nil {
		t.Fatal("performance = nil")
	}
	assert.InDelta(t, 100.0/106.0, *dept.PerformancePct, 0.0001, "earned 100h of 106h run: under 100%, bounded by physics")
	assert.False(t, dept.HasPerformanceAnomaly)
	if dept.OeePct == nil || *dept.OeePct > 1 {
		t.Errorf("oee = %v, want <= 1", dept.OeePct)
	}
}
