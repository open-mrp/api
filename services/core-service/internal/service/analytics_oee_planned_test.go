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

// oeePlannedFixture wires the two repos OEE availability now reads: the schedule-attainment repo for the scheduled machines behind capacity, and the analytics repo for output and downtime. A helper is provided per test so the schedule can be shaped to the case under test.
type oeePlannedFixture struct {
	svc       *analyticsSvcImpl
	schedule  *repositorymock.MockScheduleAttainmentRepo
	analytics *repositorymock.MockAnalyticsRepo
	// week is a Monday far enough in the past to be unambiguous history.
	week time.Time
}

// oeeFixtureMachineWeekly is the per-machine weekly capacity implied by the fixture's shift settings: 2 shifts x 8h x 5 days.
const oeeFixtureMachineWeekly = 2 * 8 * 5

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
	// Weeks start on Monday in these fixtures, matching the Monday-based helper the fixture keys on; the shift config gives one machine 80 capacity hours a week.
	scheduleRepo.EXPECT().GetSettings(gomock.Any(), gomock.Any()).
		Return(&domain.ProductionScheduleSettings{WeekStartDay: 1, ShiftsPerDay: 2, HoursPerShift: 8, WorkDaysPerWeek: 5}, nil).AnyTimes()

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

func oeeDeptPtr(s string) *string { return &s }

// Capacity scales with how many machines the plan scheduled: two knitting machines are two machines' capacity, and a department the plan never touched simply is not there.
func TestScheduledMachinesByDept_CountsPlanMachinesPerDepartment(t *testing.T) {
	f := newOeePlannedFixture(t)

	f.schedule.EXPECT().SelectAttainmentBaselines(gomock.Any(), gomock.Any()).
		Return(f.oneBaseline(), nil).Times(1)
	f.schedule.EXPECT().SumPlannedByWeek(gomock.Any(), gomock.Any()).
		Return([]domain.AttainmentPlannedRow{
			{WeekStartDate: f.week, MachineID: "mc_knit_1", ItemID: "it_1", DepartmentID: oeeDeptPtr("dp_knit")},
			{WeekStartDate: f.week, MachineID: "mc_knit_2", ItemID: "it_2", DepartmentID: oeeDeptPtr("dp_knit")},
			// Same machine on a second line in the week counts once.
			{WeekStartDate: f.week, MachineID: "mc_knit_1", ItemID: "it_3", DepartmentID: oeeDeptPtr("dp_knit")},
			{WeekStartDate: f.week, MachineID: "mc_dye_1", ItemID: "it_4", DepartmentID: oeeDeptPtr("dp_dye")},
			// A machine scheduled under no department has no availability, so it is dropped.
			{WeekStartDate: f.week, MachineID: "mc_orphan", ItemID: "it_5", DepartmentID: nil},
		}, nil).Times(1)

	byDept, apiErr := f.svc.scheduledMachinesByDept(context.Background(), "acc_1", f.week, f.week.AddDate(0, 0, 7), 1)
	require.Nil(t, apiErr)

	assert.Len(t, byDept["dp_knit"], 2, "two distinct machines, not three lines")
	assert.Len(t, byDept["dp_dye"], 1)
	_, orphan := byDept[""]
	assert.False(t, orphan, "a machine with no department is dropped")
}

// A week's capacity is its distinct scheduled machines times one machine's shift-configuration hours, per week, so the trend can prorate it.
func TestScheduledCapacityByWeek_ScalesByMachineCount(t *testing.T) {
	f := newOeePlannedFixture(t)
	weekTwo := f.week.AddDate(0, 0, 7)

	f.schedule.EXPECT().SelectAttainmentBaselines(gomock.Any(), gomock.Any()).
		Return(f.oneBaseline(), nil).Times(1)
	f.schedule.EXPECT().SumPlannedByWeek(gomock.Any(), gomock.Any()).
		Return([]domain.AttainmentPlannedRow{
			{WeekStartDate: f.week, MachineID: "mc_1", ItemID: "it_1", DepartmentID: oeeDeptPtr("dp_knit")},
			{WeekStartDate: f.week, MachineID: "mc_2", ItemID: "it_2", DepartmentID: oeeDeptPtr("dp_knit")},
			{WeekStartDate: weekTwo, MachineID: "mc_1", ItemID: "it_1", DepartmentID: oeeDeptPtr("dp_knit")},
		}, nil).Times(1)

	byWeek, apiErr := f.svc.scheduledCapacityByWeek(context.Background(), "acc_1", f.week, weekTwo.AddDate(0, 0, 7), 1, oeeFixtureMachineWeekly)
	require.Nil(t, apiErr)

	assert.InDelta(t, 2*oeeFixtureMachineWeekly, byWeek[f.week]["dp_knit"], 0.001, "two machines scheduled that week")
	assert.InDelta(t, 1*oeeFixtureMachineWeekly, byWeek[weekTwo]["dp_knit"], 0.001, "only one the next week")
}

// A version published after a past week ended did not govern it; its machines must not reach capacity. This is the same rule schedule attainment enforces, so OEE and attainment agree on what a week was.
func TestScheduledMachinesByDept_IgnoresAVersionThatWasNotLive(t *testing.T) {
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
	// The non-live republish scheduled a different machine; if it leaked in, capacity would double.
	f.schedule.EXPECT().SumPlannedByWeek(gomock.Any(), matchScheduleID("pnsc_2")).
		Return([]domain.AttainmentPlannedRow{{WeekStartDate: f.week, MachineID: "mc_phantom", ItemID: "it_1", DepartmentID: oeeDeptPtr("dp_knit")}}, nil).Times(1)
	f.schedule.EXPECT().SumPlannedByWeek(gomock.Any(), matchScheduleID("pnsc_1")).
		Return([]domain.AttainmentPlannedRow{{WeekStartDate: f.week, MachineID: "mc_knit_1", ItemID: "it_1", DepartmentID: oeeDeptPtr("dp_knit")}}, nil).Times(1)

	byDept, apiErr := f.svc.scheduledMachinesByDept(context.Background(), "acc_1", f.week, f.week.AddDate(0, 0, 7), 1)
	require.Nil(t, apiErr)

	assert.Len(t, byDept["dp_knit"], 1, "only the live version's machine counts")
	assert.True(t, byDept["dp_knit"]["mc_knit_1"])
	assert.False(t, byDept["dp_knit"]["mc_phantom"], "a version that did not govern the week schedules nothing for it")
}

// The end-to-end guard: availability is run time (capacity net of downtime) over Planned Production Time (the scheduled machines' capacity). A department the plan never scheduled has no availability at all rather than a fabricated 100%.
func TestBuildOeeByDepartment_AvailabilityUsesCapacityMinusDowntime(t *testing.T) {
	f := newOeePlannedFixture(t)

	// Read twice — once for the whole floor (quality for unscheduled departments), once scoped to the scheduled machines — so the two return the same rows here; the assertions turn on Availability, which scoping leaves untouched.
	f.analytics.EXPECT().GetOeeDepartmentData(gomock.Any(), gomock.Any()).Return([]domain.OeeDepartmentDataRow{
		{DepartmentID: "dp_knit", DepartmentName: "Knitting", GoodUnits: 100, WasteUnits: 0},
		{DepartmentID: "dp_unplanned", DepartmentName: "Sampling", GoodUnits: 50, WasteUnits: 0},
	}, nil).AnyTimes()
	f.analytics.EXPECT().GetOeeDowntimeByDepartment(gomock.Any(), gomock.Any()).Return([]domain.OeeDowntimeRow{
		// 16 hours of availability-bucket downtime in the knitting room comes out of run time.
		{DepartmentID: "dp_knit", ReasonCode: "breakdown", OeeBucket: domain.OeeBucketAvailability, DowntimeSeconds: 16 * 3600, EventCount: 2},
	}, nil).Times(1)

	f.schedule.EXPECT().SelectAttainmentBaselines(gomock.Any(), gomock.Any()).Return(f.oneBaseline(), nil).AnyTimes()
	// The knitting room scheduled two machines: 2 x 80 capacity hours over a one-week window = 160h Planned Production Time.
	f.schedule.EXPECT().SumPlannedByWeek(gomock.Any(), gomock.Any()).Return([]domain.AttainmentPlannedRow{
		{WeekStartDate: f.week, MachineID: "mc_knit_1", ItemID: "it_1", DepartmentID: oeeDeptPtr("dp_knit")},
		{WeekStartDate: f.week, MachineID: "mc_knit_2", ItemID: "it_2", DepartmentID: oeeDeptPtr("dp_knit")},
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
	assert.InDelta(t, 160*3600, knit.ScheduledSeconds, 0.001, "two machines x 80h capacity is the denominator")
	require.NotNil(t, knit.AvailabilityPct)
	assert.InDelta(t, 0.9, *knit.AvailabilityPct, 0.0001, "160h capacity less 16h downtime = 144h run / 160h")
	assert.Equal(t, 0.0, knit.OverrunSeconds, "overrun is retired")

	// The plan never scheduled the sampling room, so it has no capacity and therefore no availability — not a fabricated 100%.
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
