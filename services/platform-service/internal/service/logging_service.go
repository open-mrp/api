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

// SaveRequestLog persists a new API request log entry.
//
// 1. Insert the request log record into the repository.
func (s *loggingSvcImpl) SaveRequestLog(ctx context.Context, rl *domain.RequestLog) *apierror.APIError {
	ctx, span := loggingSvcTracer.Start(ctx, "service.logging.save_request_log")
	defer span.End()

	return s.requestLogRepo.Create(ctx, rl)
}

// GetRequestLog retrieves a single API request log by ID, scoped to the caller's account.
//
// 1. Extract and validate the caller's identity, actor type, and request_logs:read permission.
// 2. Require the Augno-Account header.
// 3. Fetch the request log from the repository by ID and account, with optional includes.
func (s *loggingSvcImpl) GetRequestLog(ctx context.Context, id string, includes []string) (*domain.RequestLogRead, *apierror.APIError) {
	ctx, span := loggingSvcTracer.Start(ctx, "service.logging.get_request_log")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainRequestLogs, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	return s.requestLogRepo.FindByID(ctx, id, identity.Target.AccountID, includes)
}

// ListRequestLogs returns a filtered, paginated list of API request logs for the caller's account.
//
// 1. Extract and validate the caller's identity, actor type, and request_logs:read permission.
// 2. Require the Augno-Account header.
// 3. Default the public_endpoint filter to true if not specified.
// 4. Query the repository with the account ID, filters, and optional includes.
func (s *loggingSvcImpl) ListRequestLogs(ctx context.Context, filter *domain.ListRequestLogsFilter, includes []string) (*domain.ListRequestLogsResult, *apierror.APIError) {
	ctx, span := loggingSvcTracer.Start(ctx, "service.logging.list_request_logs")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainRequestLogs, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	if filter.PublicEndpoint == nil {
		t := true
		filter.PublicEndpoint = &t
	}

	return s.requestLogRepo.List(ctx, identity.Target.AccountID, filter, includes)
}
