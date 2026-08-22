package notificationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request for the caller's unread notification tallies.
type UnreadCountRequest struct{}

// Returns the current user's unread tallies for the account they are acting in, for driving a notification badge.
//
// The total also counts account announcements the user has not seen, so it can be higher than the notification count alone.
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
		Extras:              apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UnreadCountRequest) (*apiresource.NotificationUnreadCount, *apierror.APIError) {
			return svc.(NotificationSvc).GetUnreadCount
		},
	})
}
