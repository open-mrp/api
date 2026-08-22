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
		GoodUnits:               90,
		WasteUnits:              10,
		StandardSecondsEarned:   3600,
		AvailabilityLossSeconds: 1800,
		NotScheduledSeconds:     3600,
	}

	// 10 planned hours, 1 of which nobody scheduled -> 9h scheduled, 0.5h down.
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

func TestComputeOeeRatios_NilWhenPlannedTimeUnknown(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{GoodUnits: 90, WasteUnits: 10, StandardSecondsEarned: 3600}
	computeOeeRatios(dept, 0)

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
		StandardSecondsEarned: 7200, // more standard time earned than run time available
	}
	computeOeeRatios(dept, 1) // 1 planned hour, no downtime -> 3600s run time

	if dept.PerformancePct == nil {
		t.Fatal("performance = nil, want the raw over-100% value")
	}
	if *dept.PerformancePct != 2 {
		t.Errorf("performance = %v, want 2 (raw, not clamped to 1)", *dept.PerformancePct)
	}
	if !dept.HasPerformanceAnomaly {
		t.Error("hasPerformanceAnomaly = false, want true; P > 1 always means a stale ideal cycle time")
	}
}

func TestComputeOeeRatios_OeeIsProductOfThree(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:               90,
		WasteUnits:              10,
		StandardSecondsEarned:   1800,
		AvailabilityLossSeconds: 1800,
	}
	computeOeeRatios(dept, 1) // 3600s scheduled, 1800s lost -> 1800s run time

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

func TestDerivePlannedHours_ScalesByMachinesAndPeriod(t *testing.T) {
	settings := &domain.ProductionScheduleSettings{
		ShiftsPerDay:    2,
		HoursPerShift:   7,
		WorkDaysPerWeek: 5,
	}
	start := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 7)

	got := derivePlannedHours(settings, map[string]int64{"dp_knit": 3}, start, end)

	// 2 shifts × 7 h × 5 days = 70 h a week, times three machines.
	assert.InDelta(t, 210, got["dp_knit"], 0.001,
		"scheduled time is machine-hours; a three-machine room is not measured against one shift")
}

// A three-day window is measured against three days of shift, not a whole week — otherwise every short range would report availability far worse than it was.
func TestDerivePlannedHours_ScalesToAPartialWeek(t *testing.T) {
	settings := &domain.ProductionScheduleSettings{ShiftsPerDay: 1, HoursPerShift: 8, WorkDaysPerWeek: 7}
	start := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)

	got := derivePlannedHours(settings, map[string]int64{"dp": 1}, start, start.AddDate(0, 0, 7))
	half := derivePlannedHours(settings, map[string]int64{"dp": 1}, start, start.AddDate(0, 0, 3))

	assert.InDelta(t, 56, got["dp"], 0.001)
	assert.InDelta(t, 24, half["dp"], 0.001)
}

// Availability has no meaning without a denominator, so an unconfigured shift pattern yields nothing rather than a guessed one.
func TestDerivePlannedHours_NoShiftPatternYieldsNothing(t *testing.T) {
	assert.Empty(t, derivePlannedHours(nil, map[string]int64{"dp": 1},
		time.Now(), time.Now().AddDate(0, 0, 7)))

	zero := &domain.ProductionScheduleSettings{ShiftsPerDay: 0, HoursPerShift: 0, WorkDaysPerWeek: 0}
	assert.Empty(t, derivePlannedHours(zero, map[string]int64{"dp": 1},
		time.Now(), time.Now().AddDate(0, 0, 7)))
}

func TestDerivePlannedHours_DepartmentWithNoMachines(t *testing.T) {
	settings := &domain.ProductionScheduleSettings{ShiftsPerDay: 2, HoursPerShift: 7, WorkDaysPerWeek: 5}
	start := time.Now()

	got := derivePlannedHours(settings, map[string]int64{"dp": 0}, start, start.AddDate(0, 0, 7))
	assert.Empty(t, got, "a department with no machines has no scheduled time")
}

// The canonical OEE example: an ideal cycle time of one minute a unit, 320 units produced, 400 minutes of run time. The department ran at 80% of its designed speed.
func TestComputeOeeRatios_PerformanceIsIdealTimeOverRunTime(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:               320,
		StandardSecondsEarned:   320 * 60,
		AvailabilityLossSeconds: 200 * 60,
	}
	// 10 planned hours less 200 minutes of availability loss leaves 400 minutes running.
	computeOeeRatios(dept, 10)

	assert.InDelta(t, 400*60.0, dept.RunTimeSeconds, 0.001)
	if dept.PerformancePct == nil {
		t.Fatal("performance = nil, want 0.8")
	}
	assert.InDelta(t, 0.8, *dept.PerformancePct, 0.0001)
	assert.False(t, dept.HasPerformanceAnomaly)
}

// Minor stops and reduced speed are speed losses, not downtime. They stay inside run time so they show up in Performance, which is the only OEE term they belong to; subtracting them the way availability losses are subtracted would make them invisible.
func TestComputeOeeRatios_PerformanceLossStaysInRunTime(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:              100,
		StandardSecondsEarned:  1800,
		PerformanceLossSeconds: 1800,
	}
	computeOeeRatios(dept, 1)

	assert.InDelta(t, 3600, dept.RunTimeSeconds, 0.001, "performance-bucket downtime must not leave run time")
	if dept.PerformancePct == nil {
		t.Fatal("performance = nil, want 0.5")
	}
	assert.InDelta(t, 0.5, *dept.PerformancePct, 0.0001, "the half hour of minor stops has to land somewhere, and Performance is where")
	if dept.AvailabilityPct == nil || *dept.AvailabilityPct != 1 {
		t.Errorf("availability = %v, want 1; minor stops are not an availability loss", dept.AvailabilityPct)
	}
}

// Performance divides by run time, so a department with no scheduled time has no Performance for the same reason it has no Availability. Reporting one anyway would mean two departments in the same table answering different questions.
func TestComputeOeeRatios_PerformanceNilWithoutRunTime(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{GoodUnits: 100, StandardSecondsEarned: 3600}
	computeOeeRatios(dept, 0)

	if dept.PerformancePct != nil {
		t.Errorf("performance = %v, want nil when run time is unknown", *dept.PerformancePct)
	}
}

// All the run time was lost to breakdowns, so nothing could have run at any speed. Zero over zero is not 0% performance.
func TestComputeOeeRatios_PerformanceNilWhenAllRunTimeLost(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{
		GoodUnits:               100,
		StandardSecondsEarned:   1800,
		AvailabilityLossSeconds: 3600,
	}
	computeOeeRatios(dept, 1)

	if dept.AvailabilityPct == nil || *dept.AvailabilityPct != 0 {
		t.Errorf("availability = %v, want 0", dept.AvailabilityPct)
	}
	if dept.PerformancePct != nil {
		t.Errorf("performance = %v, want nil; there was no run time to be fast or slow in", *dept.PerformancePct)
	}
}
