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

// Dismisses a notification, removing it from the active feed.
//
// The notification is not deleted: it can still be retrieved by ID and listed with the `dismissed` status filter. Dismissing an already-dismissed notification keeps the original dismissal time.
type MarkDismissedEndpoint struct{}

func (e *MarkDismissedEndpoint) Materialize() *apiendpoint.APIEndpoint[*MarkNotificationRequest, *apiresource.Notification] {
	return (&apiendpoint.APIEndpoint[*MarkNotificationRequest, *apiresource.Notification]{
		Title:             "Dismiss Notification",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/messaging/notifications/{id}/actions/dismiss",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeNotification,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *MarkNotificationRequest) (*apiresource.Notification, *apierror.APIError) {
			return svc.(NotificationSvc).MarkDismissed
		},
		IncludeConfig: notificationIncludeConfig(),
	})
}
