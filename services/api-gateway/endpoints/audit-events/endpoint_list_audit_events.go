package auditeventsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list audit events.
type ListAuditEventsRequest struct {
	apiresource.PaginationRequest
	// Restricts results to audit events on or after this timestamp.
	StartDate *time.Time `query:"starts_at"`
	// Restricts results to audit events on or before this timestamp.
	EndDate *time.Time `query:"ends_at"`
	// Filter by the resource type of the audited entity.
	//
	// The full set of valid values is available from the List Audit Event Resource Types endpoint.
	ResourceTypes []constants.ObjectType `query:"resource_types"`
	// Filter by the audited resource IDs.
	ResourceIDs []string `query:"resource_ids"`
	// Scope results to a root record's entire history tree.
	//
	// Returns every event whose root resource matches, covering the root record itself and all of its descendants — for example a sales order together with its lines, picks, shipments, and invoices. Both `root_resource_type` and `root_resource_id` must be supplied together; supplying only one has no effect.
	RootResourceType *constants.ObjectType `query:"root_resource_type"`
	// ID of the root record whose history tree to return.
	//
	// Only applied when paired with `root_resource_type`.
	RootResourceID *string `query:"root_resource_id"`
	// Filter by the actor identifier.
	//
	// Matches the event's `actor.id`: a user ID for `user` actors, an API key ID for `api_key` actors, or an agent ID for `agent` actors.
	ActorIDs []string `query:"actor_ids"`
	// Filter by the actor type.
	//
	// Events are recorded for actors of type `user`, `api_key`, and `agent` — the last covering changes an OpenMRP agent made on your account's behalf.
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

// Returns a paginated list of audit events, newest first.
//
// Results cover every change where your account is either the acting account or the account that was acted upon, so a customer's or supplier's changes to your records appear alongside your own. The `q` parameter searches the resource type, action, resource ID, and originating request ID.
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
