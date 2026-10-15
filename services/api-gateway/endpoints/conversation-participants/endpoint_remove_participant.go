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

func (e *RemoveParticipantEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveParticipantRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*RemoveParticipantRequest, *object.Deleted]{
		Title:             "Remove Participant",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/messaging/conversations/{id}/participants/{pid}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeConversationParticipant,
		Public:            true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveParticipantRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ParticipantSvc).RemoveParticipant, constants.ObjectTypeConversationParticipant, "pid")
		},
	})
}
