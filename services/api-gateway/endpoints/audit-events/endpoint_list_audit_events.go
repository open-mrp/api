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

// ListAuditEventsRequest is the request to list audit events with filters.
type ListAuditEventsRequest struct {
	apiresource.PaginationRequest
	// Start of date range for occurred_at.
	StartDate *time.Time `query:"start_date"`
	// End of date range for occurred_at.
	EndDate *time.Time `query:"end_date"`

	// Resource type, matching the object type of the audited entity.
	ResourceType *constants.ObjectType `query:"resource_type"`
	// Resource ID (type_id of the audited resource).
	ResourceID *string `query:"resource_id"`

	// Actor ID (user ID or API key type_id).
	ActorID *string `query:"actor_id"`
	// Audit action.
	Action *constants.AuditAction `query:"action"`

	// Actor's home account ID.
	AccountID *string `query:"account_id"`
}

type ListAuditEventsEndpoint struct{}

func (e *ListAuditEventsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAuditEventsRequest, *apiresource.List[apiresource.AuditEvent]] {
	return &apiendpoint.APIEndpoint[*ListAuditEventsRequest, *apiresource.List[apiresource.AuditEvent]]{
		Title:             "List Audit Events",
		Description:       "Returns a paginated list of audit events for the current account.",
		Method:            http.MethodGet,
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
