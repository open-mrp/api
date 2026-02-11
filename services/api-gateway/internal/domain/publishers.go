package domain

import (
	"context"

	"github.com/augno/api/shared/appctx"
)

type RequestLogPublisher interface {
	Create(ctx context.Context, rl *appctx.RequestLog) error
}
