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
	//
	// The full set of valid values is available from the List Audit Event Resource Types endpoint.
	ResourceTypes []constants.ObjectType `query:"resource_types"`
	// Filter by the audited resource IDs.
	ResourceIDs []string `query:"resource_ids"`
	// Filter by the actor identifier.
	//
	// Matches the event's `actor.id`: a user ID for `user` actors or an API key ID for `api_key` actors.
	ActorIDs []string `query:"actor_ids"`
	// Filter by the mutation type recorded on the event.
	Actions []constants.AuditAction `query:"actions"`
	// Filter by the target account the mutation was performed against.
	//
	// Narrows results to audit events whose `account` is one of the given account IDs — for example a specific customer's or supplier's account.
	AccountIDs []string `query:"account_ids"`
}

// Returns a paginated list of audit events for the current account.
type ListAuditEventsEndpoint struct{}

func (e *ListAuditEventsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAuditEventsRequest, *apiresource.List[apiresource.AuditEvent]] {
	return (&apiendpoint.APIEndpoint[*ListAuditEventsRequest, *apiresource.List[apiresource.AuditEvent]]{
		Title:             "List Audit Events",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/audit-events",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAuditEvent,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAuditEvent,
			Fields:     []string{"account", "actor", "changes", "metadata", "request"},
		}),
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestLogging: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAuditEventsRequest) (*apiresource.List[apiresource.AuditEvent], *apierror.APIError) {
			return svc.(AuditEventSvc).ListAuditEvents
		},
	})
}
