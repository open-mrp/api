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

// Request to list the caller's notifications.
type ListNotificationsRequest struct {
	apiresource.PaginationRequest
	// Filter by category.
	Category *constants.NotificationCategory `query:"category"`
	// Filter by lifecycle status.
	//
	// When omitted, the feed returns the full active feed — every non-dismissed notification (seen and unseen alike), newest first.
	Status *constants.NotificationStatus `query:"status"`
	// Filter by sender id(s).
	SenderIDs []string `query:"sender_ids"`
	// Filter by sender type(s).
	SenderTypes []constants.NotificationSenderType `query:"sender_types"`
}

// Returns the current user's notifications, most recent first.
type ListNotificationsEndpoint struct{}

func (e *ListNotificationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListNotificationsRequest, *apiresource.List[apiresource.Notification]] {
	return (&apiendpoint.APIEndpoint[*ListNotificationsRequest, *apiresource.List[apiresource.Notification]]{
		Title:               "List Notifications",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/notifications",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeNotification,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		Extras:              apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListNotificationsRequest) (*apiresource.List[apiresource.Notification], *apierror.APIError) {
			return svc.(NotificationSvc).ListNotifications
		},
		IncludeConfig: notificationIncludeConfig(),
	})
}
