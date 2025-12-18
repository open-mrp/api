package domain

import (
	"context"

	"github.com/augno/api/shared/contracts"
)

type EmailLogRepo interface {
	Create(ctx context.Context, emailLog *EmailLog) *contracts.APIError
	FindBySesMessageID(ctx context.Context, sesMessageID string) (*EmailLog, *contracts.APIError)
}
