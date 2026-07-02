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

// Request for the caller's unread notification tallies.
type UnreadCountRequest struct{}

// Returns the current user's unread notification counts.
type UnreadCountEndpoint struct{}

func (e *UnreadCountEndpoint) Materialize() *apiendpoint.APIEndpoint[*UnreadCountRequest, *apiresource.NotificationUnreadCount] {
	return (&apiendpoint.APIEndpoint[*UnreadCountRequest, *apiresource.NotificationUnreadCount]{
		Title:               "Get Notification Unread Count",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/notifications/unread-count",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeNotificationUnreadCount,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UnreadCountRequest) (*apiresource.NotificationUnreadCount, *apierror.APIError) {
			return svc.(NotificationSvc).GetUnreadCount
		},
	})
}
