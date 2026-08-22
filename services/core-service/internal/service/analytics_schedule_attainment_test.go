package service

import (
	"context"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
)

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	return parsed
}

func TestWeekStart_NormalizesToMonday(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"monday stays put", "2026-07-27T00:00:00Z", "2026-07-27"},
		{"wednesday rolls back", "2026-07-29T13:45:00Z", "2026-07-27"},
		// Sunday is the END of its week, not the start of the next one. Go's Weekday() puts Sunday at 0, so an unshifted calculation lands a week early here.
		{"sunday rolls back six days", "2026-08-02T23:59:00Z", "2026-07-27"},
		{"next monday moves on", "2026-08-03T00:00:00Z", "2026-08-03"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := weekStart(mustDate(t, tc.in)).Format("2006-01-02")
			if got != tc.want {
				t.Errorf("weekStart(%s) = %s, want %s", tc.in, got, tc.want)
			}
		})
	}
}

func TestRatio_NilOnZeroDenominator(t *testing.T) {
	// A week nobody planned has no attainment. Reporting 0% would read as a total miss and would drag every average down.
	if got := ratio(0, 0); got != nil {
		t.Errorf("ratio(0, 0) = %v, want nil", *got)
	}
	if got := ratio(50, 0); got != nil {
		t.Errorf("ratio(50, 0) = %v, want nil", *got)
	}

	got := ratio(75, 100)
	if got == nil || *got != 75 {
		t.Errorf("ratio(75, 100) = %v, want 75", got)
	}
}

func baselineRow(id string, version int32, publishedAt, start, end string, t *testing.T) domain.AttainmentBaselineRow {
	t.Helper()
	published := mustDate(t, publishedAt)
	return domain.AttainmentBaselineRow{
		ScheduleID:       id,
		Version:          version,
		PublishedAt:      &published,
		HorizonStartDate: mustDate(t, start),
		HorizonEndDate:   mustDate(t, end),
	}
}

// The single most important rule in attainment: a version published *after* a week began was not the plan the floor worked to that week.
func TestBaselineFor_IgnoresVersionsPublishedAfterTheWeek(t *testing.T) {
	week := mustDate(t, "2026-07-27T00:00:00Z")

	// Newest publish first, matching the query's ORDER BY.
	baselines := []domain.AttainmentBaselineRow{
		baselineRow("pnsc_republished", 2, "2026-07-29T10:00:00Z", "2026-07-20T00:00:00Z", "2026-10-19T00:00:00Z", t),
		baselineRow("pnsc_original", 1, "2026-07-24T09:00:00Z", "2026-07-20T00:00:00Z", "2026-10-19T00:00:00Z", t),
	}

	got := baselineFor(baselines, week, mustDate(t, "2026-12-31T00:00:00Z"))
	if got == nil {
		t.Fatal("expected a baseline")
	}
	if got.ScheduleID != "pnsc_original" {
		t.Errorf("baseline = %s, want pnsc_original — a mid-week republish must not rewrite the week that was already worked", got.ScheduleID)
	}
}

func TestBaselineFor_PicksNewestQualifyingPublish(t *testing.T) {
	week := mustDate(t, "2026-08-10T00:00:00Z")

	baselines := []domain.AttainmentBaselineRow{
		baselineRow("pnsc_newer", 2, "2026-08-03T09:00:00Z", "2026-08-03T00:00:00Z", "2026-11-01T00:00:00Z", t),
		baselineRow("pnsc_older", 1, "2026-07-24T09:00:00Z", "2026-07-20T00:00:00Z", "2026-10-19T00:00:00Z", t),
	}

	got := baselineFor(baselines, week, mustDate(t, "2026-12-31T00:00:00Z"))
	if got == nil || got.ScheduleID != "pnsc_newer" {
		t.Errorf("baseline = %v, want pnsc_newer", got)
	}
}

func TestBaselineFor_NoneWhenWeekOutsideEveryHorizon(t *testing.T) {
	week := mustDate(t, "2027-01-04T00:00:00Z")

	baselines := []domain.AttainmentBaselineRow{
		baselineRow("pnsc_old", 1, "2026-07-24T09:00:00Z", "2026-07-20T00:00:00Z", "2026-10-19T00:00:00Z", t),
	}

	if got := baselineFor(baselines, week, mustDate(t, "2026-12-31T00:00:00Z")); got != nil {
		t.Errorf("baseline = %s, want none", got.ScheduleID)
	}
}

func TestBaselineFor_SkipsUnpublished(t *testing.T) {
	week := mustDate(t, "2026-07-27T00:00:00Z")

	baselines := []domain.AttainmentBaselineRow{
		{
			ScheduleID:       "pnsc_draft",
			HorizonStartDate: mustDate(t, "2026-07-20T00:00:00Z"),
			HorizonEndDate:   mustDate(t, "2026-10-19T00:00:00Z"),
		},
	}

	if got := baselineFor(baselines, week, mustDate(t, "2026-12-31T00:00:00Z")); got != nil {
		t.Errorf("baseline = %s, want none — a draft was never a commitment", got.ScheduleID)
	}
}

// Attainment caps at what was asked for; output ratio does not. Reporting only the second would let over-building one easy SKU hide a total miss on another.
func TestToBucket_AttainmentCapsButOutputRatioDoesNot(t *testing.T) {
	acc := &attainmentAccumulator{planned: 100, actual: 150, matched: 100}

	bucket := toBucket("k", "k", acc)

	if bucket.AttainmentPct == nil || *bucket.AttainmentPct != 100 {
		t.Errorf("attainment = %v, want 100 — building more than asked is not more than 100%% adherent", bucket.AttainmentPct)
	}
	if bucket.OutputRatioPct == nil || *bucket.OutputRatioPct != 150 {
		t.Errorf("output ratio = %v, want 150 — over-production has to stay visible", bucket.OutputRatioPct)
	}
}

func TestToBucket_UnderBuildShowsInBoth(t *testing.T) {
	acc := &attainmentAccumulator{planned: 100, actual: 40, matched: 40}

	bucket := toBucket("k", "k", acc)

	if bucket.AttainmentPct == nil || *bucket.AttainmentPct != 40 {
		t.Errorf("attainment = %v, want 40", bucket.AttainmentPct)
	}
	if bucket.OutputRatioPct == nil || *bucket.OutputRatioPct != 40 {
		t.Errorf("output ratio = %v, want 40", bucket.OutputRatioPct)
	}
}

func TestToBucket_UnplannedProductionIsSurfacedNotDropped(t *testing.T) {
	// Production with nothing planned against it: the schedule-breaker number.
	acc := &attainmentAccumulator{planned: 0, actual: 250, unplanned: 250}

	bucket := toBucket("k", "k", acc)

	if bucket.UnplannedQuantity != 250 {
		t.Errorf("unplanned = %v, want 250", bucket.UnplannedQuantity)
	}
	if bucket.AttainmentPct != nil {
		t.Errorf("attainment = %v, want nil — there was no plan to attain", *bucket.AttainmentPct)
	}
}

func TestPassesFilter_EmptyMeansNoFilter(t *testing.T) {
	if !passesFilter(nil, "anything") {
		t.Error("an empty filter must mean no filter, not match nothing")
	}
	filter := toSet([]string{"mc_1"})
	if !passesFilter(filter, "mc_1") {
		t.Error("mc_1 should pass its own filter")
	}
	if passesFilter(filter, "mc_2") {
		t.Error("mc_2 should not pass a filter for mc_1")
	}
}

// A week still being worked is not history, so the guard that protects the past does not apply to it. Requiring the plan to predate Monday made every schedule published mid-week report nothing planned, which is what made the analytics page read as broken.
func TestBaselineFor_InProgressWeekUsesTheLivePlan(t *testing.T) {
	week := mustDate(t, "2026-07-27T00:00:00Z")
	// Wednesday of that same week.
	now := mustDate(t, "2026-07-29T15:00:00Z")

	baselines := []domain.AttainmentBaselineRow{
		baselineRow("pnsc_published_midweek", 2, "2026-07-29T10:00:00Z", "2026-07-20T00:00:00Z", "2026-10-19T00:00:00Z", t),
	}

	got := baselineFor(baselines, week, now)
	if got == nil {
		t.Fatal("a week in progress must measure against the plan being worked right now")
	}
	if got.ScheduleID != "pnsc_published_midweek" {
		t.Errorf("baseline = %s, want pnsc_published_midweek", got.ScheduleID)
	}
}

// The moment the week is over it becomes history, and history keeps the plan that was live at the time rather than whatever was published since.
func TestBaselineFor_CompletedWeekStillRefusesALaterPublish(t *testing.T) {
	week := mustDate(t, "2026-07-27T00:00:00Z")
	now := mustDate(t, "2026-08-05T09:00:00Z")

	baselines := []domain.AttainmentBaselineRow{
		baselineRow("pnsc_republished", 2, "2026-07-29T10:00:00Z", "2026-07-20T00:00:00Z", "2026-10-19T00:00:00Z", t),
		baselineRow("pnsc_original", 1, "2026-07-24T09:00:00Z", "2026-07-20T00:00:00Z", "2026-10-19T00:00:00Z", t),
	}

	got := baselineFor(baselines, week, now)
	if got == nil || got.ScheduleID != "pnsc_original" {
		t.Errorf("baseline = %v, want pnsc_original — a finished week keeps the plan it was worked to", got)
	}
}

// offPlanRepo answers the one call frozenAdherence makes, so the adherence arithmetic can be tested without a database.
type offPlanRepo struct {
	domain.ScheduleAttainmentRepo
	rows []domain.AttainmentDeviationRow
}

func (r offPlanRepo) CountDeviationsForBaselines(_ context.Context, _ string, _ []string) ([]domain.AttainmentDeviationRow, *apierror.APIError) {
	return r.rows, nil
}

func TestFrozenAdherence_OffPlanWorkCountsAgainstTheCommitment(t *testing.T) {
	frozenThrough := mustDate(t, "2026-08-16T00:00:00Z")
	baselines := map[string]*domain.AttainmentBaselineRow{
		"pnsc_1": {
			ScheduleID:            "pnsc_1",
			Version:               1,
			FrozenLineCount:       8,
			FrozenPlannedQuantity: 4000,
			FrozenThroughDate:     &frozenThrough,
		},
	}
	repo := offPlanRepo{rows: []domain.AttainmentDeviationRow{{
		ProductionScheduleID: "pnsc_1",
		DeviationCount:       1,
		AddedCount:           0,
		AbsDeltaQuantity:     0,
	}}}

	clean, apiErr := frozenAdherence(context.Background(), repo, "acc_1", []string{"pnsc_1"}, baselines, nil, nil)
	if apiErr != nil {
		t.Fatalf("frozenAdherence: %v", apiErr)
	}
	if clean[0].LineAdherence == nil || *clean[0].LineAdherence != 87.5 {
		t.Fatalf("adherence with one hand edit = %v, want 87.5", clean[0].LineAdherence)
	}

	// Two campaigns the floor ran that the frozen plan never asked for: 1 - (1+2)/(8+2).
	fouled, apiErr := frozenAdherence(
		context.Background(), repo, "acc_1", []string{"pnsc_1"}, baselines,
		map[string]int64{"pnsc_1": 2}, map[string]float64{"pnsc_1": 400},
	)
	if apiErr != nil {
		t.Fatalf("frozenAdherence: %v", apiErr)
	}
	if fouled[0].OffPlanLines != 2 || fouled[0].OffPlanQuantity != 400 {
		t.Errorf("off-plan figures = %d lines / %v units, want 2 / 400", fouled[0].OffPlanLines, fouled[0].OffPlanQuantity)
	}
	if fouled[0].LineAdherence == nil || *fouled[0].LineAdherence != 70 {
		t.Errorf("adherence with two off-plan campaigns = %v, want 70", fouled[0].LineAdherence)
	}
	// Off-plan units are a breach in units too: 1 - 400/4000.
	if fouled[0].UnitsAdherence == nil || *fouled[0].UnitsAdherence != 90 {
		t.Errorf("units adherence = %v, want 90", fouled[0].UnitsAdherence)
	}
}
