package service

import (
	"context"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	publishermock "github.com/open-mrp/api/services/core-service/internal/domain/mock/publisher"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/omrpctx"
	apierror "github.com/open-mrp/apikit/apierror"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type cadenceFixture struct {
	svc          *productionScheduleSvcImpl
	scheduleRepo *repositorymock.MockProductionScheduleRepo
	outbox       *subscriptionOutboxRepo
}

func newCadenceFixture(t *testing.T) *cadenceFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	scheduleRepo := repositorymock.NewMockProductionScheduleRepo(ctrl)
	inputRepo := repositorymock.NewMockProductionScheduleInputRepo(ctrl)
	enqueuer := publishermock.NewMockProductionScheduleEnqueuer(ctrl)
	outbox := &subscriptionOutboxRepo{}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewProductionScheduleRepo().Return(scheduleRepo).AnyTimes()
	repos.EXPECT().NewProductionScheduleInputRepo().Return(inputRepo).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(outbox).AnyTimes()

	// An account that never configured scheduling plans on the defaults.
	inputRepo.EXPECT().GetAccountScheduleSettings(gomock.Any(), "ac_1").Return(nil, nil).AnyTimes()
	enqueuer.EXPECT().EnqueueGeneration(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	return &cadenceFixture{
		svc:          &productionScheduleSvcImpl{repos: repos, txManager: &stubTxManager{factory: repos}, enqueuer: enqueuer},
		scheduleRepo: scheduleRepo,
		outbox:       outbox,
	}
}

// No one asked for a cadence version, so its creation is audited as the scheduler's.
func TestEnqueueScheduledGeneration_AuditsTheVersionAsTheSchedulers(t *testing.T) {
	f := newCadenceFixture(t)
	f.scheduleRepo.EXPECT().NextVersion(gomock.Any(), "ac_1").Return(int32(4), nil).Times(1)
	f.scheduleRepo.EXPECT().CreateGeneratingSchedule(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	f.scheduleRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.GetProductionScheduleParams) (*domain.ProductionSchedule, *apierror.APIError) {
			return &domain.ProductionSchedule{ID: params.ScheduleID, StatusCode: domain.ScheduleStatusGenerating}, nil
		}).Times(1)

	apiErr := f.svc.EnqueueScheduledGeneration(context.Background(), domain.EnqueueGenerationParams{AccountID: "ac_1", PlanningAsOf: time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)})
	require.Nil(t, apiErr)

	events, identities := f.outbox.audits(t)
	require.Len(t, events, 1)
	require.Equal(t, constants.AuditActionCreate, events[0].Action)
	require.Equal(t, constants.ObjectTypeProductionSchedule, events[0].ResourceType)
	requireSchedulerIdentity(t, identities[0])
}

// A cadence run that fails leaves a `failed` version behind, and the change is the scheduler's.
func TestFailScheduledGeneration_AuditsTheFailure(t *testing.T) {
	f := newCadenceFixture(t)
	params := domain.GetProductionScheduleParams{AccountID: "ac_1", ScheduleID: "ps_1"}
	gomock.InOrder(
		f.scheduleRepo.EXPECT().Get(gomock.Any(), params).Return(&domain.ProductionSchedule{ID: "ps_1", StatusCode: domain.ScheduleStatusGenerating}, nil),
		f.scheduleRepo.EXPECT().FailGeneration(gomock.Any(), "ac_1", "ps_1", "solver unavailable").Return(nil),
		f.scheduleRepo.EXPECT().Get(gomock.Any(), params).Return(&domain.ProductionSchedule{ID: "ps_1", StatusCode: "failed"}, nil),
	)
	ctx := omrpctx.WithSystemIdentity(context.Background(), constants.SystemActorScheduler, "ac_1")

	require.Nil(t, f.svc.failScheduledGeneration(ctx, "ac_1", "ps_1", "solver unavailable"))

	events, identities := f.outbox.audits(t)
	require.Len(t, events, 1)
	require.Equal(t, constants.AuditActionUpdate, events[0].Action)
	require.Equal(t, "solver unavailable", events[0].Metadata["failure_reason"])
	require.Len(t, events[0].Changes, 1)
	require.Equal(t, "status_code", events[0].Changes[0].Field)
	requireSchedulerIdentity(t, identities[0])
}

// A duplicate delivery finds the version already past `generating` and writes nothing, under any identity.
func TestRunScheduledGeneration_RedeliveryWritesNothing(t *testing.T) {
	f := newCadenceFixture(t)
	f.scheduleRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.ProductionSchedule{ID: "ps_1", StatusCode: "draft"}, nil).Times(1)

	require.Nil(t, f.svc.RunScheduledGeneration(context.Background(), domain.RunScheduledGenerationParams{AccountID: "ac_1", ScheduleID: "ps_1"}))
	events, _ := f.outbox.audits(t)
	require.Empty(t, events)
}

func requireSchedulerIdentity(t *testing.T, identity *types.Identity) {
	t.Helper()
	require.True(t, identity.IsSystem())
	require.Equal(t, constants.SystemActorScheduler.ActorID(), identity.Actor.ID)
	require.Equal(t, "ac_1", identity.Target.AccountID)
}
