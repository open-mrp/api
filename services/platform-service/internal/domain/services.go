package domain

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

type LoggingSvc interface {
	SaveRequestLog(ctx context.Context, rl *RequestLog) *apierror.APIError
}
