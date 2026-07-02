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

// Request to add a participant to a group (owner/admin).
type AddParticipantRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// The account user to add.
	AccountUserID string `json:"account_user_id" validate:"required"`
	// Role to grant the new participant.
	//
	// - `admin`: can add and remove members and rename the conversation.
	// - `member`: can post, leave, mute, and react.
	// - `viewer`: read-only access.
	//
	// `owner` is not accepted here; ownership can only be transferred via the set-role endpoint.
	Role field.Optional[constants.ParticipantRole] `json:"role,omitzero" default:"member"`
}

var sampleAddParticipantRequest = &AddParticipantRequest{
	ConversationID: apiresource.SampleConversationID,
	AccountUserID:  apiresource.SampleAccountUserID,
	Role:           field.Some(constants.ParticipantRoleMember),
}

func (*AddParticipantRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAddParticipantRequest)
}

// Adds (or reactivates) a participant in a group conversation.
type AddParticipantEndpoint struct{}

func (e *AddParticipantEndpoint) Materialize() *apiendpoint.APIEndpoint[*AddParticipantRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*AddParticipantRequest, *apiresource.Conversation]{
		Title:               "Add Participant",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/participants",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AddParticipantRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ParticipantSvc).AddParticipant
		},
	})
}
