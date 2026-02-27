package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/shared/appctx"
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

func (c *LoggingSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("logging service: repos is required")
	}
	return nil
}

func NewLoggingSvc(config *LoggingSvcConfig) domain.LoggingSvc {
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

func (s *loggingSvcImpl) GetRequestLog(ctx context.Context, id string) (*domain.RequestLogRead, *apierror.APIError) {
	ctx, span := loggingSvcTracer.Start(ctx, "service.logging.get_request_log")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckHasPermission(identity, types.PermissionDomainRequestLogs, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if identity.TargetAccountID == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Target account ID is required."))
	}

	return s.requestLogRepo.FindByID(ctx, id, *identity.TargetAccountID)
}

func (s *loggingSvcImpl) ListRequestLogs(ctx context.Context, filter *domain.ListRequestLogsFilter) (*domain.ListRequestLogsResult, *apierror.APIError) {
	ctx, span := loggingSvcTracer.Start(ctx, "service.logging.list_request_logs")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckHasPermission(identity, types.PermissionDomainRequestLogs, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if identity.TargetAccountID == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Target account ID is required."))
	}

	if filter.PublicEndpoint == nil {
		t := true
		filter.PublicEndpoint = &t
	}

	return s.requestLogRepo.List(ctx, *identity.TargetAccountID, filter)
}
