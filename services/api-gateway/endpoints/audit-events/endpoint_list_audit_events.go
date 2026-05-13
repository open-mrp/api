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
	// Restricts results to audit events on or after this timestamp.
	StartDate *time.Time `query:"start_date"`
	// Restricts results to audit events on or before this timestamp.
	EndDate *time.Time `query:"end_date"`
	// Filter by the resource type of the audited entity.
	ResourceTypes []constants.ObjectType `query:"resource_types"`
	// Filter by the audited resource IDs.
	ResourceIDs []string `query:"resource_ids"`
	// Filter by the actor identifier. `account_user.id` when `identity_type`=`user`, or an `api_key.id` when `identity_type`=`api_key`.
	ActorIDs []string `query:"actor_ids"`
	// Filter by the audit actions.
	Actions []constants.AuditAction `query:"actions"`
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
