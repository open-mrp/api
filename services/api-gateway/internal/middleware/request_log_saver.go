package middleware

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/shared/appctx"
)

type saver interface {
	Save(ctx context.Context, rl *appctx.RequestLog) error
}

type requestLogSaver struct {
	publisher domain.RequestLogPublisher
}

func NewRequestLogSaver(publisher domain.RequestLogPublisher) *requestLogSaver {
	return &requestLogSaver{publisher: publisher}
}

func (r *requestLogSaver) Save(ctx context.Context, rl *appctx.RequestLog) error {
	return r.publisher.Create(ctx, rl)
}
