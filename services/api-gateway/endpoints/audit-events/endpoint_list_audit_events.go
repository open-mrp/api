package auditeventsep

import (
	"context"
	"net/http"
	"time"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list audit events.
type ListAuditEventsRequest struct {
	object.ListRequest
	// Exact match on `resource_id`, the ID of the request that caused the change, `resource_type`, or `action`.
	//
	// A resource type or action matches by its code (`sales_order`) or its name (`Sales Order`).
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only events that occurred at or after this time.
	//
	// Defaults to 24 hours before `occurred_before`, or before now when that is also omitted, unless `resource_ids` or `root_resource_types` and `root_resource_ids` are given: a record's history is returned whole. Pass an earlier time to search further back.
	OccurredAfter *time.Time `query:"occurred_after"`
	// Only events that occurred at or before this time.
	OccurredBefore *time.Time `query:"occurred_before"`
	// Only events on resources of these types.
	ResourceTypes []constants.ObjectType `query:"resource_types"`
	// Only events on these resources.
	ResourceIDs []string `query:"resource_ids"`
	// Only events in the history of records of these types: the record itself and everything under it, such as a sales order with its lines, picks, shipments and invoices.
	//
	// Pass with `root_resource_ids`.
	RootResourceTypes []constants.ObjectType `query:"root_resource_types"`
	// Only events in the history of these records, with `root_resource_types` naming their types.
	RootResourceIDs []string `query:"root_resource_ids"`
	// Only events whose `actor.id` is one of these: a user, API key, agent or device ID.
	ActorIDs []string `query:"actor_ids"`
	// Only events whose `actor.type` is one of these.
	ActorTypes []constants.ActorType `query:"actor_types"`
	// Only events recording these actions.
	Actions []constants.AuditAction `query:"actions"`
	// Only events made by these accounts: the account the actor belongs to.
	//
	// Results always cover events where your account is either the acting account or the account acted upon; this narrows them to specific acting accounts, such as one customer's.
	ActorAccountIDs []string `query:"actor_account_ids"`
	// Only events made against these accounts (the event's `account`).
	//
	// Results always cover events where your account is either the acting account or the account acted upon; this narrows them to specific target accounts, such as one customer's or supplier's.
	TargetAccountIDs []string `query:"target_account_ids"`
}

// Returns a page of audit events, newest first.
//
// Results cover every change where your account is either the acting account or the account acted upon, so a customer's or supplier's changes to your records appear alongside your own.
type ListAuditEventsEndpoint struct{}

func (e *ListAuditEventsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAuditEventsRequest, *apiresource.List[apiresource.AuditEvent]] {
	return (&apiendpoint.APIEndpoint[*ListAuditEventsRequest, *apiresource.List[apiresource.AuditEvent]]{
		Title:             "List Audit Events",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/audit-events",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAuditEvents, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeAuditEvent,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAuditEvent,
			Fields:     []string{"account", "changes", "metadata", "request"},
		}),
		Extras: apiendpoint.APIEndpointExtras{
			HideFromRequestLog: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAuditEventsRequest) (*apiresource.List[apiresource.AuditEvent], *apierror.APIError) {
			return svc.(AuditEventSvc).ListAuditEvents
		},
	})
}
