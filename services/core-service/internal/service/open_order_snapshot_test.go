package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// idleOpenOrderSnapshotRepo is a snapshot with nothing in it and no issued orders to copy.
func idleOpenOrderSnapshotRepo(ctrl *gomock.Controller) *repositorymock.MockOpenOrderSnapshotRepo {
	repo := repositorymock.NewMockOpenOrderSnapshotRepo(ctrl)
	repo.EXPECT().ResolveOrders(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	repo.EXPECT().RefreshOrders(gomock.Any(), gomock.Any()).Return(0, nil).AnyTimes()
	repo.EXPECT().ListIssuedOrderIDsAfter(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	repo.EXPECT().ListSnapshotOrderIDsAfter(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	repo.EXPECT().FilterNotIssued(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	repo.EXPECT().SaveSync(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	return repo
}

func newOpenOrderRefresher(t *testing.T) (*SalesFactRefresher, *repositorymock.MockOpenOrderSnapshotRepo, *time.Time) {
	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockOpenOrderSnapshotRepo(ctrl)
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewOpenOrderSnapshotRepo().Return(repo).AnyTimes()
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	cfg := (&SalesFactRefresherConfig{Repos: factory, Now: func() time.Time { return now }}).WithDefaults()
	return &SalesFactRefresher{cfg: cfg}, repo, &now
}

func orderIDs(prefix string, n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("%s_%04d", prefix, i)
	}
	return ids
}

func TestTheOpenOrderPassCopiesEveryIssuedOrderThenRemovesTheRestAndRecordsItsCompletion(t *testing.T) {
	r, repo, now := newOpenOrderRefresher(t)
	ctx := context.Background()
	firstPage, lastPage := orderIDs("so_a", openOrderIssuedPage), []string{"so_z"}
	snapshot := []string{"so_a_0000", "so_gone"}
	started := *now
	gomock.InOrder(
		repo.EXPECT().ListIssuedOrderIDsAfter(gomock.Any(), "", int32(openOrderIssuedPage)).Return(firstPage, nil),
		repo.EXPECT().RefreshOrders(gomock.Any(), firstPage).Return(3, nil),
		repo.EXPECT().ListIssuedOrderIDsAfter(gomock.Any(), firstPage[len(firstPage)-1], int32(openOrderIssuedPage)).Return(lastPage, nil),
		repo.EXPECT().RefreshOrders(gomock.Any(), lastPage).Return(0, nil),
		repo.EXPECT().ListSnapshotOrderIDsAfter(gomock.Any(), "", int32(openOrderStalePage)).Return(snapshot, nil),
		repo.EXPECT().FilterNotIssued(gomock.Any(), snapshot).Return([]string{"so_gone"}, nil),
		repo.EXPECT().RefreshOrders(gomock.Any(), []string{"so_gone"}).Return(2, nil),
		repo.EXPECT().SaveSync(gomock.Any(), domain.OpenOrderSnapshotSync{PassStartedAt: &started, LastCompletedAt: &started}).Return(nil),
	)

	require.Nil(t, r.syncOpenOrders(ctx))
	require.False(t, r.backlog)
	require.Equal(t, openOrderIdle, r.openOrders.phase)
}

func TestTheOpenOrderPassWaitsItsIntervalBeforeStartingAgain(t *testing.T) {
	r, repo, now := newOpenOrderRefresher(t)
	ctx := context.Background()
	r.openOrders = openOrderPass{startedAt: *now}

	*now = now.Add(openOrderPassInterval - time.Second)
	require.Nil(t, r.syncOpenOrders(ctx))

	*now = now.Add(time.Second)
	repo.EXPECT().ListIssuedOrderIDsAfter(gomock.Any(), "", gomock.Any()).Return(nil, nil)
	repo.EXPECT().ListSnapshotOrderIDsAfter(gomock.Any(), "", gomock.Any()).Return(nil, nil)
	repo.EXPECT().FilterNotIssued(gomock.Any(), gomock.Any()).Return(nil, nil)
	repo.EXPECT().SaveSync(gomock.Any(), gomock.Any()).Return(nil)
	require.Nil(t, r.syncOpenOrders(ctx))
}

func TestAnOpenOrderPassOverItsBudgetResumesOnTheNextTick(t *testing.T) {
	r, repo, now := newOpenOrderRefresher(t)
	ctx := context.Background()
	firstPage := orderIDs("so_a", openOrderIssuedPage)
	repo.EXPECT().ListIssuedOrderIDsAfter(gomock.Any(), "", gomock.Any()).Return(firstPage, nil)
	repo.EXPECT().RefreshOrders(gomock.Any(), firstPage).DoAndReturn(func(context.Context, []string) (int, error) {
		*now = now.Add(r.cfg.ReconcileBudget)
		return 0, nil
	})

	require.Nil(t, r.syncOpenOrders(ctx))
	require.True(t, r.backlog, "an unfinished pass asks for another tick")
	require.Equal(t, openOrderIssued, r.openOrders.phase)

	repo.EXPECT().ListIssuedOrderIDsAfter(gomock.Any(), firstPage[len(firstPage)-1], gomock.Any()).Return(nil, nil)
	repo.EXPECT().ListSnapshotOrderIDsAfter(gomock.Any(), "", gomock.Any()).Return(nil, nil)
	repo.EXPECT().FilterNotIssued(gomock.Any(), gomock.Any()).Return(nil, nil)
	repo.EXPECT().SaveSync(gomock.Any(), gomock.Any()).Return(nil)
	r.backlog = false
	require.Nil(t, r.syncOpenOrders(ctx))
	require.Equal(t, openOrderIdle, r.openOrders.phase)
}

func TestDirtyMarksRefreshTheOpenOrdersTheyResolveTo(t *testing.T) {
	r, repo, _ := newOpenOrderRefresher(t)
	marks := []domain.SalesFactDirtyMark{
		{ScopeType: domain.SalesFactScopeRate, ScopeID: "rt_1", AccountID: "ac_1"},
		{ScopeType: domain.SalesFactScopeRate, ScopeID: "rt_2", AccountID: "ac_2"},
		{ScopeType: domain.SalesFactScopeInvoice, ScopeID: "iv_1", AccountID: "ac_1"},
	}
	repo.EXPECT().ResolveOrders(gomock.Any(), domain.SalesFactScopeRate, []string{"rt_1", "rt_2"}).Return([]string{"so_2", "so_1"}, nil)
	repo.EXPECT().ResolveOrders(gomock.Any(), domain.SalesFactScopeInvoice, []string{"iv_1"}).Return([]string{"so_1"}, nil)
	repo.EXPECT().RefreshOrders(gomock.Any(), []string{"so_1", "so_2"}).Return(1, nil)

	require.Nil(t, r.refreshOpenOrdersFor(context.Background(), marks))
}
