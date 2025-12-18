package middleware

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"

	"go.opentelemetry.io/otel/trace"
)

type saver interface {
	Save(ctx context.Context, rl *domain.RequestLog) error
}

type requestLogSaver struct {
	publisher domain.RequestLogPublisher
}

func NewRequestLogSaver(publisher domain.RequestLogPublisher) *requestLogSaver {
	return &requestLogSaver{publisher: publisher}
}

func (r *requestLogSaver) Save(ctx context.Context, rl *domain.RequestLog) error {
	apiErr := r.publisher.Create(ctx, rl)
	if apiErr != nil {
		return apiErr
	}
	return nil
}

type asyncSaveRequest struct {
	ctx context.Context
	rl  *domain.RequestLog
}

type asyncRequestLogSaver struct {
	ch chan asyncSaveRequest
}

func NewAsyncRequestLogSaver(buffer int, backend saver) *asyncRequestLogSaver {
	as := &asyncRequestLogSaver{ch: make(chan asyncSaveRequest, buffer)}
	go func() {
		for req := range as.ch {
			_ = backend.Save(req.ctx, req.rl)
		}
	}()
	return as
}

func (a *asyncRequestLogSaver) Save(ctx context.Context, rl *domain.RequestLog) error {
	span := trace.SpanFromContext(ctx)
	asyncCtx := trace.ContextWithSpan(context.Background(), span)

	select {
	case a.ch <- asyncSaveRequest{ctx: asyncCtx, rl: rl}:
	default:

	}
	return nil
}
