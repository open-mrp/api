package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// NotificationPreferenceItem represents a single notification preference toggle.
type NotificationPreferenceItem struct {
	// The notification type code (e.g. "invoice", "order_acknowledgement", "purchase_order_submission").
	NotificationTypeCode string `json:"notification_type_code"`
	// Whether the notification is enabled.
	Enabled bool `json:"enabled"`
}

// UpdateNotificationPreferencesRequest is the request to update notification preferences for an account user.
type UpdateNotificationPreferencesRequest struct {
	// The user ID of the account user.
	AccountUserID string `path:"id" validate:"required"`
	// The notification preferences to update.
	Preferences []NotificationPreferenceItem `json:"preferences"`
}

var sampleUpdateNotificationPreferencesRequest = &UpdateNotificationPreferencesRequest{
	Preferences: []NotificationPreferenceItem{
		{NotificationTypeCode: "invoice", Enabled: true},
		{NotificationTypeCode: "order_acknowledgement", Enabled: false},
	},
}

func (*UpdateNotificationPreferencesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateNotificationPreferencesRequest)
}

type UpdateNotificationPreferencesEndpoint struct{}

func (e *UpdateNotificationPreferencesEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateNotificationPreferencesRequest, *apiresource.AccountUser] {
	return &apiendpoint.APIEndpoint[*UpdateNotificationPreferencesRequest, *apiresource.AccountUser]{
		Title:             "Update Notification Preferences",
		Description:       "Updates notification preferences for an account user.",
		Method:            http.MethodPut,
		Route:             "/v1/identity/account-users/{id}/notification-preferences",
		Request:           &UpdateNotificationPreferencesRequest{},
		Response:          &apiresource.AccountUser{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateNotificationPreferencesRequest) (*apiresource.AccountUser, *apierror.APIError) {
			return svc.(AccountUserSvc).UpdateNotificationPreferences
		},
	}
}
