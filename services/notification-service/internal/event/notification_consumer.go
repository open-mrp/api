package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

type NotificationConsumer struct {
	rabbitmq        *messaging.RabbitMQ
	notificationSvc domain.NotificationSvc
}

func NewNotificationConsumer(rabbitmq *messaging.RabbitMQ, notificationSvc domain.NotificationSvc) *NotificationConsumer {
	return &NotificationConsumer{
		rabbitmq:        rabbitmq,
		notificationSvc: notificationSvc,
	}
}

func (c *NotificationConsumer) Listen() error {
	// Listen to email send commands
	if err := c.rabbitmq.ConsumeMessages(messaging.NotificationCmdSendEmailQueue, c.handleCommandMessage); err != nil {
		return err
	}

	// Listen to email sent events for logging
	return c.rabbitmq.ConsumeMessages(messaging.NotificationEventEmailLogQueue, c.handleEventMessage)
}

func (c *NotificationConsumer) handleCommandMessage(ctx context.Context, msg amqp091.Delivery) error {
	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return err
	}

	var payload messaging.EmailSendData
	if err := json.Unmarshal(amqpMsg.Data, &payload); err != nil {
		log.Printf("Failed to unmarshal email payload: %v", err)
		return err
	}

	if msg.RoutingKey == contracts.NotificationCmdSendEmail {
		return c.handleSendEmail(ctx, payload)
	}

	return nil
}

func (c *NotificationConsumer) handleEventMessage(ctx context.Context, msg amqp091.Delivery) error {
	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		return err
	}

	if msg.RoutingKey == contracts.NotificationEventEmailSent {
		var logData messaging.EmailLogData
		if err := json.Unmarshal(amqpMsg.Data, &logData); err != nil {
			return err
		}

		return c.handleEmailLog(ctx, logData)
	}

	return nil
}

func (c *NotificationConsumer) handleSendEmail(ctx context.Context, payload messaging.EmailSendData) error {
	sesMessageID, apiErr := c.notificationSvc.SendEmail(
		ctx,
		payload.To,
		payload.Subject,
		payload.Body,
		payload.IsBodyHTML,
		payload.SendAs,
		payload.AccountID,
		payload.SentByID,
	)

	if apiErr != nil {
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

	return c.rabbitmq.PublishMessage(ctx, contracts.NotificationEventEmailSent, contracts.AmqpMessage{
		UserID: payload.AccountID,
		Data:   logJSON,
	})
}

func (c *NotificationConsumer) handleEmailLog(ctx context.Context, logData messaging.EmailLogData) error {
	apiErr := c.notificationSvc.LogEmail(
		ctx,
		logData.SesMessageID,
		logData.AccountID,
		logData.SentByID,
		logData.Subject,
		logData.Filename,
	)

	if apiErr != nil {
		return apiErr
	}

	return nil
}

func (c *NotificationConsumer) publishEmailStatus(ctx context.Context, accountID string, success bool, errorMsg string) error {
	statusData := map[string]interface{}{
		"success":    success,
		"account_id": accountID,
	}
	if !success {
		statusData["error"] = errorMsg
	}

	statusJSON, err := json.Marshal(statusData)
	if err != nil {
		return err
	}

	routingKey := contracts.NotificationEventEmailSent
	if !success {
		routingKey = contracts.NotificationEventEmailFailed
	}

	return c.rabbitmq.PublishMessage(ctx, routingKey, contracts.AmqpMessage{
		UserID: accountID,
		Data:   statusJSON,
	})
}
