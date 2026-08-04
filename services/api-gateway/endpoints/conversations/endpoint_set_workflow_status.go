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
)

// Request to set the triage lane of a customer-service case.
type SetWorkflowStatusRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// The triage lane to move the case to.
	//
	// - `new`: opened but nobody has triaged it yet.
	// - `open`: actively being worked.
	// - `waiting_internal`: blocked on the internal team.
	// - `waiting_external`: blocked on a reply from the customer.
	// - `needs_approval`: a drafted reply is waiting for a human to approve it.
	// - `resolved`: closed out.
	WorkflowStatus constants.ConversationWorkflowStatus `json:"workflow_status" validate:"required"`
}

var sampleSetWorkflowStatusRequest = &SetWorkflowStatusRequest{
	ConversationID: apiresource.SampleConversationID,
	WorkflowStatus: constants.ConversationWorkflowStatusOpen,
}

func (*SetWorkflowStatusRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSetWorkflowStatusRequest)
}

// Moves a customer-service case to a triage lane in the support inbox.
//
// Only customer-facing cases have a triage lane; an internal conversation is rejected. The lane also advances on its own as the case progresses — an inbound customer message moves it to `waiting_internal`, a drafted reply to `needs_approval`, and an approved reply to `waiting_external` — so a lane set by hand can be overtaken by later activity.
type SetWorkflowStatusEndpoint struct{}

func (e *SetWorkflowStatusEndpoint) Materialize() *apiendpoint.APIEndpoint[*SetWorkflowStatusRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*SetWorkflowStatusRequest, *apiresource.Conversation]{
		Title:               "Set Case Status",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/actions/set-status",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *SetWorkflowStatusRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).SetWorkflowStatus
		},
	})
}
