package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/lease"
	"github.com/open-mrp/api/shared/tracing"
	"go.opentelemetry.io/otel/attribute"
)

var salesFactTracer = tracing.GetTracer("core-service.sales_fact_refresher")

const (
	salesFactLeaseName = "sales-fact-refresher"
	salesFactLeaseTTL  = time.Minute

	// salesFactPassTZ is where the daily passes' midnight falls: the least busy hour for the US accounts they serve.
	salesFactPassTZ = "America/New_York"

	// salesFactInvoiceBatch is how many dirty marks one drain lists, and how many invoices one reconcile page reads.
	salesFactInvoiceBatch = 200

	// salesFactComputeBatch bounds one recompute: the pricing join reads ~35 rows an invoice, and 25 invoices keep it under 25ms.
	salesFactComputeBatch = 25

	// salesFactFanOutThreshold is the most invoices one drain refreshes itself. A scope that resolves to
	// more (a product moved to another product line after years of sales) is fanned out into one mark
	// per invoice, which later ticks work off a batch at a time.
	salesFactFanOutThreshold = 1000

	// salesRollupDrainBatch bounds the rollup days one drain rebuilds.
	salesRollupDrainBatch = 500

	// salesBuyerDrainBatch bounds the buyer summaries one drain rebuilds.
	salesBuyerDrainBatch = 500

	// salesBuyerSweepBatch is how many buyers the summary sweep reads and rebuilds at a time.
	salesBuyerSweepBatch = 200

	// salesFactReconcileScopeID is the single scope id reconcile requests share, so they coalesce.
	salesFactReconcileScopeID = "all"
)

// SalesFactRefresherConfig configures the refresher that keeps sales_line_fact equal to its source tables.
type SalesFactRefresherConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// Lease (required) keeps the refresh to one pod at a time.
	Lease *lease.Lease

	// LeaseName (optional; default: "sales-fact-refresher") names the lease, for tests that must not contend with a running service.
	LeaseName string

	// OnFactsChanged (optional; default: no-op) is called with the accounts whose facts a refresh rewrote, so cached reports built from the old facts can be dropped.
	OnFactsChanged func(ctx context.Context, accountIDs []string)

	// TickInterval (optional; default: 5s) is how soon the refresher runs again while work is left over: a backlog larger than one batch, a reconcile or rollup pass in progress, a step that failed, or a wake another pod's lease kept it from serving. Zero or negative values are treated as unset.
	TickInterval time.Duration

	// IdleInterval (optional; default: 1m) is how often the refresher runs with nothing to wake it. That run starts the rolling recompute and reconcile passes when due, and drains any mark whose wake was lost. Zero or negative values are treated as unset.
	IdleInterval time.Duration

	// WakeDelay (optional; default: 1s) is how long the refresher waits after a mark wakes it, so a burst of edits is drained in one run. Zero or negative values are treated as unset.
	WakeDelay time.Duration

	// RecentWindow (optional; default: 7d) is how far back the rolling recompute reaches. It catches writes that mark nothing, such as the dashboard's. Zero or negative values are treated as unset.
	RecentWindow time.Duration

	// RecentInterval (optional; default: 2m) is how often the rolling recompute runs. Zero or negative values are treated as unset.
	RecentInterval time.Duration

	// PassLocation (optional; default: America/New_York) is the timezone whose midnight starts each daily pass (fact reconcile, rollup sweep, buyer summary sweep). A pass that has not started since the most recent midnight there starts on the next tick.
	PassLocation *time.Location

	// ReconcileBudget (optional; default: 2s) caps the reconcile work done in one tick, so a pass is spread across ticks instead of loading the database in one burst. Zero or negative values are treated as unset.
	ReconcileBudget time.Duration

	// Now (optional; default: time.Now) is the clock, for tests.
	Now func() time.Time
}

// WithDefaults returns the config with default values applied where unset.
func (c *SalesFactRefresherConfig) WithDefaults() *SalesFactRefresherConfig {
	if c == nil {
		c = &SalesFactRefresherConfig{}
	}
	if c.OnFactsChanged == nil {
		c.OnFactsChanged = func(context.Context, []string) {}
	}
	if c.LeaseName == "" {
		c.LeaseName = salesFactLeaseName
	}
	if c.TickInterval <= 0 {
		c.TickInterval = 5 * time.Second
	}
	if c.IdleInterval <= 0 {
		c.IdleInterval = time.Minute
	}
	if c.WakeDelay <= 0 {
		c.WakeDelay = time.Second
	}
	if c.RecentWindow <= 0 {
		c.RecentWindow = 7 * 24 * time.Hour
	}
	if c.RecentInterval <= 0 {
		c.RecentInterval = 2 * time.Minute
	}
	if c.PassLocation == nil {
		loc, err := time.LoadLocation(salesFactPassTZ)
		if err != nil {
			slog.Warn("Sales fact refresher: could not load pass timezone, falling back to UTC", "tz", salesFactPassTZ, "error", err)
			loc = time.UTC
		}
		c.PassLocation = loc
	}
	if c.ReconcileBudget <= 0 {
		c.ReconcileBudget = 2 * time.Second
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return c
}

func (c *SalesFactRefresherConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("sales fact refresher: repos is required")
	}
	if c.Lease == nil {
		return fmt.Errorf("sales fact refresher: lease is required")
	}
	return nil
}

// SalesFactRefresher keeps sales_line_fact, and the sales_fact_rollup buckets summed from it, in step with invoices. Three paths feed it, each covering what the one before misses:
//   - dirty marks, written from audit events within seconds of an edit made through this API;
//   - a rolling recompute of recent invoices, for writes that publish no audit event (the dashboard, generic rate and quantity edits);
//   - a reconcile pass over every invoice each night after midnight in PassLocation, which corrects anything older that drifted. Its first pass is the backfill.
//
// Dirty marks wake it (see Wake), so it runs within WakeDelay of an edit and otherwise only every IdleInterval.
type SalesFactRefresher struct {
	cfg *SalesFactRefresherConfig

	lastRecent time.Time

	// backlog is set during a Tick by any step that left work for a later run.
	backlog bool

	// tick is Tick, replaceable so tests can drive the loop without a database.
	tick func(ctx context.Context) bool

	wake   chan struct{}
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewSalesFactRefresher(config *SalesFactRefresherConfig) *SalesFactRefresher {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}
	s := &SalesFactRefresher{cfg: config, wake: make(chan struct{}, 1), stopCh: make(chan struct{})}
	s.tick = s.Tick
	return s
}

// Wake asks for a run within WakeDelay. It never blocks: wakes that arrive before the run are served by it.
func (s *SalesFactRefresher) Wake() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *SalesFactRefresher) Start(ctx context.Context) error {
	s.wg.Add(1)
	go s.loop(ctx)
	slog.Info("Sales fact refresher started", "tick", s.cfg.TickInterval, "recent_window", s.cfg.RecentWindow)
	return nil
}

func (s *SalesFactRefresher) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	slog.Info("Sales fact refresher stopped")
}

func (s *SalesFactRefresher) loop(ctx context.Context) {
	defer s.wg.Done()
	timer := time.NewTimer(s.cfg.TickInterval)
	defer timer.Stop()
	due := time.Now().Add(s.cfg.TickInterval)
	// woken is set by a wake until a run serves it. A run this pod could not take (another pod held the
	// lease) has not served it: that pod may have listed marks before this wake's mark was written.
	woken := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-s.wake:
			woken = true
			if at := time.Now().Add(s.cfg.WakeDelay); at.Before(due) {
				due = at
				resetTimer(timer, s.cfg.WakeDelay)
			}
		case <-timer.C:
			ran, more := false, false
			_ = s.cfg.Lease.WithLease(ctx, s.cfg.LeaseName, salesFactLeaseTTL, func(leaseCtx context.Context) error {
				ran = true
				more = s.tick(leaseCtx)
				return nil
			})
			if ran {
				woken = false
			}
			next := s.cfg.IdleInterval
			if more || woken {
				next = s.cfg.TickInterval
			}
			due = time.Now().Add(next)
			timer.Reset(next)
		}
	}
}

// resetTimer reschedules a timer whose channel has not been received from.
func resetTimer(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}

// Tick runs one round of every path and reports whether it left work for a later run. Exported for tests; production calls it under the lease.
func (s *SalesFactRefresher) Tick(ctx context.Context) bool {
	s.backlog = false
	// Days a failed or interrupted refresh left marked are rebuilt first.
	if apiErr := s.drainRollupDirty(ctx); apiErr != nil {
		s.backlog = true
		slog.ErrorContext(ctx, "Sales fact refresher: rebuilding marked rollup days failed", "error", apiErr)
	}
	// Buyers a failed or interrupted refresh left marked are rebuilt first too.
	if apiErr := s.drainBuyerDirty(ctx); apiErr != nil {
		s.backlog = true
		slog.ErrorContext(ctx, "Sales fact refresher: rebuilding marked buyer summaries failed", "error", apiErr)
	}
	if apiErr := s.drainDirty(ctx); apiErr != nil {
		s.backlog = true
		slog.ErrorContext(ctx, "Sales fact refresher: draining dirty marks failed", "error", apiErr)
	}
	if now := s.cfg.Now(); now.Sub(s.lastRecent) >= s.cfg.RecentInterval {
		if apiErr := s.refreshRecent(ctx, now); apiErr != nil {
			s.backlog = true
			slog.ErrorContext(ctx, "Sales fact refresher: rolling recompute failed", "error", apiErr)
		} else {
			s.lastRecent = now
		}
	}
	if apiErr := s.reconcile(ctx); apiErr != nil {
		s.backlog = true
		slog.ErrorContext(ctx, "Sales fact refresher: reconcile failed", "error", apiErr)
	}
	if apiErr := s.sweepRollups(ctx); apiErr != nil {
		s.backlog = true
		slog.ErrorContext(ctx, "Sales fact refresher: rollup sweep failed", "error", apiErr)
	}
	if apiErr := s.sweepBuyers(ctx); apiErr != nil {
		s.backlog = true
		slog.ErrorContext(ctx, "Sales fact refresher: buyer summary sweep failed", "error", apiErr)
	}
	return s.backlog
}

func (s *SalesFactRefresher) drainDirty(ctx context.Context) *apierror.APIError {
	ctx, span := salesFactTracer.Start(ctx, "service.sales_fact_refresher.drain_dirty")
	defer span.End()

	repo := s.cfg.Repos.NewSalesFactRepo()
	marks, apiErr := repo.ListDirty(ctx, salesFactInvoiceBatch)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	span.SetAttributes(attribute.Int("sales_fact.dirty_marks", len(marks)))
	if len(marks) == 0 {
		return nil
	}
	if len(marks) == salesFactInvoiceBatch {
		s.backlog = true
	}

	type scopeKey struct {
		scope     domain.SalesFactScope
		accountID string
	}
	byScope := make(map[scopeKey][]string)
	restart := false
	for _, m := range marks {
		if m.ScopeType == domain.SalesFactScopeReconcile {
			restart = true
			continue
		}
		k := scopeKey{m.ScopeType, m.AccountID}
		byScope[k] = append(byScope[k], m.ScopeID)
	}
	if restart {
		// A unit or base-unit change can reprice any line; the full pass is the one path that reaches them all.
		if apiErr := repo.RestartReconcile(ctx); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		slog.InfoContext(ctx, "Sales fact refresher: reconcile pass restarted by a unit change")
	}

	var invoiceIDs []string
	resolvedByAccount := make(map[string][]string)
	for k, ids := range byScope {
		resolved, apiErr := repo.ResolveInvoiceIDs(ctx, k.accountID, k.scope, ids)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		invoiceIDs = append(invoiceIDs, resolved...)
		if k.scope != domain.SalesFactScopeInvoice {
			resolvedByAccount[k.accountID] = append(resolvedByAccount[k.accountID], resolved...)
		}
	}
	slices.Sort(invoiceIDs)
	invoiceIDs = slices.Compact(invoiceIDs)
	span.SetAttributes(attribute.Int("sales_fact.resolved_invoices", len(invoiceIDs)))

	if len(invoiceIDs) > salesFactFanOutThreshold {
		// Too much for one tick to hold the lease over: mark each invoice on its own and let later ticks
		// work them off. Invoice marks already are that, so only the broader scopes are fanned out.
		for accountID, ids := range resolvedByAccount {
			if apiErr := repo.MarkInvoicesDirty(ctx, accountID, ids); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
		}
		slog.InfoContext(ctx, "Sales fact refresher: fanned a large change out into invoice marks", "invoices", len(invoiceIDs))
		s.backlog = true
		var broad []domain.SalesFactDirtyMark
		for _, m := range marks {
			if m.ScopeType != domain.SalesFactScopeInvoice {
				broad = append(broad, m)
			}
		}
		return tracing.Trace(span, repo.ClearDirty(ctx, broad))
	}

	if _, apiErr := s.refreshInvoices(ctx, invoiceIDs); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return tracing.Trace(span, repo.ClearDirty(ctx, marks))
}

func (s *SalesFactRefresher) refreshRecent(ctx context.Context, now time.Time) *apierror.APIError {
	ctx, span := salesFactTracer.Start(ctx, "service.sales_fact_refresher.refresh_recent")
	defer span.End()

	// The cap only guards against a pathological burst; a week is normally ~1k invoices.
	ids, apiErr := s.cfg.Repos.NewSalesFactRepo().ListInvoiceIDsCreatedSince(ctx, now.Add(-s.cfg.RecentWindow), 20_000)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	changed, apiErr := s.refreshInvoices(ctx, ids)
	span.SetAttributes(attribute.Int("sales_fact.invoices", len(ids)), attribute.Int("sales_fact.drift_lines", changed))
	if changed > 0 {
		slog.InfoContext(ctx, "Sales fact refresher: rolling recompute corrected facts", "lines", changed)
	}
	return tracing.Trace(span, apiErr)
}

// passDue reports whether a daily pass last started at started should start again: never started, or not since the most recent midnight in PassLocation.
func (s *SalesFactRefresher) passDue(started *time.Time, now time.Time) bool {
	if started == nil {
		return true
	}
	local := now.In(s.cfg.PassLocation)
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.cfg.PassLocation)
	return started.Before(midnight)
}

// reconcile advances the full pass by up to ReconcileBudget, starting a new pass after each midnight in PassLocation.
func (s *SalesFactRefresher) reconcile(ctx context.Context) *apierror.APIError {
	ctx, span := salesFactTracer.Start(ctx, "service.sales_fact_refresher.reconcile")
	defer span.End()

	repo := s.cfg.Repos.NewSalesFactRepo()
	state, apiErr := repo.GetSync(ctx)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	now := s.cfg.Now().UTC()
	if state.Cursor == nil {
		if !s.passDue(state.PassStartedAt, now) {
			return nil
		}
		state.Cursor = &domain.SalesFactInvoiceCursor{CreatedAt: salesFactSweepOrigin}
		state.PassStartedAt = &now
		slog.InfoContext(ctx, "Sales fact refresher: reconcile pass started", "backfill", state.LastCompletedAt == nil)
	}

	deadline := now.Add(s.cfg.ReconcileBudget)
	scanned, drift := 0, 0
	for s.cfg.Now().UTC().Before(deadline) {
		page, apiErr := repo.ListInvoicesAfter(ctx, state.Cursor, salesFactInvoiceBatch)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		ids := make([]string, len(page))
		for i, inv := range page {
			ids[i] = inv.InvoiceID
		}
		changed, apiErr := s.refreshInvoices(ctx, ids)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		scanned += len(page)
		drift += changed

		if len(page) < salesFactInvoiceBatch {
			orphans, apiErr := s.deleteOrphans(ctx)
			if apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
			drift += orphans
			completed := s.cfg.Now().UTC()
			state.Cursor = nil
			state.LastCompletedAt = &completed
			slog.InfoContext(ctx, "Sales fact refresher: reconcile pass completed", "started_at", state.PassStartedAt)
			break
		}
		last := page[len(page)-1]
		state.Cursor = &last
	}
	span.SetAttributes(attribute.Int("sales_fact.invoices", scanned), attribute.Int("sales_fact.drift_lines", drift))
	if state.Cursor != nil {
		s.backlog = true
	}
	if drift > 0 && state.LastCompletedAt != nil {
		slog.InfoContext(ctx, "Sales fact refresher: reconcile corrected facts", "lines", drift)
	}
	return tracing.Trace(span, repo.SaveSync(ctx, *state))
}

// deleteOrphans removes facts whose invoice no longer exists, which the invoice walk cannot reach. Returns how many lines it deleted.
func (s *SalesFactRefresher) deleteOrphans(ctx context.Context) (int, *apierror.APIError) {
	repo := s.cfg.Repos.NewSalesFactRepo()
	const page = 500
	deleted, after := 0, ""
	for {
		ids, apiErr := repo.ListFactInvoiceIDsAfter(ctx, after, page)
		if apiErr != nil {
			return deleted, apiErr
		}
		if len(ids) == 0 {
			return deleted, nil
		}
		existing, apiErr := repo.FilterExistingInvoiceIDs(ctx, ids)
		if apiErr != nil {
			return deleted, apiErr
		}
		live := make(map[string]struct{}, len(existing))
		for _, id := range existing {
			live[id] = struct{}{}
		}
		var orphans []string
		for _, id := range ids {
			if _, ok := live[id]; !ok {
				orphans = append(orphans, id)
			}
		}
		// Recomputing a missing invoice computes nothing, so its stored facts are deleted.
		n, apiErr := s.refreshInvoices(ctx, orphans)
		deleted += n
		if apiErr != nil {
			return deleted, apiErr
		}
		if len(ids) < page {
			return deleted, nil
		}
		after = ids[len(ids)-1]
	}
}

// salesFactSweepOrigin sorts before every invoice; the zero time.Time is below DATETIME's range.
var salesFactSweepOrigin = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

// refreshInvoices recomputes the facts of the given invoices and writes only what differs, then rebuilds
// the rollups of every day a changed line left or entered. Each such day is marked before its facts are
// written and unmarked only once its buckets are rebuilt, so a crash or a failed rebuild in between is
// repaired on the next tick rather than leaving the buckets behind their facts. Returns how many lines
// were written or deleted.
func (s *SalesFactRefresher) refreshInvoices(ctx context.Context, invoiceIDs []string) (int, *apierror.APIError) {
	repo := s.cfg.Repos.NewSalesFactRepo()
	changedLines := 0
	changedAccounts := map[string]struct{}{}
	// Whether buyers are marked, read once, at the first change.
	var tracked *bool
	for start := 0; start < len(invoiceIDs); start += salesFactComputeBatch {
		batch := invoiceIDs[start:min(start+salesFactComputeBatch, len(invoiceIDs))]
		computed, apiErr := repo.ComputeFacts(ctx, batch)
		if apiErr != nil {
			return changedLines, apiErr
		}
		stored, apiErr := repo.GetFacts(ctx, batch)
		if apiErr != nil {
			return changedLines, apiErr
		}

		upserts, deletes := diffSalesFacts(computed, stored)
		if len(upserts) == 0 && len(deletes) == 0 {
			continue
		}
		if apiErr := repo.MarkRollupDays(ctx, touchedRollupDays(upserts, deletes, stored)); apiErr != nil {
			return changedLines, apiErr
		}
		if tracked == nil {
			t, apiErr := s.buyerSummariesTracked(ctx)
			if apiErr != nil {
				return changedLines, apiErr
			}
			tracked = &t
		}
		if *tracked {
			if apiErr := repo.MarkBuyers(ctx, touchedBuyers(upserts, deletes, stored)); apiErr != nil {
				return changedLines, apiErr
			}
		}
		if apiErr := repo.UpsertFacts(ctx, upserts); apiErr != nil {
			return changedLines, apiErr
		}
		deleteIDs := make([]string, len(deletes))
		for i, f := range deletes {
			deleteIDs[i] = f.InvoiceLineID
		}
		if apiErr := repo.DeleteFacts(ctx, deleteIDs); apiErr != nil {
			return changedLines, apiErr
		}
		changedLines += len(upserts) + len(deletes)
		for _, f := range append(upserts, deletes...) {
			changedAccounts[f.AccountID] = struct{}{}
		}
	}
	if len(changedAccounts) > 0 {
		accounts := make([]string, 0, len(changedAccounts))
		for id := range changedAccounts {
			accounts = append(accounts, id)
		}
		// Reports that read the facts alone are stale now, whether or not the rollups rebuild below.
		s.cfg.OnFactsChanged(ctx, accounts)
	}
	if apiErr := s.drainRollupDirty(ctx); apiErr != nil {
		return changedLines, apiErr
	}
	return changedLines, s.drainBuyerDirty(ctx)
}

// buyerSummariesTracked reports whether changed facts must mark their buyers: once the buyer summary sweep
// has started its first pass. Before that the summaries are not served, and that pass rebuilds every buyer
// anyway, so marks would only rebuild buyers again and again while the fact backfill that precedes it
// rewrites every fact.
func (s *SalesFactRefresher) buyerSummariesTracked(ctx context.Context) (bool, *apierror.APIError) {
	state, apiErr := s.cfg.Repos.NewSalesFactRepo().GetBuyerSummarySync(ctx)
	if apiErr != nil {
		return false, apiErr
	}
	return state.PassStartedAt != nil, nil
}

// touchedBuyers returns every buyer a changed line belongs to, and for a line that changed buyer, the
// buyer it left.
func touchedBuyers(upserts, deletes, stored []domain.SalesLineFact) []domain.SalesBuyerKey {
	storedByLine := make(map[string]domain.SalesLineFact, len(stored))
	for _, f := range stored {
		storedByLine[f.InvoiceLineID] = f
	}
	seen := map[domain.SalesBuyerKey]struct{}{}
	var buyers []domain.SalesBuyerKey
	add := func(f domain.SalesLineFact) {
		k := domain.SalesBuyerKey{AccountID: f.AccountID, BuyerAccountID: f.BuyerAccountID}
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			buyers = append(buyers, k)
		}
	}
	for _, f := range upserts {
		add(f)
		if old, ok := storedByLine[f.InvoiceLineID]; ok {
			add(old)
		}
	}
	for _, f := range deletes {
		add(f)
	}
	return buyers
}

// drainBuyerDirty rebuilds a batch of marked buyers' summaries and clears each mark that was not
// re-marked meanwhile.
func (s *SalesFactRefresher) drainBuyerDirty(ctx context.Context) *apierror.APIError {
	ctx, span := salesFactTracer.Start(ctx, "service.sales_fact_refresher.drain_buyer_dirty")
	defer span.End()

	repo := s.cfg.Repos.NewSalesFactRepo()
	marks, apiErr := repo.ListBuyerDirty(ctx, salesBuyerDrainBatch)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	span.SetAttributes(attribute.Int("sales_fact.buyer_marks", len(marks)))
	if len(marks) == 0 {
		return nil
	}
	if len(marks) == salesBuyerDrainBatch {
		s.backlog = true
	}
	byAccount := map[string][]string{}
	for _, m := range marks {
		byAccount[m.Buyer.AccountID] = append(byAccount[m.Buyer.AccountID], m.Buyer.BuyerAccountID)
	}
	accounts := make([]string, 0, len(byAccount))
	for account, buyers := range byAccount {
		if apiErr := repo.RebuildBuyerSummaries(ctx, account, buyers); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		accounts = append(accounts, account)
	}
	if apiErr := repo.ClearBuyerDirty(ctx, marks); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	s.cfg.OnFactsChanged(ctx, accounts)
	return nil
}

// sweepBuyers advances the buyer summary pass by up to ReconcileBudget, rebuilding every buyer with facts
// in (account, buyer) order, then deleting the summaries of buyers the pass never reached (they have no
// facts left). Its first pass is the backfill. It needs every fact's order date and price flag, which
// facts written before those columns lack, so before it the sweep restarts the fact reconcile (which
// rewrites each fact that differs from what it computes) and waits for that pass to complete. Later
// passes start after each midnight in PassLocation and repair any summary a crash left behind its facts.
func (s *SalesFactRefresher) sweepBuyers(ctx context.Context) *apierror.APIError {
	ctx, span := salesFactTracer.Start(ctx, "service.sales_fact_refresher.sweep_buyers")
	defer span.End()

	repo := s.cfg.Repos.NewSalesFactRepo()
	state, apiErr := repo.GetBuyerSummarySync(ctx)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	now := s.cfg.Now().UTC()
	if state.FactsSince == nil {
		if apiErr := repo.RestartReconcile(ctx); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		state.FactsSince = &now
		slog.InfoContext(ctx, "Sales fact refresher: fact reconcile restarted to fill order dates for the buyer summaries")
		return tracing.Trace(span, repo.SaveBuyerSummarySync(ctx, *state))
	}
	if state.LastCompletedAt == nil && state.Cursor == nil {
		facts, apiErr := repo.GetSync(ctx)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		if facts.LastCompletedAt == nil || !facts.LastCompletedAt.After(*state.FactsSince) {
			return nil
		}
	}
	if state.Cursor == nil {
		if !s.passDue(state.PassStartedAt, now) {
			return nil
		}
		state.Cursor = &domain.SalesBuyerKey{}
		state.PassStartedAt = &now
		slog.InfoContext(ctx, "Sales fact refresher: buyer summary pass started", "backfill", state.LastCompletedAt == nil)
	}

	deadline := now.Add(s.cfg.ReconcileBudget)
	rebuilt := 0
	for s.cfg.Now().UTC().Before(deadline) {
		next, apiErr := repo.NextBuyers(ctx, *state.Cursor, salesBuyerSweepBatch)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		if len(next) == 0 {
			if apiErr := repo.DeleteBuyerSummariesRefreshedBefore(ctx, *state.PassStartedAt); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
			completed := s.cfg.Now().UTC()
			state.Cursor = nil
			state.LastCompletedAt = &completed
			slog.InfoContext(ctx, "Sales fact refresher: buyer summary pass completed", "started_at", state.PassStartedAt)
			break
		}
		byAccount := map[string][]string{}
		var order []string
		for _, k := range next {
			if _, ok := byAccount[k.AccountID]; !ok {
				order = append(order, k.AccountID)
			}
			byAccount[k.AccountID] = append(byAccount[k.AccountID], k.BuyerAccountID)
		}
		for _, account := range order {
			if apiErr := repo.RebuildBuyerSummaries(ctx, account, byAccount[account]); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
		}
		rebuilt += len(next)
		last := next[len(next)-1]
		state.Cursor = &last
	}
	span.SetAttributes(attribute.Int("sales_fact.buyers_rebuilt", rebuilt))
	if state.Cursor != nil {
		s.backlog = true
	}
	return tracing.Trace(span, repo.SaveBuyerSummarySync(ctx, *state))
}

// touchedRollupDays returns the (account, UTC day) of every changed line, and for a line that moved,
// the day it left.
func touchedRollupDays(upserts, deletes, stored []domain.SalesLineFact) []domain.SalesRollupDay {
	storedByLine := make(map[string]domain.SalesLineFact, len(stored))
	for _, f := range stored {
		storedByLine[f.InvoiceLineID] = f
	}
	seen := map[domain.SalesRollupDay]struct{}{}
	var days []domain.SalesRollupDay
	add := func(f domain.SalesLineFact) {
		d := domain.SalesRollupDay{AccountID: f.AccountID, Day: utcDay(f.InvoicedAt)}
		if _, ok := seen[d]; !ok {
			seen[d] = struct{}{}
			days = append(days, d)
		}
	}
	for _, f := range upserts {
		old, ok := storedByLine[f.InvoiceLineID]
		if ok && rollupFieldsEqual(old, f) {
			continue
		}
		add(f)
		if ok {
			add(old)
		}
	}
	for _, f := range deletes {
		add(f)
	}
	return days
}

// drainRollupDirty rebuilds a batch of marked rollup days and clears each mark that was not re-marked
// meanwhile.
func (s *SalesFactRefresher) drainRollupDirty(ctx context.Context) *apierror.APIError {
	ctx, span := salesFactTracer.Start(ctx, "service.sales_fact_refresher.drain_rollup_dirty")
	defer span.End()

	repo := s.cfg.Repos.NewSalesFactRepo()
	marks, apiErr := repo.ListRollupDirty(ctx, salesRollupDrainBatch)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	span.SetAttributes(attribute.Int("sales_fact.rollup_marks", len(marks)))
	if len(marks) == 0 {
		return nil
	}
	if len(marks) == salesRollupDrainBatch {
		s.backlog = true
	}
	days := make(map[domain.SalesRollupDay]struct{}, len(marks))
	accounts := map[string]struct{}{}
	for _, m := range marks {
		days[m.Day] = struct{}{}
		accounts[m.Day.AccountID] = struct{}{}
	}
	if apiErr := s.rebuildRollups(ctx, days); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := repo.ClearRollupDirty(ctx, marks); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	ids := make([]string, 0, len(accounts))
	for id := range accounts {
		ids = append(ids, id)
	}
	s.cfg.OnFactsChanged(ctx, ids)
	return nil
}

func (s *SalesFactRefresher) rebuildRollups(ctx context.Context, days map[domain.SalesRollupDay]struct{}) *apierror.APIError {
	repo := s.cfg.Repos.NewSalesFactRepo()
	for d := range days {
		if apiErr := repo.RebuildRollupDay(ctx, d); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// sweepRollups advances the rollup pass by up to ReconcileBudget, rebuilding every (account, day) in order. Its first pass is the backfill; later passes, started after each midnight in PassLocation, repair any bucket a crash left behind its facts and write nothing for the rest.
func (s *SalesFactRefresher) sweepRollups(ctx context.Context) *apierror.APIError {
	ctx, span := salesFactTracer.Start(ctx, "service.sales_fact_refresher.sweep_rollups")
	defer span.End()

	repo := s.cfg.Repos.NewSalesFactRepo()
	state, apiErr := repo.GetRollupSync(ctx)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	now := s.cfg.Now().UTC()
	if state.Cursor == nil {
		if !s.passDue(state.PassStartedAt, now) {
			return nil
		}
		state.Cursor = &domain.SalesRollupDay{Day: salesFactSweepOrigin}
		state.PassStartedAt = &now
		slog.InfoContext(ctx, "Sales fact refresher: rollup pass started", "backfill", state.LastCompletedAt == nil)
	}

	deadline := now.Add(s.cfg.ReconcileBudget)
	days := 0
	for s.cfg.Now().UTC().Before(deadline) {
		next, apiErr := repo.NextRollupDay(ctx, *state.Cursor)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		if next == nil {
			completed := s.cfg.Now().UTC()
			state.Cursor = nil
			state.LastCompletedAt = &completed
			slog.InfoContext(ctx, "Sales fact refresher: rollup pass completed", "started_at", state.PassStartedAt)
			break
		}
		days++
		if apiErr := repo.RebuildRollupDay(ctx, *next); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		state.Cursor = &domain.SalesRollupDay{AccountID: next.AccountID, Day: next.Day.AddDate(0, 0, 1)}
	}
	span.SetAttributes(attribute.Int("sales_fact.rollup_days", days))
	if state.Cursor != nil {
		s.backlog = true
	}
	return tracing.Trace(span, repo.SaveRollupSync(ctx, *state))
}

func utcDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// diffSalesFacts returns the computed facts that are new or differ from what is stored, and the stored facts no longer computed.
func diffSalesFacts(computed, stored []domain.SalesLineFact) (upserts, deletes []domain.SalesLineFact) {
	storedByLine := make(map[string]domain.SalesLineFact, len(stored))
	for _, f := range stored {
		storedByLine[f.InvoiceLineID] = f
	}
	for _, f := range computed {
		old, ok := storedByLine[f.InvoiceLineID]
		delete(storedByLine, f.InvoiceLineID)
		if !ok || !salesFactsEqual(old, f) {
			upserts = append(upserts, f)
		}
	}
	for _, f := range stored {
		if _, gone := storedByLine[f.InvoiceLineID]; gone {
			deletes = append(deletes, f)
		}
	}
	return upserts, deletes
}

func salesFactsEqual(a, b domain.SalesLineFact) bool {
	return a.AccountID == b.AccountID &&
		a.InvoicedAt.Equal(b.InvoicedAt) &&
		a.InvoiceID == b.InvoiceID &&
		a.SalesOrderID == b.SalesOrderID &&
		a.SalesOrderTypeCode == b.SalesOrderTypeCode &&
		a.BuyerAccountID == b.BuyerAccountID &&
		ptrEqual(a.SalesRepID, b.SalesRepID) &&
		ptrEqual(a.OrderDiscountID, b.OrderDiscountID) &&
		a.ProductID == b.ProductID &&
		a.ItemID == b.ItemID &&
		a.ProductLineID == b.ProductLineID &&
		ptrEqual(a.QuantityBase, b.QuantityBase) &&
		ptrEqual(a.TotalInvoiced, b.TotalInvoiced) &&
		ptrEqual(a.TotalCost, b.TotalCost) &&
		timePtrEqual(a.OrderedAt, b.OrderedAt) &&
		a.IsPriced == b.IsPriced
}

// rollupFieldsEqual is whether two facts put the same amounts in the same rollup buckets. The order date
// and price flag feed only the buyer summaries, so a change to them alone (their backfill included)
// leaves the rollups as they are.
func rollupFieldsEqual(a, b domain.SalesLineFact) bool {
	a.OrderedAt, a.IsPriced = b.OrderedAt, b.IsPriced
	return salesFactsEqual(a, b)
}

func timePtrEqual(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func ptrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// SalesFactMarker turns audit events into dirty marks for the refresher.
type SalesFactMarker struct {
	repos    domain.RepoFactory
	onMarked func()
}

// NewSalesFactMarker returns a marker that calls onMarked (nil for none) after each mark it writes, to wake the refresher.
func NewSalesFactMarker(repos domain.RepoFactory, onMarked func()) *SalesFactMarker {
	if onMarked == nil {
		onMarked = func() {}
	}
	return &SalesFactMarker{repos: repos, onMarked: onMarked}
}

// HandleAuditEvent marks the scope an event can change facts for. Every replica receives every event, so a mark is written once per replica; marking is idempotent.
func (m *SalesFactMarker) HandleAuditEvent(ctx context.Context, e audit.ObservedEvent) {
	scope, scopeID, ok := salesFactScopeFor(e)
	if !ok || e.AccountID == "" || scopeID == "" {
		return
	}
	if apiErr := m.repos.NewSalesFactRepo().MarkDirty(ctx, scope, scopeID, e.AccountID); apiErr != nil {
		// The rolling recompute and reconcile pass still pick the change up, just later.
		slog.WarnContext(ctx, "Sales fact marker: failed to mark scope", "scope", scope, "id", scopeID, "error", apiErr)
		return
	}
	m.onMarked()
}

// salesFactScopeFor maps an audit event to the scope whose facts it can change, and the scope id to mark.
func salesFactScopeFor(e audit.ObservedEvent) (domain.SalesFactScope, string, bool) {
	changed := func(fields ...string) bool {
		for _, c := range e.ChangedFields {
			for _, f := range fields {
				if c == f || strings.HasPrefix(c, f+".") {
					return true
				}
			}
		}
		return false
	}
	update := e.Action == constants.AuditActionUpdate
	switch e.ResourceType {
	case constants.ObjectTypeInvoice:
		return domain.SalesFactScopeInvoice, e.ResourceID, true
	case constants.ObjectTypeSalesOrder:
		return domain.SalesFactScopeSalesOrder, e.ResourceID, true
	case constants.ObjectTypeSalesOrderLine:
		return domain.SalesFactScopeSalesOrderLine, e.ResourceID, true
	case constants.ObjectTypeProduct:
		// Every other product edit leaves facts alone, and a product can sit on thousands of invoices.
		return domain.SalesFactScopeProduct, e.ResourceID, changed("product_line_id")
	case constants.ObjectTypeQuantity:
		// An invoice line's quantity edited on its own, through the generic quantity endpoint.
		return domain.SalesFactScopeQuantity, e.ResourceID, update
	case constants.ObjectTypeRate:
		// An order line's price or cost edited on its own, through the generic rate endpoint.
		return domain.SalesFactScopeRate, e.ResourceID, update
	case constants.ObjectTypeItem:
		// A new category can mean a new base unit, and so a new base quantity on every line.
		return domain.SalesFactScopeItem, e.ResourceID, update && changed("item_category_id")
	case constants.ObjectTypeCustomer:
		// A merge deletes the customers merged away; their orders now belong to another buyer.
		return domain.SalesFactScopeBuyer, e.ResourceID, e.Action == constants.AuditActionDelete
	case constants.ObjectTypeUnit:
		return domain.SalesFactScopeReconcile, salesFactReconcileScopeID,
			update && changed("ratio_numerator", "ratio_denominator", "offset_numerator", "offset_denominator")
	case constants.ObjectTypeUnitGroup:
		return domain.SalesFactScopeReconcile, salesFactReconcileScopeID, update && changed("base_unit")
	case constants.ObjectTypeItemCategory:
		return domain.SalesFactScopeReconcile, salesFactReconcileScopeID, update && changed("unit_group_id")
	}
	return "", "", false
}
