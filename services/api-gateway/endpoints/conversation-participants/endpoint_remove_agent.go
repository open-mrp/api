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
	// The agent participant ID to remove.
	ParticipantID string `path:"pid" validate:"required"`
}

// Removes an agent participant from a conversation.
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
