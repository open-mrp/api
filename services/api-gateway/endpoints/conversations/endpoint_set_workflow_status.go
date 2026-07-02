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
	WorkflowStatus constants.ConversationWorkflowStatus `json:"workflow_status" validate:"required"`
}

var sampleSetWorkflowStatusRequest = &SetWorkflowStatusRequest{
	ConversationID: apiresource.SampleConversationID,
	WorkflowStatus: constants.ConversationWorkflowStatusOpen,
}

func (*SetWorkflowStatusRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSetWorkflowStatusRequest)
}

// Sets the triage lane of an external customer-service case.
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
