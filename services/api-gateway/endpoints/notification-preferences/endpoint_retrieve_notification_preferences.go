package notificationpreferenceep

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

// Request to retrieve your notification preferences.
type RetrieveNotificationPreferencesRequest struct{}

// Retrieves your notification preferences in the account you're acting in.
//
// `default` and `auto_subscribe` show what applies to you, including what you inherit from your role and your account when you haven't set them yourself. `types` lists only the types you've set.
type RetrieveNotificationPreferencesEndpoint struct{}

func (e *RetrieveNotificationPreferencesEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveNotificationPreferencesRequest, *apiresource.NotificationPreferences] {
	return (&apiendpoint.APIEndpoint[*RetrieveNotificationPreferencesRequest, *apiresource.NotificationPreferences]{
		Title:             "Retrieve Notification Preferences",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             preferencesRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeNotificationPreferences,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveNotificationPreferencesRequest) (*apiresource.NotificationPreferences, *apierror.APIError) {
			return svc.(NotificationPreferenceSvc).GetNotificationPreferences
		},
	})
}
