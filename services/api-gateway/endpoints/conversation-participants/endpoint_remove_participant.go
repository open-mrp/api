package participantep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to remove a participant from a group conversation.
type RemoveParticipantRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// The participant to remove (its `id` from the conversation's participants, not the account user's id).
	ParticipantID string `path:"pid" validate:"required"`
}

// Removes a participant from a group conversation.
//
// Only an owner or admin can remove someone, participants cannot be removed from a direct message, and callers cannot remove themselves — leave the conversation instead. Use the remove-agent endpoint for agent participants.
//
// The removed member immediately loses access to the conversation, but their earlier messages stay in the thread and a system event records the removal. Adding them back later reactivates the same membership.
type RemoveParticipantEndpoint struct{}

func (e *RemoveParticipantEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveParticipantRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*RemoveParticipantRequest, *apiresource.EmptyResource]{
		Title:               "Remove Participant",
		Method:              http.MethodDelete,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/participants/{pid}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveParticipantRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ParticipantSvc).RemoveParticipant
		},
	})
}
