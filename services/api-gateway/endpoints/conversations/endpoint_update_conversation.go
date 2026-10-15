package conversationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to rename a conversation.
type UpdateConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// The conversation's title. Send `null` to clear it.
	Title field.Clearable[string] `json:"title,omitzero"`
}

var sampleUpdateConversationRequest = &UpdateConversationRequest{
	ConversationID: apiresource.SampleConversationID,
	Title:          field.Set("Fulfillment war room"),
}

func (*UpdateConversationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateConversationRequest)
}

// Updates a conversation.
//
// Only the fields you send change. Anyone in a direct message can rename it; in a group, only its owners and admins can.
type UpdateConversationEndpoint struct{}

func (e *UpdateConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateConversationRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*UpdateConversationRequest, *apiresource.Conversation]{
		Title:             "Update Conversation",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/messaging/conversations/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         false,
		ObjectType:        constants.ObjectTypeConversation,
		IncludeConfig:     conversationIncludeConfig(),
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).UpdateConversation
		},
	})
}
