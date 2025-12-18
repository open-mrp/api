package tracing

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTracedPublisherSpanNameUsesRoutingKey(t *testing.T) {
	origTP := otel.GetTracerProvider()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	err := TracedPublisher(context.Background(), "app", "notification.cmd.send_email", amqp.Publishing{}, func(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
		return nil
	})
	require.NoError(t, err)

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "rabbitmq.publish notification.cmd.send_email", spans[0].Name())
}

func TestTracedConsumerSpanNamePrefersRoutingKey(t *testing.T) {
	origTP := otel.GetTracerProvider()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	delivery := amqp.Delivery{
		RoutingKey: "notification.event.email_sent",
		Exchange:   "app",
		Headers:    amqp.Table{},
	}
	err := TracedConsumer(delivery, "notification_event_email_log", func(ctx context.Context, _ amqp.Delivery) error {
		return nil
	})
	require.NoError(t, err)

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "rabbitmq.consume notification.event.email_sent", spans[0].Name())
}

func TestNormalizeMessagingNameHandlesSeparators(t *testing.T) {
	cases := map[string]string{
		"app/notification.cmd.send_email": "app.notification.cmd.send_email",
		"Notification-Cmd.SendEmail":      "notification_cmd.send_email",
		"":                                "unknown",
	}

	for input, expected := range cases {
		require.Equal(t, expected, normalizeMessagingName(input), input)
	}
}
