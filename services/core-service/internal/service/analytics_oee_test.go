package service

import (
	"testing"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/stretchr/testify/assert"
)

func oeeTimePtr(t time.Time) *time.Time {
	return &t
}

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

func scanRow(machineID, departmentID string, scannedAt time.Time, idealSeconds float64) domain.OeeScanIntervalRow {
	return domain.OeeScanIntervalRow{
		MachineID:    machineID,
		DepartmentID: departmentID,
		ScannedAt:    oeeTimePtr(scannedAt),
		IdealSeconds: idealSeconds,
	}
}

// The canonical example: one ticket comes off machine 1 at 6:00, the next at 7:10. The second ticket's 60 units should have taken 60 minutes, so it ran at 60/70.
func TestComputeMeasuredPerformance_DiffsConsecutiveScansPerMachine(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 7, 27, 6, 0, 0, 0, time.UTC)
	rows := []domain.OeeScanIntervalRow{
		scanRow("mc_1", "dp_knit", base, 3000),
		scanRow("mc_1", "dp_knit", base.Add(70*time.Minute), 3600),
	}

	got := computeMeasuredPerformance(rows, nil)
	sample := got["dp_knit"]
	if sample == nil {
		t.Fatal("expected a sample for dp_knit")
	}
	assert.InDelta(t, 3600, sample.idealSeconds, 0.001)
	assert.InDelta(t, 4200, sample.actualSeconds, 0.001)
	assert.Equal(t, int64(1), sample.tickets, "the first scan has no predecessor, so only one ticket samples")

	dept := &domain.OeeDepartment{}
	applyMeasuredPerformance(dept, sample)
	computeOeeRatios(dept, 0)
	if dept.PerformancePct == nil {
		t.Fatal("performance = nil, want 3600/4200; measured P needs no planned time")
	}
	assert.InDelta(t, 0.857, *dept.PerformancePct, 0.001)
	assert.Equal(t, string(constants.OeePerformanceBasisScanIntervals), dept.PerformanceBasis)
}

// A gap between machines is meaningless: the last scan on machine A must not become the predecessor of the first scan on machine B.
func TestComputeMeasuredPerformance_ResetsAtMachineBoundary(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 7, 27, 6, 0, 0, 0, time.UTC)
	rows := []domain.OeeScanIntervalRow{
		scanRow("mc_1", "dp_knit", base, 600),
		scanRow("mc_2", "dp_knit", base.Add(10*time.Minute), 600),
	}

	got := computeMeasuredPerformance(rows, nil)
	assert.Empty(t, got, "each machine's first scan has no predecessor")
}

// Downtime already charged to Availability must come out of the gap, or the same stoppage would drag both A and P down. Performance-bucket downtime stays in: minor stops are exactly the loss measured P exists to capture.
func TestComputeMeasuredPerformance_SubtractsNonPerformanceDowntime(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 7, 27, 6, 0, 0, 0, time.UTC)
	rows := []domain.OeeScanIntervalRow{
		scanRow("mc_1", "dp_knit", base, 600),
		scanRow("mc_1", "dp_knit", base.Add(70*time.Minute), 3600),
	}
	downtime := buildMachineDowntime([]domain.OeeMachineDowntimeIntervalRow{
		{
			MachineID: "mc_1",
			OeeBucket: domain.OeeBucketAvailability,
			StartedAt: base.Add(10 * time.Minute),
			EndedAt:   oeeTimePtr(base.Add(20 * time.Minute)),
		},
		{
			MachineID: "mc_1",
			OeeBucket: domain.OeeBucketPerformance,
			StartedAt: base.Add(30 * time.Minute),
			EndedAt:   oeeTimePtr(base.Add(40 * time.Minute)),
		},
	})

	sample := computeMeasuredPerformance(rows, downtime)["dp_knit"]
	if sample == nil {
		t.Fatal("expected a sample for dp_knit")
	}
	// 70 min gap minus the 10 min availability stoppage; the performance-bucket stop stays.
	assert.InDelta(t, 3600, sample.actualSeconds, 0.001)
}

// Two overlapping events describing the same stoppage must not subtract the same seconds twice.
func TestBuildMachineDowntime_MergesOverlappingIntervals(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 7, 27, 6, 0, 0, 0, time.UTC)
	intervals := buildMachineDowntime([]domain.OeeMachineDowntimeIntervalRow{
		{MachineID: "mc_1", OeeBucket: domain.OeeBucketAvailability, StartedAt: base, EndedAt: oeeTimePtr(base.Add(20 * time.Minute))},
		{MachineID: "mc_1", OeeBucket: domain.OeeBucketQuality, StartedAt: base.Add(10 * time.Minute), EndedAt: oeeTimePtr(base.Add(30 * time.Minute))},
	})["mc_1"]

	if len(intervals) != 1 {
		t.Fatalf("intervals = %d, want 1 merged interval", len(intervals))
	}
	assert.InDelta(t, 30*60, downtimeWithinGap(intervals, base.Add(-time.Hour), base.Add(time.Hour)), 0.001)
}

// An overnight gap is a break in production, not slow running. Charging it to P would make the first ticket of every shift catastrophic.
func TestComputeMeasuredPerformance_ExcludesOutlierGaps(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 7, 27, 15, 0, 0, 0, time.UTC)
	rows := []domain.OeeScanIntervalRow{
		scanRow("mc_1", "dp_knit", base, 600),
		// 16 hours later: overnight. Ideal 1h, cap = max(4x3600, 2h) = 4h < 16h.
		scanRow("mc_1", "dp_knit", base.Add(16*time.Hour), 3600),
		// 30 minutes after that: a normal gap, sampled.
		scanRow("mc_1", "dp_knit", base.Add(16*time.Hour+30*time.Minute), 1500),
	}

	sample := computeMeasuredPerformance(rows, nil)["dp_knit"]
	if sample == nil {
		t.Fatal("expected a sample for dp_knit")
	}
	assert.Equal(t, int64(1), sample.tickets, "the overnight gap must be excluded")
	assert.InDelta(t, 1500, sample.idealSeconds, 0.001)
	assert.InDelta(t, 1800, sample.actualSeconds, 0.001)
}

// A ticket with no ideal cycle time configured cannot be judged, but its scan is still real: it must anchor the next ticket's gap, not vanish from the timeline.
func TestComputeMeasuredPerformance_NoIdealTimeStillAnchorsNextGap(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 7, 27, 6, 0, 0, 0, time.UTC)
	rows := []domain.OeeScanIntervalRow{
		scanRow("mc_1", "dp_knit", base, 600),
		scanRow("mc_1", "dp_knit", base.Add(10*time.Minute), 0),
		scanRow("mc_1", "dp_knit", base.Add(40*time.Minute), 1500),
	}

	sample := computeMeasuredPerformance(rows, nil)["dp_knit"]
	if sample == nil {
		t.Fatal("expected a sample for dp_knit")
	}
	assert.Equal(t, int64(1), sample.tickets)
	assert.InDelta(t, 1800, sample.actualSeconds, 0.001, "the gap runs from the unjudgeable ticket's scan, not before it")
}

// Measured performance wins over the shift-pattern fallback, and the fallback is labelled as such when it is all there is.
func TestComputeOeeRatios_MeasuredPerformanceBeatsFallback(t *testing.T) {
	t.Parallel()

	measured := &domain.OeeDepartment{
		GoodUnits:             100,
		StandardSecondsEarned: 7200,
		MeasuredIdealSeconds:  3600,
		MeasuredRunSeconds:    4200,
	}
	computeOeeRatios(measured, 1)
	if measured.PerformancePct == nil {
		t.Fatal("performance = nil, want measured value")
	}
	assert.InDelta(t, 3600.0/4200.0, *measured.PerformancePct, 0.0001)
	assert.Equal(t, string(constants.OeePerformanceBasisScanIntervals), measured.PerformanceBasis)
	assert.False(t, measured.HasPerformanceAnomaly)

	fallback := &domain.OeeDepartment{GoodUnits: 100, StandardSecondsEarned: 1800}
	computeOeeRatios(fallback, 1)
	if fallback.PerformancePct == nil {
		t.Fatal("performance = nil, want fallback estimate")
	}
	assert.InDelta(t, 0.5, *fallback.PerformancePct, 0.0001)
	assert.Equal(t, string(constants.OeePerformanceBasisRunTimeEstimate), fallback.PerformanceBasis)
}

// Measured P > 1 is possible when a scan was missed and two tickets share one gap, or when the ideal cycle time is stale. Same rule as before: report raw, flag it.
func TestComputeOeeRatios_MeasuredPerformanceAnomalyFlagged(t *testing.T) {
	t.Parallel()

	dept := &domain.OeeDepartment{GoodUnits: 100, MeasuredIdealSeconds: 4200, MeasuredRunSeconds: 3600}
	computeOeeRatios(dept, 0)

	if dept.PerformancePct == nil {
		t.Fatal("performance = nil, want raw over-100% value")
	}
	assert.InDelta(t, 4200.0/3600.0, *dept.PerformancePct, 0.0001)
	assert.True(t, dept.HasPerformanceAnomaly)
}
