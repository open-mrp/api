package participantep

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

// Request to add an agent participant to a conversation.
type AddAgentParticipantRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// The agent to add.
	AgentConfigID string `json:"agent_config_id" validate:"required"`
	// Controls when this agent responds to human messages in the conversation.
	//
	// - `mention`: responds only when @mentioned by one of its trigger keywords.
	// - `keyword`: responds whenever a message contains one of its trigger keywords.
	// - `always`: responds to every human message in the conversation.
	TriggerPolicy field.Optional[constants.AgentTriggerPolicy] `json:"trigger_policy,omitzero" default:"mention"`
	// For keyword/mention policies, the keywords (or mention handles) that trigger the agent.
	TriggerKeywords []string `json:"trigger_keywords,omitzero"`
}

var sampleAddAgentParticipantRequest = &AddAgentParticipantRequest{
	ConversationID:  apiresource.SampleConversationID,
	AgentConfigID:   apiresource.SampleAgentDefinitionID,
	TriggerPolicy:   field.Some(constants.AgentTriggerPolicyMention),
	TriggerKeywords: []string{"forecast"},
}

func (*AddAgentParticipantRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAddAgentParticipantRequest)
}

// Adds (or re-activates) an agent participant in a conversation with a trigger policy.
type AddAgentParticipantEndpoint struct{}

func (e *AddAgentParticipantEndpoint) Materialize() *apiendpoint.APIEndpoint[*AddAgentParticipantRequest, *apiresource.ConversationParticipant] {
	return (&apiendpoint.APIEndpoint[*AddAgentParticipantRequest, *apiresource.ConversationParticipant]{
		Title:               "Add Agent Participant",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/agents",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversationParticipant,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AddAgentParticipantRequest) (*apiresource.ConversationParticipant, *apierror.APIError) {
			return svc.(ParticipantSvc).AddAgentParticipant
		},
	})
}
