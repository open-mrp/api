package domain

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

type LoggingSvc interface {
	// SaveRequestLog saves a request log to the database.
	//
	//  1. Calls the request log repository to create the request log.
	SaveRequestLog(ctx context.Context, rl *RequestLog) *apierror.APIError
}
