package event

import (
	"context"
	"encoding/json"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var notificationPublisherTracer = tracing.GetTracer("auth-service.notification_publisher")

type notificationPublisherImpl struct {
	rabbitmq *messaging.RabbitMQ
}

func NewNotificationPublisher(rabbitmq *messaging.RabbitMQ) domain.NotificationPublisher {
	return &notificationPublisherImpl{
		rabbitmq: rabbitmq,
	}
}

func (p *notificationPublisherImpl) PublishSendEmail(ctx context.Context, to []string, subject, body string, isBodyHTML bool, sendAs *string, accountID string, sentByID *string) *contracts.APIError {
	ctx, span := notificationPublisherTracer.Start(ctx, "event.notification_publisher.publish_send_email")
	defer span.End()

	payload := messaging.EmailSendData{
		To:         to,
		Subject:    subject,
		Body:       body,
		IsBodyHTML: isBodyHTML,
		SendAs:     sendAs,
		AccountID:  accountID,
		SentByID:   sentByID,
	}

	emailJSON, err := json.Marshal(payload)
	if err != nil {
		return tracing.Trace(span, contracts.NewInternalError(err, "Failed to marshal email send data."))
	}

	err = p.rabbitmq.PublishMessage(ctx, contracts.NotificationCmdSendEmail, contracts.AmqpMessage{
		UserID: accountID,
		Data:   emailJSON,
	})

	if err != nil {
		return tracing.Trace(span, contracts.NewInternalError(err, "Failed to publish email send data."))
	}

	return nil
}
