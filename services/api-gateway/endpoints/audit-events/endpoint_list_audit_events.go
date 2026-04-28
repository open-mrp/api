package auditeventsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list audit events.
type ListAuditEventsRequest struct {
	apiresource.PaginationRequest
	// Start of date range for occurred_at.
	StartDate *time.Time `query:"start_date"`
	// End of date range for occurred_at.
	EndDate *time.Time `query:"end_date"`
	// Resource types of the audited entity (repeatable, exact match).
	ResourceTypes []constants.ObjectType `query:"resource_types"`
	// Audited resource IDs (repeatable, exact match).
	ResourceIDs []string `query:"resource_id"`
	// Actor IDs (repeatable, exact match). Each value is a user.id when
	// identity_type=user, or an api_key.type_id when identity_type=api_key —
	// not an account_user.id.
	ActorIDs []string `query:"actor_id"`
	// Audit actions (repeatable, exact match).
	Actions []constants.AuditAction `query:"action"`
}

type ListAuditEventsEndpoint struct{}

func (e *ListAuditEventsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAuditEventsRequest, *apiresource.List[apiresource.AuditEvent]] {
	return &apiendpoint.APIEndpoint[*ListAuditEventsRequest, *apiresource.List[apiresource.AuditEvent]]{
		Title:             "List Audit Events",
		Description:       "Returns a paginated list of audit events for the current account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/audit-events",
		Request:           &ListAuditEventsRequest{},
		Response:          &apiresource.List[apiresource.AuditEvent]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAuditEvent,
			Fields:     []string{"actor", "changes", "metadata"},
		}),
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestLogging: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAuditEventsRequest) (*apiresource.List[apiresource.AuditEvent], *apierror.APIError) {
			return svc.(AuditEventSvc).ListAuditEvents
		},
	}
}
