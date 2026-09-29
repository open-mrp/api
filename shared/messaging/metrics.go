package messaging

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/open-mrp/api/shared/appctx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const meterName = "github.com/open-mrp/api/shared/messaging"

// gaugeQueryTimeout bounds each backlog query a metrics collection runs, so a slow database delays one export rather than stalling the reader.
const gaugeQueryTimeout = 5 * time.Second

var inboxHandleDurationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300}

const (
	inboxOutcomeSuccess   = "success"
	inboxOutcomeError     = "error"
	inboxOutcomeDiscarded = "discarded"
	inboxOutcomeIgnored   = "ignored"
)

// inboxBacklogStatus is the status label on the messaging.inbox.messages gauge. It splits InboxStatusReceived by whether an attempt has already failed, because a retrying message and a fresh one need different responses.
type inboxBacklogStatus string

const (
	inboxBacklogReceived  inboxBacklogStatus = "received"
	inboxBacklogFailed    inboxBacklogStatus = "failed"
	inboxBacklogDiscarded inboxBacklogStatus = "discarded"
)

var inboxBacklogStatuses = []inboxBacklogStatus{inboxBacklogReceived, inboxBacklogFailed, inboxBacklogDiscarded}

// OutboxStatusStat is one status's share of a service's unpublished outbox rows.
type OutboxStatusStat struct {
	Status OutboxStatus
	Count  int64
	// OldestAge is how long ago the oldest row in this status was created.
	OldestAge time.Duration
}

// OutboxStatsRepo reads the outbox backlog the messaging gauges report.
type OutboxStatsRepo interface {
	// OutboxStats returns the service's 'pending' and 'failed' outbox rows grouped by status. Statuses with no rows may be omitted.
	OutboxStats(ctx context.Context, serviceName string) ([]OutboxStatusStat, error)
}

// InboxHandlerStat is one (handler, status, failed) group of a service's unfinished or discarded inbox rows.
type InboxHandlerStat struct {
	Handler string
	Status  InboxStatus
	// Failed is true when a previous attempt recorded failed_at.
	Failed bool
	Count  int64
	// OldestAge is how long ago the oldest row in this group was received.
	OldestAge time.Duration
}

// InboxStatsRepo reads the inbox backlog the messaging gauges report.
type InboxStatsRepo interface {
	// InboxStats returns the service's 'received' and 'discarded' inbox rows grouped by handler, status, and whether failed_at is set. Groups with no rows may be omitted.
	InboxStats(ctx context.Context, serviceName string) ([]InboxHandlerStat, error)
}

// messagingMetrics holds the synchronous instruments the enqueuer and inbox consumer record into. A nil *messagingMetrics records nothing.
type messagingMetrics struct {
	outboxPublished     metric.Int64Counter
	outboxPublishErrors metric.Int64Counter
	inboxHandled        metric.Int64Counter
	inboxHandleDuration metric.Float64Histogram
}

// defaultMessagingMetrics binds to the global MeterProvider, which forwards to the provider InitProvider installs even when the instruments are created first.
var defaultMessagingMetrics = sync.OnceValue(func() *messagingMetrics {
	return newMessagingMetrics(otel.Meter(meterName))
})

func newMessagingMetrics(meter metric.Meter) *messagingMetrics {
	var m messagingMetrics
	var errs [4]error
	m.outboxPublished, errs[0] = meter.Int64Counter("messaging.outbox.published",
		metric.WithDescription("Outbox messages the broker confirmed."))
	m.outboxPublishErrors, errs[1] = meter.Int64Counter("messaging.outbox.publish_errors",
		metric.WithDescription("Outbox publish attempts that failed and were rescheduled."))
	m.inboxHandled, errs[2] = meter.Int64Counter("messaging.inbox.handled",
		metric.WithDescription("Inbox handler invocations by outcome."))
	m.inboxHandleDuration, errs[3] = meter.Float64Histogram("messaging.inbox.handle_duration",
		metric.WithDescription("Inbox handler run time."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(inboxHandleDurationBuckets...))
	if err := errors.Join(errs[:]...); err != nil {
		otel.Handle(err)
	}
	return &m
}

func (m *messagingMetrics) recordPublished(ctx context.Context, messageType string) {
	if m == nil {
		return
	}
	m.outboxPublished.Add(ctx, 1, metric.WithAttributes(attribute.String("message_type", messageType)))
}

func (m *messagingMetrics) recordPublishError(ctx context.Context, messageType string) {
	if m == nil {
		return
	}
	m.outboxPublishErrors.Add(ctx, 1, metric.WithAttributes(attribute.String("message_type", messageType)))
}

func (m *messagingMetrics) recordHandled(ctx context.Context, handler string, elapsed time.Duration, err error) {
	if m == nil {
		return
	}
	handlerAttr := attribute.String("handler", handler)
	m.inboxHandled.Add(ctx, 1, metric.WithAttributes(handlerAttr, attribute.String("outcome", inboxOutcome(err))))
	m.inboxHandleDuration.Record(ctx, elapsed.Seconds(), metric.WithAttributes(handlerAttr))
}

// inboxOutcome classifies a handler's return value the way Wrap acts on it.
func inboxOutcome(err error) string {
	switch {
	case err == nil, errors.Is(err, ErrInboxAlreadyCompleted):
		return inboxOutcomeSuccess
	case errors.Is(err, errInboxIgnored):
		return inboxOutcomeIgnored
	case errors.Is(err, ErrInboxDiscarded):
		return inboxOutcomeDiscarded
	default:
		return inboxOutcomeError
	}
}

// wrappedHandlers records every handler name InboxConsumer.Wrap has seen in this process, so the inbox gauges report zero for a drained handler instead of letting its series go stale.
var wrappedHandlers sync.Map

// RegisterOutboxGauges reports serviceName's outbox backlog on every metrics collection: messaging.outbox.messages by status and messaging.outbox.oldest_pending_age. Every replica reports the same rows, so dashboards should aggregate with max, not sum.
func RegisterOutboxGauges(serviceName string, repo OutboxStatsRepo) (metric.Registration, error) {
	return registerOutboxGauges(otel.Meter(meterName), serviceName, repo)
}

func registerOutboxGauges(meter metric.Meter, serviceName string, repo OutboxStatsRepo) (metric.Registration, error) {
	messages, err := meter.Int64ObservableGauge("messaging.outbox.messages",
		metric.WithDescription("Unpublished outbox rows by status; failed rows exhausted max_attempts."))
	if err != nil {
		return nil, err
	}
	oldestPending, err := meter.Float64ObservableGauge("messaging.outbox.oldest_pending_age",
		metric.WithDescription("Age of the oldest pending outbox row, or 0 when there is none."),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	return meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		ctx, cancel := context.WithTimeout(appctx.WithNoTrace(ctx), gaugeQueryTimeout)
		defer cancel()

		stats, err := repo.OutboxStats(ctx, serviceName)
		if err != nil {
			slog.Warn("Failed to collect outbox gauges", "service", serviceName, "error", err)
			return nil
		}

		counts := map[OutboxStatus]int64{OutboxStatusPending: 0, OutboxStatusFailed: 0}
		var oldest time.Duration
		for _, s := range stats {
			if _, ok := counts[s.Status]; !ok {
				continue
			}
			counts[s.Status] += s.Count
			if s.Status == OutboxStatusPending {
				oldest = max(oldest, s.OldestAge)
			}
		}

		for _, status := range []OutboxStatus{OutboxStatusPending, OutboxStatusFailed} {
			o.ObserveInt64(messages, counts[status], metric.WithAttributes(attribute.String("status", string(status))))
		}
		o.ObserveFloat64(oldestPending, oldest.Seconds())
		return nil
	}, messages, oldestPending)
}

// RegisterInboxGauges reports serviceName's inbox backlog on every metrics collection: messaging.inbox.messages by handler and status, and messaging.inbox.oldest_unprocessed_age by handler. Processed and ignored rows are left out; they are healthy and only bounded by the purge. Every replica reports the same rows, so dashboards should aggregate with max, not sum.
func RegisterInboxGauges(serviceName string, repo InboxStatsRepo) (metric.Registration, error) {
	return registerInboxGauges(otel.Meter(meterName), serviceName, repo)
}

func registerInboxGauges(meter metric.Meter, serviceName string, repo InboxStatsRepo) (metric.Registration, error) {
	messages, err := meter.Int64ObservableGauge("messaging.inbox.messages",
		metric.WithDescription("Unfinished and discarded inbox rows by handler and status."))
	if err != nil {
		return nil, err
	}
	oldestUnprocessed, err := meter.Float64ObservableGauge("messaging.inbox.oldest_unprocessed_age",
		metric.WithDescription("Age of the handler's oldest received inbox row, or 0 when there is none."),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	return meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		ctx, cancel := context.WithTimeout(appctx.WithNoTrace(ctx), gaugeQueryTimeout)
		defer cancel()

		stats, err := repo.InboxStats(ctx, serviceName)
		if err != nil {
			slog.Warn("Failed to collect inbox gauges", "service", serviceName, "error", err)
			return nil
		}

		type handlerBacklog struct {
			counts map[inboxBacklogStatus]int64
			oldest time.Duration
		}
		backlogs := map[string]*handlerBacklog{}
		backlogFor := func(handler string) *handlerBacklog {
			b, ok := backlogs[handler]
			if !ok {
				b = &handlerBacklog{counts: map[inboxBacklogStatus]int64{}}
				backlogs[handler] = b
			}
			return b
		}

		wrappedHandlers.Range(func(handler, _ any) bool {
			backlogFor(handler.(string))
			return true
		})
		for _, s := range stats {
			status, ok := classifyInboxBacklog(s.Status, s.Failed)
			if !ok {
				continue
			}
			b := backlogFor(s.Handler)
			b.counts[status] += s.Count
			if s.Status == InboxStatusReceived {
				b.oldest = max(b.oldest, s.OldestAge)
			}
		}

		for handler, b := range backlogs {
			handlerAttr := attribute.String("handler", handler)
			for _, status := range inboxBacklogStatuses {
				o.ObserveInt64(messages, b.counts[status], metric.WithAttributes(handlerAttr, attribute.String("status", string(status))))
			}
			o.ObserveFloat64(oldestUnprocessed, b.oldest.Seconds(), metric.WithAttributes(handlerAttr))
		}
		return nil
	}, messages, oldestUnprocessed)
}

func classifyInboxBacklog(status InboxStatus, failed bool) (inboxBacklogStatus, bool) {
	switch status {
	case InboxStatusReceived:
		if failed {
			return inboxBacklogFailed, true
		}
		return inboxBacklogReceived, true
	case InboxStatusDiscarded:
		return inboxBacklogDiscarded, true
	default:
		return "", false
	}
}
