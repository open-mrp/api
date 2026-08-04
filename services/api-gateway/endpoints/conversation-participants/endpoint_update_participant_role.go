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
)

// Request to change a participant's role in a conversation.
type UpdateParticipantRoleRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// The participant whose role is changing (its `id` from the conversation's participants, not the account user's id).
	ParticipantID string `path:"pid" validate:"required"`
	// The role to assign to the participant.
	//
	// - `owner`: can rename or delete the conversation and manage members and roles.
	// - `admin`: can add and remove members and rename the conversation.
	// - `member`: can post, leave, mute, and react.
	// - `viewer`: read-only access.
	Role constants.ParticipantRole `json:"role" validate:"required"`
}

var sampleUpdateParticipantRoleRequest = &UpdateParticipantRoleRequest{
	ConversationID: apiresource.SampleConversationID,
	ParticipantID:  apiresource.SampleConversationParticipantID,
	Role:           constants.ParticipantRoleAdmin,
}

func (*UpdateParticipantRoleRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateParticipantRoleRequest)
}

// Changes a participant's role in a conversation and returns the updated conversation.
//
// Only the conversation's owner can change roles, and agent and system participants are rejected — they hold no role that can be changed. This is also the only way to grant `owner`: the promoted member gains full control while the caller keeps their own owner role, so a conversation can have more than one owner.
//
// A change of role posts a system event to the thread; setting a participant to the role they already hold is a no-op.
type UpdateParticipantRoleEndpoint struct{}

func (e *UpdateParticipantRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateParticipantRoleRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*UpdateParticipantRoleRequest, *apiresource.Conversation]{
		Title:               "Update Participant Role",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/participants/{pid}/actions/set-role",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateParticipantRoleRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ParticipantSvc).UpdateParticipantRole
		},
	})
}
