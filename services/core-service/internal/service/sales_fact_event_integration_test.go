//go:build integration

package service

import (
	"context"
	"database/sql"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/lease"
)

// These prove the event-driven refresher keeps sales_line_fact and sales_fact_rollup equal to their
// source with no help from the idle run: each pod's IdleInterval outlasts the test, and the edited
// invoices are older than the rolling recompute's window, so only a wake can refresh them.

// eventPod is one core-service replica: a refresher under the shared lease and the marker that wakes it.
type eventPod struct {
	*SalesFactRefresher
	marker *SalesFactMarker
	runs   atomic.Int32
}

func startEventPod(t *testing.T, pool *sql.DB, repos domain.RepoFactory, holder string) *eventPod {
	t.Helper()
	p := &eventPod{}
	p.SalesFactRefresher = NewSalesFactRefresher(&SalesFactRefresherConfig{
		Repos:           repos,
		Lease:           lease.NewWithHolder(repository.NewLeaseRepo(sqlc.New(pool)), holder),
		LeaseName:       salesFactTestLease,
		TickInterval:    100 * time.Millisecond,
		IdleInterval:    time.Hour,
		WakeDelay:       50 * time.Millisecond,
		ReconcileBudget: time.Minute,
	})
	p.tick = func(ctx context.Context) bool {
		more := p.Tick(ctx)
		p.runs.Add(1)
		return more
	}
	p.marker = NewSalesFactMarker(repos, p.Wake)
	if err := p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Stop)
	return p
}

// publish delivers an audit event the way the fanout does: to every replica.
func publish(ctx context.Context, e audit.ObservedEvent, pods ...*eventPod) {
	for _, p := range pods {
		p.marker.HandleAuditEvent(ctx, e)
	}
}

// oldSeedInvoice returns an invoice of the seed account, with facts, that the rolling recompute does not reach.
func oldSeedInvoice(t *testing.T, ctx context.Context, pool *sql.DB) string {
	t.Helper()
	var id string
	err := pool.QueryRowContext(ctx, `SELECT i.id FROM invoice i
WHERE i.account_id = ? AND i.id <> ? AND i.created_at < NOW() - INTERVAL 30 DAY
  AND EXISTS (SELECT 1 FROM sales_line_fact f WHERE f.invoice_id = i.id)
ORDER BY i.created_at, i.id LIMIT 1`, seedAccountID, seedInvoiceID).Scan(&id)
	if err != nil {
		t.Fatalf("no invoice older than the recompute window; seed it first: %v", err)
	}
	return id
}

// editedLine bumps the quantity of an invoice's first line by one, reverting it at cleanup.
type editedLine struct {
	invoiceID  string
	quantityID string
	invoicedAt time.Time
}

func bumpFirstLine(t *testing.T, ctx context.Context, pool *sql.DB, invoiceID string) editedLine {
	t.Helper()
	e := editedLine{invoiceID: invoiceID}
	if err := pool.QueryRowContext(ctx, `SELECT il.quantity_id, i.created_at FROM invoice_line il JOIN invoice i ON i.id = il.invoice_id
WHERE il.invoice_id = ? ORDER BY il.id LIMIT 1`, invoiceID).Scan(&e.quantityID, &e.invoicedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.ExecContext(ctx, `UPDATE quantity SET value = value + 1 WHERE id = ?`, e.quantityID); err != nil {
		t.Fatal(err)
	}
	return e
}

// restoreEdits reverts the edits and refreshes their invoices, so later tests start from an accurate table.
func restoreEdits(t *testing.T, pool *sql.DB, repos domain.RepoFactory, edits ...editedLine) {
	t.Cleanup(func() {
		ctx := context.Background()
		r := newTestRefresher(repos, pool)
		for _, e := range edits {
			_, _ = pool.ExecContext(ctx, `UPDATE quantity SET value = value - 1 WHERE id = ?`, e.quantityID)
			_ = repos.NewSalesFactRepo().MarkDirty(ctx, domain.SalesFactScopeInvoice, e.invoiceID, seedAccountID)
		}
		_ = r.drainDirty(ctx)
	})
}

// factsStale reports how many of the invoices' fact lines differ from what their source computes.
func factsStale(t *testing.T, ctx context.Context, repos domain.RepoFactory, invoiceIDs ...string) int {
	t.Helper()
	computed, apiErr := repos.NewSalesFactRepo().ComputeFacts(ctx, invoiceIDs)
	if apiErr != nil {
		t.Fatal(apiErr)
	}
	stored, apiErr := repos.NewSalesFactRepo().GetFacts(ctx, invoiceIDs)
	if apiErr != nil {
		t.Fatal(apiErr)
	}
	upserts, deletes := diffSalesFacts(computed, stored)
	return len(upserts) + len(deletes)
}

// requireAccurate fails unless every invoice's facts match its source and every month's rollup matches its facts.
func requireAccurate(t *testing.T, ctx context.Context, pool *sql.DB, repos domain.RepoFactory) {
	t.Helper()
	if n := factsStale(t, ctx, repos, allInvoiceIDs(t, ctx, pool)...); n != 0 {
		t.Fatalf("%d fact lines differ from their source", n)
	}
	var months int
	if err := pool.QueryRowContext(ctx, `SELECT COUNT(*) FROM (
  SELECT f.account_id, f.month, f.inv, f.line_total, r.inv AS r_inv, r.line_total AS r_lines FROM
    (SELECT account_id, DATE_FORMAT(invoiced_at, '%Y-%m-01') AS month, SUM(total_invoiced) AS inv, COUNT(*) AS line_total
     FROM sales_line_fact GROUP BY account_id, month) f
  LEFT JOIN
    (SELECT account_id, DATE_FORMAT(bucket_start, '%Y-%m-01') AS month, SUM(total_invoiced) AS inv, SUM(line_count) AS line_total
     FROM sales_fact_rollup WHERE dimension = 'total' AND product_line_key = '' AND grain = 'month' GROUP BY account_id, month) r
  ON r.account_id = f.account_id AND r.month = f.month
) m WHERE r_inv IS NULL OR r_inv <> inv OR r_lines <> line_total`).Scan(&months); err != nil {
		t.Fatal(err)
	}
	if months != 0 {
		t.Fatalf("%d account-months have a rollup that differs from their facts", months)
	}
}

func waitAccurate(t *testing.T, ctx context.Context, repos domain.RepoFactory, within time.Duration, invoiceIDs ...string) {
	t.Helper()
	deadline := time.Now().Add(within)
	for factsStale(t, ctx, repos, invoiceIDs...) != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("invoices %v still stale after %s", invoiceIDs, within)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func totalRuns(pods ...*eventPod) int32 {
	var n int32
	for _, p := range pods {
		n += p.runs.Load()
	}
	return n
}

// settle waits until the pods have finished their startup runs and gone idle.
func settle(t *testing.T, pods ...*eventPod) {
	t.Helper()
	total := func() int32 { return totalRuns(pods...) }
	deadline := time.Now().Add(30 * time.Second)
	for total() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("no pod ran at startup")
		}
		time.Sleep(20 * time.Millisecond)
	}
	for last := total(); ; {
		time.Sleep(500 * time.Millisecond)
		if now := total(); now == last {
			return
		} else {
			last = now
		}
	}
}

// pauseLiveRefresher holds the production refresher's lease for the rest of the test, so a core-service
// running against the same database (a local dev stack) cannot drain the marks these tests write and
// mask a refresher that failed to.
func pauseLiveRefresher(t *testing.T, pool *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	held, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		l := lease.NewWithHolder(repository.NewLeaseRepo(sqlc.New(pool)), "sales-fact-integration-test")
		for ctx.Err() == nil {
			// Returns without running while the live refresher holds it; its runs are short, so retry.
			_ = l.WithLease(ctx, salesFactLeaseName, 30*time.Second, func(leaseCtx context.Context) error {
				close(held)
				<-leaseCtx.Done()
				return nil
			})
			select {
			case <-held:
				return
			case <-time.After(20 * time.Millisecond):
			}
		}
	}()
	t.Cleanup(func() { cancel(); <-done })
	select {
	case <-held:
	case <-time.After(30 * time.Second):
		t.Fatal("could not take the live refresher's lease")
	}
}

// accurateBaseline brings every fact and rollup in line with its source before a test edits anything.
func accurateBaseline(t *testing.T, ctx context.Context, pool *sql.DB, repos domain.RepoFactory) {
	t.Helper()
	pauseLiveRefresher(t, pool)
	r := newTestRefresher(repos, pool)
	forceReconcile(t, ctx, pool, r)
	forceRollupSweep(t, ctx, pool, r)
	requireAccurate(t, ctx, pool, repos)
}

func TestSalesFactEventsKeepFactsAndRollupsAccurateAcrossPods(t *testing.T) {
	ctx := context.Background()
	pool, repos := salesFactDB(t)
	accurateBaseline(t, ctx, pool, repos)

	old := oldSeedInvoice(t, ctx, pool)
	byQuantity := bumpFirstLine(t, ctx, pool, old)
	byInvoice := bumpFirstLine(t, ctx, pool, seedInvoiceID)
	restoreEdits(t, pool, repos, byQuantity, byInvoice)
	if factsStale(t, ctx, repos, old, seedInvoiceID) == 0 {
		t.Fatal("the edits changed no facts; the test proves nothing")
	}

	a := startEventPod(t, pool, repos, "pod-a")
	b := startEventPod(t, pool, repos, "pod-b")
	settle(t, a, b)
	// The startup run's rolling recompute reaches the seed invoice; re-edit it so only the wake can fix it.
	byInvoiceAgain := bumpFirstLine(t, ctx, pool, seedInvoiceID)
	restoreEdits(t, pool, repos, byInvoiceAgain)
	idleRuns := totalRuns(a, b)
	time.Sleep(300 * time.Millisecond)
	if totalRuns(a, b) != idleRuns || factsStale(t, ctx, repos, old) == 0 || factsStale(t, ctx, repos, seedInvoiceID) == 0 {
		t.Fatal("a pod ran or facts refreshed with no event: the test cannot tell a wake from an idle run")
	}

	// A quantity edit resolves to its invoice through the quantity scope; an invoice event marks it directly.
	publish(ctx, audit.ObservedEvent{AccountID: seedAccountID, Action: constants.AuditActionUpdate, ResourceType: constants.ObjectTypeQuantity, ResourceID: byQuantity.quantityID, ChangedFields: []string{"value"}}, a, b)
	publish(ctx, audit.ObservedEvent{AccountID: seedAccountID, Action: constants.AuditActionUpdate, ResourceType: constants.ObjectTypeInvoice, ResourceID: seedInvoiceID}, a, b)

	waitAccurate(t, ctx, repos, 10*time.Second, old, seedInvoiceID)
	settle(t, a, b)
	if totalRuns(a, b) == idleRuns {
		t.Fatal("the facts were refreshed by no pod")
	}
	requireAccurate(t, ctx, pool, repos)
	marks, apiErr := repos.NewSalesFactRepo().ListDirty(ctx, 1000)
	if apiErr != nil {
		t.Fatal(apiErr)
	}
	for _, m := range marks {
		if m.ScopeID == old || m.ScopeID == seedInvoiceID || m.ScopeID == byQuantity.quantityID {
			t.Fatalf("mark %+v was left after its refresh", m)
		}
	}
}

func TestSalesFactMarkIsRefreshedAfterAnotherPodReleasesTheLease(t *testing.T) {
	ctx := context.Background()
	pool, repos := salesFactDB(t)
	accurateBaseline(t, ctx, pool, repos)

	a := startEventPod(t, pool, repos, "pod-a")
	settle(t, a)

	// Pod B is mid-run and only pod A hears the event, so pod A must keep trying until it gets the lease.
	leases := repository.NewLeaseRepo(sqlc.New(pool))
	if ok, err := leases.Acquire(ctx, salesFactTestLease, "pod-b", time.Minute); err != nil || !ok {
		t.Fatalf("pod-b acquire = %v, %v", ok, err)
	}
	released := false
	t.Cleanup(func() {
		if !released {
			_ = leases.Release(context.Background(), salesFactTestLease, "pod-b")
		}
	})

	old := oldSeedInvoice(t, ctx, pool)
	edit := bumpFirstLine(t, ctx, pool, old)
	restoreEdits(t, pool, repos, edit)
	blocked := a.runs.Load()
	publish(ctx, audit.ObservedEvent{AccountID: seedAccountID, Action: constants.AuditActionUpdate, ResourceType: constants.ObjectTypeInvoice, ResourceID: old}, a)

	time.Sleep(500 * time.Millisecond)
	if a.runs.Load() != blocked || factsStale(t, ctx, repos, old) == 0 {
		t.Fatal("refreshed while another pod held the lease")
	}

	if err := leases.Release(ctx, salesFactTestLease, "pod-b"); err != nil {
		t.Fatal(err)
	}
	released = true

	waitAccurate(t, ctx, repos, 10*time.Second, old)
	settle(t, a)
	if a.runs.Load() == blocked {
		t.Fatal("the facts were refreshed by no pod")
	}
	requireAccurate(t, ctx, pool, repos)
}

func TestSalesFactBurstOfEditsIsFullyRefreshed(t *testing.T) {
	ctx := context.Background()
	pool, repos := salesFactDB(t)
	accurateBaseline(t, ctx, pool, repos)

	rows, err := pool.QueryContext(ctx, `SELECT i.id FROM invoice i
WHERE i.account_id = ? AND i.created_at < NOW() - INTERVAL 30 DAY
  AND EXISTS (SELECT 1 FROM sales_line_fact f WHERE f.invoice_id = i.id)
ORDER BY i.created_at, i.id LIMIT 25`, seedAccountID)
	if err != nil {
		t.Fatal(err)
	}
	var invoices []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		invoices = append(invoices, id)
	}
	_ = rows.Close()
	if len(invoices) < 10 {
		t.Fatalf("only %d old invoices; seed more", len(invoices))
	}

	a := startEventPod(t, pool, repos, "pod-a")
	b := startEventPod(t, pool, repos, "pod-b")
	settle(t, a, b)

	// Edits and their events interleave, as they do in production: some land mid-run.
	idleRuns := totalRuns(a, b)
	var edits []editedLine
	for _, id := range invoices {
		e := bumpFirstLine(t, ctx, pool, id)
		edits = append(edits, e)
		publish(ctx, audit.ObservedEvent{AccountID: seedAccountID, Action: constants.AuditActionUpdate, ResourceType: constants.ObjectTypeQuantity, ResourceID: e.quantityID, ChangedFields: []string{"value"}}, a, b)
		time.Sleep(20 * time.Millisecond)
	}
	restoreEdits(t, pool, repos, edits...)

	waitAccurate(t, ctx, repos, 20*time.Second, invoices...)
	settle(t, a, b)
	if totalRuns(a, b) == idleRuns {
		t.Fatal("the facts were refreshed by no pod")
	}
	requireAccurate(t, ctx, pool, repos)
}
