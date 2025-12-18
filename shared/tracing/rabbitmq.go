package tracing

import (
	"context"
	"strings"
	"unicode"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// amqpHeadersCarrier implements the TextMapCarrier interface for AMQP headers
type amqpHeadersCarrier amqp.Table

func (c amqpHeadersCarrier) Get(key string) string {
	if v, ok := c[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (c amqpHeadersCarrier) Set(key string, value string) {
	c[key] = value
}

func (c amqpHeadersCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// TracedPublisher wraps the RabbitMQ publish function with tracing.
// Creates producer spans named: rabbitmq.publish <routingKey>
func TracedPublisher(ctx context.Context, exchange, routingKey string, msg amqp.Publishing, publish func(context.Context, string, string, amqp.Publishing) error) error {
	tracer := otel.GetTracerProvider().Tracer("rabbitmq")

	spanName := "rabbitmq.publish " + normalizeMessagingName(routingKey)
	ctx, span := tracer.Start(ctx, spanName,
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination", exchange),
			attribute.String("messaging.destination_kind", "exchange"),
			attribute.String("messaging.rabbitmq.routing_key", routingKey),
			attribute.String("messaging.operation", "publish"),
		),
	)
	defer span.End()

	// Inject trace context into message headers
	if msg.Headers == nil {
		msg.Headers = make(amqp.Table)
	}
	carrier := amqpHeadersCarrier(msg.Headers)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	msg.Headers = amqp.Table(carrier)

	if err := publish(ctx, exchange, routingKey, msg); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

// TracedConsumer wraps the RabbitMQ message handler with tracing.
// Creates consumer spans named: rabbitmq.consume [queue]
// queueName should be provided to identify the consuming queue.
func TracedConsumer(delivery amqp.Delivery, queueName string, handler func(context.Context, amqp.Delivery) error) error {
	// Extract trace context from message headers
	carrier := amqpHeadersCarrier(delivery.Headers)
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)

	tracer := otel.GetTracerProvider().Tracer("rabbitmq")

	nameSource := delivery.RoutingKey
	if nameSource == "" {
		nameSource = queueName
	}

	spanName := "rabbitmq.consume " + normalizeMessagingName(nameSource)
	ctx, span := tracer.Start(ctx, spanName,
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination", queueName),
			attribute.String("messaging.destination_kind", "queue"),
			attribute.String("messaging.rabbitmq.exchange", delivery.Exchange),
			attribute.String("messaging.rabbitmq.routing_key", delivery.RoutingKey),
			attribute.String("messaging.operation", "consume"),
		),
	)
	defer span.End()

	if err := handler(ctx, delivery); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

func normalizeMessagingName(name string) string {
	if name == "" {
		return "unknown"
	}
	name = strings.ReplaceAll(name, "/", ".")
	name = strings.ReplaceAll(name, " ", "_")

	var b strings.Builder
	var prev rune
	var hasPrev bool

	writeSep := func(sep rune) {
		if !hasPrev || prev != sep {
			b.WriteRune(sep)
			prev = sep
			hasPrev = true
		}
	}

	for _, r := range name {
		switch {
		case r == '.':
			writeSep('.')
		case r == '_' || r == '-':
			writeSep('_')
		case unicode.IsUpper(r):
			if hasPrev && prev != '.' && prev != '_' && !unicode.IsUpper(prev) {
				b.WriteRune('_')
			}
			lower := unicode.ToLower(r)
			b.WriteRune(lower)
			prev = lower
			hasPrev = true
		default:
			lower := unicode.ToLower(r)
			b.WriteRune(lower)
			prev = lower
			hasPrev = true
		}
	}

	out := b.String()
	if out == "" {
		return "unknown"
	}
	return out
}
