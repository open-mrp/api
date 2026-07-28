package auditeventsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
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
	// Scope results to a root record's entire history tree.
	//
	// Every event whose root resource matches, including the root itself and all of its descendants (for example a sales order together with its lines, picks, shipments, and invoices). Both `root_resource_type` and `root_resource_id` must be supplied together.
	RootResourceType *constants.ObjectType `query:"root_resource_type"`
	// Filter by the root resource.
	RootResourceID *string `query:"root_resource_id"`
	// Filter by the actor identifier.
	//
	// Matches the event's `actor.id`: a user ID for `user` actors or an API key ID for `api_key` actors.
	ActorIDs []string `query:"actor_ids"`
	// Filter by the actor type.
	ActorTypes []constants.ActorType `query:"actor_types"`
	// Filter by the mutation type recorded on the event.
	Actions []constants.AuditAction `query:"actions"`
	// Filter by the _acting_ account: the account that performed the mutation.
	//
	// Results are always scoped to events where your account is either the acting account or the target account; this narrows that set to specific acting accounts — for example a specific customer's account that mutated a resource on your account.
	ActorAccountIDs []string `query:"actor_account_ids"`
	// Filter by the _target_ account the mutation was performed against (the event's `account`).
	//
	// Results are always scoped to events where your account is either the acting account or the target account; this narrows that set to specific target accounts — for example a specific customer's or supplier's account.
	TargetAccountIDs []string `query:"target_account_ids"`
}

// Returns a paginated list of audit events for the current account.
type ListAuditEventsEndpoint struct{}

func (e *ListAuditEventsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAuditEventsRequest, *apiresource.List[apiresource.AuditEvent]] {
	return (&apiendpoint.APIEndpoint[*ListAuditEventsRequest, *apiresource.List[apiresource.AuditEvent]]{
		Title:               "List Audit Events",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/core/audit-events",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAuditEvents, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeAuditEvent,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAuditEvent,
			Fields:     []string{"account", "actor", "changes", "metadata", "request"},
		}),
		Extras: apiendpoint.APIEndpointExtras{
			HideFromRequestLog: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAuditEventsRequest) (*apiresource.List[apiresource.AuditEvent], *apierror.APIError) {
			return svc.(AuditEventSvc).ListAuditEvents
		},
	})
}
