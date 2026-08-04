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

// Request to retrieve a single notification addressed to the caller.
type RetrieveNotificationRequest struct {
	// Notification ID.
	NotificationID string `path:"id" validate:"required"`
}

// Retrieves a single notification by ID.
//
// Only notifications addressed to the current user are visible; another user's notification is reported as not found. Dismissed notifications remain retrievable.
type RetrieveNotificationEndpoint struct{}

func (e *RetrieveNotificationEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveNotificationRequest, *apiresource.Notification] {
	return (&apiendpoint.APIEndpoint[*RetrieveNotificationRequest, *apiresource.Notification]{
		Title:               "Retrieve Notification",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/notifications/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeNotification,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveNotificationRequest) (*apiresource.Notification, *apierror.APIError) {
			return svc.(NotificationSvc).GetNotification
		},
		IncludeConfig: notificationIncludeConfig(),
	})
}
