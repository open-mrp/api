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

	repo.EXPECT().ComputeFacts(ctx, []string{"iv_1"}).Return([]domain.SalesLineFact{changed}, nil)
	repo.EXPECT().GetFacts(ctx, []string{"iv_1"}).Return(nil, nil)
	gomock.InOrder(
		repo.EXPECT().MarkRollupDays(ctx, []domain.SalesRollupDay{day}).Return(nil),
		repo.EXPECT().MarkBuyers(ctx, []domain.SalesBuyerKey{buyer}).Return(nil),
		repo.EXPECT().UpsertFacts(ctx, []domain.SalesLineFact{changed}).Return(nil),
		repo.EXPECT().DeleteFacts(ctx, []string{}).Return(nil),
		repo.EXPECT().ListRollupDirty(gomock.Any(), int32(salesRollupDrainBatch)).Return([]domain.SalesRollupDirtyMark{mark}, nil),
		repo.EXPECT().RebuildRollupDay(gomock.Any(), day).Return(nil),
		repo.EXPECT().RebuildRollupMonth(gomock.Any(), "ac_1", utcMonth(day.Day)).Return(nil),
		repo.EXPECT().ClearRollupDirty(gomock.Any(), mark).Return(nil),
		repo.EXPECT().ListBuyerDirty(gomock.Any(), int32(salesBuyerDrainBatch)).Return([]domain.SalesBuyerDirtyMark{buyerMark}, nil),
		repo.EXPECT().RebuildBuyerSummaries(gomock.Any(), "ac_1", []string{buyer.BuyerAccountID}).Return(nil),
		repo.EXPECT().ClearBuyerDirty(gomock.Any(), buyerMark).Return(nil),
	)

	n, apiErr := r.refreshInvoices(ctx, []string{"iv_1"})

	require.Nil(t, apiErr)
	require.Equal(t, 1, n)
	require.Equal(t, [][]string{{"ac_1"}, {"ac_1"}, {"ac_1"}}, *invalidated, "invalidated for the facts, the rollups and the buyer summaries")
}

func TestAFailedBuyerRebuildLeavesTheBuyerMarkedForTheNextTick(t *testing.T) {
	r, repo, _ := newMockRefresher(t)
	mark := domain.SalesBuyerDirtyMark{Buyer: domain.SalesBuyerKey{AccountID: "ac_1", BuyerAccountID: "ac_buyer"}, MarkedAt: time.Now()}

	repo.EXPECT().ListBuyerDirty(gomock.Any(), gomock.Any()).Return([]domain.SalesBuyerDirtyMark{mark}, nil)
	repo.EXPECT().RebuildBuyerSummaries(gomock.Any(), "ac_1", []string{"ac_buyer"}).Return(apierror.NewInternalError(nil, "lock wait timeout"))
	repo.EXPECT().ClearBuyerDirty(gomock.Any(), gomock.Any()).Times(0)

	require.NotNil(t, r.drainBuyerDirty(context.Background()))
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
	repo.EXPECT().ListBuyerDirty(gomock.Any(), gomock.Any()).Return(nil, nil)
	repo.EXPECT().ClearDirty(gomock.Any(), restart).Return(nil)

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
			repo.EXPECT().ComputeFacts(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().GetFacts(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().ClearDirty(gomock.Any(), gomock.Any()).Return(nil).Times(salesFactInvoiceBatch)
		}},
		{"more rollup days than one batch", func(repo *repositorymock.MockSalesFactRepo) {
			repo.EXPECT().ListRollupDirty(gomock.Any(), gomock.Any()).Return(fullRollup, nil)
			repo.EXPECT().RebuildRollupDay(gomock.Any(), gomock.Any()).Return(nil).Times(salesRollupDrainBatch)
			repo.EXPECT().RebuildRollupMonth(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			repo.EXPECT().ClearRollupDirty(gomock.Any(), gomock.Any()).Return(nil).Times(salesRollupDrainBatch)
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
			repo.EXPECT().ClearBuyerDirty(gomock.Any(), gomock.Any()).Return(nil).Times(salesBuyerDrainBatch)
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
	repo.EXPECT().ClearDirty(gomock.Any(), product).Return(nil)

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
			repo.EXPECT().ComputeFacts(gomock.Any(), gomock.Any()).Return(nil, nil)
			repo.EXPECT().GetFacts(gomock.Any(), gomock.Any()).Return(nil, nil)
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
	repo.EXPECT().RebuildRollupMonth(gomock.Any(), "ac_1", utcMonth(day.Day)).Return(nil)
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

func TestABuyerSweepRebuildsEachAccountsBuyersAndDropsThoseItNeverReached(t *testing.T) {
	r, repo, _ := newMockRefresher(t)
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	r.cfg.Now = func() time.Time { return now }
	since := now.Add(-time.Hour)
	filled := now.Add(-time.Minute)
	repo.EXPECT().GetBuyerSummarySync(gomock.Any()).Return(&domain.SalesBuyerSummarySync{FactsSince: &since}, nil)
	repo.EXPECT().GetSync(gomock.Any()).Return(&domain.SalesFactSync{LastCompletedAt: &filled}, nil)
	page := []domain.SalesBuyerKey{{AccountID: "ac_1", BuyerAccountID: "ac_a"}, {AccountID: "ac_1", BuyerAccountID: "ac_b"}, {AccountID: "ac_2", BuyerAccountID: "ac_c"}}
	gomock.InOrder(
		repo.EXPECT().NextBuyers(gomock.Any(), domain.SalesBuyerKey{}, int32(salesBuyerSweepBatch)).Return(page, nil),
		repo.EXPECT().RebuildBuyerSummaries(gomock.Any(), "ac_1", []string{"ac_a", "ac_b"}).Return(nil),
		repo.EXPECT().RebuildBuyerSummaries(gomock.Any(), "ac_2", []string{"ac_c"}).Return(nil),
		repo.EXPECT().NextBuyers(gomock.Any(), page[2], int32(salesBuyerSweepBatch)).Return(nil, nil),
		repo.EXPECT().DeleteBuyerSummariesRefreshedBefore(gomock.Any(), now).Return(nil),
		repo.EXPECT().SaveBuyerSummarySync(gomock.Any(), domain.SalesBuyerSummarySync{FactsSince: &since, PassStartedAt: &now, LastCompletedAt: &now}).Return(nil),
	)

	require.Nil(t, r.sweepBuyers(context.Background()))
	require.False(t, r.backlog)
}

func ptrTime(t time.Time) *time.Time { return &t }
