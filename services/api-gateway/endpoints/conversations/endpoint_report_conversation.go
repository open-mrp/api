package conversationep

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

// Request to report a conversation (optionally a specific message) for abuse.
type ReportConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// Why the conversation or message is being reported, in free-form text.
	Reason string `json:"reason" validate:"required"`
	// The specific message being reported.
	//
	// Omit to report the conversation as a whole.
	MessageID field.Optional[string] `json:"message_id,omitzero"`
}

var sampleReportConversationRequest = &ReportConversationRequest{
	ConversationID: apiresource.SampleConversationID,
	Reason:         "spam",
	MessageID:      field.Some(apiresource.SampleMessageID),
}

func (*ReportConversationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleReportConversationRequest)
}

// Files an abuse report against a conversation, or against one message within it, and returns the conversation.
//
// Only an active participant can report a conversation. The report is recorded for review and changes nothing about the conversation itself — it is not hidden, muted, or removed.
type ReportConversationEndpoint struct{}

func (e *ReportConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*ReportConversationRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*ReportConversationRequest, *apiresource.Conversation]{
		Title:               "Report Conversation",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/actions/report",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ReportConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).ReportConversation
		},
	})
}
