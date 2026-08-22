package messageep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to cancel a pending scheduled message.
type CancelScheduledRequest struct {
	// The id of the scheduled message to cancel.
	MessageID string `path:"id" validate:"required"`
}

// Cancels a message that was scheduled for a future send, so it is never delivered.
//
// You can only cancel a message you scheduled yourself, and only while it is still waiting to go out — once it has been delivered or has otherwise left the scheduled state, the request fails. The canceled message is kept as a record and never appears in the conversation.
type CancelScheduledEndpoint struct{}

func (e *CancelScheduledEndpoint) Materialize() *apiendpoint.APIEndpoint[*CancelScheduledRequest, *apiresource.Message] {
	return (&apiendpoint.APIEndpoint[*CancelScheduledRequest, *apiresource.Message]{
		Title:               "Cancel Scheduled Message",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/messages/{id}/actions/cancel",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeChatMessage,
		IncludeConfig:       messageIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CancelScheduledRequest) (*apiresource.Message, *apierror.APIError) {
			return svc.(MessageSvc).CancelScheduled
		},
	})
}
