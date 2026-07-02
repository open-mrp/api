package participantep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to remove a participant from a group (owner/admin).
type RemoveParticipantRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// Participant ID.
	ParticipantID string `path:"pid" validate:"required"`
}

// Removes a participant from a group conversation.
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
