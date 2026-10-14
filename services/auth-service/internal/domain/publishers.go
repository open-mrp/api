package domain

import (
	"context"

	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"
)

type NotificationPublisher interface {
	PublishSendEmail(ctx context.Context, data messaging.EmailSendData) *apierror.APIError
}
