package tracing

import (
	"context"
	"strings"
	"unicode"

	"github.com/open-mrp/api/shared/appctx"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// amqpHeadersCarrier adapts an amqp.Table (map[string]any) to the propagation.TextMapCarrier interface so the OpenTelemetry propagator can inject/extract W3C TraceContext headers into/from AMQP message headers. Only string values are supported by Get; non-string header values are silently ignored.
type amqpHeadersCarrier amqp.Table

// Get returns the string value for key, or "" if the key is missing or not a string.
func (c amqpHeadersCarrier) Get(key string) string {
	if v, ok := c[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Set stores a string value in the AMQP headers table.
func (c amqpHeadersCarrier) Set(key string, value string) {
	c[key] = value
}

// Keys returns all header names, satisfying the TextMapCarrier interface.
func (c amqpHeadersCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// TracedPublisher wraps a RabbitMQ publish call with an OpenTelemetry producer span. The span is named "rabbitmq.publish <normalized_routing_key>" and carries messaging.* semantic attributes (system, destination exchange, routing key, operation). Trace context is injected into the AMQP message headers so downstream consumers can link their spans to this publish.
//
// If the context has tracing disabled via [WithNoTrace] (e.g. outbox background publishing), the publish function is called directly with no span overhead.
func TracedPublisher(ctx context.Context, exchange, routingKey string, msg amqp.Publishing, publish func(context.Context, string, string, amqp.Publishing) error) error {
	if !appctx.ShouldTrace(ctx) {
		return publish(ctx, exchange, routingKey, msg)
	}

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

// TracedConsumer wraps a RabbitMQ message handler with an OpenTelemetry consumer span. It extracts trace context from the delivery's AMQP headers (injected by [TracedPublisher]) so the consumer span becomes a child of the producer span, forming a complete publish → consume trace.
//
// The span is named "rabbitmq.consume <normalized_name>" where the name is derived from the delivery's routing key (preferred) or the queueName fallback. Messaging semantic attributes (system, destination queue, exchange, routing key, operation) are attached.
//
// Unlike [TracedPublisher], this function does not check [ShouldTrace] because consumer spans are always desirable — the consumer has no way to know whether the publisher suppressed tracing, and the extracted parent context handles that naturally.
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

// normalizeMessagingName converts a routing key or queue name into a clean, lowercase span-name suffix. It applies the following transformations:
//   - Slashes are replaced with dots (e.g. "notification/cmd" → "notification.cmd").
//   - Spaces are replaced with underscores.
//   - PascalCase boundaries get underscore separators (e.g. "SendEmail" → "send_email").
//   - Consecutive separators are collapsed.
//   - Empty input returns "unknown".
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
