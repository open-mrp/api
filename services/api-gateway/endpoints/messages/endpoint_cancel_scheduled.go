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

// Request to cancel a pending scheduled message.
type CancelScheduledRequest struct {
	// Message ID (the scheduled message).
	MessageID string `path:"id" validate:"required"`
}

// Cancels a scheduled message the caller created (status becomes `canceled`).
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
