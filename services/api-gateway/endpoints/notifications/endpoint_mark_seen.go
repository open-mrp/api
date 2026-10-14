package notificationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to advance the state of a single notification addressed to the caller.
type MarkNotificationRequest struct {
	// Notification ID.
	NotificationID string `path:"id" validate:"required"`
}

// Marks a notification as seen, as when it is surfaced to the user without being opened.
//
// Seeing a notification removes it from the unread count but leaves it in the feed. Repeating the call keeps the original seen time.
type MarkSeenEndpoint struct{}

func (e *MarkSeenEndpoint) Materialize() *apiendpoint.APIEndpoint[*MarkNotificationRequest, *apiresource.Notification] {
	return (&apiendpoint.APIEndpoint[*MarkNotificationRequest, *apiresource.Notification]{
		Title:             "Mark Notification Seen",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/messaging/notifications/{id}/actions/seen",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeNotification,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *MarkNotificationRequest) (*apiresource.Notification, *apierror.APIError) {
			return svc.(NotificationSvc).MarkSeen
		},
		IncludeConfig: notificationIncludeConfig(),
	})
}
