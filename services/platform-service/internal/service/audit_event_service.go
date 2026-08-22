package service

import (
	"context"
	"fmt"

	authtypes "github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

var auditEventSvcTracer = tracing.GetTracer("platform-service.audit_event_service")

type auditEventSvcImpl struct {
	auditEventRepo domain.AuditEventRepo
	outboxRepo     messaging.OutboxRepo
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
		outboxRepo:     config.Repos.NewOutboxRepo(),
	}
}

func (s *auditEventSvcImpl) SaveAuditEvent(ctx context.Context, event *domain.AuditEvent) *apierror.APIError {
	ctx, span := auditEventSvcTracer.Start(ctx, "service.audit_event.save_audit_event")
	defer span.End()

	if apiErr := s.auditEventRepo.Create(ctx, event); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Follower fan-out runs after the row is persisted so the acting user is registered as a follower before any later event is processed. Both steps are idempotent (duplicate-tolerant), so a retry after a partial failure completes the remainder.
	return tracing.Trace(span, s.notifySalesOrderFollowers(ctx, event))
}

func (s *auditEventSvcImpl) BatchGetResourceCreators(ctx context.Context, resourceType string, resourceIDs []string) ([]domain.ResourceCreator, *apierror.APIError) {
	ctx, span := auditEventSvcTracer.Start(ctx, "service.audit_event.batch_get_resource_creators")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	// Deliberately NOT gated by the audit read permission: "who created this resource" follows the resource's own visibility, so any assigned actor (internal or customer) may resolve it, scoped to their target account.
	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account header is required."))
	}

	if len(resourceIDs) == 0 {
		return nil, nil
	}

	return s.auditEventRepo.BatchGetResourceCreators(ctx, identity.Target.AccountID, resourceType, resourceIDs)
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
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account header is required."))
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
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account header is required."))
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
