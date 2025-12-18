package domain

import (
	"context"

	"github.com/augno/api/shared/contracts"
)

type NotificationPublisher interface {
	PublishSendEmail(ctx context.Context, to []string, subject, body string, isBodyHTML bool, sendAs *string, accountID string, sentByID *string) *contracts.APIError
}
