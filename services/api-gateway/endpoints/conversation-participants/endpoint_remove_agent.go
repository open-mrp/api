package participantep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to remove an agent participant from a conversation.
type RemoveAgentParticipantRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// The agent's participant record in the conversation (its `id` from the conversation's participants, not the agent's own id).
	ParticipantID string `path:"pid" validate:"required"`
}

// Removes an AI agent from a conversation so it stops responding there.
//
// In an internal group conversation only an owner or admin can remove an agent; in a direct message or a customer-facing case any active participant can. The agent's earlier messages stay in the thread, and it can be added back later.
type RemoveAgentParticipantEndpoint struct{}

func (e *RemoveAgentParticipantEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveAgentParticipantRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*RemoveAgentParticipantRequest, *apiresource.EmptyResource]{
		Title:               "Remove Agent Participant",
		Method:              http.MethodDelete,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/agents/{pid}",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveAgentParticipantRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ParticipantSvc).RemoveAgentParticipant
		},
	})
}
