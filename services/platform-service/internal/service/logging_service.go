package service

import (
	"context"

	"github.com/augno/api/services/platform-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var loggingSvcTracer = tracing.GetTracer("platform-service.logging_service")

type loggingSvcImpl struct {
	requestLogRepo domain.RequestLogRepo
}

type LoggingSvcConfig struct {
	Repos domain.RepoFactory
}

func NewLoggingSvc(config LoggingSvcConfig) domain.LoggingSvc {
	return &loggingSvcImpl{
		requestLogRepo: config.Repos.NewRequestLogRepo(),
	}
}

func (s *loggingSvcImpl) SaveRequestLog(ctx context.Context, rl *domain.RequestLog) *apierror.APIError {
	ctx, span := loggingSvcTracer.Start(ctx, "service.logging.save_request_log")
	defer span.End()

	return s.requestLogRepo.Create(ctx, rl)
}
