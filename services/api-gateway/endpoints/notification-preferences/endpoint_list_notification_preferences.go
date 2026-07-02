package notificationpreferenceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list the caller's notification preferences.
type ListNotificationPreferencesRequest struct{}

// Lists the caller's notification channel preferences (global default + per-category overrides).
type ListNotificationPreferencesEndpoint struct{}

func (e *ListNotificationPreferencesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListNotificationPreferencesRequest, *apiresource.List[apiresource.NotificationPreference]] {
	return (&apiendpoint.APIEndpoint[*ListNotificationPreferencesRequest, *apiresource.List[apiresource.NotificationPreference]]{
		Title:               "List Notification Preferences",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/preferences",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeNotificationPreference,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListNotificationPreferencesRequest) (*apiresource.List[apiresource.NotificationPreference], *apierror.APIError) {
			return svc.(NotificationPreferenceSvc).ListNotificationPreferences
		},
	})
}
