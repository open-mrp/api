package participantep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
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

func (e *RemoveAgentParticipantEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveAgentParticipantRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*RemoveAgentParticipantRequest, *object.Deleted]{
		Title:             "Remove Agent Participant",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/messaging/conversations/{id}/agents/{pid}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeConversationParticipant,
		Public:            false,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveAgentParticipantRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ParticipantSvc).RemoveAgentParticipant, constants.ObjectTypeConversationParticipant, "pid")
		},
	})
}
