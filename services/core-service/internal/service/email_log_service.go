package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var emailLogSvcTracer = tracing.GetTracer("core-service.email_log_service")

type emailLogSvcImpl struct {
	repos domain.RepoFactory
}

type EmailLogSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory
}

func (c *EmailLogSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("email log service: repos is required")
	}
	return nil
}

func NewEmailLogSvc(config *EmailLogSvcConfig) domain.EmailLogSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &emailLogSvcImpl{
		repos: config.Repos,
	}
}

func (s *emailLogSvcImpl) ListEmailLogs(ctx context.Context, params domain.ListEmailLogsParams) (*domain.ListEmailLogsResult, *apierror.APIError) {
	ctx, span := emailLogSvcTracer.Start(ctx, "service.email_log.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainEmailLogs, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewEmailLogRepo().List(ctx, params)
}

func (s *emailLogSvcImpl) GetEmailLog(ctx context.Context, params domain.GetEmailLogParams) (*domain.EmailLog, *apierror.APIError) {
	ctx, span := emailLogSvcTracer.Start(ctx, "service.email_log.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainEmailLogs, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewEmailLogRepo().Get(ctx, params)
}
