package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/open-mrp/api/shared/appctx"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestTracedPublisherSpanNameUsesRoutingKey(t *testing.T) {
	// Not parallel: mutates global otel tracer provider.
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
	// Not parallel: mutates global otel tracer provider.
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
	t.Parallel()
	cases := map[string]string{
		"app/notification.cmd.send_email": "app.notification.cmd.send_email",
		"Notification-Cmd.SendEmail":      "notification_cmd.send_email",
		"":                                "unknown",
	}

	for input, expected := range cases {
		require.Equal(t, expected, normalizeMessagingName(input), input)
	}
}

func TestTracedPublisherToConsumerPropagatesTraceContext(t *testing.T) {
	// Not parallel: mutates global otel tracer provider and propagator.
	origTP := otel.GetTracerProvider()
	origProp := otel.GetTextMapPropagator()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(newPropagator())
	defer func() {
		otel.SetTracerProvider(origTP)
		otel.SetTextMapPropagator(origProp)
	}()

	ctx, root := tp.Tracer("test").Start(context.Background(), "root")

	var published amqp.Publishing
	err := TracedPublisher(ctx, "app", "notification.cmd.send_email", amqp.Publishing{}, func(_ context.Context, _, _ string, msg amqp.Publishing) error {
		published = msg
		return nil
	})
	require.NoError(t, err)
	root.End()

	require.NotEmpty(t, published.Headers["traceparent"], "publisher must inject traceparent into message headers")

	var handlerSpanCtx trace.SpanContext
	err = TracedConsumer(amqp.Delivery{
		RoutingKey: "notification.cmd.send_email",
		Exchange:   "app",
		Headers:    published.Headers,
	}, "notification_cmd_send_email", func(ctx context.Context, _ amqp.Delivery) error {
		handlerSpanCtx = trace.SpanContextFromContext(ctx)
		return nil
	})
	require.NoError(t, err)

	publishSpan := requireSpanNamed(t, spanRecorder, "rabbitmq.publish notification.cmd.send_email")
	consumeSpan := requireSpanNamed(t, spanRecorder, "rabbitmq.consume notification.cmd.send_email")

	require.Equal(t, publishSpan.SpanContext().TraceID(), consumeSpan.SpanContext().TraceID(), "consumer must join the publisher's trace")
	require.Equal(t, publishSpan.SpanContext().SpanID(), consumeSpan.Parent().SpanID(), "consumer span must be a child of the publish span")
	require.Equal(t, consumeSpan.SpanContext(), handlerSpanCtx, "handler context must carry the consumer span")
}

func TestTracedPublisherWithNoTraceInjectsNoHeaders(t *testing.T) {
	// Not parallel: mutates global otel tracer provider and propagator.
	origTP := otel.GetTracerProvider()
	origProp := otel.GetTextMapPropagator()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(newPropagator())
	defer func() {
		otel.SetTracerProvider(origTP)
		otel.SetTextMapPropagator(origProp)
	}()

	ctx, root := tp.Tracer("test").Start(context.Background(), "root")
	defer root.End()

	var published amqp.Publishing
	err := TracedPublisher(appctx.WithNoTrace(ctx), "app", "notification.cmd.send_email", amqp.Publishing{Headers: amqp.Table{"x": "y"}}, func(_ context.Context, _, _ string, msg amqp.Publishing) error {
		published = msg
		return nil
	})
	require.NoError(t, err)

	require.NotContains(t, published.Headers, "traceparent", "suppressed publish must not inject trace context")
	require.Empty(t, spanRecorder.Ended(), "suppressed publish must not create a span")
}

func TestTracedPublisherRecordsPublishError(t *testing.T) {
	// Not parallel: mutates global otel tracer provider.
	origTP := otel.GetTracerProvider()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	publishErr := errors.New("channel closed")
	err := TracedPublisher(context.Background(), "app", "notification.cmd.send_email", amqp.Publishing{}, func(context.Context, string, string, amqp.Publishing) error {
		return publishErr
	})
	require.ErrorIs(t, err, publishErr)

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, codes.Error, spans[0].Status().Code)
	require.Equal(t, publishErr.Error(), spans[0].Status().Description)
}

func TestTracedPublisherPreservesExistingHeaders(t *testing.T) {
	// Not parallel: mutates global otel tracer provider and propagator.
	origTP := otel.GetTracerProvider()
	origProp := otel.GetTextMapPropagator()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(tracetest.NewSpanRecorder()))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(newPropagator())
	defer func() {
		otel.SetTracerProvider(origTP)
		otel.SetTextMapPropagator(origProp)
	}()

	ctx, root := tp.Tracer("test").Start(context.Background(), "root")
	defer root.End()

	var published amqp.Publishing
	err := TracedPublisher(ctx, "app", "notification.cmd.send_email", amqp.Publishing{Headers: amqp.Table{"x-idempotency-key": "key_123"}}, func(_ context.Context, _, _ string, msg amqp.Publishing) error {
		published = msg
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, "key_123", published.Headers["x-idempotency-key"], "injection must not drop caller headers")
	require.NotEmpty(t, published.Headers["traceparent"])
}

func TestTracedConsumerUnlinkableHeadersStartRootSpan(t *testing.T) {
	// Not parallel: mutates global otel tracer provider and propagator.
	tests := []struct {
		name    string
		headers amqp.Table
	}{
		{
			name:    "nil headers",
			headers: nil,
		},
		{
			name:    "non-string traceparent",
			headers: amqp.Table{"traceparent": []byte("00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")},
		},
		{
			name:    "malformed traceparent",
			headers: amqp.Table{"traceparent": "not-a-traceparent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origTP := otel.GetTracerProvider()
			origProp := otel.GetTextMapPropagator()
			spanRecorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
			otel.SetTracerProvider(tp)
			otel.SetTextMapPropagator(newPropagator())
			defer func() {
				otel.SetTracerProvider(origTP)
				otel.SetTextMapPropagator(origProp)
			}()

			err := TracedConsumer(amqp.Delivery{RoutingKey: "notification.event.email_sent", Headers: tt.headers}, "queue", func(context.Context, amqp.Delivery) error {
				return nil
			})
			require.NoError(t, err)

			spans := spanRecorder.Ended()
			require.Len(t, spans, 1)
			require.False(t, spans[0].Parent().IsValid(), "unusable headers must yield a root span")
			require.True(t, spans[0].SpanContext().IsValid())
		})
	}
}

func TestTracedConsumerRecordsHandlerError(t *testing.T) {
	// Not parallel: mutates global otel tracer provider.
	origTP := otel.GetTracerProvider()
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	handlerErr := errors.New("handler blew up")
	err := TracedConsumer(amqp.Delivery{RoutingKey: "notification.event.email_sent", Headers: amqp.Table{}}, "queue", func(context.Context, amqp.Delivery) error {
		return handlerErr
	})
	require.ErrorIs(t, err, handlerErr)

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, codes.Error, spans[0].Status().Code)
	require.Equal(t, handlerErr.Error(), spans[0].Status().Description)
}

// The handler context is derived from context.Background(), so nothing a caller cancels can reach it. Pinned here because a shutdown that must interrupt in-flight handlers has to change this.
func TestTracedConsumerHandlerContextIsNotCancellable(t *testing.T) {
	// Not parallel: mutates global otel tracer provider.
	origTP := otel.GetTracerProvider()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(tracetest.NewSpanRecorder()))
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	var handlerCtx context.Context
	err := TracedConsumer(amqp.Delivery{RoutingKey: "notification.event.email_sent"}, "queue", func(ctx context.Context, _ amqp.Delivery) error {
		handlerCtx = ctx
		return nil
	})
	require.NoError(t, err)

	require.Nil(t, handlerCtx.Done(), "handler context has no cancellation channel")
	require.NoError(t, handlerCtx.Err())
	_, hasDeadline := handlerCtx.Deadline()
	require.False(t, hasDeadline)
}

func requireSpanNamed(t *testing.T, recorder *tracetest.SpanRecorder, name string) sdktrace.ReadOnlySpan {
	t.Helper()
	for _, span := range recorder.Ended() {
		if span.Name() == name {
			return span
		}
	}
	t.Fatalf("no ended span named %q", name)
	return nil
}
