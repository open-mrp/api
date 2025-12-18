package service

import (
	"context"

	"github.com/augno/api/services/logging-service/internal/domain"
	"github.com/augno/api/services/logging-service/internal/infrastructure/repository"
	"github.com/augno/api/services/logging-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/tracing"
)

var loggingSvcTracer = tracing.GetTracer("logging-service.logging_service")

type loggingSvcImpl struct {
	requestLogRepo domain.RequestLogRepo
}

type LoggingSvcConfig struct {
	Repos RepoFactory
}

func NewLoggingSvc(config LoggingSvcConfig) domain.LoggingSvc {
	return &loggingSvcImpl{
		requestLogRepo: config.Repos.NewRequestLogRepo(),
	}
}

func (s *loggingSvcImpl) SaveRequestLog(ctx context.Context, rl *domain.RequestLog) *contracts.APIError {
	ctx, span := loggingSvcTracer.Start(ctx, "service.logging.save_request_log")
	defer span.End()

	return s.requestLogRepo.Create(ctx, rl)
}

// RepoFactory builds repositories for the logging service.
type RepoFactory interface {
	NewRequestLogRepo() domain.RequestLogRepo
}

// RepoFactoryImpl is the concrete implementation used by the service.
type RepoFactoryImpl struct {
	db *sqlc.Queries
}

func NewRepoFactory(db *sqlc.Queries) RepoFactory {
	return &RepoFactoryImpl{db: db}
}

func (f *RepoFactoryImpl) NewRequestLogRepo() domain.RequestLogRepo {
	return repository.NewRequestLogRepo(f.db)
}
