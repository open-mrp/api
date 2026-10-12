package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
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

var deliveryFactTracer = tracing.GetTracer("core-service.delivery_fact_refresher")

const (
	deliveryFactLeaseName = "delivery-fact-refresher"
	deliveryFactLeaseTTL  = time.Minute

	// deliveryFactMarkBatch is how many dirty marks one drain lists.
	deliveryFactMarkBatch = 200

	// deliveryFactOrderPage is how many orders one reconcile page lists.
	deliveryFactOrderPage = 200

	// deliveryFactFanOutThreshold is the most orders one drain refreshes itself. A scope that resolves to
	// more (a product moved to another product line after years of sales) is fanned out into one mark per
	// order, which later ticks work off a batch at a time.
	deliveryFactFanOutThreshold = 1000

	// deliveryFactTargetStatement is the duration a recompute is steered toward: half the 50ms budget,
	// since a batch of orders whose lines are cold costs far more than the same batch warm.
	deliveryFactTargetStatement = 25 * time.Millisecond

	// deliveryFactMaxBatch caps the orders one recompute reads.
	deliveryFactMaxBatch = 100
)

// DeliveryFactRefresherConfig configures the refresher that keeps delivery_order_fact equal to its source tables.
type DeliveryFactRefresherConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// TxManager (required) writes each batch's facts in one transaction, so a reader never sees an order half rewritten.
	TxManager TransactionManager

	// Lease (required) keeps the refresh to one pod at a time.
	Lease *lease.Lease

	// LeaseName (optional; default: "delivery-fact-refresher") names the lease, for tests that must not contend with a running service.
	LeaseName string

	// OnFactsChanged (optional; default: no-op) is called with the accounts whose facts a refresh rewrote, so cached reports built from the old facts can be dropped.
	OnFactsChanged func(ctx context.Context, accountIDs []string)

	// TickInterval (optional; default: 5s) is how soon the refresher runs again while work is left over. Zero or negative values are treated as unset.
	TickInterval time.Duration

	// IdleInterval (optional; default: 1m) is how often the refresher runs with nothing to wake it. Zero or negative values are treated as unset.
	IdleInterval time.Duration

	// WakeDelay (optional; default: 1s) is how long the refresher waits after a mark wakes it, so a burst of edits is drained in one run. Zero or negative values are treated as unset.
	WakeDelay time.Duration

	// PassLocation (optional; default: America/New_York) is the timezone whose midnight starts each daily reconcile pass.
	PassLocation *time.Location

	// ReconcileBudget (optional; default: 1s) caps the reconcile work done in one tick, so a pass is spread across ticks instead of loading the database in one burst. Zero or negative values are treated as unset.
	ReconcileBudget time.Duration

	// Now (optional; default: time.Now) is the clock, for tests.
	Now func() time.Time
}

// WithDefaults returns the config with default values applied where unset.
func (c *DeliveryFactRefresherConfig) WithDefaults() *DeliveryFactRefresherConfig {
	if c == nil {
		c = &DeliveryFactRefresherConfig{}
	}
	if c.LeaseName == "" {
		c.LeaseName = deliveryFactLeaseName
	}
	if c.OnFactsChanged == nil {
		c.OnFactsChanged = func(context.Context, []string) {}
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
	if c.PassLocation == nil {
		loc, err := time.LoadLocation(salesFactPassTZ)
		if err != nil {
			slog.Warn("Delivery fact refresher: could not load pass timezone, falling back to UTC", "tz", salesFactPassTZ, "error", err)
			loc = time.UTC
		}
		c.PassLocation = loc
	}
	if c.ReconcileBudget <= 0 {
		c.ReconcileBudget = time.Second
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return c
}

func (c *DeliveryFactRefresherConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("delivery fact refresher: repos is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("delivery fact refresher: tx manager is required")
	}
	if c.Lease == nil {
		return fmt.Errorf("delivery fact refresher: lease is required")
	}
	return nil
}

// DeliveryFactRefresher keeps delivery_order_fact in step with the orders it sums. Two paths feed it:
//   - dirty marks, written from audit events within seconds of an edit;
//   - a reconcile pass over every sales order each night after midnight in PassLocation, which corrects
//     anything a mark missed. Its first pass is the backfill, and the report reads the facts only once
//     one has completed.
//
// Recomputes are steered toward deliveryFactTargetStatement: a batch that ran long is followed by a smaller one.
type DeliveryFactRefresher struct {
	cfg *DeliveryFactRefresherConfig

	// batch is how many orders the next recompute reads.
	batch int
	// backlog is set during a Tick by any step that left work for a later run.
	backlog bool

	wake   chan struct{}
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewDeliveryFactRefresher(config *DeliveryFactRefresherConfig) *DeliveryFactRefresher {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &DeliveryFactRefresher{cfg: config, batch: 10, wake: make(chan struct{}, 1), stopCh: make(chan struct{})}
}

// Wake asks for a run within WakeDelay. It never blocks: wakes that arrive before the run are served by it.
func (s *DeliveryFactRefresher) Wake() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *DeliveryFactRefresher) Start(ctx context.Context) error {
	s.wg.Add(1)
	go s.loop(ctx)
	slog.Info("Delivery fact refresher started", "tick", s.cfg.TickInterval)
	return nil
}

func (s *DeliveryFactRefresher) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	slog.Info("Delivery fact refresher stopped")
}

func (s *DeliveryFactRefresher) loop(ctx context.Context) {
	defer s.wg.Done()
	timer := time.NewTimer(s.cfg.TickInterval)
	defer timer.Stop()
	due := time.Now().Add(s.cfg.TickInterval)
	// woken is set by a wake until a run serves it; a run another pod's lease kept from this one has not.
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
			_ = s.cfg.Lease.WithLease(ctx, s.cfg.LeaseName, deliveryFactLeaseTTL, func(leaseCtx context.Context) error {
				ran = true
				more = s.Tick(leaseCtx)
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

// Tick drains marks and advances the reconcile pass, and reports whether it left work for a later run.
// Exported for tests; production calls it under the lease.
func (s *DeliveryFactRefresher) Tick(ctx context.Context) bool {
	s.backlog = false
	if apiErr := s.drainDirty(ctx); apiErr != nil {
		s.backlog = true
		slog.ErrorContext(ctx, "Delivery fact refresher: draining dirty marks failed", "error", apiErr)
	}
	if apiErr := s.reconcile(ctx); apiErr != nil {
		s.backlog = true
		slog.ErrorContext(ctx, "Delivery fact refresher: reconcile failed", "error", apiErr)
	}
	return s.backlog
}

func (s *DeliveryFactRefresher) drainDirty(ctx context.Context) *apierror.APIError {
	ctx, span := deliveryFactTracer.Start(ctx, "service.delivery_fact_refresher.drain_dirty")
	defer span.End()

	repo := s.cfg.Repos.NewDeliveryFactRepo()
	marks, apiErr := repo.ListDirty(ctx, deliveryFactMarkBatch)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	span.SetAttributes(attribute.Int("delivery_fact.dirty_marks", len(marks)))
	if len(marks) == 0 {
		return nil
	}
	if len(marks) == deliveryFactMarkBatch {
		s.backlog = true
	}

	type scopeKey struct {
		scope     domain.DeliveryFactScope
		accountID string
	}
	byScope := make(map[scopeKey][]string)
	for _, m := range marks {
		if m.ScopeType == domain.DeliveryFactScopeReconcile {
			if apiErr := repo.RestartReconcile(ctx); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
			slog.InfoContext(ctx, "Delivery fact refresher: reconcile pass restarted by a customer merge")
			continue
		}
		k := scopeKey{m.ScopeType, m.AccountID}
		byScope[k] = append(byScope[k], m.ScopeID)
	}

	var orderIDs []string
	resolvedByAccount := make(map[string][]string)
	for k, ids := range byScope {
		resolved, apiErr := repo.ResolveSalesOrderIDs(ctx, k.scope, ids, deliveryFactFanOutThreshold)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		orderIDs = append(orderIDs, resolved...)
		if k.scope != domain.DeliveryFactScopeSalesOrder {
			resolvedByAccount[k.accountID] = append(resolvedByAccount[k.accountID], resolved...)
		}
	}
	slices.Sort(orderIDs)
	orderIDs = slices.Compact(orderIDs)
	span.SetAttributes(attribute.Int("delivery_fact.resolved_orders", len(orderIDs)))

	if len(orderIDs) > deliveryFactFanOutThreshold {
		// A scope this broad may reach more orders than were resolved; the reconcile pass is the one path
		// sure to reach them all, so it restarts, and the orders resolved so far are marked meanwhile.
		for accountID, ids := range resolvedByAccount {
			if apiErr := repo.MarkSalesOrdersDirty(ctx, accountID, ids); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
		}
		if apiErr := repo.RestartReconcile(ctx); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		slog.InfoContext(ctx, "Delivery fact refresher: fanned a large change out into order marks", "orders", len(orderIDs))
		s.backlog = true
		var broad []domain.DeliveryFactDirtyMark
		for _, m := range marks {
			if m.ScopeType != domain.DeliveryFactScopeSalesOrder {
				broad = append(broad, m)
			}
		}
		return tracing.Trace(span, repo.ClearDirty(ctx, broad))
	}

	if _, apiErr := s.refreshOrders(ctx, orderIDs); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return tracing.Trace(span, repo.ClearDirty(ctx, marks))
}

// passDue reports whether a pass last started at started should start again: never started, or not since the most recent midnight in PassLocation.
func (s *DeliveryFactRefresher) passDue(started *time.Time, now time.Time) bool {
	if started == nil {
		return true
	}
	local := now.In(s.cfg.PassLocation)
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.cfg.PassLocation)
	return started.Before(midnight)
}

// reconcile advances the full pass by up to ReconcileBudget, starting a new pass after each midnight in PassLocation.
func (s *DeliveryFactRefresher) reconcile(ctx context.Context) *apierror.APIError {
	ctx, span := deliveryFactTracer.Start(ctx, "service.delivery_fact_refresher.reconcile")
	defer span.End()

	repo := s.cfg.Repos.NewDeliveryFactRepo()
	state, apiErr := repo.GetSync(ctx)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	now := s.cfg.Now().UTC()
	if state.CursorSalesOrderID == nil {
		if !s.passDue(state.PassStartedAt, now) {
			return nil
		}
		start := ""
		state.CursorSalesOrderID, state.PassStartedAt = &start, &now
		slog.InfoContext(ctx, "Delivery fact refresher: reconcile pass started", "backfill", state.LastCompletedAt == nil)
	}

	deadline := now.Add(s.cfg.ReconcileBudget)
	scanned, drift := 0, 0
	for s.cfg.Now().UTC().Before(deadline) {
		page, apiErr := repo.ListSalesOrderIDsAfter(ctx, *state.CursorSalesOrderID, deliveryFactOrderPage)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		changed, apiErr := s.refreshOrders(ctx, page)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		scanned += len(page)
		drift += changed

		if len(page) < deliveryFactOrderPage {
			orphans, apiErr := s.deleteOrphans(ctx)
			if apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
			drift += orphans
			completed := s.cfg.Now().UTC()
			state.CursorSalesOrderID, state.LastCompletedAt = nil, &completed
			slog.InfoContext(ctx, "Delivery fact refresher: reconcile pass completed", "started_at", state.PassStartedAt)
			break
		}
		last := page[len(page)-1]
		state.CursorSalesOrderID = &last
	}
	span.SetAttributes(attribute.Int("delivery_fact.orders", scanned), attribute.Int("delivery_fact.drift_rows", drift))
	if state.CursorSalesOrderID != nil {
		s.backlog = true
	}
	if drift > 0 && state.LastCompletedAt != nil {
		slog.InfoContext(ctx, "Delivery fact refresher: reconcile corrected facts", "rows", drift)
	}
	return tracing.Trace(span, repo.SaveSync(ctx, *state))
}

// deleteOrphans removes facts whose order no longer exists, which the order walk cannot reach. Returns how many rows it deleted.
func (s *DeliveryFactRefresher) deleteOrphans(ctx context.Context) (int, *apierror.APIError) {
	repo := s.cfg.Repos.NewDeliveryFactRepo()
	const page = 500
	deleted, after := 0, ""
	for {
		ids, apiErr := repo.ListFactSalesOrderIDsAfter(ctx, after, page)
		if apiErr != nil || len(ids) == 0 {
			return deleted, apiErr
		}
		existing, apiErr := repo.FilterExistingSalesOrderIDs(ctx, ids)
		if apiErr != nil {
			return deleted, apiErr
		}
		var orphans []string
		for _, id := range ids {
			if !slices.Contains(existing, id) {
				orphans = append(orphans, id)
			}
		}
		// Recomputing a missing order computes nothing, so its stored facts are deleted.
		n, apiErr := s.refreshOrders(ctx, orphans)
		deleted += n
		if apiErr != nil || len(ids) < page {
			return deleted, apiErr
		}
		after = ids[len(ids)-1]
	}
}

// refreshOrders recomputes the facts of the given orders and writes only what differs, each batch in one
// transaction. Returns how many rows were written or deleted.
func (s *DeliveryFactRefresher) refreshOrders(ctx context.Context, orderIDs []string) (int, *apierror.APIError) {
	repo := s.cfg.Repos.NewDeliveryFactRepo()
	changedRows := 0
	changedAccounts := map[string]struct{}{}
	defer func() {
		if len(changedAccounts) > 0 {
			ids := make([]string, 0, len(changedAccounts))
			for id := range changedAccounts {
				ids = append(ids, id)
			}
			s.cfg.OnFactsChanged(ctx, ids)
		}
	}()
	for start := 0; start < len(orderIDs); {
		batch := orderIDs[start:min(start+s.batch, len(orderIDs))]
		began := time.Now()
		computed, apiErr := repo.ComputeFacts(ctx, batch)
		if apiErr != nil {
			return changedRows, apiErr
		}
		s.batch = nextDeliveryBatch(s.batch, time.Since(began))
		start += len(batch)

		stored, apiErr := repo.GetFacts(ctx, batch)
		if apiErr != nil {
			return changedRows, apiErr
		}
		upserts, deletes := diffDeliveryFacts(computed, stored)
		if len(upserts)+len(deletes) == 0 {
			continue
		}
		if apiErr := s.cfg.TxManager.WithTx(ctx, func(txCtx context.Context, repos domain.RepoFactory) *apierror.APIError {
			return repos.NewDeliveryFactRepo().WriteFacts(txCtx, upserts, deletes)
		}); apiErr != nil {
			return changedRows, apiErr
		}
		changedRows += len(upserts) + len(deletes)
		for _, f := range slices.Concat(upserts, deletes) {
			changedAccounts[f.OwnerAccountID] = struct{}{}
		}
	}
	return changedRows, nil
}

// nextDeliveryBatch steers the recompute batch so a statement lands near deliveryFactTargetStatement:
// shrink in proportion when over it, grow gently when well under it.
func nextDeliveryBatch(batch int, took time.Duration) int {
	switch {
	case took > deliveryFactTargetStatement:
		scaled := int(float64(batch) * float64(deliveryFactTargetStatement) / float64(took) * 0.8)
		return max(1, min(scaled, batch-1))
	case took < deliveryFactTargetStatement/2:
		return min(deliveryFactMaxBatch, max(batch+1, batch*3/2))
	default:
		return batch
	}
}

// deliveryFactKey is a fact row's full primary key.
func deliveryFactKey(f domain.DeliveryOrderFact) string {
	return strings.Join([]string{f.OwnerAccountID, f.ShipByDate.Format(time.DateOnly), f.SalesOrderID, f.ProductLineKey}, "\x00")
}

// diffDeliveryFacts returns the computed rows that are new or differ from what is stored, and the stored
// rows whose key no computed row has.
func diffDeliveryFacts(computed, stored []domain.DeliveryOrderFact) (upserts, deletes []domain.DeliveryOrderFact) {
	have := make(map[string]domain.DeliveryOrderFact, len(stored))
	for _, f := range stored {
		have[deliveryFactKey(f)] = f
	}
	want := make(map[string]struct{}, len(computed))
	for _, f := range computed {
		k := deliveryFactKey(f)
		want[k] = struct{}{}
		if old, ok := have[k]; !ok || !deliveryFactsEqual(old, f) {
			upserts = append(upserts, f)
		}
	}
	for _, f := range stored {
		if _, ok := want[deliveryFactKey(f)]; !ok {
			deletes = append(deletes, f)
		}
	}
	return upserts, deletes
}

func deliveryFactsEqual(a, b domain.DeliveryOrderFact) bool {
	return a.SalesOrderNumber == b.SalesOrderNumber && a.BuyerAccountID == b.BuyerAccountID &&
		ptrEqual(a.SalesRepID, b.SalesRepID) && a.StatusCode == b.StatusCode &&
		timePtrEqual(a.IssuedAt, b.IssuedAt) && timePtrEqual(a.FirstShipAt, b.FirstShipAt) && timePtrEqual(a.CompletedAt, b.CompletedAt) &&
		int32PtrEqual(a.LeadTimeDays, b.LeadTimeDays) && ptrEqual(a.LeadTimeSourceCode, b.LeadTimeSourceCode) &&
		a.QuantityOrdered == b.QuantityOrdered && a.QuantityPacked == b.QuantityPacked
}

func int32PtrEqual(a, b *int32) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// DeliveryFactMarker turns audit events into delivery dirty marks for the refresher.
type DeliveryFactMarker struct {
	repos    domain.RepoFactory
	onMarked func()
}

// NewDeliveryFactMarker returns a marker that calls onMarked (nil for none) after each mark it writes, to wake the refresher.
func NewDeliveryFactMarker(repos domain.RepoFactory, onMarked func()) *DeliveryFactMarker {
	if onMarked == nil {
		onMarked = func() {}
	}
	return &DeliveryFactMarker{repos: repos, onMarked: onMarked}
}

// HandleAuditEvent marks the scope an event can change delivery facts for. Every replica receives every event, so a mark is written once per replica; marking is idempotent.
func (m *DeliveryFactMarker) HandleAuditEvent(ctx context.Context, e audit.ObservedEvent) {
	scope, scopeID, ok := deliveryFactScopeFor(e)
	if !ok || e.AccountID == "" || scopeID == "" {
		return
	}
	if apiErr := m.repos.NewDeliveryFactRepo().MarkDirty(ctx, scope, scopeID, e.AccountID); apiErr != nil {
		// The reconcile pass still picks the change up, just later.
		slog.WarnContext(ctx, "Delivery fact marker: failed to mark scope", "scope", scope, "id", scopeID, "error", apiErr)
		return
	}
	m.onMarked()
}

// deliveryFactScopeFor maps an audit event to the scope whose delivery facts it can change, and the scope id to mark.
func deliveryFactScopeFor(e audit.ObservedEvent) (domain.DeliveryFactScope, string, bool) {
	// Lines, picks and pick lines name their order as the event's root, which is all a refresh needs.
	if e.RootResourceType == constants.ObjectTypeSalesOrder && e.RootResourceID != "" {
		return domain.DeliveryFactScopeSalesOrder, e.RootResourceID, true
	}
	update := e.Action == constants.AuditActionUpdate
	switch e.ResourceType {
	case constants.ObjectTypeSalesOrder:
		return domain.DeliveryFactScopeSalesOrder, e.ResourceID, true
	case constants.ObjectTypeSalesOrderLine:
		return domain.DeliveryFactScopeSalesOrderLine, e.ResourceID, true
	case constants.ObjectTypePick:
		return domain.DeliveryFactScopePick, e.ResourceID, true
	case constants.ObjectTypePickLine:
		return domain.DeliveryFactScopePickLine, e.ResourceID, true
	case constants.ObjectTypeQuantity:
		// An order line's or pick line's quantity edited on its own, through the generic quantity endpoint.
		return domain.DeliveryFactScopeQuantity, e.ResourceID, update
	case constants.ObjectTypeProduct:
		// Every other product edit leaves the facts alone, and a product can sit on thousands of orders.
		return domain.DeliveryFactScopeProduct, e.ResourceID, update && auditChanged(e, "product_line_id", "product_type_code")
	case constants.ObjectTypeCustomer:
		// A merge deletes the customers merged away; their orders now belong to another buyer.
		return domain.DeliveryFactScopeReconcile, "all", e.Action == constants.AuditActionDelete
	}
	return "", "", false
}

// auditChanged reports whether an update touched any of the fields, or a field nested under one.
func auditChanged(e audit.ObservedEvent, fields ...string) bool {
	for _, c := range e.ChangedFields {
		for _, f := range fields {
			if c == f || strings.HasPrefix(c, f+".") {
				return true
			}
		}
	}
	return false
}
