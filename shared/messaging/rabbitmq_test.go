package messaging

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/id"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

func TestPublishWithReconnectRetriesClosedChannel(t *testing.T) {
	t.Parallel()
	t.Helper()

	ctx := context.Background()
	reconnectCalls := 0
	publishCalls := 0

	rmq := &rabbitMQ{}
	rmq.reconnectFunc = func(ctx context.Context) error {
		reconnectCalls++
		// mark channel/connection as open so subsequent ensureChannel checks pass
		rmq.conn = &amqp.Connection{}
		rmq.Channel = &amqp.Channel{}
		return nil
	}
	rmq.publishFunc = func(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
		publishCalls++
		if publishCalls == 1 {
			return amqp.ErrClosed
		}
		return nil
	}

	err := rmq.publishWithReconnect(ctx, "app", "test.key", amqp.Publishing{})
	require.NoError(t, err)
	require.Equal(t, 2, reconnectCalls) // initial ensure + retry after closed error
	require.Equal(t, 2, publishCalls)   // initial publish + retry publish
}

func TestPublishWithReconnectPropagatesNonRecoverableError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	expectedErr := errors.New("boom")
	reconnectCalls := 0
	publishCalls := 0

	rmq := &rabbitMQ{}
	rmq.reconnectFunc = func(ctx context.Context) error {
		reconnectCalls++
		rmq.conn = &amqp.Connection{}
		rmq.Channel = &amqp.Channel{}
		return nil
	}
	rmq.publishFunc = func(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
		publishCalls++
		return expectedErr
	}

	err := rmq.publishWithReconnect(ctx, "app", "test.key", amqp.Publishing{})
	require.ErrorIs(t, err, expectedErr)
	require.Equal(t, 1, publishCalls)
	require.Equal(t, 1, reconnectCalls) // only the initial ensureChannel reconnect
}

func TestPublishWithReconnectReturnsReconnectError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	reconnectCalls := 0
	publishCalls := 0
	expectedReconnectErr := errors.New("reconnect failed")

	rmq := &rabbitMQ{}
	rmq.reconnectFunc = func(ctx context.Context) error {
		reconnectCalls++
		// first reconnect (ensureChannel) succeeds, second fails
		if reconnectCalls == 1 {
			rmq.conn = &amqp.Connection{}
			rmq.Channel = &amqp.Channel{}
			return nil
		}
		return expectedReconnectErr
	}
	rmq.publishFunc = func(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
		publishCalls++
		return amqp.ErrClosed
	}

	err := rmq.publishWithReconnect(ctx, "app", "test.key", amqp.Publishing{})
	require.ErrorIs(t, err, expectedReconnectErr)
	require.Equal(t, 1, publishCalls)   // publish failed and never retried because reconnect failed
	require.Equal(t, 2, reconnectCalls) // ensureChannel + reconnect attempt after publish error
}

func TestShouldReconnectRecognizesExpectedErrors(t *testing.T) {
	t.Parallel()
	require.True(t, shouldReconnect(amqp.ErrClosed))
	require.True(t, shouldReconnect(&amqp.Error{Code: amqp.ChannelError}))
	require.True(t, shouldReconnect(errors.New("Exception (504) Reason: \"channel/connection is not open\"")))

	require.False(t, shouldReconnect(nil))
	require.False(t, shouldReconnect(errors.New("other error")))
}

func TestRabbitMQConfigValidate(t *testing.T) {
	t.Parallel()

	valid := func() *RabbitMQConfig {
		return &RabbitMQConfig{
			URI:               "amqp://guest:guest@localhost:5672/",
			ConnectionTimeout: time.Minute,
			MaxRetries:        3,
			InitialRetryWait:  time.Second,
			MaxRetryWait:      2 * time.Second,
			PrefetchCount:     1,
			ReconnectDelay:    time.Second,
		}
	}

	tests := []struct {
		name    string
		mutate  func(*RabbitMQConfig)
		wantErr string
	}{
		{name: "valid", mutate: func(*RabbitMQConfig) {}},
		{name: "empty URI", mutate: func(c *RabbitMQConfig) { c.URI = "" }, wantErr: "URI is empty"},
		{name: "non-positive connection timeout", mutate: func(c *RabbitMQConfig) { c.ConnectionTimeout = 0 }, wantErr: "connection timeout must be positive"},
		{name: "negative connection timeout", mutate: func(c *RabbitMQConfig) { c.ConnectionTimeout = -time.Second }, wantErr: "connection timeout must be positive"},
		{name: "non-positive max retries", mutate: func(c *RabbitMQConfig) { c.MaxRetries = 0 }, wantErr: "max retries must be positive"},
		{name: "non-positive initial retry wait", mutate: func(c *RabbitMQConfig) { c.InitialRetryWait = 0 }, wantErr: "initial retry wait must be positive"},
		{name: "max retry wait below initial", mutate: func(c *RabbitMQConfig) { c.MaxRetryWait = c.InitialRetryWait - time.Millisecond }, wantErr: "max retry wait must be >= initial retry wait"},
		{name: "non-positive prefetch count", mutate: func(c *RabbitMQConfig) { c.PrefetchCount = 0 }, wantErr: "prefetch count must be positive"},
		{name: "negative prefetch count", mutate: func(c *RabbitMQConfig) { c.PrefetchCount = -8 }, wantErr: "prefetch count must be positive"},
		{name: "non-positive reconnect delay", mutate: func(c *RabbitMQConfig) { c.ReconnectDelay = 0 }, wantErr: "reconnect delay must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := valid()
			tt.mutate(cfg)

			err := cfg.validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestRabbitMQConfigValidateRejectsNilConfig(t *testing.T) {
	t.Parallel()

	var cfg *RabbitMQConfig
	require.ErrorContains(t, cfg.validate(), "config is nil")
}

func TestNewRabbitMQRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	// Defaults fill every optional field, so an empty URI is the only way a config reaches validate() invalid.
	_, err := NewRabbitMQ(context.Background(), &RabbitMQConfig{})
	require.ErrorContains(t, err, "URI is empty")
}

func TestNewRabbitMQReturnsDialError(t *testing.T) {
	t.Parallel()

	_, err := NewRabbitMQ(context.Background(), &RabbitMQConfig{
		URI:              "://not-a-valid-amqp-uri",
		MaxRetries:       1,
		InitialRetryWait: time.Millisecond,
		MaxRetryWait:     2 * time.Millisecond,
	})
	require.ErrorContains(t, err, "failed to connect to rabbitMQ")
}

func TestReconnectReusesHealthyConnection(t *testing.T) {
	t.Parallel()

	conn := &amqp.Connection{}
	ch := &amqp.Channel{}
	rmq := &rabbitMQ{conn: conn, Channel: ch}

	require.NoError(t, rmq.reconnect(context.Background()))

	require.Same(t, conn, rmq.conn)
	require.Same(t, ch, rmq.Channel)
	require.NotNil(t, rmq.publishFunc)
	require.NotNil(t, rmq.reconnectFunc)
}

func TestReconnectLeavesBrokerUnreadyWhenDialFails(t *testing.T) {
	t.Parallel()

	rmq := &rabbitMQ{
		uri:              "://not-a-valid-amqp-uri",
		maxRetries:       1,
		initialRetryWait: time.Millisecond,
		maxRetryWait:     2 * time.Millisecond,
	}

	require.Error(t, rmq.reconnect(context.Background()))
	require.Nil(t, rmq.conn)
	require.Nil(t, rmq.Channel)
	require.False(t, rmq.IsReady())
}

func TestIsReady(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rmq  *rabbitMQ
		want bool
	}{
		{name: "never connected", rmq: &rabbitMQ{}, want: false},
		{name: "connection without channel", rmq: &rabbitMQ{conn: &amqp.Connection{}}, want: false},
		{name: "channel without connection", rmq: &rabbitMQ{Channel: &amqp.Channel{}}, want: false},
		{name: "connection and channel open", rmq: &rabbitMQ{conn: &amqp.Connection{}, Channel: &amqp.Channel{}}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.rmq.IsReady())
		})
	}
}

func TestCloseIsIdempotentOnNeverConnectedBroker(t *testing.T) {
	t.Parallel()

	rmq := &rabbitMQ{}
	rmq.Close()
	rmq.Close()

	require.False(t, rmq.IsReady())
}

func TestPublishAfterCloseReconnectsInsteadOfFailing(t *testing.T) {
	t.Parallel()

	var published []amqp.Publishing
	reconnects := 0

	rmq := &rabbitMQ{}
	rmq.reconnectFunc = func(context.Context) error {
		reconnects++
		rmq.conn = &amqp.Connection{}
		rmq.Channel = &amqp.Channel{}
		return nil
	}
	rmq.publishFunc = func(_ context.Context, _, _ string, msg amqp.Publishing) error {
		published = append(published, msg)
		return nil
	}

	rmq.Close()

	require.NoError(t, rmq.PublishMessage(context.Background(), ApplicationExchange, "test.key", contracts.AmqpMessage{}))
	require.Equal(t, 1, reconnects)
	require.Len(t, published, 1)
	require.True(t, strings.HasPrefix(published[0].MessageId, string(id.MessageIDPrefix)))
	require.True(t, rmq.IsReady())
}

func TestConsumeRedeclaresInstanceQueueOnEveryReconnect(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	declares := 0

	rmq := &rabbitMQ{reconnectDelay: time.Millisecond}
	// conn stays nil so consumerChannel fails every pass, driving the loop back through its reconnect path.
	rmq.reconnectFunc = func(context.Context) error { return nil }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, rmq.consume(ctx, "ws.events.instance", func() error {
		mu.Lock()
		defer mu.Unlock()
		declares++
		return nil
	}, func(context.Context, amqp.Delivery) error { return nil }))

	count := func() int {
		mu.Lock()
		defer mu.Unlock()
		return declares
	}

	require.Eventually(t, func() bool { return count() >= 3 }, 5*time.Second, time.Millisecond,
		"instance queue must be re-declared after every reconnect")

	cancel()

	prev := -1
	require.Eventually(t, func() bool {
		cur := count()
		stable := cur == prev
		prev = cur
		return stable
	}, 5*time.Second, 20*time.Millisecond, "consumer kept looping after its context was cancelled")
}

func TestConsumeRetriesWhenReconnectFails(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	attempts := 0

	rmq := &rabbitMQ{reconnectDelay: time.Millisecond}
	rmq.reconnectFunc = func(context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		attempts++
		return errors.New("broker down")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	declared := false
	require.NoError(t, rmq.consume(ctx, "some.queue", func() error {
		mu.Lock()
		defer mu.Unlock()
		declared = true
		return nil
	}, func(context.Context, amqp.Delivery) error { return nil }))

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return attempts >= 3
	}, 5*time.Second, time.Millisecond, "consumer must keep retrying while the broker is unreachable")

	mu.Lock()
	defer mu.Unlock()
	require.False(t, declared, "queue must not be declared while there is no channel")
}
