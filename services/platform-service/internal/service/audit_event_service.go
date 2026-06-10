package service

import (
	"context"
	"fmt"

	authtypes "github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var auditEventSvcTracer = tracing.GetTracer("platform-service.audit_event_service")

type auditEventSvcImpl struct {
	auditEventRepo domain.AuditEventRepo
}

type AuditEventSvcConfig struct {
	// Repos (required) is the repository factory for audit event persistence.
	Repos domain.RepoFactory
}

func (c *AuditEventSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("audit event service: repos is required")
	}
	return nil
}

func NewAuditEventSvc(config *AuditEventSvcConfig) domain.AuditEventSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &auditEventSvcImpl{
		auditEventRepo: config.Repos.NewAuditEventRepo(),
	}
}

func (s *auditEventSvcImpl) SaveAuditEvent(ctx context.Context, event *domain.AuditEvent) *apierror.APIError {
	ctx, span := auditEventSvcTracer.Start(ctx, "service.audit_event.save_audit_event")
	defer span.End()

	return s.auditEventRepo.Create(ctx, event)
}

func (s *auditEventSvcImpl) GetAuditEvent(ctx context.Context, id string, includes []string) (*domain.AuditEventRead, *apierror.APIError) {
	ctx, span := auditEventSvcTracer.Start(ctx, "service.audit_event.get_audit_event")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(authtypes.PermissionDomainAuditEvents, authtypes.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	return s.auditEventRepo.FindByID(ctx, id, identity.Target.AccountID, includes)
}

func (s *auditEventSvcImpl) ListAuditEvents(ctx context.Context, filter *domain.ListAuditEventsFilter, includes []string) (*domain.ListAuditEventsResult, *apierror.APIError) {
	ctx, span := auditEventSvcTracer.Start(ctx, "service.audit_event.list_audit_events")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(authtypes.PermissionDomainAuditEvents, authtypes.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	return s.auditEventRepo.List(ctx, identity.Target.AccountID, filter, includes)
}

func (s *auditEventSvcImpl) ListAuditEventResourceTypes(ctx context.Context) ([]string, *apierror.APIError) {
	_, span := auditEventSvcTracer.Start(ctx, "service.audit_event.list_audit_event_resource_types")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(authtypes.PermissionDomainAuditEvents, authtypes.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return constants.ObjectType("").EnumValues(), nil
}
