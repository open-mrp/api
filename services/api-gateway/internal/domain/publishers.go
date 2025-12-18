package domain

import (
	"context"

	"github.com/augno/api/shared/contracts"
)

type RequestLogPublisher interface {
	Create(ctx context.Context, rl *RequestLog) *contracts.APIError
}
