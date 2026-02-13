package service

import (
	"context"
	"fmt"

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

// WithDefaults returns a new LoggingSvcConfig with zero-value fields replaced by defaults.
func (c *LoggingSvcConfig) WithDefaults() *LoggingSvcConfig {
	if c == nil {
		c = &LoggingSvcConfig{}
	}
	return &LoggingSvcConfig{
		Repos: c.Repos,
	}
}

func (c *LoggingSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("logging service: repos is required")
	}
	return nil
}

func NewLoggingSvc(config *LoggingSvcConfig) domain.LoggingSvc {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &loggingSvcImpl{
		requestLogRepo: config.Repos.NewRequestLogRepo(),
	}
}

func (s *loggingSvcImpl) SaveRequestLog(ctx context.Context, rl *domain.RequestLog) *apierror.APIError {
	ctx, span := loggingSvcTracer.Start(ctx, "service.logging.save_request_log")
	defer span.End()

	return s.requestLogRepo.Create(ctx, rl)
}
