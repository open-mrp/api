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

// Marks a notification dismissed, clearing it from your active feed.
//
// The notification isn't deleted: you can still retrieve it, and list it with `statuses[]=dismissed`. Dismissing it again keeps the first dismissal time.
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
		ObjectType:        constants.ObjectTypeNotification,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *MarkNotificationRequest) (*apiresource.Notification, *apierror.APIError) {
			return svc.(NotificationSvc).MarkDismissed
		},
		IncludeConfig: notificationIncludeConfig(),
	})
}
