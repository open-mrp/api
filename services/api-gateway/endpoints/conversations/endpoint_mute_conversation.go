package conversationep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to mute a conversation for the caller.
type MuteConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// When the mute expires.
	//
	// Omit to mute indefinitely.
	MutedUntil field.Optional[time.Time] `json:"muted_until,omitzero"`
}

var sampleMuteConversationRequest = &MuteConversationRequest{
	ConversationID: apiresource.SampleConversationID,
	MutedUntil:     field.Some(time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)),
}

func (*MuteConversationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleMuteConversationRequest)
}

// Mutes a conversation's notifications for the caller only, leaving the other participants unaffected.
//
// While muted the caller gets no notification or email for new messages, though the conversation still accumulates an unread count. A direct @mention pierces the mute and still raises a notification.
type MuteConversationEndpoint struct{}

func (e *MuteConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*MuteConversationRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*MuteConversationRequest, *apiresource.Conversation]{
		Title:               "Mute Conversation",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/actions/mute",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		AgentTool:           false,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *MuteConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).MuteConversation
		},
	})
}
