package conversationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to assign a customer-service case to a single owner — a user or a team.
//
// The owner is a polymorphic (`assignee_resource_type`, `assignee_resource_id`) reference; omit both fields to clear the assignment.
type AssignConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// What kind of owner the case is being assigned to.
	//
	// - `account_user`: an individual teammate takes the case.
	// - `account_group`: a team takes the case, so anyone on it can pick it up.
	AssigneeResourceType field.Optional[constants.ConversationAssigneeType] `json:"assignee_resource_type,omitzero"`
	// The owner's id, an `account_user` or `account_group` matching `assignee_resource_type`.
	//
	// Omit this and `assignee_resource_type` to clear the assignment.
	AssigneeResourceID field.Optional[string] `json:"assignee_resource_id,omitzero"`
}

var sampleAssignConversationRequest = &AssignConversationRequest{
	ConversationID:       apiresource.SampleConversationID,
	AssigneeResourceType: field.Some(constants.ConversationAssigneeTypeAccountUser),
	AssigneeResourceID:   field.Some(apiresource.SampleAccountUserID),
}

func (*AssignConversationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAssignConversationRequest)
}

// Assigns an external customer-service case to an owner — a user or a team — or clears the assignment.
//
// Only customer-facing cases can be assigned; assigning an internal conversation is rejected. The support inbox can then be filtered to a single assignee, or to the cases nobody owns yet.
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
