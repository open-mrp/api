package domain

import (
	"context"

	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

type NotificationPublisher interface {
	PublishSendEmail(ctx context.Context, data messaging.EmailSendData) *apierror.APIError
}
