package participantep

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
	// For the keyword and mention policies, the keywords (or mention handles) that trigger the agent.
	//
	// Matching is case-insensitive and looks anywhere in the message body: under `keyword` the bare word is matched, under `mention` it must appear as `@keyword`. Replying directly to one of the agent's own messages always reaches it, so an agent left without keywords still answers replies but nothing else.
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

// Adds an AI agent to a conversation so it can respond to messages there.
//
// Adding an agent that is already a participant is not an error: its trigger policy and keywords are replaced with the ones supplied here, and an agent that had been removed is put back. That makes this endpoint the way to change when an existing agent responds, without removing and re-adding it.
//
// In an internal group conversation only an owner or admin can add an agent; in a direct message or a customer-facing case any active participant can.
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
