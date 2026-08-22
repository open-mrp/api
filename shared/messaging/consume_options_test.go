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
