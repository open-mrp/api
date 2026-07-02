package conversationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list the caller's conversations.
//
// Supplying a topic resource returns the conversations anchored to that business record; supplying any case filter (or `audience=customer`) returns the support inbox of external customer-service cases.
type ListConversationsRequest struct {
	apiresource.PaginationRequest
	// Filter by conversation type.
	Type *constants.ConversationType `query:"type"`
	// Filter by conversation audience direction.
	Audience *constants.ConversationAudience `query:"audience"`
	// Filter by conversation visibility.
	Status *constants.ConversationListStatus `query:"status" default:"active"`
	// Restrict to conversations anchored to a business record of this type (with `topic_resource_id`).
	TopicResourceType *constants.ObjectType `query:"topic_resource_type"`
	// The id of the anchoring business record (with `topic_resource_type`).
	TopicResourceID *string `query:"topic_resource_id"`
	// Support inbox: filter external cases to a single triage lane.
	WorkflowStatus *constants.ConversationWorkflowStatus `query:"workflow_status"`
	// Support inbox: filter to cases owned by this assignee (a user or a team), matched by id.
	AssigneeResourceID *string `query:"assignee_resource_id"`
	// Support inbox: restrict to cases with no assignee.
	Unassigned bool `query:"unassigned"`
	// Support inbox: include archived (resolved-and-closed) cases.
	IncludeArchived bool `query:"include_archived"`
}

// isInboxQuery reports whether any support-inbox filter is set (or audience is customer),
// selecting the external-case inbox branch of the list.
func (r *ListConversationsRequest) isInboxQuery() bool {
	if r.WorkflowStatus != nil || r.AssigneeResourceID != nil || r.Unassigned || r.IncludeArchived {
		return true
	}
	return r.Audience != nil && *r.Audience == constants.ConversationAudienceCustomer
}

// isByRecordQuery reports whether a topic-resource anchor filter is set.
func (r *ListConversationsRequest) isByRecordQuery() bool {
	return r.TopicResourceType != nil && r.TopicResourceID != nil && *r.TopicResourceID != ""
}

// Returns the caller's conversations, most-recently-active first.
type ListConversationsEndpoint struct{}

func (e *ListConversationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListConversationsRequest, *apiresource.List[apiresource.Conversation]] {
	return (&apiendpoint.APIEndpoint[*ListConversationsRequest, *apiresource.List[apiresource.Conversation]]{
		Title:               "List Conversations",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListConversationsRequest) (*apiresource.List[apiresource.Conversation], *apierror.APIError) {
			return svc.(ConversationSvc).ListConversations
		},
	})
}
