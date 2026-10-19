package notificationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list the caller's notifications.
type ListNotificationsRequest struct {
	object.ListRequest
	// Case-insensitive substring match on `title` or `body`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Returns only notifications of these types.
	Types []constants.NotificationCategory `query:"types"`
	// Returns only notifications in these states. Omit it for every state; your active feed is `unseen`, `seen` and `read`.
	Statuses []constants.NotificationStatus `query:"statuses"`
	// Returns only notifications sent by these actors. A person is identified by their account user ID.
	SenderIDs []string `query:"sender_ids"`
	// Returns only notifications sent by these kinds of actor. Notifications OpenMRP raises itself are `system` and have no `sender`.
	SenderTypes []constants.NotificationSenderType `query:"sender_types"`
}

// Returns a list of your notifications, newest first.
//
// Only your own notifications in the account you're acting in are listed. Callers who aren't a person in the account, such as an API key, get an empty list.
type ListNotificationsEndpoint struct{}

func (e *ListNotificationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListNotificationsRequest, *apiresource.List[apiresource.Notification]] {
	return (&apiendpoint.APIEndpoint[*ListNotificationsRequest, *apiresource.List[apiresource.Notification]]{
		Title:             "List Notifications",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/messaging/notifications",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeNotification,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}}},
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListNotificationsRequest) (*apiresource.List[apiresource.Notification], *apierror.APIError) {
			return svc.(NotificationSvc).ListNotifications
		},
		IncludeConfig: notificationIncludeConfig(),
	})
}
