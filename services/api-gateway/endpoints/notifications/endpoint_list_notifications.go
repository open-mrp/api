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
	// Return only notifications of this category, such as `chat.mention` or `order.updated`.
	Category *constants.NotificationCategory `query:"category"`
	// Return only notifications in this lifecycle state.
	//
	// When omitted, the response is the active feed: every notification that has not been dismissed, whatever its seen or read state. Pass `dismissed` to review notifications that were cleared out of the feed.
	Status *constants.NotificationStatus `query:"status"`
	// Return only notifications sent by these actors.
	//
	// A notification sent by a person is attributed to their account user id, not their user id.
	SenderIDs []string `query:"sender_ids"`
	// Return only notifications sent by these kinds of actor.
	//
	// Notifications raised by the platform itself are attributed to the `system` sender type but are returned without a sender.
	SenderTypes []constants.NotificationSenderType `query:"sender_types"`
}

// Lists the notifications addressed to the current user, newest first.
//
// The feed is personal and scoped to the account being acted in, so it never includes another user's notifications. Callers with no user membership in that account, such as an API key, get an empty list rather than an error.
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
