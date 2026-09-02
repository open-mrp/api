package service

import (
	"context"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// oeePlannedFixture wires the two repos OEE availability now reads: the schedule-attainment repo for the planned denominator, and the analytics repo for output and downtime. A helper is provided per test so the schedule can be shaped to the case under test.
type oeePlannedFixture struct {
	svc       *analyticsSvcImpl
	schedule  *repositorymock.MockScheduleAttainmentRepo
	analytics *repositorymock.MockAnalyticsRepo
	// week is a Monday far enough in the past to be unambiguous history.
	week time.Time
}

func newOeePlannedFixture(t *testing.T) *oeePlannedFixture {
	t.Helper()
	ctrl := gomock.NewController(t)

	schedule := repositorymock.NewMockScheduleAttainmentRepo(ctrl)
	analytics := repositorymock.NewMockAnalyticsRepo(ctrl)
	scheduleRepo := repositorymock.NewMockProductionScheduleRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewScheduleAttainmentRepo().Return(schedule).AnyTimes()
	repos.EXPECT().NewAnalyticsRepo().Return(analytics).AnyTimes()
	repos.EXPECT().NewProductionScheduleRepo().Return(scheduleRepo).AnyTimes()
	// Weeks start on Monday in these fixtures, matching the Monday-based helper the fixture keys on.
	scheduleRepo.EXPECT().GetSettings(gomock.Any(), gomock.Any()).
		Return(&domain.ProductionScheduleSettings{WeekStartDay: 1}, nil).AnyTimes()

	return &oeePlannedFixture{
		svc:       &analyticsSvcImpl{repos: repos},
		schedule:  schedule,
		analytics: analytics,
		week:      weekStart(time.Now().UTC().AddDate(0, 0, -21)),
	}
}

// oneBaseline returns a single published version, live for the fixture's week and having frozen its whole horizon, so it owns every week the tests measure.
func (f *oeePlannedFixture) oneBaseline() []domain.AttainmentBaselineRow {
	published := f.week.AddDate(0, 0, -3)
	frozenThrough := f.week.AddDate(0, 0, 70)
	return []domain.AttainmentBaselineRow{{
		ScheduleID:        "pnsc_1",
		Version:           1,
		HorizonStartDate:  f.week.AddDate(0, 0, -7),
		HorizonEndDate:    f.week.AddDate(0, 0, 70),
		PublishedAt:       &published,
		FrozenThroughDate: &frozenThrough,
	}}
}

// The bug in one assertion: a one-week window used to report a denominator inflated by weeks it never scheduled and machines that no longer run. Grounded in the plan, it is exactly what the schedule put on that week — 150 run hours plus a 600-minute changeover is 160 machine-hours — and a department the plan never touched simply is not there.
func TestScheduledHoursByWeek_IsWhatThePlanScheduled(t *testing.T) {
	f := newOeePlannedFixture(t)

	f.schedule.EXPECT().SelectAttainmentBaselines(gomock.Any(), gomock.Any()).
		Return(f.oneBaseline(), nil).Times(1)
	f.schedule.EXPECT().SumScheduledHoursByDepartmentWeek(gomock.Any(), gomock.Any()).
		Return([]domain.ScheduledHoursRow{
			{WeekStartDate: f.week, DepartmentID: "dp_knit", PlannedRunHours: 150, PlannedChangeoverMinutes: 600},
		}, nil).Times(1)

	byWeek, apiErr := f.svc.scheduledHoursByWeek(context.Background(), "acc_1", f.week, f.week.AddDate(0, 0, 7), 1)
	require.Nil(t, apiErr)

	assert.InDelta(t, 160, byWeek[f.week]["dp_knit"], 0.001,
		"run hours plus changeover are both scheduled machine time")
	_, phantom := byWeek[f.week]["dp_dye"]
	assert.False(t, phantom, "a department the plan never scheduled contributes no hours")
}

// A quarter-long window does not balloon the denominator: only the weeks the plan actually scheduled contribute, so an under-loaded quarter is not measured against a full one.
func TestScheduledHoursByWeek_OnlyScheduledWeeksCount(t *testing.T) {
	f := newOeePlannedFixture(t)
	weekTwo := f.week.AddDate(0, 0, 7)

	f.schedule.EXPECT().SelectAttainmentBaselines(gomock.Any(), gomock.Any()).
		Return(f.oneBaseline(), nil).Times(1)
	// Two of the window's weeks were scheduled; the rest were not, so they never appear.
	f.schedule.EXPECT().SumScheduledHoursByDepartmentWeek(gomock.Any(), gomock.Any()).
		Return([]domain.ScheduledHoursRow{
			{WeekStartDate: f.week, DepartmentID: "dp_knit", PlannedRunHours: 160},
			{WeekStartDate: weekTwo, DepartmentID: "dp_knit", PlannedRunHours: 120},
		}, nil).Times(1)

	windowStart := f.week
	windowEnd := f.week.AddDate(0, 0, 84)
	byWeek, apiErr := f.svc.scheduledHoursByWeek(context.Background(), "acc_1", windowStart, windowEnd, 1)
	require.Nil(t, apiErr)

	assert.InDelta(t, 280, proratedScheduledHours(byWeek, windowStart, windowEnd)["dp_knit"], 0.001,
		"the window sums only the weeks that were scheduled, not every calendar week in the range")
	assert.Len(t, byWeek, 2)
}

// A version published after a past week ended did not govern it; its hours must not reach the denominator. This is the same rule schedule attainment enforces, so OEE and attainment agree on what a week was.
func TestScheduledHoursByWeek_IgnoresAVersionThatWasNotLive(t *testing.T) {
	f := newOeePlannedFixture(t)

	livePublished := f.week.AddDate(0, 0, -3)
	liveFrozen := f.week.AddDate(0, 0, 7)
	republishedAfterWeek := f.week.AddDate(0, 0, 9) // published after the week closed, so it never governed it
	republishedFrozen := f.week.AddDate(0, 0, 16)
	baselines := []domain.AttainmentBaselineRow{
		// Newest-publish first, matching the query's ORDER BY.
		{ScheduleID: "pnsc_2", Version: 2, PublishedAt: &republishedAfterWeek, HorizonStartDate: f.week.AddDate(0, 0, -7), HorizonEndDate: f.week.AddDate(0, 0, 70), FrozenThroughDate: &republishedFrozen},
		{ScheduleID: "pnsc_1", Version: 1, PublishedAt: &livePublished, HorizonStartDate: f.week.AddDate(0, 0, -7), HorizonEndDate: f.week.AddDate(0, 0, 70), FrozenThroughDate: &liveFrozen},
	}

	f.schedule.EXPECT().SelectAttainmentBaselines(gomock.Any(), gomock.Any()).Return(baselines, nil).Times(1)
	// The non-live republish carries a wildly different number; if it leaked in, the denominator would be wrong.
	f.schedule.EXPECT().SumScheduledHoursByDepartmentWeek(gomock.Any(), matchScheduleID("pnsc_2")).
		Return([]domain.ScheduledHoursRow{{WeekStartDate: f.week, DepartmentID: "dp_knit", PlannedRunHours: 9999}}, nil).Times(1)
	f.schedule.EXPECT().SumScheduledHoursByDepartmentWeek(gomock.Any(), matchScheduleID("pnsc_1")).
		Return([]domain.ScheduledHoursRow{{WeekStartDate: f.week, DepartmentID: "dp_knit", PlannedRunHours: 160}}, nil).Times(1)

	byWeek, apiErr := f.svc.scheduledHoursByWeek(context.Background(), "acc_1", f.week, f.week.AddDate(0, 0, 7), 1)
	require.Nil(t, apiErr)

	assert.InDelta(t, 160, byWeek[f.week]["dp_knit"], 0.001,
		"only the version that was live for the week may set its denominator")
}

// The end-to-end guard: availability is (scheduled - downtime) / scheduled, and scheduled is the plan's machine-hours. A department the plan never scheduled has no availability at all rather than a fabricated 100%.
func TestBuildOeeByDepartment_AvailabilityUsesScheduledHours(t *testing.T) {
	f := newOeePlannedFixture(t)

	// Read twice — once for the whole floor (estimated quality for unscheduled departments), once scoped to the scheduled machines — so the two return the same rows here; the assertions turn on Availability, which scoping leaves untouched.
	f.analytics.EXPECT().GetOeeDepartmentData(gomock.Any(), gomock.Any()).Return([]domain.OeeDepartmentDataRow{
		{DepartmentID: "dp_knit", DepartmentName: "Knitting", GoodUnits: 100, WasteUnits: 0},
		{DepartmentID: "dp_unplanned", DepartmentName: "Sampling", GoodUnits: 50, WasteUnits: 0},
	}, nil).AnyTimes()
	f.analytics.EXPECT().GetOeeEstimatedRuntime(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
	f.analytics.EXPECT().GetOeeDowntimeByDepartment(gomock.Any(), gomock.Any()).Return([]domain.OeeDowntimeRow{
		// 16 hours of availability-bucket downtime in the knitting room.
		{DepartmentID: "dp_knit", ReasonCode: "breakdown", OeeBucket: domain.OeeBucketAvailability, DowntimeSeconds: 16 * 3600, EventCount: 2},
	}, nil).Times(1)

	// Baselines are read for both the scheduled hours and the scheduled machines.
	f.schedule.EXPECT().SelectAttainmentBaselines(gomock.Any(), gomock.Any()).Return(f.oneBaseline(), nil).AnyTimes()
	f.schedule.EXPECT().SumScheduledHoursByDepartmentWeek(gomock.Any(), gomock.Any()).Return([]domain.ScheduledHoursRow{
		{WeekStartDate: f.week, DepartmentID: "dp_knit", PlannedRunHours: 160},
	}, nil).Times(1)
	// The knitting room scheduled one machine; its output is what the scoped read measures.
	f.schedule.EXPECT().SumPlannedByWeek(gomock.Any(), gomock.Any()).Return([]domain.AttainmentPlannedRow{
		{WeekStartDate: f.week, MachineID: "mc_knit_1", ItemID: "it_1", PlannedQuantity: 100, PlannedRunHours: 160},
	}, nil).Times(1)

	departments, apiErr := f.svc.buildOeeByDepartment(context.Background(), domain.AnalyzeOeeParams{
		AccountID: "acc_1",
		StartDate: f.week,
		EndDate:   f.week.AddDate(0, 0, 7),
	})
	require.Nil(t, apiErr)

	byID := map[string]domain.OeeDepartment{}
	for _, dept := range departments {
		byID[dept.DepartmentID] = dept
	}

	knit := byID["dp_knit"]
	assert.InDelta(t, 160*3600, knit.ScheduledSeconds, 0.001, "the denominator is the scheduled machine-hours")
	require.NotNil(t, knit.AvailabilityPct)
	assert.InDelta(t, 0.9, *knit.AvailabilityPct, 0.0001, "(160 - 16) / 160")

	// The plan never scheduled the sampling room, so it has no denominator and therefore no availability — not a fabricated 100%.
	unplanned := byID["dp_unplanned"]
	assert.Nil(t, unplanned.AvailabilityPct, "a department with no plan has no availability")
}

// matchScheduleID matches a SumPlannedByWeekParams carrying the given production schedule id, so the two per-baseline reads can return different rows.
func matchScheduleID(id string) gomock.Matcher {
	return scheduleIDMatcher{id: id}
}

type scheduleIDMatcher struct{ id string }

func (m scheduleIDMatcher) Matches(x any) bool {
	params, ok := x.(domain.SumPlannedByWeekParams)
	return ok && params.ProductionScheduleID == m.id
}

func (m scheduleIDMatcher) String() string { return "schedule id " + m.id }
