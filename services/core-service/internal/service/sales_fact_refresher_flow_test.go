package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/open-mrp/api/shared/errors"
)

func newMockRefresher(t *testing.T) (*SalesFactRefresher, *repositorymock.MockSalesFactRepo, *[][]string) {
	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockSalesFactRepo(ctrl)
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewSalesFactRepo().Return(repo).AnyTimes()
	var invalidated [][]string
	cfg := (&SalesFactRefresherConfig{
		Repos:          factory,
		OnFactsChanged: func(_ context.Context, accounts []string) { invalidated = append(invalidated, accounts) },
	}).WithDefaults()
	return &SalesFactRefresher{cfg: cfg}, repo, &invalidated
}

func TestRefreshMarksTheDayBeforeWritingItsFactsAndClearsItAfterTheRebuild(t *testing.T) {
	r, repo, invalidated := newMockRefresher(t)
	ctx := context.Background()
	changed := fact("il_1", "10")
	day := domain.SalesRollupDay{AccountID: "ac_1", Day: utcDay(changed.InvoicedAt)}
	mark := domain.SalesRollupDirtyMark{Day: day, MarkedAt: time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)}

	repo.EXPECT().ComputeFacts(ctx, []string{"iv_1"}).Return([]domain.SalesLineFact{changed}, nil)
	repo.EXPECT().GetFacts(ctx, []string{"iv_1"}).Return(nil, nil)
	gomock.InOrder(
		repo.EXPECT().MarkRollupDays(ctx, []domain.SalesRollupDay{day}).Return(nil),
		repo.EXPECT().UpsertFacts(ctx, []domain.SalesLineFact{changed}).Return(nil),
		repo.EXPECT().DeleteFacts(ctx, []string{}).Return(nil),
		repo.EXPECT().ListRollupDirty(gomock.Any(), int32(salesRollupDrainBatch)).Return([]domain.SalesRollupDirtyMark{mark}, nil),
		repo.EXPECT().RebuildRollupDay(gomock.Any(), day).Return(nil),
		repo.EXPECT().RebuildRollupMonth(gomock.Any(), "ac_1", utcMonth(day.Day)).Return(nil),
		repo.EXPECT().ClearRollupDirty(gomock.Any(), mark).Return(nil),
	)

	n, apiErr := r.refreshInvoices(ctx, []string{"iv_1"})

	require.Nil(t, apiErr)
	require.Equal(t, 1, n)
	require.Equal(t, [][]string{{"ac_1"}, {"ac_1"}}, *invalidated, "invalidated once for the facts, once for the rollups")
}

func TestAFailedRebuildLeavesTheDayMarkedForTheNextTick(t *testing.T) {
	r, repo, _ := newMockRefresher(t)
	ctx := context.Background()
	day := domain.SalesRollupDay{AccountID: "ac_1", Day: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}
	mark := domain.SalesRollupDirtyMark{Day: day, MarkedAt: time.Now()}

	repo.EXPECT().ListRollupDirty(gomock.Any(), gomock.Any()).Return([]domain.SalesRollupDirtyMark{mark}, nil)
	repo.EXPECT().RebuildRollupDay(gomock.Any(), day).Return(apierror.NewInternalError(nil, "lock wait timeout"))
	repo.EXPECT().ClearRollupDirty(gomock.Any(), gomock.Any()).Times(0)

	require.NotNil(t, r.drainRollupDirty(ctx))
}

func TestFactsThatDidNotChangeMarkNothing(t *testing.T) {
	r, repo, invalidated := newMockRefresher(t)
	same := fact("il_1", "10")
	repo.EXPECT().ComputeFacts(gomock.Any(), gomock.Any()).Return([]domain.SalesLineFact{same}, nil)
	repo.EXPECT().GetFacts(gomock.Any(), gomock.Any()).Return([]domain.SalesLineFact{same}, nil)
	repo.EXPECT().MarkRollupDays(gomock.Any(), gomock.Any()).Times(0)
	repo.EXPECT().UpsertFacts(gomock.Any(), gomock.Any()).Times(0)
	repo.EXPECT().ListRollupDirty(gomock.Any(), gomock.Any()).Return(nil, nil)

	n, apiErr := r.refreshInvoices(context.Background(), []string{"iv_1"})
	require.Nil(t, apiErr)
	require.Zero(t, n)
	require.Empty(t, *invalidated)
}

func TestALargeChangeIsFannedOutIntoInvoiceMarks(t *testing.T) {
	r, repo, _ := newMockRefresher(t)
	product := domain.SalesFactDirtyMark{ScopeType: domain.SalesFactScopeProduct, ScopeID: "pr_1", AccountID: "ac_1"}
	invoices := make([]string, salesFactFanOutThreshold+1)
	for i := range invoices {
		invoices[i] = fmt.Sprintf("iv_%05d", i)
	}

	repo.EXPECT().ListDirty(gomock.Any(), gomock.Any()).Return([]domain.SalesFactDirtyMark{product}, nil)
	repo.EXPECT().ResolveInvoiceIDs(gomock.Any(), "ac_1", domain.SalesFactScopeProduct, []string{"pr_1"}).Return(invoices, nil)
	repo.EXPECT().MarkInvoicesDirty(gomock.Any(), "ac_1", invoices).Return(nil)
	repo.EXPECT().ClearDirty(gomock.Any(), product).Return(nil)
	repo.EXPECT().ComputeFacts(gomock.Any(), gomock.Any()).Times(0)

	require.Nil(t, r.drainDirty(context.Background()))
}

func TestAUnitChangeRestartsTheReconcilePass(t *testing.T) {
	r, repo, _ := newMockRefresher(t)
	restart := domain.SalesFactDirtyMark{ScopeType: domain.SalesFactScopeReconcile, ScopeID: salesFactReconcileScopeID, AccountID: "ac_1"}

	repo.EXPECT().ListDirty(gomock.Any(), gomock.Any()).Return([]domain.SalesFactDirtyMark{restart}, nil)
	repo.EXPECT().RestartReconcile(gomock.Any()).Return(nil)
	repo.EXPECT().ResolveInvoiceIDs(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	repo.EXPECT().ListRollupDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
	repo.EXPECT().ClearDirty(gomock.Any(), restart).Return(nil)

	require.Nil(t, r.drainDirty(context.Background()))
}
