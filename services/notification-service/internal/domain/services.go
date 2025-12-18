package domain

import (
	"context"

	"github.com/augno/api/shared/contracts"
)

type NotificationSvc interface {
	SendEmail(ctx context.Context, to []string, subject, body string, isBodyHtml bool, sendAs *string, accountID string, sentByID *string) (*string, *contracts.APIError)
	LogEmail(ctx context.Context, sesMessageID string, accountID string, sentByID *string, subject string, filename *string) *contracts.APIError
}

type EmailSender interface {
	Send(ctx context.Context, to []string, subject, body string, isBodyHtml bool, sendAs *string) (*string, *contracts.APIError)
}
