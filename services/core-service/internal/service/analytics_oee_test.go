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

func TestComputeOeeRatios_NotScheduledLeavesDenominator(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:             90,
		WasteUnits:            10,
		StandardSecondsEarned: 3600,
		NotScheduledSeconds:   3600,
	}

	// 10 planned hours, 1 of which nobody scheduled -> 9h scheduled. The machines ran a
	// measured 9h less half an hour -> availability is that run time over the 9h scheduled.
	computeOeeRatios(dept, 10, 9*3600-1800)

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

func TestComputeOeeRatios_NilWhenPlannedTimeUnknown(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{GoodUnits: 90, WasteUnits: 10, StandardSecondsEarned: 3600}
	computeOeeRatios(dept, 0, 0)

	// An unscheduled department has no OEE. Reporting 0% would read as a real result.
	if dept.AvailabilityPct != nil {
		t.Errorf("availability = %v, want nil when planned time is unknown", *dept.AvailabilityPct)
	}
	if dept.OeePct != nil {
		t.Errorf("oee = %v, want nil when planned time is unknown", *dept.OeePct)
	}
	// Quality needs no planned time, so it is still measurable.
	if dept.QualityPct == nil {
		t.Error("quality = nil, want 0.9; quality does not depend on planned time")
	}
}

func TestComputeOeeRatios_FlagsPerformanceAnomalyWithoutClamping(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:             100,
		WasteUnits:            0,
		StandardSecondsEarned: 7200, // more standard time earned than the machines were measured running
	}
	// The scheduled machines were measured running 3600s but earned 7200s of standard time:
	// impossible at a correct rate, so P > 1 flags a stale/optimistic labor rate.
	computeOeeRatios(dept, 1, 3600)

	if dept.PerformancePct == nil {
		t.Fatal("performance = nil, want the raw over-100% value")
	}
	if *dept.PerformancePct != 2 {
		t.Errorf("performance = %v, want 2 (raw, not clamped to 1)", *dept.PerformancePct)
	}
	if !dept.HasPerformanceAnomaly {
		t.Error("hasPerformanceAnomaly = false, want true; P > 1 means earned standard time exceeds measured run time")
	}
}

func TestComputeOeeRatios_OeeIsProductOfThree(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:             90,
		WasteUnits:            10,
		StandardSecondsEarned: 1800,
	}
	computeOeeRatios(dept, 1, 1800) // 3600s scheduled, machines measured running 1800s

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
	computeOeeRatios(dept, 0, 0)

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
	computeOeeRatios(dept, 1, 3600)

	if dept.PerformancePct != nil {
		t.Errorf("performance = %v, want nil; no standard time was earned", *dept.PerformancePct)
	}
}

// The per-department table reports the whole window as one figure. A window that covers each week in full takes all of that week's hours, so the weeks add up per department.
func TestProratedScheduledHours_SumsWholeWeeksPerDepartment(t *testing.T) {
	w1 := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC) // Monday
	w2 := w1.AddDate(0, 0, 7)

	got := proratedScheduledHours(map[time.Time]map[string]float64{
		w1: {"dp_knit": 80, "dp_dye": 40},
		w2: {"dp_knit": 60},
	}, w1, w2.AddDate(0, 0, 7))

	assert.InDelta(t, 140, got["dp_knit"], 0.001, "two full weeks of a department add up")
	assert.InDelta(t, 40, got["dp_dye"], 0.001)
}

// A window shorter than a week takes a proportional slice of that week's scheduled hours, the same slice the old day-count proration took — so availability is not divided by a whole week the range never covered.
func TestProratedScheduledHours_TakesAPartialWeekInProportion(t *testing.T) {
	monday := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)

	full := proratedScheduledHours(map[time.Time]map[string]float64{monday: {"dp": 140}}, monday, monday.AddDate(0, 0, 7))
	oneDay := proratedScheduledHours(map[time.Time]map[string]float64{monday: {"dp": 140}}, monday, monday.AddDate(0, 0, 1))

	assert.InDelta(t, 140, full["dp"], 0.001)
	assert.InDelta(t, 20, oneDay["dp"], 0.001, "one day of a week is a seventh of its scheduled hours")
}

func TestProratedScheduledHours_EmptyScheduleYieldsNothing(t *testing.T) {
	// No published plan over the window means no denominator, which computeOeeRatios turns into a nil availability rather than a fabricated one.
	now := time.Now()
	assert.Empty(t, proratedScheduledHours(nil, now, now.AddDate(0, 0, 7)))
	assert.Empty(t, proratedScheduledHours(map[time.Time]map[string]float64{}, now, now.AddDate(0, 0, 7)))
}

// An empty filter means every department; a non-empty one keeps only what was asked for.
func TestFilterDeptHours_HonoursTheFilter(t *testing.T) {
	hours := map[string]float64{"dp_knit": 80, "dp_dye": 40}

	assert.Equal(t, hours, filterDeptHours(hours, nil), "no filter keeps everything")

	got := filterDeptHours(hours, map[string]bool{"dp_knit": true})
	assert.Equal(t, map[string]float64{"dp_knit": 80}, got)
}

// The canonical OEE example: an ideal cycle time of one minute a unit, 320 units produced, 400 minutes of run time. The department ran at 80% of its designed speed.
func TestComputeOeeRatios_PerformanceIsIdealTimeOverRunTime(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:             320,
		StandardSecondsEarned: 320 * 60,
	}
	// 10 planned hours; the machines were measured running 400 minutes.
	computeOeeRatios(dept, 10, 400*60)

	assert.InDelta(t, 400*60.0, dept.RunTimeSeconds, 0.001)
	if dept.PerformancePct == nil {
		t.Fatal("performance = nil, want 0.8")
	}
	assert.InDelta(t, 0.8, *dept.PerformancePct, 0.0001)
	assert.False(t, dept.HasPerformanceAnomaly)
}

// Minor stops and reduced speed are speed losses, not downtime. Because Operating Time spans the machine's whole run, they stay inside Performance's denominator and surface there — the only OEE term they belong to. A machine that ran a full measured hour but earned only half an hour of standard time ran at half speed.
func TestComputeOeeRatios_PerformanceLossStaysInRunTime(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:              100,
		StandardSecondsEarned:  1800,
		PerformanceLossSeconds: 1800, // logged for the Pareto; not subtracted from run time
	}
	computeOeeRatios(dept, 1, 3600) // scheduled 1h, machines measured running the full hour

	assert.InDelta(t, 3600, dept.RunTimeSeconds, 0.001, "the machine ran the whole scheduled hour")
	if dept.PerformancePct == nil {
		t.Fatal("performance = nil, want 0.5")
	}
	assert.InDelta(t, 0.5, *dept.PerformancePct, 0.0001, "the half hour of minor stops has to land somewhere, and Performance is where")
	if dept.AvailabilityPct == nil || *dept.AvailabilityPct != 1 {
		t.Errorf("availability = %v, want 1; minor stops are not an availability loss", dept.AvailabilityPct)
	}
}

// Performance divides by operating time, so a department with no scheduled time has no Performance for the same reason it has no Availability. Reporting one anyway would mean two departments in the same table answering different questions.
func TestComputeOeeRatios_PerformanceNilWithoutRunTime(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{GoodUnits: 100, StandardSecondsEarned: 3600}
	computeOeeRatios(dept, 0, 0)

	if dept.PerformancePct != nil {
		t.Errorf("performance = %v, want nil when planned time is unknown", *dept.PerformancePct)
	}
}

// The scheduled machines never scanned, so there is no measured run time: nothing could have run at any speed. Zero operating time is no Performance, not 0%.
func TestComputeOeeRatios_PerformanceNilWhenNeverRan(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:             100,
		StandardSecondsEarned: 1800,
	}
	computeOeeRatios(dept, 1, 0) // scheduled 1h, but the machines were never measured running

	if dept.AvailabilityPct == nil || *dept.AvailabilityPct != 0 {
		t.Errorf("availability = %v, want 0", dept.AvailabilityPct)
	}
	if dept.PerformancePct != nil {
		t.Errorf("performance = %v, want nil; there was no run time to be fast or slow in", *dept.PerformancePct)
	}
}

// Factory Physics keeps OEE a chain of nested ratios, so Performance is bounded by physics: the ideal time for the output cannot exceed the time the machine was measured running. A plant that out-runs its schedule (Carolon's case) reads as 100% available with overrun reported apart, not >100% Performance.
func TestComputeOeeRatios_OverrunCapsAvailabilityNotPerformance(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:             1000,
		StandardSecondsEarned: 128 * 3600, // 128h of ideal work
	}
	// Scheduled 106h, but the machines were measured running 128h at their real rate.
	computeOeeRatios(dept, 106, 128*3600)

	assert.InDelta(t, 128*3600.0, dept.OperatingTimeSeconds, 0.001)
	assert.InDelta(t, 106*3600.0, dept.RunTimeSeconds, 0.001, "run time counted toward availability is capped at scheduled")
	assert.InDelta(t, 22*3600.0, dept.OverrunSeconds, 0.001, "the 22h over schedule is overrun, reported apart")
	if dept.AvailabilityPct == nil || *dept.AvailabilityPct != 1 {
		t.Errorf("availability = %v, want 1; overtime does not exceed 100%% available", dept.AvailabilityPct)
	}
	if dept.PerformancePct == nil {
		t.Fatal("performance = nil, want 1.0")
	}
	assert.InDelta(t, 1.0, *dept.PerformancePct, 0.0001, "ran 128h and earned 128h of standard time: 100%%, not >100%%")
	assert.False(t, dept.HasPerformanceAnomaly, "a plant that out-runs its schedule at rate is not a data anomaly")
	if dept.OeePct == nil || *dept.OeePct > 1 {
		t.Errorf("oee = %v, want <= 1", dept.OeePct)
	}
}
