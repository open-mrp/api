package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/retry"
	"github.com/augno/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ApplicationExchange = "app"
	DeadLetterExchange  = "dlx"
)

type RabbitMQ struct {
	uri string

	conn    *amqp.Connection
	Channel *amqp.Channel

	mu            sync.Mutex
	publishFunc   func(context.Context, string, string, amqp.Publishing) error
	reconnectFunc func(context.Context) error
}

func NewRabbitMQ(uri string) (*RabbitMQ, error) {
	rmq := &RabbitMQ{
		uri: uri,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := rmq.reconnect(ctx); err != nil {
		return nil, err
	}

	return rmq, nil
}

type MessageHandler func(context.Context, amqp.Delivery) error

func (r *RabbitMQ) ConsumeMessages(queueName string, handler MessageHandler) error {
	go func() {
		for {
			if err := r.ensureChannel(context.Background()); err != nil {
				log.Printf("Failed to ensure channel for queue %s: %v. Retrying in 5s...", queueName, err)
				time.Sleep(5 * time.Second)
				continue
			}

			// Set prefetch count to 1 for fair dispatch
			// This tells RabbitMQ not to give more than one message to a service at a time.
			// The worker will only get the next message after it has acknowledged the previous one.
			err := r.Channel.Qos(
				1,     // prefetchCount: Limit to 1 unacknowledged message per consumer
				0,     // prefetchSize: No specific limit on message size
				false, // global: Apply prefetchCount to each consumer individually
			)
			if err != nil {
				log.Printf("Failed to set QoS for queue %s: %v. Retrying in 5s...", queueName, err)
				time.Sleep(5 * time.Second)
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
				log.Printf("Failed to start consume for queue %s: %v. Retrying in 5s...", queueName, err)
				time.Sleep(5 * time.Second)
				continue
			}

			log.Printf("Started consuming from queue: %s", queueName)

			for msg := range msgs {
				if err := tracing.TracedConsumer(msg, queueName, func(ctx context.Context, d amqp.Delivery) error {

					cfg := retry.DefaultConfig()
					err := retry.WithBackoff(ctx, cfg, func() error {
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
						headers["x-retry-count"] = cfg.MaxRetries
						d.Headers = headers

						// Reject without requeue - message will go to the DLQ
						_ = d.Reject(false)
						return err
					}

					// Only Ack if the handler succeeds
					if ackErr := msg.Ack(false); ackErr != nil {
						log.Printf("ERROR: Failed to Ack message: %v. Message body: %s", ackErr, msg.Body)
					}

					return nil
				}); err != nil {
					log.Printf("Error processing message: %v", err)
				}
			}

			log.Printf("Consumption loop ended for queue %s. Reconnecting...", queueName)
			time.Sleep(1 * time.Second)
		}
	}()

	return nil
}

func (r *RabbitMQ) PublishMessage(ctx context.Context, routingKey string, message contracts.AmqpMessage) error {
	log.Printf("Publishing message with routing key: %s", routingKey)

	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         jsonMsg,
	}

	return tracing.TracedPublisher(ctx, ApplicationExchange, routingKey, msg, r.publishWithReconnect)
}

func (r *RabbitMQ) publish(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	return r.Channel.PublishWithContext(ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		msg,
	)
}

func (r *RabbitMQ) publishWithReconnect(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
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

func (r *RabbitMQ) ensureChannel(ctx context.Context) error {
	r.mu.Lock()
	needsReconnect := r.conn == nil || r.conn.IsClosed() || r.Channel == nil || r.Channel.IsClosed()
	r.mu.Unlock()

	if !needsReconnect {
		return nil
	}

	return r.reconnectFunc(ctx)
}

func (r *RabbitMQ) reconnect(ctx context.Context) error {
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

func (r *RabbitMQ) connect(ctx context.Context) (*amqp.Connection, *amqp.Channel, error) {
	cfg := retry.Config{
		MaxRetries:  10,
		InitialWait: 1 * time.Second,
		MaxWait:     10 * time.Second,
	}

	var conn *amqp.Connection
	if err := retry.WithBackoff(ctx, cfg, func() error {
		c, err := amqp.Dial(r.uri)
		if err != nil {
			return fmt.Errorf("failed to connect to RabbitMQ: %v", err)
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

func (r *RabbitMQ) setupDeadLetterExchange() error {
	// Declare the dead letter exchange
	err := r.Channel.ExchangeDeclare(
		DeadLetterExchange,
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
		DeadLetterExchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind dead letter queue: %v", err)
	}

	return nil
}

func (r *RabbitMQ) setupExchangesAndQueues() error {
	// First setup the DLQ exchange and queue
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
		[]string{contracts.NotificationCmdSendEmail},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Notification event queues
	if err := r.declareAndBindQueue(
		NotifyEmailStatusQueue,
		[]string{contracts.NotificationEventEmailSent, contracts.NotificationEventEmailFailed},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Email log event queue (internal to notification service)
	if err := r.declareAndBindQueue(
		NotificationEventEmailLogQueue,
		[]string{contracts.NotificationEventEmailSent},
		ApplicationExchange,
	); err != nil {
		return err
	}

	// Request log event queue (handled by logging-service)
	if err := r.declareAndBindQueue(
		LoggingEventRequestLogQueue,
		[]string{contracts.LoggingEventRequestLogged},
		ApplicationExchange,
	); err != nil {
		return err
	}

	return nil
}

func (r *RabbitMQ) declareAndBindQueue(queueName string, messageTypes []string, exchange string) error {
	// Add dead letter configuration
	args := amqp.Table{
		"x-dead-letter-exchange": DeadLetterExchange,
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

// IsReady verifies that the RabbitMQ connection and channel are open and ready for use.
// This is useful for startup warmup to ensure the connection is established before
// the service reports itself as healthy.
func (r *RabbitMQ) IsReady() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.conn != nil && !r.conn.IsClosed() && r.Channel != nil && !r.Channel.IsClosed()
}

func (r *RabbitMQ) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Channel != nil {
		_ = r.Channel.Close()
	}
	if r.conn != nil {
		_ = r.conn.Close()
	}
}
