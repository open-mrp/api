package event

import (
	"context"
	"encoding/json"

	"github.com/open-mrp/api/services/auth-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

var notificationPublisherTracer = tracing.GetTracer("auth-service.notification_publisher")

// reposContextKey is the context key for passing the repo factory.
type reposContextKey struct{}

// WithRepos adds a RepoFactory to the context so the outbox publisher can access it.
func WithRepos(ctx context.Context, repos domain.RepoFactory) context.Context {
	return context.WithValue(ctx, reposContextKey{}, repos)
}

// GetReposFromContext retrieves the RepoFactory from the context.
func GetReposFromContext(ctx context.Context) (domain.RepoFactory, bool) {
	repos, ok := ctx.Value(reposContextKey{}).(domain.RepoFactory)
	return repos, ok
}

// outboxNotificationPublisher writes notification commands to the outbox table instead of publishing directly to RabbitMQ, so the message commits atomically with the surrounding transaction and is delivered out-of-band by the outbox drain.
type outboxNotificationPublisher struct{}

// NewOutboxNotificationPublisher creates a notification publisher that writes to the outbox table for reliable, transactionally-committed message delivery.
func NewOutboxNotificationPublisher() domain.NotificationPublisher {
	return &outboxNotificationPublisher{}
}

func (p *outboxNotificationPublisher) PublishSendEmail(ctx context.Context, data messaging.EmailSendData) *apierror.APIError {
	ctx, span := notificationPublisherTracer.Start(ctx, "event.outbox_notification_publisher.publish_send_email")
	defer span.End()

	repos, ok := GetReposFromContext(ctx)
	if !ok {
		return tracing.Trace(span, apierror.NewInternalError(nil, "RepoFactory not found in context for outbox publisher."))
	}

	emailJSON, err := json.Marshal(data)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal email send data."))
	}

	msg := contracts.AmqpMessage{
		Data: emailJSON,
	}

	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	outboxInput := messaging.OutboxMessageInput{
		ServiceName: "auth-service",
		MessageType: string(contracts.NotificationCmdSendEmail),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.NotificationCmdSendEmail),
		Payload:     msg,
	}

	outboxRepo := repos.NewOutboxRepo()
	_, err = outboxRepo.Create(ctx, outboxInput)

	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to create outbox message."))
	}

	return nil
}
