package domain

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

type LoggingSvc interface {
	// SaveRequestLog persists a request log entry.
	SaveRequestLog(ctx context.Context, rl *RequestLog) *apierror.APIError

	// GetRequestLog returns a single request log by ID, scoped to the
	// caller's target account.
	GetRequestLog(ctx context.Context, id string) (*RequestLogRead, *apierror.APIError)

	// ListRequestLogs returns a filtered, paginated list of request logs
	// scoped to the caller's target account.
	ListRequestLogs(ctx context.Context, filter *ListRequestLogsFilter) (*ListRequestLogsResult, *apierror.APIError)
}
