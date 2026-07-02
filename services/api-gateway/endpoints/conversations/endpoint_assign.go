package conversationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to assign a customer-service case to a single owner — a user or a team.
//
// The owner is a polymorphic (`assignee_resource_type`, `assignee_resource_id`) reference; omit both fields to clear the assignment.
type AssignConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// The owner's resource type: `account_user` (a teammate) or `account_group` (a team).
	AssigneeResourceType field.Optional[string] `json:"assignee_resource_type,omitzero"`
	// The owner's id, an `account_user` or `account_group` matching `assignee_resource_type`.
	//
	// Omit this and `assignee_resource_type` to clear the assignment.
	AssigneeResourceID field.Optional[string] `json:"assignee_resource_id,omitzero"`
}

var sampleAssignConversationRequest = &AssignConversationRequest{
	ConversationID:       apiresource.SampleConversationID,
	AssigneeResourceType: field.Some(string(constants.ObjectTypeAccountUser)),
	AssigneeResourceID:   field.Some(apiresource.SampleAccountUserID),
}

func (*AssignConversationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAssignConversationRequest)
}

// Assigns an external customer-service case to an owner — a user or a team — or clears the assignment.
type AssignConversationEndpoint struct{}

func (e *AssignConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*AssignConversationRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*AssignConversationRequest, *apiresource.Conversation]{
		Title:               "Assign Case",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/actions/assign",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AssignConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).AssignConversation
		},
	})
}
