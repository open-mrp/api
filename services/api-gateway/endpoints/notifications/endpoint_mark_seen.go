package notificationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to transition a single notification owned by the caller.
type MarkNotificationRequest struct {
	// Notification ID.
	NotificationID string `path:"id" validate:"required"`
}

// Marks a notification as seen.
type MarkSeenEndpoint struct{}

func (e *MarkSeenEndpoint) Materialize() *apiendpoint.APIEndpoint[*MarkNotificationRequest, *apiresource.Notification] {
	return (&apiendpoint.APIEndpoint[*MarkNotificationRequest, *apiresource.Notification]{
		Title:               "Mark Notification Seen",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/notifications/{id}/actions/seen",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeNotification,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *MarkNotificationRequest) (*apiresource.Notification, *apierror.APIError) {
			return svc.(NotificationSvc).MarkSeen
		},
		IncludeConfig: notificationIncludeConfig(),
	})
}
