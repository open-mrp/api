package domain

import (
	"context"

	"github.com/augno/api/shared/contracts"
)

type RequestLogRepo interface {
	Create(ctx context.Context, requestLog *RequestLog) *contracts.APIError
}
