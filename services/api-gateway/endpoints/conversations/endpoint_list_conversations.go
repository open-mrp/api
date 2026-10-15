package conversationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list the caller's conversations.
//
// Supplying `record_type` and `record_id` returns that record's discussions; supplying any case filter (or `audience=customer`) returns the support inbox of external customer-service cases.
type ListConversationsRequest struct {
	object.ListRequest
	// Filter by conversation type.
	Type *constants.ConversationType `query:"type"`
	// Only conversations that also include any of these account users. With `limit=1`, this finds your latest conversation with someone.
	ParticipantIDs []string `query:"participant_ids" validate:"omitempty,max=100,dive,required"`
	// Filter by whether the conversation is team-only or customer-facing.
	//
	// - `internal`: threads the customer never sees — direct messages, group threads, and record discussions.
	// - `customer`: external customer-service cases the customer takes part in, from the portal or a bridged email thread.
	Audience *constants.ConversationAudience `query:"audience"`
	// Filter by your own membership: `active` (conversations you haven't hidden) or `hidden`. Defaults to `active`.
	Memberships []constants.ConversationListStatus `query:"memberships"`
	// Restrict to the discussions of a record of this type, together with `record_id`.
	//
	// Matches both conversations about the record and conversations that link it, which is what powers the "discussions on this record" view.
	RecordType *constants.RecordType `query:"record_type"`
	// The ID of the record, together with `record_type`.
	RecordID *string `query:"record_id"`
	// Filter the support inbox to a single triage lane.
	//
	// - `new`: opened but nobody has triaged it yet.
	// - `open`: actively being worked.
	// - `waiting_internal`: blocked on the internal team.
	// - `waiting_external`: blocked on a reply from the customer.
	// - `needs_approval`: a drafted reply is waiting for a human to approve it.
	// - `resolved`: closed out.
	//
	// The working inbox hides resolved cases unless you ask for this lane explicitly.
	WorkflowStatus *constants.ConversationWorkflowStatus `query:"workflow_status"`
	// Filter the support inbox to cases owned by this assignee, an account user or an account group.
	AssigneeResourceID *string `query:"assignee_resource_id"`
	// Restrict the support inbox to cases nobody has been assigned yet.
	Unassigned bool `query:"unassigned"`
	// Return the archived support inbox instead of the working one.
	//
	// This swaps the view rather than widening it: archived cases are returned and unarchived ones are left out.
	IncludeArchived bool `query:"include_archived"`
}

// isInboxQuery reports whether any support-inbox filter is set (or audience is customer), selecting the external-case inbox branch of the list.
func (r *ListConversationsRequest) isInboxQuery() bool {
	if r.WorkflowStatus != nil || r.AssigneeResourceID != nil || r.Unassigned || r.IncludeArchived {
		return true
	}
	return r.Audience != nil && *r.Audience == constants.ConversationAudienceCustomer
}

// isByRecordQuery reports whether a topic-resource anchor filter is set.
func (r *ListConversationsRequest) isByRecordQuery() bool {
	return r.RecordType != nil && r.RecordID != nil && *r.RecordID != ""
}

// listStatus is the chat service's hidden-state filter for the requested memberships: both reads every conversation.
func (r *ListConversationsRequest) listStatus() string {
	active, hidden := len(r.Memberships) == 0, false
	for _, m := range r.Memberships {
		switch m {
		case constants.ConversationListStatusActive:
			active = true
		case constants.ConversationListStatusHidden:
			hidden = true
		}
	}
	switch {
	case active && hidden:
		return "all"
	case hidden:
		return string(constants.ConversationListStatusHidden)
	default:
		return string(constants.ConversationListStatusActive)
	}
}

// Returns a list of your conversations, most recently active first.
//
// A customer portal user sees only their own support case with the vendor, and an empty list until they have contacted support.
type ListConversationsEndpoint struct{}

func (e *ListConversationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListConversationsRequest, *apiresource.List[apiresource.Conversation]] {
	return (&apiendpoint.APIEndpoint[*ListConversationsRequest, *apiresource.List[apiresource.Conversation]]{
		Title:             "List Conversations",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/messaging/conversations",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeConversation,
		IncludeConfig:     conversationIncludeConfig(),
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListConversationsRequest) (*apiresource.List[apiresource.Conversation], *apierror.APIError) {
			return svc.(ConversationSvc).ListConversations
		},
	})
}
