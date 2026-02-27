package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/services/notification-service/internal/email"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type NotificationConsumer struct {
	rabbitmq         messaging.MessageBroker
	notificationSvc  domain.NotificationSvc
	inboxConsumer    *messaging.InboxConsumer
	templateRenderer email.TemplateRenderer
	tracer           trace.Tracer
}

func NewNotificationConsumer(rabbitmq messaging.MessageBroker, notificationSvc domain.NotificationSvc, inboxRepo messaging.InboxRepo, templateRenderer email.TemplateRenderer, tracer trace.Tracer) *NotificationConsumer {
	return &NotificationConsumer{
		rabbitmq:         rabbitmq,
		notificationSvc:  notificationSvc,
		inboxConsumer:    messaging.NewInboxConsumer(inboxRepo, "notification-service"),
		templateRenderer: templateRenderer,
		tracer:           tracer,
	}
}

func (c *NotificationConsumer) Listen(ctx context.Context) error {
	// Listen to email send commands
	if err := c.rabbitmq.ConsumeMessages(ctx, messaging.NotificationCmdSendEmailQueue, c.inboxConsumer.Wrap("notification.send_email", c.handleCommandMessage)); err != nil {
		return err
	}

	// Listen to email sent events for logging
	return c.rabbitmq.ConsumeMessages(ctx, messaging.NotificationEventEmailLogQueue, c.inboxConsumer.Wrap("notification.email_log", c.handleEventMessage))
}

func (c *NotificationConsumer) handleCommandMessage(ctx context.Context, msg amqp091.Delivery) error {
	// Start a new root span (not linked to original request trace)
	ctx, span := c.tracer.Start(ctx, "consumer.handle_command",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		span.RecordError(err)
		return err
	}

	if amqpMsg.Identity != nil {
		ctx = appctx.WithIdentity(ctx, amqpMsg.Identity)
	}
	if amqpMsg.RequestID != "" {
		ctx = appctx.WithRequestID(ctx, amqpMsg.RequestID)
	}

	var payload messaging.EmailSendData
	if err := json.Unmarshal(amqpMsg.Data, &payload); err != nil {
		log.Printf("Failed to unmarshal email payload: %v", err)
		span.RecordError(err)
		return err
	}

	if msg.RoutingKey == string(contracts.NotificationCmdSendEmail) {
		return c.handleSendEmail(ctx, payload)
	}

	return nil
}

func (c *NotificationConsumer) handleEventMessage(ctx context.Context, msg amqp091.Delivery) error {
	// Start a new root span (not linked to original request trace)
	ctx, span := c.tracer.Start(ctx, "consumer.handle_event",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		span.RecordError(err)
		return err
	}

	if amqpMsg.Identity != nil {
		ctx = appctx.WithIdentity(ctx, amqpMsg.Identity)
	}
	if amqpMsg.RequestID != "" {
		ctx = appctx.WithRequestID(ctx, amqpMsg.RequestID)
	}

	if msg.RoutingKey == string(contracts.NotificationEventEmailSent) {
		var logData messaging.EmailLogData
		if err := json.Unmarshal(amqpMsg.Data, &logData); err != nil {
			span.RecordError(err)
			return err
		}

		return c.handleEmailLog(ctx, logData)
	}

	return nil
}

func (c *NotificationConsumer) handleSendEmail(ctx context.Context, payload messaging.EmailSendData) error {
	ctx, span := c.tracer.Start(ctx, "consumer.send_email",
		trace.WithAttributes(
			attribute.String("email.template_id", string(payload.TemplateID)),
			attribute.StringSlice("email.to", payload.To),
			attribute.String("email.subject", payload.Subject),
		),
	)
	defer span.End()

	// Render the email template
	body, apiErr := c.templateRenderer.RenderTemplate(ctx, payload.TemplateID, payload.Params)
	if apiErr != nil {
		log.Printf("Failed to render email template %s: %v", payload.TemplateID, apiErr)
		span.RecordError(apiErr)
		return apiErr
	}

	sesMessageID, apiErr := c.notificationSvc.SendEmail(
		ctx,
		domain.EmailSendData{
			To:        payload.To,
			Subject:   payload.Subject,
			Body:      body,
			SendAs:    payload.SendAs,
			AccountID: payload.AccountID,
			SentByID:  payload.SentByID,
		},
	)

	if apiErr != nil {
		span.RecordError(apiErr)
		// Publish failure event
		errorMsg := apiErr.PublicMessage
		if apiErr.InternalMessage != "" {
			errorMsg = apiErr.InternalMessage
		}
		if err := c.publishEmailStatus(ctx, payload.AccountID, false, errorMsg); err != nil {
			log.Printf("Failed to publish email failure status: %v", err)
		}
		return apiErr
	}

	span.SetAttributes(attribute.String("email.ses_message_id", *sesMessageID))

	// Publish email sent event with logging data (this triggers log creation)
	if err := c.publishEmailLogEvent(ctx, *sesMessageID, payload); err != nil {
		// Don't fail the message - email was sent successfully
		log.Printf("Failed to publish email log event: %v", err)
	}

	// Publish success event for external consumers
	if err := c.publishEmailStatus(ctx, payload.AccountID, true, ""); err != nil {
		log.Printf("Failed to publish email success status: %v", err)
	}

	return nil
}

func (c *NotificationConsumer) publishEmailLogEvent(ctx context.Context, sesMessageID string, payload messaging.EmailSendData) error {
	logData := messaging.EmailLogData{
		SesMessageID: sesMessageID,
		AccountID:    payload.AccountID,
		SentByID:     payload.SentByID,
		Subject:      payload.Subject,
		Filename:     nil,
	}

	logJSON, err := json.Marshal(logData)
	if err != nil {
		return err
	}

	msg := contracts.AmqpMessage{
		Data: logJSON,
	}

	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	return c.rabbitmq.PublishMessage(ctx, messaging.ApplicationExchange, string(contracts.NotificationEventEmailSent), msg)
}

func (c *NotificationConsumer) handleEmailLog(ctx context.Context, logData messaging.EmailLogData) error {
	ctx, span := c.tracer.Start(ctx, "consumer.log_email",
		trace.WithAttributes(
			attribute.String("email.ses_message_id", logData.SesMessageID),
			attribute.String("email.subject", logData.Subject),
		),
	)
	defer span.End()

	apiErr := c.notificationSvc.LogEmail(
		ctx,
		domain.EmailLogData{
			SesMessageID: logData.SesMessageID,
			AccountID:    logData.AccountID,
			SentByID:     logData.SentByID,
			Subject:      logData.Subject,
			Filename:     logData.Filename,
		},
	)

	if apiErr != nil {
		span.RecordError(apiErr)
		return apiErr
	}

	return nil
}

func (c *NotificationConsumer) publishEmailStatus(ctx context.Context, userID *string, success bool, errorMsg string) error {
	statusData := map[string]any{
		"success": success,
		"user_id": userID,
	}
	if !success {
		statusData["error"] = errorMsg
	}

	statusJSON, err := json.Marshal(statusData)
	if err != nil {
		return err
	}

	routingKey := string(contracts.NotificationEventEmailSent)
	if !success {
		routingKey = string(contracts.NotificationEventEmailFailed)
	}

	msg := contracts.AmqpMessage{
		Data: statusJSON,
	}

	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	return c.rabbitmq.PublishMessage(ctx, messaging.ApplicationExchange, routingKey, msg)
}
