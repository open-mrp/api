package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

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

	// salesFactInvoiceBatch bounds one recompute: ~4 lines an invoice keeps the pricing join under ~1k rows.
	salesFactInvoiceBatch = 200
)

// SalesFactRefresherConfig configures the refresher that keeps sales_line_fact equal to its source tables.
type SalesFactRefresherConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// Lease (required) keeps the refresh to one pod at a time.
	Lease *lease.Lease

	// OnFactsChanged (optional; default: no-op) is called with the accounts whose facts a refresh rewrote, so cached reports built from the old facts can be dropped.
	OnFactsChanged func(ctx context.Context, accountIDs []string)

	// TickInterval (optional; default: 5s) is how often dirty marks are drained. Zero or negative values are treated as unset.
	TickInterval time.Duration

	// RecentWindow (optional; default: 7d) is how far back the rolling recompute reaches. It catches writes that mark nothing, such as the dashboard's. Zero or negative values are treated as unset.
	RecentWindow time.Duration

	// RecentInterval (optional; default: 2m) is how often the rolling recompute runs. Zero or negative values are treated as unset.
	RecentInterval time.Duration

	// ReconcileEvery (optional; default: 24h) is how long after a full reconcile pass starts the next one does. Zero or negative values are treated as unset.
	ReconcileEvery time.Duration

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
	if c.TickInterval <= 0 {
		c.TickInterval = 5 * time.Second
	}
	if c.RecentWindow <= 0 {
		c.RecentWindow = 7 * 24 * time.Hour
	}
	if c.RecentInterval <= 0 {
		c.RecentInterval = 2 * time.Minute
	}
	if c.ReconcileEvery <= 0 {
		c.ReconcileEvery = 24 * time.Hour
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

// SalesFactRefresher keeps sales_line_fact in step with invoices. Three paths feed it, each covering what the one before misses:
//   - dirty marks, written from audit events within seconds of an edit made through this API;
//   - a rolling recompute of recent invoices, for writes that publish no audit event (the dashboard, generic rate and quantity edits);
//   - a daily reconcile pass over every invoice, which corrects anything older that drifted. Its first pass is the backfill.
type SalesFactRefresher struct {
	cfg *SalesFactRefresherConfig

	lastRecent time.Time

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewSalesFactRefresher(config *SalesFactRefresherConfig) *SalesFactRefresher {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &SalesFactRefresher{cfg: config, stopCh: make(chan struct{})}
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
	ticker := time.NewTicker(s.cfg.TickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			_ = s.cfg.Lease.WithLease(ctx, salesFactLeaseName, salesFactLeaseTTL, func(leaseCtx context.Context) error {
				s.Tick(leaseCtx)
				return nil
			})
		}
	}
}

// Tick runs one round of every path. Exported for tests; production calls it under the lease.
func (s *SalesFactRefresher) Tick(ctx context.Context) {
	if apiErr := s.drainDirty(ctx); apiErr != nil {
		slog.ErrorContext(ctx, "Sales fact refresher: draining dirty marks failed", "error", apiErr)
	}
	if now := s.cfg.Now(); now.Sub(s.lastRecent) >= s.cfg.RecentInterval {
		if apiErr := s.refreshRecent(ctx, now); apiErr != nil {
			slog.ErrorContext(ctx, "Sales fact refresher: rolling recompute failed", "error", apiErr)
		} else {
			s.lastRecent = now
		}
	}
	if apiErr := s.reconcile(ctx); apiErr != nil {
		slog.ErrorContext(ctx, "Sales fact refresher: reconcile failed", "error", apiErr)
	}
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

	byScope := make(map[domain.SalesFactScope][]string)
	for _, m := range marks {
		byScope[m.ScopeType] = append(byScope[m.ScopeType], m.ScopeID)
	}
	var invoiceIDs []string
	for scope, ids := range byScope {
		resolved, apiErr := repo.ResolveInvoiceIDs(ctx, scope, ids)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		invoiceIDs = append(invoiceIDs, resolved...)
	}
	slices.Sort(invoiceIDs)
	invoiceIDs = slices.Compact(invoiceIDs)

	if _, apiErr := s.refreshInvoices(ctx, invoiceIDs); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	for _, m := range marks {
		if apiErr := repo.ClearDirty(ctx, m); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
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

// reconcile advances the full pass by up to ReconcileBudget, starting a new pass once ReconcileEvery has passed since the last began.
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
		if state.PassStartedAt != nil && now.Sub(*state.PassStartedAt) < s.cfg.ReconcileEvery {
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
	if drift > 0 && state.LastCompletedAt != nil {
		slog.InfoContext(ctx, "Sales fact refresher: reconcile corrected facts", "lines", drift)
	}
	return tracing.Trace(span, repo.SaveSync(ctx, *state))
}

// deleteOrphans removes facts whose invoice no longer exists, which the invoice walk cannot reach. Returns how many lines it deleted.
func (s *SalesFactRefresher) deleteOrphans(ctx context.Context) (int, *apierror.APIError) {
	repo := s.cfg.Repos.NewSalesFactRepo()
	const page = 2000
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

// refreshInvoices recomputes the facts of the given invoices and writes only what differs. Returns how many lines were written or deleted.
func (s *SalesFactRefresher) refreshInvoices(ctx context.Context, invoiceIDs []string) (int, *apierror.APIError) {
	repo := s.cfg.Repos.NewSalesFactRepo()
	changedLines := 0
	changedAccounts := map[string]struct{}{}
	for start := 0; start < len(invoiceIDs); start += salesFactInvoiceBatch {
		batch := invoiceIDs[start:min(start+salesFactInvoiceBatch, len(invoiceIDs))]
		computed, apiErr := repo.ComputeFacts(ctx, batch)
		if apiErr != nil {
			return changedLines, apiErr
		}
		stored, apiErr := repo.GetFacts(ctx, batch)
		if apiErr != nil {
			return changedLines, apiErr
		}

		upserts, deletes := diffSalesFacts(computed, stored)
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
		for _, f := range upserts {
			changedAccounts[f.AccountID] = struct{}{}
		}
		for _, f := range deletes {
			changedAccounts[f.AccountID] = struct{}{}
		}
	}
	if len(changedAccounts) > 0 {
		accounts := make([]string, 0, len(changedAccounts))
		for id := range changedAccounts {
			accounts = append(accounts, id)
		}
		s.cfg.OnFactsChanged(ctx, accounts)
	}
	return changedLines, nil
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
		ptrEqual(a.TotalCost, b.TotalCost)
}

func ptrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// SalesFactMarker turns audit events into dirty marks for the refresher.
type SalesFactMarker struct {
	repos domain.RepoFactory
}

func NewSalesFactMarker(repos domain.RepoFactory) *SalesFactMarker {
	return &SalesFactMarker{repos: repos}
}

// HandleAuditEvent marks the scope an event can change facts for. Every replica receives every event, so a mark is written once per replica; marking is idempotent.
func (m *SalesFactMarker) HandleAuditEvent(ctx context.Context, e audit.ObservedEvent) {
	scope, ok := salesFactScopeFor(e)
	if !ok || e.AccountID == "" || e.ResourceID == "" {
		return
	}
	if apiErr := m.repos.NewSalesFactRepo().MarkDirty(ctx, scope, e.ResourceID, e.AccountID); apiErr != nil {
		// The rolling recompute and reconcile pass still pick the change up, just later.
		slog.WarnContext(ctx, "Sales fact marker: failed to mark scope", "scope", scope, "id", e.ResourceID, "error", apiErr)
	}
}

func salesFactScopeFor(e audit.ObservedEvent) (domain.SalesFactScope, bool) {
	switch e.ResourceType {
	case constants.ObjectTypeInvoice:
		return domain.SalesFactScopeInvoice, true
	case constants.ObjectTypeSalesOrder:
		return domain.SalesFactScopeSalesOrder, true
	case constants.ObjectTypeSalesOrderLine:
		return domain.SalesFactScopeSalesOrderLine, true
	case constants.ObjectTypeProduct:
		// Every other product edit leaves facts alone, and a product can sit on thousands of invoices.
		return domain.SalesFactScopeProduct, slices.Contains(e.ChangedFields, "product_line_id")
	}
	return "", false
}
