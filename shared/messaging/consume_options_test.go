package messaging

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/retry"
	amqp "github.com/rabbitmq/amqp091-go"
)

// fastConsumerRetry keeps the failure-path tests fast; behavior (retry then
// dead-letter) is identical to the production defaults.
func fastConsumerRetry() *retry.Config {
	return (&retry.Config{
		MaxRetries:  1,
		InitialWait: time.Millisecond,
		MaxWait:     2 * time.Millisecond,
	}).WithDefaults()
}

// fakeAcknowledger records ack/reject outcomes for a delivery.
type fakeAcknowledger struct {
	mu       sync.Mutex
	acked    bool
	rejected bool
	requeue  bool
}

func (f *fakeAcknowledger) Ack(_ uint64, _ bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.acked = true
	return nil
}

func (f *fakeAcknowledger) Nack(_ uint64, _ bool, requeue bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rejected = true
	f.requeue = requeue
	return nil
}

func (f *fakeAcknowledger) Reject(_ uint64, requeue bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rejected = true
	f.requeue = requeue
	return nil
}

func TestWithConcurrencyOption(t *testing.T) {
	t.Parallel()

	options := ConsumeOptions{Concurrency: 1}
	WithConcurrency(8)(&options)

	if options.Concurrency != 8 {
		t.Errorf("expected concurrency 8, got %d", options.Concurrency)
	}
}

func TestProcessDeliveryAcksOnSuccess(t *testing.T) {
	t.Parallel()

	r := &rabbitMQ{}
	ack := &fakeAcknowledger{}
	delivery := amqp.Delivery{Acknowledger: ack, Body: []byte(`{}`)}

	r.processDelivery(context.Background(), "test-queue", func(context.Context, amqp.Delivery) error {
		return nil
	}, delivery)

	if !ack.acked {
		t.Error("expected delivery to be acked on handler success")
	}
	if ack.rejected {
		t.Error("did not expect delivery to be rejected")
	}
}

func TestProcessDeliveryRejectsToDLQOnFailure(t *testing.T) {
	t.Parallel()

	r := &rabbitMQ{consumerRetry: fastConsumerRetry()}
	ack := &fakeAcknowledger{}
	delivery := amqp.Delivery{Acknowledger: ack, Body: []byte(`{}`)}

	calls := 0
	r.processDelivery(context.Background(), "test-queue", func(context.Context, amqp.Delivery) error {
		calls++
		return errors.New("handler failure")
	}, delivery)

	if !ack.rejected {
		t.Error("expected delivery to be rejected after retry exhaustion")
	}
	if ack.requeue {
		t.Error("expected reject without requeue (message should dead-letter)")
	}
	if ack.acked {
		t.Error("did not expect delivery to be acked")
	}
	if calls < 2 {
		t.Errorf("expected the handler to be retried before dead-lettering, got %d calls", calls)
	}
}

func TestProcessDeliverySkipsWhenContextCancelled(t *testing.T) {
	t.Parallel()

	r := &rabbitMQ{}
	ack := &fakeAcknowledger{}
	delivery := amqp.Delivery{Acknowledger: ack, Body: []byte(`{}`)}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	r.processDelivery(ctx, "test-queue", func(context.Context, amqp.Delivery) error {
		called = true
		return nil
	}, delivery)

	if called {
		t.Error("expected handler to be skipped when context is cancelled")
	}
}

// failingAcknowledger reports ack/reject outcomes and can fail either operation, the way a broker does when the channel dies mid-delivery.
type failingAcknowledger struct {
	mu        sync.Mutex
	ackErr    error
	rejectErr error
	acks      int
	rejects   int
}

func (f *failingAcknowledger) Ack(_ uint64, _ bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.acks++
	return f.ackErr
}

func (f *failingAcknowledger) Nack(_ uint64, _ bool, _ bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rejects++
	return f.rejectErr
}

func (f *failingAcknowledger) Reject(_ uint64, _ bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rejects++
	return f.rejectErr
}

func TestProcessDeliverySwallowsAckFailure(t *testing.T) {
	t.Parallel()

	r := &rabbitMQ{}
	ack := &failingAcknowledger{ackErr: errors.New("channel closed")}
	delivery := amqp.Delivery{Acknowledger: ack, Body: []byte(`{}`)}

	r.processDelivery(context.Background(), "test-queue", func(context.Context, amqp.Delivery) error {
		return nil
	}, delivery)

	if ack.acks != 1 {
		t.Errorf("expected exactly one ack attempt, got %d", ack.acks)
	}
	// A failed ack must not become a reject: the broker will redeliver the message, and dead-lettering it here would discard work the handler already completed.
	if ack.rejects != 0 {
		t.Errorf("expected no reject after a failed ack, got %d", ack.rejects)
	}
}

func TestProcessDeliverySwallowsRejectFailure(t *testing.T) {
	t.Parallel()

	r := &rabbitMQ{consumerRetry: fastConsumerRetry()}
	ack := &failingAcknowledger{rejectErr: errors.New("channel closed")}
	delivery := amqp.Delivery{Acknowledger: ack, Body: []byte(`{}`)}

	r.processDelivery(context.Background(), "test-queue", func(context.Context, amqp.Delivery) error {
		return errors.New("handler failure")
	}, delivery)

	if ack.rejects != 1 {
		t.Errorf("expected exactly one reject attempt, got %d", ack.rejects)
	}
	if ack.acks != 0 {
		t.Errorf("expected no ack, got %d", ack.acks)
	}
}

func TestProcessDeliveryStampsDeadLetterHeaders(t *testing.T) {
	t.Parallel()

	retryCfg := fastConsumerRetry()
	r := &rabbitMQ{consumerRetry: retryCfg}
	headers := amqp.Table{"x-existing": "kept"}
	delivery := amqp.Delivery{
		Acknowledger: &fakeAcknowledger{},
		Headers:      headers,
		Exchange:     "app",
		RoutingKey:   "sales_order.created",
		Body:         []byte(`{}`),
	}

	r.processDelivery(context.Background(), "test-queue", func(context.Context, amqp.Delivery) error {
		return errors.New("handler failure")
	}, delivery)

	if got := headers["x-death-reason"]; got != "handler failure" {
		t.Errorf("expected the handler error as x-death-reason, got %v", got)
	}
	if got := headers["x-origin-exchange"]; got != "app" {
		t.Errorf("expected x-origin-exchange app, got %v", got)
	}
	if got := headers["x-original-routing-key"]; got != "sales_order.created" {
		t.Errorf("expected x-original-routing-key sales_order.created, got %v", got)
	}
	if got := headers["x-retry-count"]; got != retryCfg.MaxRetries {
		t.Errorf("expected x-retry-count %d, got %v", retryCfg.MaxRetries, got)
	}
	if got := headers["x-existing"]; got != "kept" {
		t.Errorf("expected pre-existing headers to be preserved, got %v", got)
	}
}

func TestProcessDeliveryHandlerContextIsDetachedFromConsumer(t *testing.T) {
	t.Parallel()

	r := &rabbitMQ{consumerRetry: fastConsumerRetry()}
	ack := &fakeAcknowledger{}
	delivery := amqp.Delivery{Acknowledger: ack, Body: []byte(`{}`)}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	calls := 0
	var lastHandlerErr error
	r.processDelivery(ctx, "test-queue", func(handlerCtx context.Context, _ amqp.Delivery) error {
		calls++
		cancel()
		lastHandlerErr = handlerCtx.Err()
		return errors.New("handler failure")
	}, delivery)

	// TracedConsumer starts the handler context from context.Background(), so shutdown cannot interrupt an in-flight handler or its retries.
	if lastHandlerErr != nil {
		t.Errorf("expected the handler context to be unaffected by consumer cancellation, got %v", lastHandlerErr)
	}
	if calls != 2 {
		t.Errorf("expected the handler to be retried after consumer cancellation, got %d calls", calls)
	}
	if !ack.rejected {
		t.Error("expected the delivery to be dead-lettered after retry exhaustion")
	}
}
