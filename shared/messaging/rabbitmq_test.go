package messaging

import (
	"context"
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

func TestPublishWithReconnectRetriesClosedChannel(t *testing.T) {
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
	require.True(t, shouldReconnect(amqp.ErrClosed))
	require.True(t, shouldReconnect(&amqp.Error{Code: amqp.ChannelError}))
	require.True(t, shouldReconnect(errors.New("Exception (504) Reason: \"channel/connection is not open\"")))

	require.False(t, shouldReconnect(nil))
	require.False(t, shouldReconnect(errors.New("other error")))
}
