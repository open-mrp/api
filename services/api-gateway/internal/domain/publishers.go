package domain

import (
	"context"

	"github.com/open-mrp/apikit/appctx"
)

type RequestLogPublisher interface {
	Create(ctx context.Context, rl *appctx.RequestLog) error
}
