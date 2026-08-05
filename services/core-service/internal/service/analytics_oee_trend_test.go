package service

import (
	"testing"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/stretchr/testify/assert"
)

// Monday 3 August 2026.
var oeeTrendMonday = time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)

func TestOeeTrendBuckets_CutsOnMondays(t *testing.T) {
	t.Parallel()

	buckets := oeeTrendBuckets(oeeTrendMonday, oeeTrendMonday.AddDate(0, 0, 21))

	if len(buckets) != 3 {
		t.Fatalf("buckets = %d, want 3 whole weeks", len(buckets))
	}
	for i, bucket := range buckets {
		assert.Equal(t, oeeTrendMonday.AddDate(0, 0, i*7), bucket.start)
		assert.Equal(t, oeeTrendMonday.AddDate(0, 0, (i+1)*7), bucket.end)
	}
}

// A range that ends mid-week must report the days it covers, not a whole week: padding the last bucket out to Sunday would report a plant that had stopped producing.
func TestOeeTrendBuckets_ClipsPartialWeeksAtBothEnds(t *testing.T) {
	t.Parallel()

	start := oeeTrendMonday.AddDate(0, 0, 2)                   // Wednesday
	end := oeeTrendMonday.AddDate(0, 0, 10).Add(9 * time.Hour) // the Thursday after next, mid-morning
	buckets := oeeTrendBuckets(start, end)

	if len(buckets) != 2 {
		t.Fatalf("buckets = %d, want 2", len(buckets))
	}
	assert.Equal(t, start, buckets[0].start, "the first bucket starts where the window does")
	assert.Equal(t, oeeTrendMonday.AddDate(0, 0, 7), buckets[0].end, "and ends on the next Monday")
	assert.Equal(t, end, buckets[1].end, "the last bucket ends where the window does")
}

func TestOeeTrendBuckets_EmptyWindow(t *testing.T) {
	t.Parallel()

	assert.Empty(t, oeeTrendBuckets(oeeTrendMonday, oeeTrendMonday))
	assert.Empty(t, oeeTrendBuckets(oeeTrendMonday, oeeTrendMonday.AddDate(0, 0, -1)))
}

// A stoppage that runs from Friday into Monday belongs partly to each week. Charging it whole to the week it started in would make one week look worse and the next look untouched.
func TestOeeTrendDowntimeInBucket_SplitsAcrossWeekBoundary(t *testing.T) {
	t.Parallel()

	rows := []domain.OeeDowntimeIntervalRow{{
		DepartmentID: "dp_knit",
		OeeBucket:    domain.OeeBucketAvailability,
		StartedAt:    oeeTrendMonday.AddDate(0, 0, 6).Add(20 * time.Hour), // Sunday 20:00
		EndedAt:      oeeTrendMonday.AddDate(0, 0, 7).Add(6 * time.Hour),  // Monday 06:00
	}}

	first := oeeTrendDowntimeInBucket(rows, nil, oeeTrendMonday, oeeTrendMonday.AddDate(0, 0, 7))
	second := oeeTrendDowntimeInBucket(rows, nil, oeeTrendMonday.AddDate(0, 0, 7), oeeTrendMonday.AddDate(0, 0, 14))

	assert.InDelta(t, 4*3600.0, first["dp_knit"].availability, 0.001, "the four hours before midnight belong to week one")
	assert.InDelta(t, 6*3600.0, second["dp_knit"].availability, 0.001, "the six hours after it belong to week two")
}

// Minor stops and quality holds are logged, but they must not leave run time: they are speed and quality losses, and subtracting them from the denominator would hide them.
func TestOeeTrendDowntimeInBucket_OnlyAvailabilityAndNotScheduledChargeDenominators(t *testing.T) {
	t.Parallel()

	rows := []domain.OeeDowntimeIntervalRow{
		{DepartmentID: "dp", OeeBucket: domain.OeeBucketPerformance, StartedAt: oeeTrendMonday, EndedAt: oeeTrendMonday.Add(time.Hour)},
		{DepartmentID: "dp", OeeBucket: domain.OeeBucketQuality, StartedAt: oeeTrendMonday, EndedAt: oeeTrendMonday.Add(time.Hour)},
		{DepartmentID: "dp", OeeBucket: domain.OeeBucketNotScheduled, StartedAt: oeeTrendMonday, EndedAt: oeeTrendMonday.Add(2 * time.Hour)},
	}

	totals := oeeTrendDowntimeInBucket(rows, nil, oeeTrendMonday, oeeTrendMonday.AddDate(0, 0, 7))["dp"]

	assert.Zero(t, totals.availability)
	assert.InDelta(t, 2*3600.0, totals.notScheduled, 0.001)
	assert.Equal(t, int64(3), totals.events, "every logged event still counts as data, whatever it charges")
}

func TestOeeTrendDowntimeInBucket_HonoursDepartmentFilter(t *testing.T) {
	t.Parallel()

	rows := []domain.OeeDowntimeIntervalRow{
		{DepartmentID: "dp_keep", OeeBucket: domain.OeeBucketAvailability, StartedAt: oeeTrendMonday, EndedAt: oeeTrendMonday.Add(time.Hour)},
		{DepartmentID: "dp_drop", OeeBucket: domain.OeeBucketAvailability, StartedAt: oeeTrendMonday, EndedAt: oeeTrendMonday.Add(time.Hour)},
	}

	totals := oeeTrendDowntimeInBucket(rows, map[string]bool{"dp_keep": true}, oeeTrendMonday, oeeTrendMonday.AddDate(0, 0, 7))

	assert.Len(t, totals, 1)
	assert.NotNil(t, totals["dp_keep"])
}

// The roll-up is weighted by seconds. A department that ran one hour must not weigh as heavily as one that ran all week, which is what averaging their percentages would do.
func TestBuildOeeTrendPeriod_WeightsBySecondsNotByDepartment(t *testing.T) {
	t.Parallel()

	bucket := oeeTrendBucket{start: oeeTrendMonday, end: oeeTrendMonday.AddDate(0, 0, 7)}
	planned := map[string]float64{"dp_big": 100, "dp_small": 1}
	output := map[string]domain.OeeTrendDepartmentWeekRow{
		// The big room runs at half speed, the small one at full speed.
		"dp_big":   {GoodUnits: 100, StandardSecondsEarned: 100 * 3600 / 2},
		"dp_small": {GoodUnits: 10, StandardSecondsEarned: 1 * 3600},
	}

	period := buildOeeTrendPeriod(bucket, planned, output, nil)

	if period.PerformancePct == nil {
		t.Fatal("performance = nil, want a weighted value")
	}
	// (180000 + 3600) / (363600) — dominated by the big room, not the mean of 50% and 100%.
	assert.InDelta(t, 0.5049, *period.PerformancePct, 0.001)
	assert.InDelta(t, 101*3600.0, period.ScheduledSeconds, 0.001)
	assert.InDelta(t, 110, period.GoodUnits, 0.001)
}

// A department with no machines has no scheduled time, so it has no availability and no performance. Counting its output in quality would leave the three terms describing different plants.
func TestBuildOeeTrendPeriod_ExcludesDepartmentsWithoutScheduledTime(t *testing.T) {
	t.Parallel()

	bucket := oeeTrendBucket{start: oeeTrendMonday, end: oeeTrendMonday.AddDate(0, 0, 7)}
	output := map[string]domain.OeeTrendDepartmentWeekRow{
		"dp_scheduled": {GoodUnits: 90, WasteUnits: 10, StandardSecondsEarned: 3600},
		"unassigned":   {GoodUnits: 0, WasteUnits: 500, StandardSecondsEarned: 3600},
	}

	period := buildOeeTrendPeriod(bucket, map[string]float64{"dp_scheduled": 1}, output, nil)

	assert.InDelta(t, 100, period.GoodUnits+period.WasteUnits, 0.001, "the unscheduled department's output is not counted")
	if period.QualityPct == nil {
		t.Fatal("quality = nil, want 0.9")
	}
	assert.InDelta(t, 0.9, *period.QualityPct, 0.0001)
}

// Not-scheduled time leaves the denominator; availability losses leave run time. The trend has to agree with the per-department table on both, which is why it runs the same computeOeeRatios per department before summing.
func TestBuildOeeTrendPeriod_AppliesDowntimeToTheRightDenominator(t *testing.T) {
	t.Parallel()

	bucket := oeeTrendBucket{start: oeeTrendMonday, end: oeeTrendMonday.AddDate(0, 0, 7)}
	downtime := map[string]*oeeTrendDowntimeTotals{
		"dp": {availability: 3600, notScheduled: 7200, events: 4},
	}
	output := map[string]domain.OeeTrendDepartmentWeekRow{"dp": {GoodUnits: 100, StandardSecondsEarned: 3600}}

	period := buildOeeTrendPeriod(bucket, map[string]float64{"dp": 10}, output, downtime)

	assert.InDelta(t, 8*3600.0, period.ScheduledSeconds, 0.001, "10 planned hours less 2 nobody scheduled")
	assert.InDelta(t, 7*3600.0, period.RunTimeSeconds, 0.001, "less the hour of logged breakdown")
	assert.True(t, period.HasDowntimeData)
	assert.Equal(t, int64(4), period.DowntimeEventCount)
	assert.InDelta(t, 7.0/8.0, *period.AvailabilityPct, 0.0001)
	assert.InDelta(t, 3600.0/(7*3600.0), *period.PerformancePct, 0.0001)
}

// A week nobody scheduled has no OEE. Reporting 0% would read as a catastrophic week rather than a shut plant.
func TestBuildOeeTrendPeriod_NilRatiosWithoutScheduledTime(t *testing.T) {
	t.Parallel()

	bucket := oeeTrendBucket{start: oeeTrendMonday, end: oeeTrendMonday.AddDate(0, 0, 7)}
	period := buildOeeTrendPeriod(bucket, nil, nil, nil)

	assert.Nil(t, period.AvailabilityPct)
	assert.Nil(t, period.PerformancePct)
	assert.Nil(t, period.QualityPct)
	assert.Nil(t, period.OeePct)
	assert.Equal(t, bucket.start, period.StartsAt)
	assert.Equal(t, bucket.end, period.EndsAt)
}

func TestBuildOeeTrendPeriod_OeeIsProductOfThree(t *testing.T) {
	t.Parallel()

	bucket := oeeTrendBucket{start: oeeTrendMonday, end: oeeTrendMonday.AddDate(0, 0, 7)}
	output := map[string]domain.OeeTrendDepartmentWeekRow{"dp": {GoodUnits: 90, WasteUnits: 10, StandardSecondsEarned: 1800}}
	downtime := map[string]*oeeTrendDowntimeTotals{"dp": {availability: 1800, events: 1}}

	period := buildOeeTrendPeriod(bucket, map[string]float64{"dp": 1}, output, downtime)

	if period.OeePct == nil {
		t.Fatal("oee = nil, want the product of the three terms")
	}
	assert.InDelta(t, *period.AvailabilityPct**period.PerformancePct**period.QualityPct, *period.OeePct, 0.000001)
}

// The week key comes off the row's own date, so a row already sitting on a Monday and one mid-week land in the same bucket.
func TestIndexOeeTrendOutput_KeysOnTheWeekMonday(t *testing.T) {
	t.Parallel()

	rows := []domain.OeeTrendDepartmentWeekRow{
		{WeekStart: oeeTrendMonday, DepartmentID: "dp_a", GoodUnits: 1},
		{WeekStart: oeeTrendMonday.AddDate(0, 0, 3), DepartmentID: "dp_b", GoodUnits: 2},
		{WeekStart: oeeTrendMonday.AddDate(0, 0, 7), DepartmentID: "dp_a", GoodUnits: 3},
	}

	indexed := indexOeeTrendOutput(rows, nil)

	assert.Len(t, indexed[oeeTrendMonday], 2)
	assert.Len(t, indexed[oeeTrendMonday.AddDate(0, 0, 7)], 1)
}
