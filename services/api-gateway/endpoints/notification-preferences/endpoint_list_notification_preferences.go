package notificationpreferenceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list the caller's notification preferences.
type ListNotificationPreferencesRequest struct{}

// Lists the current user's notification preferences for the account they are acting in: their global default plus any per-category overrides.
//
// Only preferences the user has explicitly set are returned, so an empty list means everything falls back to the standard behavior — in-app notifications on, email and push off.
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
