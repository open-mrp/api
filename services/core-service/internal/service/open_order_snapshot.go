package service

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
	"go.opentelemetry.io/otel/attribute"
)

const (
	// openOrderPassInterval is how often the open-order snapshot is compared against every issued sales
	// order. Few orders are issued at once, so the pass is cheap, and it is the only path that sees writes
	// publishing no audit event.
	openOrderPassInterval = time.Minute

	// openOrderIssuedPage is how many issued orders one step of the pass refreshes.
	openOrderIssuedPage = 50

	// openOrderStalePage is how many snapshot orders one step of the pass checks are still issued.
	openOrderStalePage = 200
)

// openOrderPhase is the step of the snapshot pass in progress.
type openOrderPhase int

const (
	openOrderIdle openOrderPhase = iota
	// openOrderIssued refreshes every issued sales order, which copies new ones in.
	openOrderIssued
	// openOrderStale removes the orders the snapshot holds that are no longer issued.
	openOrderStale
)

// openOrderPass is the snapshot pass's progress in this pod. It is not persisted: a pass interrupted by a
// restart or a lease handover starts again, which costs one more pass over a small set.
type openOrderPass struct {
	phase     openOrderPhase
	cursor    string
	startedAt time.Time
}

// syncOpenOrders advances the snapshot pass by up to ReconcileBudget, starting one every openOrderPassInterval.
// The first pass to complete is the backfill: the reports read the snapshot only after it.
func (s *SalesFactRefresher) syncOpenOrders(ctx context.Context) *apierror.APIError {
	ctx, span := salesFactTracer.Start(ctx, "service.sales_fact_refresher.sync_open_orders")
	defer span.End()

	p := &s.openOrders
	now := s.cfg.Now().UTC()
	if p.phase == openOrderIdle {
		if !p.startedAt.IsZero() && now.Sub(p.startedAt) < openOrderPassInterval {
			return nil
		}
		*p = openOrderPass{phase: openOrderIssued, startedAt: now}
	}

	repo := s.cfg.Repos.NewOpenOrderSnapshotRepo()
	deadline := now.Add(s.cfg.ReconcileBudget)
	changed := 0
	for p.phase != openOrderIdle && s.cfg.Now().UTC().Before(deadline) {
		switch p.phase {
		case openOrderIssued:
			ids, apiErr := repo.ListIssuedOrderIDsAfter(ctx, p.cursor, openOrderIssuedPage)
			if apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
			n, apiErr := s.refreshOpenOrders(ctx, repo, ids)
			changed += n
			if apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
			if len(ids) < openOrderIssuedPage {
				p.phase, p.cursor = openOrderStale, ""
				continue
			}
			p.cursor = ids[len(ids)-1]
		case openOrderStale:
			ids, apiErr := repo.ListSnapshotOrderIDsAfter(ctx, p.cursor, openOrderStalePage)
			if apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
			stale, apiErr := repo.FilterNotIssued(ctx, ids)
			if apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
			n, apiErr := s.refreshOpenOrders(ctx, repo, stale)
			changed += n
			if apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
			if len(ids) < openOrderStalePage {
				completed := s.cfg.Now().UTC()
				if apiErr := repo.SaveSync(ctx, domain.OpenOrderSnapshotSync{PassStartedAt: &p.startedAt, LastCompletedAt: &completed}); apiErr != nil {
					return tracing.Trace(span, apiErr)
				}
				p.phase = openOrderIdle
				continue
			}
			p.cursor = ids[len(ids)-1]
		}
	}
	span.SetAttributes(attribute.Int("open_order_snapshot.changed_rows", changed))
	if changed > 0 {
		slog.InfoContext(ctx, "Sales fact refresher: open-order pass corrected the snapshot", "rows", changed)
	}
	if p.phase != openOrderIdle {
		s.backlog = true
	}
	return nil
}

// refreshOpenOrdersFor refreshes the snapshot orders a batch of dirty marks can change, before the marks
// are cleared; the pass would correct them within openOrderPassInterval regardless.
func (s *SalesFactRefresher) refreshOpenOrdersFor(ctx context.Context, marks []domain.SalesFactDirtyMark) *apierror.APIError {
	ctx, span := salesFactTracer.Start(ctx, "service.sales_fact_refresher.refresh_open_orders")
	defer span.End()

	byScope := make(map[domain.SalesFactScope][]string)
	for _, m := range marks {
		byScope[m.ScopeType] = append(byScope[m.ScopeType], m.ScopeID)
	}
	repo := s.cfg.Repos.NewOpenOrderSnapshotRepo()
	var orderIDs []string
	for scope, ids := range byScope {
		resolved, apiErr := repo.ResolveOrders(ctx, scope, ids)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		orderIDs = append(orderIDs, resolved...)
	}
	slices.Sort(orderIDs)
	orderIDs = slices.Compact(orderIDs)
	span.SetAttributes(attribute.Int("open_order_snapshot.orders", len(orderIDs)))
	_, apiErr := s.refreshOpenOrders(ctx, repo, orderIDs)
	return tracing.Trace(span, apiErr)
}

// refreshOpenOrders refreshes the given orders' snapshot rows; none is no work. The open-order reports
// are sales reports, so the changed accounts' sales reports are dropped as after a fact refresh.
func (s *SalesFactRefresher) refreshOpenOrders(ctx context.Context, repo domain.OpenOrderSnapshotRepo, orderIDs []string) (int, *apierror.APIError) {
	if len(orderIDs) == 0 {
		return 0, nil
	}
	accounts, changed, apiErr := repo.RefreshOrders(ctx, orderIDs)
	if len(accounts) > 0 {
		s.cfg.OnFactsChanged(ctx, accounts)
	}
	return changed, apiErr
}
