package service

import (
	"context"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// attainmentFixture wires the one repo buildScheduleAttainment reads, with a published version covering a single week that scheduled machine 51 only.
type attainmentFixture struct {
	svc  *analyticsSvcImpl
	repo *repositorymock.MockScheduleAttainmentRepo
	week time.Time
}

func newAttainmentFixture(t *testing.T) *attainmentFixture {
	t.Helper()
	ctrl := gomock.NewController(t)

	repo := repositorymock.NewMockScheduleAttainmentRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewScheduleAttainmentRepo().Return(repo).AnyTimes()

	// The fixture's weeks are Mondays, so the account measures on a Monday-start week.
	psRepo := repositorymock.NewMockProductionScheduleRepo(ctrl)
	repos.EXPECT().NewProductionScheduleRepo().Return(psRepo).AnyTimes()
	psRepo.EXPECT().GetSettings(gomock.Any(), gomock.Any()).
		Return(&domain.ProductionScheduleSettings{WeekStartDay: 1}, nil).AnyTimes()

	// A week far enough in the past that it is unambiguously history, published before it began.
	week := weekStart(time.Now().UTC().AddDate(0, 0, -21))
	published := week.AddDate(0, 0, -3)
	frozenThrough := week.AddDate(0, 0, 6)

	repo.EXPECT().SelectAttainmentBaselines(gomock.Any(), gomock.Any()).Return([]domain.AttainmentBaselineRow{{
		ScheduleID:            "pnsc_1",
		Version:               1,
		HorizonStartDate:      week.AddDate(0, 0, -7),
		HorizonEndDate:        week.AddDate(0, 0, 70),
		PublishedAt:           &published,
		FrozenThroughDate:     &frozenThrough,
		FrozenLineCount:       1,
		FrozenPlannedQuantity: 1000,
	}}, nil).AnyTimes()

	repo.EXPECT().SumPlannedByWeek(gomock.Any(), gomock.Any()).Return([]domain.AttainmentPlannedRow{{
		WeekStartDate:   week,
		MachineID:       "mc_51",
		ItemID:          "it_a",
		PlannedQuantity: 1000,
		PlannedRunHours: 40,
		LineCount:       1,
	}}, nil).AnyTimes()

	repo.EXPECT().GetMachineLabels(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]domain.AttainmentLabelRow{{ID: "mc_51", Label: "51"}}, nil).AnyTimes()

	repo.EXPECT().CountDeviationsForBaselines(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil).AnyTimes()

	return &attainmentFixture{
		svc:  &analyticsSvcImpl{repos: repos},
		repo: repo,
		week: week,
	}
}

func (f *attainmentFixture) run(t *testing.T, actuals []domain.AttainmentActualRow) *domain.ScheduleAttainmentResult {
	t.Helper()
	f.repo.EXPECT().SumActualsByWeek(gomock.Any(), gomock.Any()).Return(actuals, nil).Times(1)

	result, apiErr := f.svc.buildScheduleAttainment(context.Background(), domain.AnalyzeScheduleAttainmentParams{
		AccountID: "acc_1",
		StartDate: f.week.AddDate(0, 0, -7),
		EndDate:   time.Now().UTC(),
		GroupBy:   string(constants.AttainmentGroupByMachine),
	})
	require.Nil(t, apiErr)
	require.NotNil(t, result)
	return result
}

func machineID(id string) *string { return &id }

// The whole point of the scoping: a plant that schedules two knitting machines and scans a hundred other work centres was measuring the entire factory against a plan that only ever covered two of it, and "output vs target" came out meaningless.
func TestBuildScheduleAttainment_IgnoresMachinesThePlanNeverScheduled(t *testing.T) {
	f := newAttainmentFixture(t)

	result := f.run(t, []domain.AttainmentActualRow{
		{WeekStartDate: f.week, MachineID: machineID("mc_51"), ItemID: "it_a", ActualQuantity: 800, WasteQuantity: 10, BatchCount: 8},
		// A machine no published version scheduled. None of this may reach any figure.
		{WeekStartDate: f.week, MachineID: machineID("mc_99"), ItemID: "it_z", ActualQuantity: 5000, WasteQuantity: 900, BatchCount: 60},
	})

	require.EqualValues(t, 1, result.ScheduledMachineCount)
	require.Len(t, result.Buckets, 1, "only the scheduled machine may appear as a bucket")
	require.Equal(t, "51", result.Buckets[0].Label)

	require.EqualValues(t, 800, result.Totals.ActualQuantity)
	require.EqualValues(t, 1000, result.Totals.PlannedQuantity)
	require.EqualValues(t, 0, result.Totals.UnplannedQuantity)
	require.EqualValues(t, 10, result.Totals.WasteQuantity)
	require.NotNil(t, result.Totals.OutputRatioPct)
	require.EqualValues(t, 80, *result.Totals.OutputRatioPct)
}

// Work on a machine the plan DID schedule, for something the plan never asked for, is unplanned output — and inside a frozen window it is also a breach of the commitment that no deviation log records.
func TestBuildScheduleAttainment_OffPlanWorkOnAScheduledMachineCountsTwice(t *testing.T) {
	f := newAttainmentFixture(t)

	result := f.run(t, []domain.AttainmentActualRow{
		{WeekStartDate: f.week, MachineID: machineID("mc_51"), ItemID: "it_a", ActualQuantity: 1000, BatchCount: 10},
		{WeekStartDate: f.week, MachineID: machineID("mc_51"), ItemID: "it_rush", ActualQuantity: 250, BatchCount: 3},
	})

	require.EqualValues(t, 250, result.Totals.UnplannedQuantity)

	require.Len(t, result.FrozenAdherence, 1)
	adherence := result.FrozenAdherence[0]
	require.EqualValues(t, 1, adherence.OffPlanLines)
	require.EqualValues(t, 250, adherence.OffPlanQuantity)
	// 1 - 1/(1+1): half of what the frozen week turned out to contain was never committed to.
	require.NotNil(t, adherence.LineAdherence)
	require.EqualValues(t, 50, *adherence.LineAdherence)
}
