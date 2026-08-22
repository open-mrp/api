package domain

import (
	"context"

	apierror "github.com/open-mrp/api/shared/errors"
)

type LoggingSvc interface {
	// SaveRequestLog persists a request log entry.
	SaveRequestLog(ctx context.Context, rl *RequestLog) *apierror.APIError

	// GetRequestLog returns a single request log by ID, scoped to the caller's target account.
	GetRequestLog(ctx context.Context, id string, includes []string) (*RequestLogRead, *apierror.APIError)

	// ListRequestLogs returns a filtered, paginated list of request logs scoped to the caller's target account.
	ListRequestLogs(ctx context.Context, filter *ListRequestLogsFilter, includes []string) (*ListRequestLogsResult, *apierror.APIError)
}

type AuditEventSvc interface {
	// SaveAuditEvent persists a single audit event. It is intended to be invoked by the platform-service's async consumer pipeline.
	SaveAuditEvent(ctx context.Context, event *AuditEvent) *apierror.APIError

	// GetAuditEvent returns a single audit event by ID, scoped to the caller's target account.
	GetAuditEvent(ctx context.Context, id string, includes []string) (*AuditEventRead, *apierror.APIError)

	// ListAuditEvents returns a filtered, cursor-paginated list of audit events scoped to the caller's target account.
	ListAuditEvents(ctx context.Context, filter *ListAuditEventsFilter, includes []string) (*ListAuditEventsResult, *apierror.APIError)

	// ListAuditEventResourceTypes returns the full set of resource types that may appear on audit events.
	ListAuditEventResourceTypes(ctx context.Context) ([]string, *apierror.APIError)

	// BatchGetResourceCreators returns the creating actor for a batch of resources, derived from each resource's `create` audit event, scoped to the caller's target account. Requires only an assigned actor (not the audit read permission) so it can back a `created_by` include.
	BatchGetResourceCreators(ctx context.Context, resourceType string, resourceIDs []string) ([]ResourceCreator, *apierror.APIError)
}
