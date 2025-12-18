package domain

import (
	"context"

	"github.com/augno/api/shared/contracts"
)

type LoggingSvc interface {
	SaveRequestLog(ctx context.Context, rl *RequestLog) *contracts.APIError
}
