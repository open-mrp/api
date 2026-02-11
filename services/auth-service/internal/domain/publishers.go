package domain

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"
)

type NotificationPublisher interface {
	PublishSendEmail(ctx context.Context, data messaging.EmailSendData) *apierror.APIError
}
