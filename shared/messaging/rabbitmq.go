// Package messaging provides rabbitMQ integration, outbox/inbox patterns, and background workers for reliable asynchronous communication between microservices.
package messaging

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/retry"
	"github.com/augno/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// ApplicationExchange is the primary topic exchange that all services publish to and consume from. Messages are routed by their routing key (message type) to the appropriate queue bindings.
	ApplicationExchange = "app"
	// deadLetterExchange receives messages that have been rejected by consumers after exhausting retries. All application queues are configured with x-dead-letter-exchange pointing here, so failed messages are preserved for inspection rather than lost.
	deadLetterExchange = "dlx"
)

const (
	defaultConnectionTimeout = 2 * time.Minute
	defaultMaxRetries        = 10
	defaultInitialRetryWait  = 1 * time.Second
	defaultMaxRetryWait      = 10 * time.Second
	defaultPrefetchCount     = 1
	defaultReconnectDelay    = 5 * time.Second
)

// rabbitMQ manages a single AMQP connection and channel to a rabbitMQ broker. It implements the MessageBroker interface and handles automatic reconnection on channel/connection failures, declares the full exchange and queue topology on each (re)connect, and provides thread-safe publish and consume operations.
type rabbitMQ struct {
	uri               string
	connectionTimeout time.Duration
	maxRetries        int
	initialRetryWait  time.Duration
	maxRetryWait      time.Duration
	prefetchCount     int
	reconnectDelay    time.Duration

	conn    *amqp.Connection
	Channel *amqp.Channel

	mu            sync.Mutex
	publishFunc   func(context.Context, string, string, amqp.Publishing) error
	reconnectFunc func(context.Context) error

	// consumerRetry overrides the per-delivery handler retry policy; nil means the package retry defaults. Indirected for testability.
	consumerRetry *retry.Config
}

// RabbitMQConfig represents the configuration for the rabbitMQ client.
type RabbitMQConfig struct {
	// URI (required) is the rabbitMQ connection URI.
	URI string

	// ConnectionTimeout (optional; default: 2m) is the overall timeout for the initial connection attempt.
	ConnectionTimeout time.Duration

	// MaxRetries (optional; default: 10) is the maximum number of connection dial retries.
	MaxRetries int

	// InitialRetryWait (optional; default: 1s) is the starting backoff interval between connection retries.
	InitialRetryWait time.Duration

	// MaxRetryWait (optional; default: 10s) is the maximum backoff interval between connection retries.
	MaxRetryWait time.Duration

	// PrefetchCount (optional; default: 1) is the QoS prefetch limit per consumer.
	PrefetchCount int

	// ReconnectDelay (optional; default: 5s) is how long to wait before retrying after a consumer failure.
	ReconnectDelay time.Duration
}

// WithDefaults returns a new RabbitMQConfig with all zero-value optional fields replaced by production defaults. It is safe to call on a nil receiver. The original config is not mutated; a copy is always returned.
func (c *RabbitMQConfig) WithDefaults() *RabbitMQConfig {
	if c == nil {
		c = &RabbitMQConfig{}
	}

	return &RabbitMQConfig{
		URI:               c.URI,
		ConnectionTimeout: cmp.Or(c.ConnectionTimeout, defaultConnectionTimeout),
		MaxRetries:        cmp.Or(c.MaxRetries, defaultMaxRetries),
		InitialRetryWait:  cmp.Or(c.InitialRetryWait, defaultInitialRetryWait),
		MaxRetryWait:      cmp.Or(c.MaxRetryWait, defaultMaxRetryWait),
		PrefetchCount:     cmp.Or(c.PrefetchCount, defaultPrefetchCount),
		ReconnectDelay:    cmp.Or(c.ReconnectDelay, defaultReconnectDelay),
	}
}

func (c *RabbitMQConfig) validate() error {
	if c == nil {
		return fmt.Errorf("rabbitMQ: config is nil")
	}
	if c.URI == "" {
		return fmt.Errorf("rabbitMQ: URI is empty")
	}
	if c.ConnectionTimeout <= 0 {
		return fmt.Errorf("rabbitMQ: connection timeout must be positive")
	}
	if c.MaxRetries <= 0 {
		return fmt.Errorf("rabbitMQ: max retries must be positive")
	}
	if c.InitialRetryWait <= 0 {
		return fmt.Errorf("rabbitMQ: initial retry wait must be positive")
	}
	if c.MaxRetryWait < c.InitialRetryWait {
		return fmt.Errorf("rabbitMQ: max retry wait must be >= initial retry wait")
	}
	if c.PrefetchCount <= 0 {
		return fmt.Errorf("rabbitMQ: prefetch count must be positive")
	}
	if c.ReconnectDelay <= 0 {
		return fmt.Errorf("rabbitMQ: reconnect delay must be positive")
	}
	return nil
}

// NewRabbitMQ creates a new rabbitMQ client connected to the given AMQP URI. It dials the broker with exponential backoff, declares the full exchange/queue topology, and verifies the connection is ready. On success the returned client is guaranteed to have an open connection and channel.
func NewRabbitMQ(ctx context.Context, config *RabbitMQConfig) (MessageBroker, error) {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}

	rmq := &rabbitMQ{
		uri:               config.URI,
		connectionTimeout: config.ConnectionTimeout,
		maxRetries:        config.MaxRetries,
		initialRetryWait:  config.InitialRetryWait,
		maxRetryWait:      config.MaxRetryWait,
		prefetchCount:     config.PrefetchCount,
		reconnectDelay:    config.ReconnectDelay,
	}

	connectCtx, cancel := context.WithTimeout(ctx, config.ConnectionTimeout)
	defer cancel()

	if err := rmq.reconnect(connectCtx); err != nil {
		return nil, err
	}

	if !rmq.IsReady() {
		return nil, fmt.Errorf("rabbitMQ connection is not ready")
	}

	return rmq, nil
}

// ConsumeMessages starts consuming from the given queue in a background goroutine. The goroutine runs an infinite loop that re-establishes the channel and consumer on any connection interruption. Deliveries are processed by a pool of Concurrency worker goroutines (default 1); for each delivery:
//  1. The message is wrapped in a traced span via tracing.TracedConsumer.
//  2. The handler is called with exponential backoff retries (via retry.WithBackoff).
//  3. On success the delivery is ACKed. On exhausted retries the delivery is rejected
//     without requeue, sending it to the dead-letter queue with diagnostic headers
//     (x-death-reason, x-retry-count, etc.).
//
// With the default Concurrency of 1, QoS prefetch is the broker default (1) and each message is fully processed before the next is delivered — strict in-order consumption. With Concurrency > 1, prefetch is raised to 2x the worker count so workers stay fed, and messages on this queue are processed (and ACKed) out of order; see ConsumeOptions.Concurrency for when that is safe.
func (r *rabbitMQ) ConsumeMessages(ctx context.Context, queueName string, handler MessageHandler, opts ...ConsumeOption) error {
	options := ConsumeOptions{Concurrency: 1}
	for _, opt := range opts {
		opt(&options)
	}
	concurrency := max(options.Concurrency, 1)

	prefetch := r.prefetchCount
	if concurrency > 1 {
		prefetch = concurrency * 2
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("Context cancelled, stopping consumer", "queue", queueName)
				return
			default:
			}

			if err := r.ensureChannel(ctx); err != nil {
				slog.Error("Failed to ensure channel, retrying", "queue", queueName, "error", err, "retry_delay", r.reconnectDelay)
				select {
				case <-ctx.Done():
					return
				case <-time.After(r.reconnectDelay):
				}
				continue
			}

			err := r.Channel.Qos(
				prefetch, // prefetchCount
				0,        // prefetchSize
				false,    // global
			)
			if err != nil {
				slog.Error("Failed to set QoS, retrying", "queue", queueName, "error", err, "retry_delay", r.reconnectDelay)
				select {
				case <-ctx.Done():
					return
				case <-time.After(r.reconnectDelay):
				}
				continue
			}

			msgs, err := r.Channel.Consume(
				queueName, // queue
				"",        // consumer
				false,     // auto-ack
				false,     // exclusive
				false,     // no-local
				false,     // no-wait
				nil,       // args
			)
			if err != nil {
				slog.Error("Failed to start consume, retrying", "queue", queueName, "error", err, "retry_delay", r.reconnectDelay)
				select {
				case <-ctx.Done():
					return
				case <-time.After(r.reconnectDelay):
				}
				continue
			}

			slog.Info("Started consuming", "queue", queueName, "concurrency", concurrency, "prefetch", prefetch)

			// Workers exit when the deliveries channel closes (connection loss or shutdown). The pool is re-created on each (re)connect.
			var wg sync.WaitGroup
			for range concurrency {
				wg.Go(func() {
					for msg := range msgs {
						r.processDelivery(ctx, queueName, handler, msg)
					}
				})
			}

			workersDone := make(chan struct{})
			go func() {
				wg.Wait()
				close(workersDone)
			}()

			select {
			case <-ctx.Done():
				slog.Info("Context cancelled, stopping consumer", "queue", queueName)
				return
			case <-workersDone:
			}

			slog.Info("Consumption loop ended, reconnecting", "queue", queueName)
			select {
			case <-ctx.Done():
				return
			case <-time.After(r.reconnectDelay):
			}
		}
	}()

	return nil
}

// processDelivery handles one AMQP delivery: traced span, handler invocation with backoff retries, ACK on success, and rejection to the dead-letter queue (with diagnostic headers) when retries are exhausted.
func (r *rabbitMQ) processDelivery(ctx context.Context, queueName string, handler MessageHandler, msg amqp.Delivery) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	if err := tracing.TracedConsumer(msg, queueName, func(ctx context.Context, d amqp.Delivery) error {
		retryCfg := r.consumerRetry
		if retryCfg == nil {
			retryCfg = new(retry.Config).WithDefaults()
		}
		err := retry.WithBackoff(ctx, retryCfg, func() error {
			return handler(ctx, d)
		})
		if err != nil {
			// Add failure context before sending to the DLQ
			headers := amqp.Table{}
			if d.Headers != nil {
				headers = d.Headers
			}

			headers["x-death-reason"] = err.Error()
			headers["x-origin-exchange"] = d.Exchange
			headers["x-original-routing-key"] = d.RoutingKey
			headers["x-retry-count"] = retryCfg.MaxRetries
			d.Headers = headers

			// Reject without requeue - message will go to the DLQ
			_ = d.Reject(false)
			return err
		}

		// Only Ack if the handler succeeds
		if ackErr := msg.Ack(false); ackErr != nil {
			slog.Error("Failed to Ack message", "error", ackErr, "body", string(msg.Body))
		}

		return nil
	}); err != nil {
		slog.Error("Error processing message", "error", err)
	}
}

// PublishMessage serializes an AmqpMessage to JSON and publishes it to the given exchange with the specified routing key. If the message has no MessageID, one is auto-generated using the shared id package (msg_ prefix, 22 chars). The publish is traced via tracing.TracedPublisher and uses publishWithReconnect to transparently recover from connection failures.
func (r *rabbitMQ) PublishMessage(ctx context.Context, exchange, routingKey string, message contracts.AmqpMessage) error {
	if message.MessageID == "" {
		length := id.IDLength22
		msgID, _ := id.GenID(id.MessageIDPrefix, &length)
		message.MessageID = msgID
	}

	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	msg := amqp.Publishing{
		MessageId:    message.MessageID,
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         jsonMsg,
	}

	return tracing.TracedPublisher(ctx, exchange, routingKey, msg, r.publishWithReconnect)
}

// publish sends a single AMQP publishing to the given exchange and routing key. This is the low-level publish that assumes the channel is already open. It is called through publishFunc, which is indirected for testability.
func (r *rabbitMQ) publish(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	return r.Channel.PublishWithContext(ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		msg,
	)
}

// publishWithReconnect is a resilient publish wrapper. It first ensures the channel is open (reconnecting if needed), then attempts the publish. If the publish fails with a recoverable error (closed connection, channel error, or connection forced), it reconnects once and retries. Non-recoverable errors are returned immediately.
func (r *rabbitMQ) publishWithReconnect(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	if err := r.ensureChannel(ctx); err != nil {
		return err
	}

	if err := r.publishFunc(ctx, exchange, routingKey, msg); err != nil {
		if shouldReconnect(err) {
			if recErr := r.reconnectFunc(ctx); recErr != nil {
				return fmt.Errorf("reconnect after publish failure: %w", recErr)
			}
			return r.publishFunc(ctx, exchange, routingKey, msg)
		}
		return err
	}

	return nil
}

// ensureChannel checks whether the connection and channel are still open and triggers a reconnect if either has been closed. The check is performed under the mutex to get a consistent snapshot of both conn and Channel state.
func (r *rabbitMQ) ensureChannel(ctx context.Context) error {
	r.mu.Lock()
	needsReconnect := r.conn == nil || r.conn.IsClosed() || r.Channel == nil || r.Channel.IsClosed()
	r.mu.Unlock()

	if !needsReconnect {
		return nil
	}

	return r.reconnectFunc(ctx)
}

// reconnect establishes a fresh AMQP connection and channel, replacing any stale resources. It is called both during initial startup (from NewRabbitMQ) and during runtime recovery (from ensureChannel or publishWithReconnect).
//
// The method is guarded by mu and performs a double-check: if another goroutine already reconnected while we were waiting for the lock, the existing healthy connection is reused. On the first call it also initializes publishFunc and reconnectFunc to their default implementations.
//
// After dialing and opening a channel, it declares the full exchange/queue topology via setupExchangesAndQueues. If topology setup fails, the connection is closed and the error is returned.
func (r *rabbitMQ) reconnect(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.publishFunc == nil {
		r.publishFunc = r.publish
	}
	if r.reconnectFunc == nil {
		r.reconnectFunc = r.reconnect
	}

	if r.conn != nil && !r.conn.IsClosed() && r.Channel != nil && !r.Channel.IsClosed() {
		return nil
	}

	// Clean up any stale resources before reconnecting.
	if r.Channel != nil {
		_ = r.Channel.Close()
		r.Channel = nil
	}
	if r.conn != nil {
		_ = r.conn.Close()
		r.conn = nil
	}

	conn, ch, err := r.connect(ctx)
	if err != nil {
		return err
	}

	r.conn = conn
	r.Channel = ch

	if err := r.setupExchangesAndQueues(); err != nil {
		r.Close()
		return fmt.Errorf("failed to setup exchanges and queues: %v", err)
	}

	return nil
}

// connect dials the AMQP broker with exponential backoff (up to 10 retries, 1–10s wait) and opens a channel on the new connection. Returns both the connection and channel on success. If the dial exhausts retries or the channel open fails, all resources are cleaned up before returning the error.
func (r *rabbitMQ) connect(ctx context.Context) (*amqp.Connection, *amqp.Channel, error) {
	cfg := retry.Config{
		MaxRetries:  r.maxRetries,
		InitialWait: r.initialRetryWait,
		MaxWait:     r.maxRetryWait,
	}

	var conn *amqp.Connection
	if err := retry.WithBackoff(ctx, &cfg, func() error {
		c, err := amqp.Dial(r.uri)
		if err != nil {
			return fmt.Errorf("failed to connect to rabbitMQ: %v", err)
		}
		conn = c
		return nil
	}); err != nil {
		return nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to create channel: %v", err)
	}

	return conn, ch, nil
}

// shouldReconnect returns true if the given error indicates a connection or channel level failure that can be recovered by re-dialing the broker. It matches against amqp.ErrClosed, AMQP channel/connection error codes, and the "not open" string pattern. Returning false means the error is application-level and reconnecting would not help (e.g. exchange not found, permission denied).
func shouldReconnect(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, amqp.ErrClosed) {
		return true
	}

	var amqpErr *amqp.Error
	if errors.As(err, &amqpErr) {
		return amqpErr != nil && (amqpErr.Code == amqp.ChannelError || amqpErr.Code == amqp.ConnectionForced)
	}

	return strings.Contains(err.Error(), "channel/connection is not open")
}

// setupDeadLetterExchange declares the dead-letter exchange (DLX), its catch-all queue, and binds them with the "#" wildcard routing key. Any message rejected by a consumer (after retry exhaustion) is routed here for post-mortem inspection.
func (r *rabbitMQ) setupDeadLetterExchange() error {
	err := r.Channel.ExchangeDeclare(
		deadLetterExchange,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter exchange: %v", err)
	}

	// Declare the dead letter queue
	q, err := r.Channel.QueueDeclare(
		DeadLetterQueue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter queue: %v", err)
	}

	// Bind the queue to the exchange with a wildcard routing key
	err = r.Channel.QueueBind(
		q.Name,
		"#", // wildcard routing key to catch all messages
		deadLetterExchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind dead letter queue: %v", err)
	}

	return nil
}

// setupExchangesAndQueues declares the full AMQP topology: the dead-letter exchange, the application topic exchange, and all service queues with their routing key bindings. This is called on every (re)connect to ensure the topology exists even if the broker was reset. All queues are durable and configured with x-dead-letter-exchange so rejected messages flow to the DLQ.
func (r *rabbitMQ) setupExchangesAndQueues() error {
	if err := r.setupDeadLetterExchange(); err != nil {
		return err
	}

	err := r.Channel.ExchangeDeclare(
		ApplicationExchange, // name
		"topic",             // type
		true,                // durable
		false,               // auto-deleted
		false,               // internal
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %s: %v", ApplicationExchange, err)
	}
	// Notification command queue
	if err := r.declareAndBindQueue(
		NotificationCmdSendEmailQueue,
		[]string{string(contracts.NotificationCmdSendEmail)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Notification event queues
	if err := r.declareAndBindQueue(
		NotifyEmailStatusQueue,
		[]string{string(contracts.NotificationEventEmailSent), string(contracts.NotificationEventEmailFailed)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Email log event queue (internal to notification service)
	if err := r.declareAndBindQueue(
		NotificationEventEmailLogQueue,
		[]string{string(contracts.NotificationEventEmailSent)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// In-app messaging fan-out command queue (handled by notification-service)
	if err := r.declareAndBindQueue(
		NotificationCmdFanoutQueue,
		[]string{string(contracts.NotificationCmdFanout), string(contracts.NotificationCmdSendMessage)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Agent chat-reply command queue (handled by notification-service)
	if err := r.declareAndBindQueue(
		NotificationCmdAgentReplyQueue,
		[]string{string(contracts.NotificationCmdAgentReply)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Agent chat-reply streaming patch queue (handled by notification-service). Best-effort partial body updates for an in-flight streaming reply; not inbox-deduped.
	if err := r.declareAndBindQueue(
		NotificationCmdAgentReplyPatchQueue,
		[]string{string(contracts.NotificationCmdAgentReplyPatch)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Request log event queue (handled by platform-service)
	if err := r.declareAndBindQueue(
		LoggingEventRequestLogQueue,
		[]string{string(contracts.LoggingEventRequestLogged)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Audit event queue (handled by platform-service)
	if err := r.declareAndBindQueue(
		PlatformEventAuditLogQueue,
		[]string{string(contracts.PlatformEventAuditLogged)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Core purge account data command queue (handled by core-service)
	if err := r.declareAndBindQueue(
		CoreCmdPurgeAccountDataQueue,
		[]string{string(contracts.CoreCmdPurgeAccountData)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Core seed sandbox command queue (handled by core-service)
	if err := r.declareAndBindQueue(
		CoreCmdSeedSandboxQueue,
		[]string{string(contracts.CoreCmdSeedSandbox)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Core execute production step command queue (handled by core-service)
	if err := r.declareAndBindQueue(
		CoreCmdExecuteProductionStepQueue,
		[]string{string(contracts.CoreCmdExecuteProductionStep)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Core sales order created event queue (handled by core-service)
	if err := r.declareAndBindQueue(
		CoreEventSalesOrderCreatedQueue,
		[]string{string(contracts.CoreEventSalesOrderCreated)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Core HubSpot sync command queue (handled by core-service) — bound to both preview and execute commands
	if err := r.declareAndBindQueue(
		CoreCmdHubspotSyncQueue,
		[]string{string(contracts.CoreCmdHubspotSyncPreview), string(contracts.CoreCmdHubspotSyncExecute)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Billing stripe webhook event queue (handled by billing-service)
	if err := r.declareAndBindQueue(
		BillingEventStripeWebhookQueue,
		[]string{string(contracts.BillingEventStripeWebhook)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Billing sync seats command queue (handled by billing-service)
	if err := r.declareAndBindQueue(
		BillingCmdSyncSeatsQueue,
		[]string{string(contracts.BillingCmdSyncSeats)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Billing report seat change command queue (handled by billing-service)
	if err := r.declareAndBindQueue(
		BillingCmdReportSeatChangeQueue,
		[]string{string(contracts.BillingCmdReportSeatChange)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Agent command queue: execute run (handled by agent-service)
	if err := r.declareAndBindQueue(
		AgentCmdExecuteRunQueue,
		[]string{string(contracts.AgentCmdExecuteRun)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Agent command queue: chat run (from notification-service — start an agent run from a chat mention)
	if err := r.declareAndBindQueue(
		AgentCmdChatRunQueue,
		[]string{string(contracts.AgentCmdChatRun)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Agent command queue: execute action (handled by agent-service)
	if err := r.declareAndBindQueue(
		AgentCmdExecuteActionQueue,
		[]string{string(contracts.AgentCmdExecuteAction)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Agent command queue: continue run (handled by agent-service)
	if err := r.declareAndBindQueue(
		AgentCmdContinueRunQueue,
		[]string{string(contracts.AgentCmdContinueRun)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Agent event queue: run completed
	if err := r.declareAndBindQueue(
		AgentEventRunCompletedQueue,
		[]string{string(contracts.AgentEventRunCompleted)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Agent event queue: run step (for WebSocket streaming)
	if err := r.declareAndBindQueue(
		AgentEventRunStepQueue,
		[]string{string(contracts.AgentEventRunStep)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Notification realtime-delivery queue (for WebSocket fan-out of the bell + live chat)
	if err := r.declareAndBindQueue(
		NotificationEventDeliveredQueue,
		[]string{string(contracts.NotificationEventDelivered), string(contracts.NotificationEventConversationUpdated)},
		ApplicationExchange,
	); err != nil {
		return err
	}

	return nil
}

// declareAndBindQueue declares a durable queue with dead-letter routing to the DLX and binds it to the given exchange for each of the specified message type routing keys. This is the building block used by setupExchangesAndQueues for each application queue.
func (r *rabbitMQ) declareAndBindQueue(queueName string, messageTypes []string, exchange string) error {
	args := amqp.Table{
		"x-dead-letter-exchange": deadLetterExchange,
	}

	q, err := r.Channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		args,      // arguments with DLX config
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %v", queueName, err)
	}

	for _, msg := range messageTypes {
		if err := r.Channel.QueueBind(
			q.Name,   // queue name
			msg,      // routing key
			exchange, // exchange
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind queue to %s: %v", queueName, err)
		}
	}

	return nil
}

// IsReady reports whether the broker connection and channel are ready for use.
func (r *rabbitMQ) IsReady() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.conn != nil && !r.conn.IsClosed() && r.Channel != nil && !r.Channel.IsClosed()
}

// Close shuts down the AMQP channel and connection. Safe to call multiple times.
func (r *rabbitMQ) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Channel != nil {
		_ = r.Channel.Close()
	}
	if r.conn != nil {
		_ = r.conn.Close()
	}
}
