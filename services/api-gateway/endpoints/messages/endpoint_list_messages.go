package messageep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list a conversation's messages.
type ListMessagesRequest struct {
	apiresource.PaginationRequest
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// Which set of the conversation's messages to return.
	//
	// Left unset, you get the delivered timeline. Pass `draft` for the case's reply drafts awaiting approval, or `scheduled` for the messages you yourself have queued for a future send, soonest first. Those two ignore paging and come back in a single response.
	Status *constants.MessageStatus `query:"status"`
	// Return only messages that come after this position in the timeline.
	//
	// Use it to catch up after a dropped realtime connection: pass the sequence of the last message you already have to fetch everything since.
	AfterSequence *int64 `query:"after_sequence"`
}

// Returns the messages in a conversation, newest first.
//
// You must be an active participant. A customer reading their own case receives only the messages meant for them — internal team notes are never included.
type ListMessagesEndpoint struct{}

func (e *ListMessagesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMessagesRequest, *apiresource.List[apiresource.Message]] {
	return (&apiendpoint.APIEndpoint[*ListMessagesRequest, *apiresource.List[apiresource.Message]]{
		Title:               "List Messages",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/messages",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeChatMessage,
		IncludeConfig:       messageIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMessagesRequest) (*apiresource.List[apiresource.Message], *apierror.APIError) {
			return svc.(MessageSvc).ListMessages
		},
	})
}
