package messaging

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type stubOutboxStatsRepo struct {
	stats       []OutboxStatusStat
	err         error
	serviceName string
	hasDeadline bool
}

func (r *stubOutboxStatsRepo) OutboxStats(ctx context.Context, serviceName string) ([]OutboxStatusStat, error) {
	r.serviceName = serviceName
	_, r.hasDeadline = ctx.Deadline()
	return r.stats, r.err
}

type stubInboxStatsRepo struct {
	stats       []InboxHandlerStat
	err         error
	serviceName string
}

func (r *stubInboxStatsRepo) InboxStats(_ context.Context, serviceName string) ([]InboxHandlerStat, error) {
	r.serviceName = serviceName
	return r.stats, r.err
}

func newTestMeterProvider(t *testing.T) (*sdkmetric.MeterProvider, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	return provider, reader
}

func collectMetrics(t *testing.T, reader *sdkmetric.ManualReader) map[string]metricdata.Metrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	byName := map[string]metricdata.Metrics{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			byName[m.Name] = m
		}
	}
	return byName
}

// attrKey renders a data point's attributes as "k=v,k=v" so expectations read as literals.
func attrKey(set attribute.Set) string {
	parts := make([]string, 0, set.Len())
	for _, kv := range set.ToSlice() {
		parts = append(parts, string(kv.Key)+"="+kv.Value.String())
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func int64Points(t *testing.T, m metricdata.Metrics) map[string]int64 {
	t.Helper()
	points := map[string]int64{}
	switch data := m.Data.(type) {
	case metricdata.Gauge[int64]:
		for _, dp := range data.DataPoints {
			points[attrKey(dp.Attributes)] = dp.Value
		}
	case metricdata.Sum[int64]:
		for _, dp := range data.DataPoints {
			points[attrKey(dp.Attributes)] = dp.Value
		}
	default:
		t.Fatalf("%s: unexpected data type %T", m.Name, m.Data)
	}
	return points
}

func float64GaugePoints(t *testing.T, m metricdata.Metrics) map[string]float64 {
	t.Helper()
	data, ok := m.Data.(metricdata.Gauge[float64])
	require.True(t, ok, "%s: unexpected data type %T", m.Name, m.Data)
	points := map[string]float64{}
	for _, dp := range data.DataPoints {
		points[attrKey(dp.Attributes)] = dp.Value
	}
	return points
}

func TestOutboxGauges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		stats          []OutboxStatusStat
		expectedCounts map[string]int64
		expectedAge    float64
	}{
		{
			name:           "empty outbox reports zeros",
			stats:          nil,
			expectedCounts: map[string]int64{"status=pending": 0, "status=failed": 0},
			expectedAge:    0,
		},
		{
			name: "pending and failed rows",
			stats: []OutboxStatusStat{
				{Status: OutboxStatusPending, Count: 12, OldestAge: 90 * time.Second},
				{Status: OutboxStatusFailed, Count: 2, OldestAge: 72 * time.Hour},
			},
			expectedCounts: map[string]int64{"status=pending": 12, "status=failed": 2},
			expectedAge:    90,
		},
		{
			name: "failed rows do not set the pending age",
			stats: []OutboxStatusStat{
				{Status: OutboxStatusFailed, Count: 1, OldestAge: time.Hour},
			},
			expectedCounts: map[string]int64{"status=pending": 0, "status=failed": 1},
			expectedAge:    0,
		},
		{
			name: "published rows are ignored",
			stats: []OutboxStatusStat{
				{Status: OutboxStatusPublished, Count: 1000, OldestAge: time.Hour},
				{Status: OutboxStatusPending, Count: 1, OldestAge: 1500 * time.Millisecond},
			},
			expectedCounts: map[string]int64{"status=pending": 1, "status=failed": 0},
			expectedAge:    1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			provider, reader := newTestMeterProvider(t)
			repo := &stubOutboxStatsRepo{stats: tt.stats}

			_, err := registerOutboxGauges(provider.Meter("test"), "core-service", repo)
			require.NoError(t, err)

			metrics := collectMetrics(t, reader)
			require.Equal(t, "core-service", repo.serviceName)
			require.True(t, repo.hasDeadline, "the gauge query must be bounded")
			require.Equal(t, tt.expectedCounts, int64Points(t, metrics["messaging.outbox.messages"]))
			require.Equal(t, map[string]float64{"": tt.expectedAge}, float64GaugePoints(t, metrics["messaging.outbox.oldest_pending_age"]))
			require.Equal(t, "s", metrics["messaging.outbox.oldest_pending_age"].Unit)
		})
	}
}

func TestOutboxGaugesSkipObservationOnQueryError(t *testing.T) {
	t.Parallel()
	provider, reader := newTestMeterProvider(t)

	_, err := registerOutboxGauges(provider.Meter("test"), "core-service", &stubOutboxStatsRepo{err: errors.New("db down")})
	require.NoError(t, err)

	metrics := collectMetrics(t, reader)
	require.NotContains(t, metrics, "messaging.outbox.messages")
	require.NotContains(t, metrics, "messaging.outbox.oldest_pending_age")
}

func TestInboxGauges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		stats          []InboxHandlerStat
		expectedCounts map[string]int64
		expectedAges   map[string]float64
	}{
		{
			name: "received rows split by failure",
			stats: []InboxHandlerStat{
				{Handler: "h.a", Status: InboxStatusReceived, Count: 4, OldestAge: 10 * time.Second},
				{Handler: "h.a", Status: InboxStatusReceived, Failed: true, Count: 2, OldestAge: 5 * time.Minute},
			},
			expectedCounts: map[string]int64{
				"handler=h.a,status=received":  4,
				"handler=h.a,status=failed":    2,
				"handler=h.a,status=discarded": 0,
			},
			expectedAges: map[string]float64{"handler=h.a": 300},
		},
		{
			name: "discarded rows count but do not age the backlog",
			stats: []InboxHandlerStat{
				{Handler: "h.b", Status: InboxStatusDiscarded, Failed: true, Count: 3, OldestAge: 48 * time.Hour},
			},
			expectedCounts: map[string]int64{
				"handler=h.b,status=received":  0,
				"handler=h.b,status=failed":    0,
				"handler=h.b,status=discarded": 3,
			},
			expectedAges: map[string]float64{"handler=h.b": 0},
		},
		{
			name: "terminal healthy statuses are not reported",
			stats: []InboxHandlerStat{
				{Handler: "h.c", Status: InboxStatusProcessed, Count: 100000, OldestAge: time.Hour},
				{Handler: "h.c", Status: InboxStatusIgnored, Count: 50, OldestAge: time.Hour},
			},
			expectedCounts: map[string]int64{},
			expectedAges:   map[string]float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			provider, reader := newTestMeterProvider(t)
			repo := &stubInboxStatsRepo{stats: tt.stats}

			_, err := registerInboxGauges(provider.Meter("test"), "core-service", repo)
			require.NoError(t, err)

			metrics := collectMetrics(t, reader)
			require.Equal(t, "core-service", repo.serviceName)

			// Other tests in the package register wrapped handlers, which this gauge zero-fills; only this case's handlers are asserted.
			counts := int64Points(t, metrics["messaging.inbox.messages"])
			ages := float64GaugePoints(t, metrics["messaging.inbox.oldest_unprocessed_age"])
			for _, s := range tt.stats {
				for key := range counts {
					if strings.HasPrefix(key, "handler="+s.Handler+",") {
						require.Contains(t, tt.expectedCounts, key)
					}
				}
			}
			for key, expected := range tt.expectedCounts {
				require.Equal(t, expected, counts[key], key)
			}
			for key, expected := range tt.expectedAges {
				require.Equal(t, expected, ages[key], key)
			}
		})
	}
}

func TestInboxGaugesReportZeroForWrappedHandlersWithNoRows(t *testing.T) {
	t.Parallel()
	provider, reader := newTestMeterProvider(t)
	handler := fmt.Sprintf("test.zero_fill.%d", time.Now().UnixNano())
	NewInboxConsumer(&mockInboxRepo{}, "core-service").Wrap(handler, func(context.Context, amqp.Delivery) error { return nil })

	_, err := registerInboxGauges(provider.Meter("test"), "core-service", &stubInboxStatsRepo{})
	require.NoError(t, err)

	metrics := collectMetrics(t, reader)
	counts := int64Points(t, metrics["messaging.inbox.messages"])
	for _, status := range []string{"received", "failed", "discarded"} {
		key := "handler=" + handler + ",status=" + status
		require.Contains(t, counts, key)
		require.Zero(t, counts[key])
	}
	ages := float64GaugePoints(t, metrics["messaging.inbox.oldest_unprocessed_age"])
	require.Contains(t, ages, "handler="+handler)
	require.Zero(t, ages["handler="+handler])
}

func TestInboxGaugesSkipObservationOnQueryError(t *testing.T) {
	t.Parallel()
	provider, reader := newTestMeterProvider(t)

	_, err := registerInboxGauges(provider.Meter("test"), "core-service", &stubInboxStatsRepo{err: errors.New("db down")})
	require.NoError(t, err)

	metrics := collectMetrics(t, reader)
	require.NotContains(t, metrics, "messaging.inbox.messages")
	require.NotContains(t, metrics, "messaging.inbox.oldest_unprocessed_age")
}

func TestEnqueuerCountsPublishesAndErrorsByMessageType(t *testing.T) {
	t.Parallel()
	provider, reader := newTestMeterProvider(t)

	batch := outboxBatch(0, 3)
	batch[2].MessageType = "other.event"
	repo := &mockEnqueuerRepo{batches: [][]*OutboxMessage{batch}}
	broker := &mockEnqueuerBroker{failKeys: map[string]bool{"rk_1": true}}
	e := newTestEnqueuer(t, repo, broker, 10)
	e.metrics = newMessagingMetrics(provider.Meter("test"))

	e.processBatch()

	metrics := collectMetrics(t, reader)
	require.Equal(t, map[string]int64{
		"message_type=test.event":  1,
		"message_type=other.event": 1,
	}, int64Points(t, metrics["messaging.outbox.published"]))
	require.Equal(t, map[string]int64{
		"message_type=test.event": 1,
	}, int64Points(t, metrics["messaging.outbox.publish_errors"]))
}

func TestInboxConsumerRecordsHandledOutcomes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		handler         func(c *InboxConsumer) MessageHandler
		expectedOutcome string
	}{
		{
			name: "success",
			handler: func(*InboxConsumer) MessageHandler {
				return func(context.Context, amqp.Delivery) error { return nil }
			},
			expectedOutcome: "success",
		},
		{
			name: "handler error",
			handler: func(*InboxConsumer) MessageHandler {
				return func(context.Context, amqp.Delivery) error { return errors.New("boom") }
			},
			expectedOutcome: "error",
		},
		{
			name: "discarded",
			handler: func(c *InboxConsumer) MessageHandler {
				return func(ctx context.Context, _ amqp.Delivery) error { return c.Discard(ctx, "malformed") }
			},
			expectedOutcome: "discarded",
		},
		{
			name: "ignored",
			handler: func(c *InboxConsumer) MessageHandler {
				return func(ctx context.Context, _ amqp.Delivery) error { return c.Ignore(ctx, "not ours") }
			},
			expectedOutcome: "ignored",
		},
		{
			name: "completed concurrently",
			handler: func(*InboxConsumer) MessageHandler {
				return func(context.Context, amqp.Delivery) error { return ErrInboxAlreadyCompleted }
			},
			expectedOutcome: "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			provider, reader := newTestMeterProvider(t)
			consumer := NewInboxConsumer(&mockInboxRepo{}, "core-service")
			consumer.metrics = newMessagingMetrics(provider.Meter("test"))

			_ = consumer.Wrap("test.handler", tt.handler(consumer))(context.Background(), deliveryWithMessageID("msg_1"))

			metrics := collectMetrics(t, reader)
			require.Equal(t, map[string]int64{
				"handler=test.handler,outcome=" + tt.expectedOutcome: 1,
			}, int64Points(t, metrics["messaging.inbox.handled"]))

			duration := metrics["messaging.inbox.handle_duration"]
			require.Equal(t, "s", duration.Unit)
			hist, ok := duration.Data.(metricdata.Histogram[float64])
			require.True(t, ok)
			require.Len(t, hist.DataPoints, 1)
			require.Equal(t, "handler=test.handler", attrKey(hist.DataPoints[0].Attributes))
			require.Equal(t, uint64(1), hist.DataPoints[0].Count)
			require.Equal(t, inboxHandleDurationBuckets, hist.DataPoints[0].Bounds)
		})
	}
}

func TestInboxConsumerDoesNotRecordSkippedDuplicates(t *testing.T) {
	t.Parallel()
	provider, reader := newTestMeterProvider(t)
	consumer := NewInboxConsumer(&mockInboxRepo{
		tryInsertFn: func(context.Context, InboxRecordInput) (int64, error) { return 0, mysqlDupError() },
		getByMessageAndHandler: func(context.Context, string, string) (*InboxRecord, error) {
			return &InboxRecord{ID: 1, Status: InboxStatusProcessed}, nil
		},
	}, "core-service")
	consumer.metrics = newMessagingMetrics(provider.Meter("test"))

	require.NoError(t, consumer.Wrap("test.handler", func(context.Context, amqp.Delivery) error {
		t.Fatal("a processed duplicate must not run the handler")
		return nil
	})(context.Background(), deliveryWithMessageID("msg_1")))

	require.NotContains(t, collectMetrics(t, reader), "messaging.inbox.handled")
}
