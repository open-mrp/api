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
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

func newMockRefresher(t *testing.T) (*SalesFactRefresher, *repositorymock.MockSalesFactRepo, *[][]string) {
	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockSalesFactRepo(ctrl)
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewSalesFactRepo().Return(repo).AnyTimes()
	// Unknown line counts pack invoices by salesFactComputeBatch alone.
	repo.EXPECT().CountInvoiceLines(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
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
	buyer := domain.SalesBuyerKey{AccountID: "ac_1", BuyerAccountID: changed.BuyerAccountID}
	buyerMark := domain.SalesBuyerDirtyMark{Buyer: buyer, MarkedAt: mark.MarkedAt}

	started := mark.MarkedAt
	repo.EXPECT().ComputeFacts(ctx, []string{"iv_1"}).Return([]domain.SalesLineFact{changed}, nil)
	repo.EXPECT().GetFacts(ctx, []string{"iv_1"}).Return(nil, nil)
	repo.EXPECT().GetBuyerSummarySync(ctx).Return(&domain.SalesBuyerSummarySync{PassStartedAt: &started}, nil)
	gomock.InOrder(
		repo.EXPECT().MarkRollupDays(ctx, []domain.SalesRollupDay{day}).Return(nil),
		repo.EXPECT().MarkBuyers(ctx, []domain.SalesBuyerKey{buyer}).Return(nil),
		repo.EXPECT().UpsertFacts(ctx, []domain.SalesLineFact{changed}).Return(nil),
		repo.EXPECT().DeleteFacts(ctx, []string{}).Return(nil),
		repo.EXPECT().ListRollupDirty(gomock.Any(), int32(salesRollupDrainBatch)).Return([]domain.SalesRollupDirtyMark{mark}, nil),
		repo.EXPECT().RebuildRollupDay(gomock.Any(), day).Return(nil),
		repo.EXPECT().ClearRollupDirty(gomock.Any(), []domain.SalesRollupDirtyMark{mark}).Return(nil),
		repo.EXPECT().ListBuyerDirty(gomock.Any(), int32(salesBuyerDrainBatch)).Return([]domain.SalesBuyerDirtyMark{buyerMark}, nil),
		repo.EXPECT().RebuildBuyerSummaries(gomock.Any(), "ac_1", []string{buyer.BuyerAccountID}).Return(nil),
		repo.EXPECT().ClearBuyerDirty(gomock.Any(), []domain.SalesBuyerDirtyMark{buyerMark}).Return(nil),
	)

	n, apiErr := r.refreshInvoices(ctx, []string{"iv_1"})

	require.Nil(t, apiErr)
	require.Equal(t, 1, n)
	require.Equal(t, [][]string{{"ac_1"}, {"ac_1"}, {"ac_1"}}, *invalidated, "invalidated for the facts, the rollups and the buyer summaries")
}

// Before the summary sweep's first pass, which rebuilds every buyer anyway, changed facts mark no buyers:
// the fact backfill that precedes it rewrites every fact, and marks would rebuild each buyer per batch.
func TestNoBuyersAreMarkedBeforeTheFirstSummaryPass(t *testing.T) {
	r, repo, _ := newMockRefresher(t)
	ctx := context.Background()
	since := time.Date(2026, 9, 30, 17, 32, 0, 0, time.UTC)
	changed := fact("il_1", "10")
	repo.EXPECT().ComputeFacts(ctx, gomock.Any()).Return([]domain.SalesLineFact{changed}, nil).Times(2)
	repo.EXPECT().GetFacts(ctx, gomock.Any()).Return(nil, nil).Times(2)
	repo.EXPECT().GetBuyerSummarySync(ctx).Return(&domain.SalesBuyerSummarySync{FactsSince: &since}, nil).Times(1)
	repo.EXPECT().MarkRollupDays(ctx, gomock.Any()).Return(nil).Times(2)
	repo.EXPECT().MarkBuyers(gomock.Any(), gomock.Any()).Times(0)
	repo.EXPECT().UpsertFacts(ctx, gomock.Any()).Return(nil).Times(2)
	repo.EXPECT().DeleteFacts(ctx, gomock.Any()).Return(nil).Times(2)
	repo.EXPECT().ListRollupDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
	repo.EXPECT().ListBuyerDirty(gomock.Any(), gomock.Any()).Return(nil, nil)

	invoices := make([]string, salesFactComputeBatch+1) // two batches, one read of the sweep's state
	for i := range invoices {
		invoices[i] = fmt.Sprintf("iv_%04d", i)
	}
	_, apiErr := r.refreshInvoices(ctx, invoices)
	require.Nil(t, apiErr)
}

func TestAFailedBuyerRebuildLeavesTheBuyerMarkedForTheNextTick(t *testing.T) {
	r, repo, _ := newMockRefresher(t)
	mark := domain.SalesBuyerDirtyMark{Buyer: domain.SalesBuyerKey{AccountID: "ac_1", BuyerAccountID: "ac_buyer"}, MarkedAt: time.Now()}

	repo.EXPECT().ListBuyerDirty(gomock.Any(), gomock.Any()).Return([]domain.SalesBuyerDirtyMark{mark}, nil)
	repo.EXPECT().RebuildBuyerSummaries(gomock.Any(), "ac_1", []string{"ac_buyer"}).Return(apierror.NewInternalError(nil, "lock wait timeout"))
	repo.EXPECT().ClearBuyerDirty(gomock.Any(), gomock.Any()).Times(0)

	require.NotNil(t, r.drainBuyerDirty(context.Background()))
}

// A voided shipment's invoice is gone: its facts must go, and its mark clear, even while a buyer rebuild
// keeps failing. Otherwise the oldest marks are retried forever and every newer one waits behind them.
func TestAFailedBuyerRebuildStillClearsTheInvoiceMarksWhoseFactsWereWritten(t *testing.T) {
	r, repo, _ := newMockRefresher(t)
	voided := domain.SalesFactDirtyMark{ScopeType: domain.SalesFactScopeInvoice, ScopeID: "iv_1", AccountID: "ac_1"}
	stored := fact("il_1", "10")
	stored.BuyerAccountID = "ac_buyer"
	buyer := domain.SalesBuyerKey{AccountID: "ac_1", BuyerAccountID: stored.BuyerAccountID}
	started := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	repo.EXPECT().ListDirty(gomock.Any(), gomock.Any()).Return([]domain.SalesFactDirtyMark{voided}, nil)
	repo.EXPECT().ResolveInvoiceIDs(gomock.Any(), "ac_1", domain.SalesFactScopeInvoice, []string{"iv_1"}).Return([]string{"iv_1"}, nil)
	repo.EXPECT().ComputeFacts(gomock.Any(), []string{"iv_1"}).Return(nil, nil)
	repo.EXPECT().GetFacts(gomock.Any(), []string{"iv_1"}).Return([]domain.SalesLineFact{stored}, nil)
	repo.EXPECT().GetBuyerSummarySync(gomock.Any()).Return(&domain.SalesBuyerSummarySync{PassStartedAt: &started}, nil)
	repo.EXPECT().MarkRollupDays(gomock.Any(), gomock.Any()).Return(nil)
	repo.EXPECT().MarkBuyers(gomock.Any(), []domain.SalesBuyerKey{buyer}).Return(nil)
	repo.EXPECT().UpsertFacts(gomock.Any(), gomock.Len(0)).Return(nil)
	repo.EXPECT().DeleteFacts(gomock.Any(), []string{"il_1"}).Return(nil)
	repo.EXPECT().ListRollupDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
	repo.EXPECT().ListBuyerDirty(gomock.Any(), gomock.Any()).Return([]domain.SalesBuyerDirtyMark{{Buyer: buyer, MarkedAt: started}}, nil)
	repo.EXPECT().RebuildBuyerSummaries(gomock.Any(), "ac_1", []string{buyer.BuyerAccountID}).Return(apierror.NewInternalError(nil, "missing index"))
	repo.EXPECT().ClearBuyerDirty(gomock.Any(), gomock.Any()).Times(0)
	repo.EXPECT().ClearDirty(gomock.Any(), []domain.SalesFactDirtyMark{voided}).Return(nil)

	require.Nil(t, r.drainDirty(context.Background()))
	require.True(t, r.backlog, "the buyer stays marked for the next tick")
}

func TestALineThatChangedBuyerMarksBothBuyers(t *testing.T) {
	old := fact("il_1", "10")
	moved := old
	moved.BuyerAccountID = "ac_other"
	gone := fact("il_2", "5")
	require.ElementsMatch(t, []domain.SalesBuyerKey{
		{AccountID: "ac_1", BuyerAccountID: "ac_other"},
		{AccountID: "ac_1", BuyerAccountID: old.BuyerAccountID},
	}, touchedBuyers([]domain.SalesLineFact{moved}, []domain.SalesLineFact{gone}, []domain.SalesLineFact{old, gone}))
}

func TestAnOrderDateOrPriceFlagChangeRebuildsNoRollups(t *testing.T) {
	old := fact("il_1", "10")
	filled := old
	orderedAt := time.Date(2026, 8, 30, 9, 0, 0, 0, time.UTC)
	filled.OrderedAt, filled.IsPriced = &orderedAt, true
	require.Empty(t, touchedRollupDays([]domain.SalesLineFact{filled}, nil, []domain.SalesLineFact{old}),
		"the backfill of these columns must not rebuild every rollup day")
	require.Len(t, touchedBuyers([]domain.SalesLineFact{filled}, nil, []domain.SalesLineFact{old}), 1, "but it does rebuild the buyer")
	require.False(t, salesFactsEqual(old, filled), "and the fact itself is rewritten")
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
	repo.EXPECT().MarkBuyers(gomock.Any(), gomock.Any()).Times(0)
	repo.EXPECT().ListRollupDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
	repo.EXPECT().ListBuyerDirty(gomock.Any(), gomock.Any()).Return(nil, nil)

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
	repo.EXPECT().ClearDirty(gomock.Any(), []domain.SalesFactDirtyMark{product}).Return(nil)
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
	repo.EXPECT().ListBuyerDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
	repo.EXPECT().ClearDirty(gomock.Any(), []domain.SalesFactDirtyMark{restart}).Return(nil)

	require.Nil(t, r.drainDirty(context.Background()))
}

// expectQuietPasses sets up a tick whose rolling recompute, reconcile and rollup sweep have nothing due.
func expectQuietPasses(r *SalesFactRefresher, repo *repositorymock.MockSalesFactRepo) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	r.cfg.Now = func() time.Time { return now }
	r.lastRecent = now
	started := now.Add(-time.Hour)
	repo.EXPECT().GetSync(gomock.Any()).Return(&domain.SalesFactSync{PassStartedAt: &started, LastCompletedAt: &started}, nil).AnyTimes()
	repo.EXPECT().GetRollupSync(gomock.Any()).Return(&domain.SalesRollupSync{PassStartedAt: &started, LastCompletedAt: &started}, nil).AnyTimes()
	repo.EXPECT().GetBuyerSummarySync(gomock.Any()).Return(&domain.SalesBuyerSummarySync{FactsSince: &started, PassStartedAt: &started, LastCompletedAt: &started}, nil).AnyTimes()
}

func TestAnIdleTickLeavesNoBacklog(t *testing.T) {
	r, repo, _ := newMockRefresher(t)
	expectQuietPasses(r, repo)
	repo.EXPECT().ListRollupDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
	repo.EXPECT().ListBuyerDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
	repo.EXPECT().ListDirty(gomock.Any(), gomock.Any()).Return(nil, nil)

	require.False(t, r.Tick(context.Background()))
}

func TestATickReportsABacklog(t *testing.T) {
	fullDirty := make([]domain.SalesFactDirtyMark, salesFactInvoiceBatch)
	for i := range fullDirty {
		fullDirty[i] = domain.SalesFactDirtyMark{ScopeType: domain.SalesFactScopeInvoice, ScopeID: fmt.Sprintf("iv_%03d", i), AccountID: "ac_1"}
	}
	fullRollup := make([]domain.SalesRollupDirtyMark, salesRollupDrainBatch)
	for i := range fullRollup {
		fullRollup[i] = domain.SalesRollupDirtyMark{Day: domain.SalesRollupDay{AccountID: "ac_1", Day: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i)}}
	}
	fullBuyers := make([]domain.SalesBuyerDirtyMark, salesBuyerDrainBatch)
	for i := range fullBuyers {
		fullBuyers[i] = domain.SalesBuyerDirtyMark{Buyer: domain.SalesBuyerKey{AccountID: "ac_1", BuyerAccountID: fmt.Sprintf("ac_b%03d", i)}}
	}
	fail := apierror.NewInternalError(nil, "lock wait timeout")

	tests := []struct {
		name   string
		expect func(repo *repositorymock.MockSalesFactRepo)
	}{
		{"more dirty marks than one batch", func(repo *repositorymock.MockSalesFactRepo) {
			repo.EXPECT().ListRollupDirty(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
			repo.EXPECT().ListDirty(gomock.Any(), gomock.Any()).Return(fullDirty, nil)
			repo.EXPECT().ResolveInvoiceIDs(gomock.Any(), "ac_1", domain.SalesFactScopeInvoice, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ string, _ domain.SalesFactScope, ids []string) ([]string, *apierror.APIError) {
					return ids, nil
				})
			repo.EXPECT().ComputeFacts(gomock.Any(), gomock.Any()).Return(nil, nil).Times(computeBatches(salesFactInvoiceBatch))
			repo.EXPECT().GetFacts(gomock.Any(), gomock.Any()).Return(nil, nil).Times(computeBatches(salesFactInvoiceBatch))
			repo.EXPECT().ClearDirty(gomock.Any(), gomock.Len(salesFactInvoiceBatch)).Return(nil)
		}},
		{"more rollup days than one batch", func(repo *repositorymock.MockSalesFactRepo) {
			repo.EXPECT().ListRollupDirty(gomock.Any(), gomock.Any()).Return(fullRollup, nil)
			repo.EXPECT().RebuildRollupDay(gomock.Any(), gomock.Any()).Return(nil).Times(salesRollupDrainBatch)
			repo.EXPECT().ClearRollupDirty(gomock.Any(), gomock.Len(salesRollupDrainBatch)).Return(nil)
			repo.EXPECT().ListDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
		}},
		{"a failed rollup rebuild", func(repo *repositorymock.MockSalesFactRepo) {
			repo.EXPECT().ListRollupDirty(gomock.Any(), gomock.Any()).Return(fullRollup[:1], nil)
			repo.EXPECT().RebuildRollupDay(gomock.Any(), gomock.Any()).Return(fail)
			repo.EXPECT().ListDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
		}},
		{"more buyers than one batch", func(repo *repositorymock.MockSalesFactRepo) {
			repo.EXPECT().ListRollupDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().ListBuyerDirty(gomock.Any(), gomock.Any()).Return(fullBuyers, nil)
			repo.EXPECT().RebuildBuyerSummaries(gomock.Any(), "ac_1", gomock.Len(salesBuyerDrainBatch)).Return(nil)
			repo.EXPECT().ClearBuyerDirty(gomock.Any(), gomock.Len(salesBuyerDrainBatch)).Return(nil)
			repo.EXPECT().ListDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
		}},
		{"a failed buyer rebuild", func(repo *repositorymock.MockSalesFactRepo) {
			repo.EXPECT().ListRollupDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().ListBuyerDirty(gomock.Any(), gomock.Any()).Return(fullBuyers[:1], nil)
			repo.EXPECT().RebuildBuyerSummaries(gomock.Any(), gomock.Any(), gomock.Any()).Return(fail)
			repo.EXPECT().ListDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
		}},
		{"a failed dirty drain", func(repo *repositorymock.MockSalesFactRepo) {
			repo.EXPECT().ListRollupDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().ListDirty(gomock.Any(), gomock.Any()).Return(nil, fail)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, repo, _ := newMockRefresher(t)
			expectQuietPasses(r, repo)
			tt.expect(repo)
			repo.EXPECT().ListBuyerDirty(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

			require.True(t, r.Tick(context.Background()))
		})
	}
}

func TestAFanOutLeavesABacklog(t *testing.T) {
	r, repo, _ := newMockRefresher(t)
	product := domain.SalesFactDirtyMark{ScopeType: domain.SalesFactScopeProduct, ScopeID: "pr_1", AccountID: "ac_1"}
	invoices := make([]string, salesFactFanOutThreshold+1)
	for i := range invoices {
		invoices[i] = fmt.Sprintf("iv_%05d", i)
	}
	repo.EXPECT().ListDirty(gomock.Any(), gomock.Any()).Return([]domain.SalesFactDirtyMark{product}, nil)
	repo.EXPECT().ResolveInvoiceIDs(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(invoices, nil)
	repo.EXPECT().MarkInvoicesDirty(gomock.Any(), "ac_1", invoices).Return(nil)
	repo.EXPECT().ClearDirty(gomock.Any(), []domain.SalesFactDirtyMark{product}).Return(nil)

	require.Nil(t, r.drainDirty(context.Background()))
	require.True(t, r.backlog, "the invoice marks the fan-out wrote are drained by later runs")
}

func TestAReconcilePassInProgressLeavesABacklog(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	page := make([]domain.SalesFactInvoiceCursor, salesFactInvoiceBatch)
	for i := range page {
		page[i] = domain.SalesFactInvoiceCursor{CreatedAt: now.AddDate(-1, 0, 0), InvoiceID: fmt.Sprintf("iv_%03d", i)}
	}
	for _, tt := range []struct {
		name    string
		page    []domain.SalesFactInvoiceCursor
		backlog bool
	}{
		{"pass stops at its budget", page, true},
		{"pass completes", page[:3], false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, repo, _ := newMockRefresher(t)
			clock := now
			// Each read of the clock advances it, so the budget runs out after one page.
			r.cfg.Now = func() time.Time { clock = clock.Add(r.cfg.ReconcileBudget / 2); return clock }
			started := now.AddDate(0, 0, -2)
			repo.EXPECT().GetSync(gomock.Any()).Return(&domain.SalesFactSync{PassStartedAt: &started, LastCompletedAt: &started}, nil)
			repo.EXPECT().ListInvoicesAfter(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.page, nil)
			repo.EXPECT().ComputeFacts(gomock.Any(), gomock.Any()).Return(nil, nil).Times(computeBatches(len(tt.page)))
			repo.EXPECT().GetFacts(gomock.Any(), gomock.Any()).Return(nil, nil).Times(computeBatches(len(tt.page)))
			repo.EXPECT().ListRollupDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().ListBuyerDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
			if !tt.backlog {
				repo.EXPECT().ListFactInvoiceIDsAfter(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			}
			repo.EXPECT().SaveSync(gomock.Any(), gomock.Any()).Return(nil)

			require.Nil(t, r.reconcile(context.Background()))
			require.Equal(t, tt.backlog, r.backlog)
		})
	}
}

func TestARollupPassInProgressLeavesABacklog(t *testing.T) {
	r, repo, _ := newMockRefresher(t)
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	clock := now
	r.cfg.Now = func() time.Time { clock = clock.Add(r.cfg.ReconcileBudget / 2); return clock }
	day := domain.SalesRollupDay{AccountID: "ac_1", Day: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo.EXPECT().GetRollupSync(gomock.Any()).Return(&domain.SalesRollupSync{Cursor: &domain.SalesRollupDay{Day: salesFactSweepOrigin}, PassStartedAt: &now}, nil)
	repo.EXPECT().NextRollupDay(gomock.Any(), gomock.Any()).Return(&day, nil)
	repo.EXPECT().RebuildRollupDay(gomock.Any(), day).Return(nil)
	repo.EXPECT().SaveRollupSync(gomock.Any(), gomock.Any()).Return(nil)

	require.Nil(t, r.sweepRollups(context.Background()))
	require.True(t, r.backlog)
}

func newMockMarker(t *testing.T) (*SalesFactMarker, *repositorymock.MockSalesFactRepo, *int) {
	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockSalesFactRepo(ctrl)
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewSalesFactRepo().Return(repo).AnyTimes()
	wakes := 0
	return NewSalesFactMarker(factory, func() { wakes++ }), repo, &wakes
}

func TestTheMarkerWakesTheRefresherAfterItsMarkIsWritten(t *testing.T) {
	m, repo, wakes := newMockMarker(t)
	repo.EXPECT().MarkDirty(gomock.Any(), domain.SalesFactScopeInvoice, "iv_1", "ac_1").DoAndReturn(
		func(context.Context, domain.SalesFactScope, string, string) *apierror.APIError {
			require.Zero(t, *wakes, "woken before the mark was written")
			return nil
		})

	m.HandleAuditEvent(context.Background(), audit.ObservedEvent{AccountID: "ac_1", ResourceType: constants.ObjectTypeInvoice, ResourceID: "iv_1"})

	require.Equal(t, 1, *wakes)
}

func TestTheMarkerDoesNotWakeTheRefresherWithoutAMark(t *testing.T) {
	for name, e := range map[string]audit.ObservedEvent{
		"unrelated event":    {AccountID: "ac_1", ResourceType: constants.ObjectTypeShipment, ResourceID: "sh_1"},
		"no account":         {ResourceType: constants.ObjectTypeInvoice, ResourceID: "iv_1"},
		"irrelevant edit":    {AccountID: "ac_1", Action: constants.AuditActionUpdate, ResourceType: constants.ObjectTypeProduct, ResourceID: "pr_1", ChangedFields: []string{"name"}},
		"mark write failure": {AccountID: "ac_1", ResourceType: constants.ObjectTypeInvoice, ResourceID: "iv_fail"},
	} {
		t.Run(name, func(t *testing.T) {
			m, repo, wakes := newMockMarker(t)
			repo.EXPECT().MarkDirty(gomock.Any(), gomock.Any(), "iv_fail", gomock.Any()).Return(apierror.NewInternalError(nil, "down")).AnyTimes()

			m.HandleAuditEvent(context.Background(), e)

			require.Zero(t, *wakes)
		})
	}
}

func TestAMarkerWithoutAWakeStillMarks(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockSalesFactRepo(ctrl)
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewSalesFactRepo().Return(repo).AnyTimes()
	repo.EXPECT().MarkDirty(gomock.Any(), domain.SalesFactScopeInvoice, "iv_1", "ac_1").Return(nil)

	NewSalesFactMarker(factory, nil).HandleAuditEvent(context.Background(), audit.ObservedEvent{AccountID: "ac_1", ResourceType: constants.ObjectTypeInvoice, ResourceID: "iv_1"})
}

func TestTheFirstBuyerSweepRestartsTheFactReconcileToFillOrderDates(t *testing.T) {
	r, repo, _ := newMockRefresher(t)
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	r.cfg.Now = func() time.Time { return now }
	repo.EXPECT().GetBuyerSummarySync(gomock.Any()).Return(&domain.SalesBuyerSummarySync{}, nil)
	gomock.InOrder(
		repo.EXPECT().RestartReconcile(gomock.Any()).Return(nil),
		repo.EXPECT().SaveBuyerSummarySync(gomock.Any(), domain.SalesBuyerSummarySync{FactsSince: &now}).Return(nil),
	)
	repo.EXPECT().NextBuyers(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	require.Nil(t, r.sweepBuyers(context.Background()))
}

func TestTheFirstBuyerSweepWaitsForAReconcilePassCompletedAfterItsRestart(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	since := now.Add(-time.Hour)
	for name, completed := range map[string]*time.Time{
		"no pass completed yet":           nil,
		"the last pass predates the fill": ptrTime(since.Add(-time.Minute)),
	} {
		t.Run(name, func(t *testing.T) {
			r, repo, _ := newMockRefresher(t)
			r.cfg.Now = func() time.Time { return now }
			repo.EXPECT().GetBuyerSummarySync(gomock.Any()).Return(&domain.SalesBuyerSummarySync{FactsSince: &since}, nil)
			repo.EXPECT().GetSync(gomock.Any()).Return(&domain.SalesFactSync{LastCompletedAt: completed}, nil)
			repo.EXPECT().NextBuyers(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			repo.EXPECT().SaveBuyerSummarySync(gomock.Any(), gomock.Any()).Times(0)

			require.Nil(t, r.sweepBuyers(context.Background()))
		})
	}
}

func TestABuyerSweepRebuildsStaleBuyersAndDropsThoseWithoutFacts(t *testing.T) {
	r, repo, _ := newMockRefresher(t)
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	r.cfg.Now = func() time.Time { return now }
	since := now.Add(-time.Hour)
	filled := now.Add(-time.Minute)
	earlier, later := now.Add(-48*time.Hour), now.Add(-24*time.Hour)
	repo.EXPECT().GetBuyerSummarySync(gomock.Any()).Return(&domain.SalesBuyerSummarySync{FactsSince: &since}, nil)
	repo.EXPECT().GetSync(gomock.Any()).Return(&domain.SalesFactSync{LastCompletedAt: &filled}, nil)

	key := func(account, buyer string) domain.SalesBuyerKey {
		return domain.SalesBuyerKey{AccountID: account, BuyerAccountID: buyer}
	}
	page := []domain.SalesBuyerKey{key("ac_1", "ac_a"), key("ac_1", "ac_b"), key("ac_1", "ac_c"), key("ac_2", "ac_d")}
	gomock.InOrder(
		repo.EXPECT().NextBuyers(gomock.Any(), domain.SalesBuyerKey{}, int32(salesBuyerSweepBatch)).Return(page, nil),
		// ac_a is current, ac_b has a fact newer than its summary, ac_c and ac_d have none, ac_gone has no facts.
		repo.EXPECT().ListBuyerSummaryRefreshes(gomock.Any(), domain.SalesBuyerKey{}, &page[3]).Return(map[domain.SalesBuyerKey]time.Time{
			key("ac_1", "ac_a"): later, key("ac_1", "ac_b"): earlier, key("ac_1", "ac_gone"): earlier,
		}, nil),
		repo.EXPECT().LatestBuyerFactRefreshes(gomock.Any(), "ac_1", []string{"ac_a", "ac_b", "ac_c"}).Return(map[string]time.Time{
			"ac_a": earlier, "ac_b": later, "ac_c": earlier,
		}, nil),
		repo.EXPECT().RebuildBuyerSummaries(gomock.Any(), "ac_1", []string{"ac_b", "ac_c"}).Return(nil),
		repo.EXPECT().LatestBuyerFactRefreshes(gomock.Any(), "ac_2", []string{"ac_d"}).Return(map[string]time.Time{"ac_d": earlier}, nil),
		repo.EXPECT().RebuildBuyerSummaries(gomock.Any(), "ac_2", []string{"ac_d"}).Return(nil),
		repo.EXPECT().DeleteBuyerSummaries(gomock.Any(), []domain.SalesBuyerKey{key("ac_1", "ac_gone")}).Return(nil),
		repo.EXPECT().NextBuyers(gomock.Any(), page[3], int32(salesBuyerSweepBatch)).Return(nil, nil),
		// Past the last buyer with facts, every remaining summary is gone.
		repo.EXPECT().ListBuyerSummaryRefreshes(gomock.Any(), page[3], nil).Return(map[domain.SalesBuyerKey]time.Time{key("ac_3", "ac_e"): earlier}, nil),
		repo.EXPECT().DeleteBuyerSummaries(gomock.Any(), []domain.SalesBuyerKey{key("ac_3", "ac_e")}).Return(nil),
		repo.EXPECT().SaveBuyerSummarySync(gomock.Any(), domain.SalesBuyerSummarySync{FactsSince: &since, PassStartedAt: &now, LastCompletedAt: &now}).Return(nil),
	)

	require.Nil(t, r.sweepBuyers(context.Background()))
	require.False(t, r.backlog)
}

func TestPackInvoicesByLines(t *testing.T) {
	ids := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = fmt.Sprintf("iv_%02d", i)
		}
		return out
	}
	tests := []struct {
		name  string
		ids   []string
		lines map[string]int
		want  []int
	}{
		{"small invoices fill a batch to the invoice cap", ids(30), nil, []int{25, 5}},
		{"lines close a batch early", ids(3), map[string]int{"iv_00": 40, "iv_01": 30, "iv_02": 10}, []int{1, 2}},
		{"a large invoice stands alone", ids(3), map[string]int{"iv_00": 5, "iv_01": 177, "iv_02": 5}, []int{1, 1, 1}},
		{"none", nil, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sizes []int
			var flat []string
			for _, b := range packInvoicesByLines(tt.ids, tt.lines) {
				sizes = append(sizes, len(b))
				flat = append(flat, b...)
			}
			require.Equal(t, tt.want, sizes)
			require.Equal(t, tt.ids, flat)
		})
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

// The daily passes start after midnight Eastern rather than 24h after the last start, which drifted the rollup rebuild into business hours.
func TestADailyPassStartsOnlyAfterMidnightEastern(t *testing.T) {
	r, _, _ := newMockRefresher(t)
	et, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	at := func(y int, m time.Month, d, h, min int) time.Time { return time.Date(y, m, d, h, min, 0, 0, et) }

	tests := []struct {
		name    string
		started *time.Time
		now     time.Time
		due     bool
	}{
		{"never started", nil, at(2026, 10, 1, 11, 0), true},
		{"started this morning, now afternoon", ptrTime(at(2026, 10, 1, 11, 7)), at(2026, 10, 1, 13, 0), false},
		{"started just after midnight, now just before the next", ptrTime(at(2026, 10, 1, 0, 1)), at(2026, 10, 1, 23, 59), false},
		{"started before midnight, now just after", ptrTime(at(2026, 10, 1, 23, 30)), at(2026, 10, 2, 0, 1), true},
		{"started yesterday afternoon, now afternoon", ptrTime(at(2026, 9, 30, 13, 32)), at(2026, 10, 1, 13, 0), true},
		{"midnight follows the DST change", ptrTime(at(2026, 11, 1, 0, 30)), at(2026, 11, 1, 23, 30), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.due, r.passDue(tt.started, tt.now.UTC()))
		})
	}
}

// computeBatches is how many recomputes refreshInvoices runs for n invoices.
func computeBatches(n int) int {
	return (n + salesFactComputeBatch - 1) / salesFactComputeBatch
}
