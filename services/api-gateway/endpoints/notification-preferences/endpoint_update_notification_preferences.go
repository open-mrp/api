package notificationpreferenceep

import (
	"context"
	"net/http"
	"time"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to update your notification preferences.
type UpdateNotificationPreferencesRequest struct {
	// Where your notifications go when you haven't set their type in `types`. Send `null` to go back to your role's or account's default.
	Default field.Clearable[apiresource.NotificationSettingInput] `json:"default,omitzero"`
	// Your settings for single notification types, keyed by type such as `order.updated`. A type sent as `null` goes back to your default; types you leave out are unchanged.
	Types map[string]*apiresource.NotificationSettingInput `json:"types,omitzero"`
	// When you're subscribed to records automatically: `created`, `assigned`, `mentioned`, `edited`. Send `null` to go back to your role's or account's list.
	AutoSubscribe field.Clearable[[]constants.AutoSubscribeTrigger] `json:"auto_subscribe,omitzero"`
	// Holds email and push notifications until this time; in-app notifications still arrive. Send `null` to resume now.
	PausedUntil field.Clearable[time.Time] `json:"paused_until,omitzero"`
}

var sampleUpdateNotificationPreferencesRequest = &UpdateNotificationPreferencesRequest{
	Types: map[string]*apiresource.NotificationSettingInput{
		string(constants.NotificationCategoryOrderUpdated): {
			Channels: []constants.NotificationChannel{constants.NotificationChannelInApp, constants.NotificationChannelEmail},
			Digest:   field.Some(constants.NotificationDigestInstant),
		},
	},
	AutoSubscribe: field.Set([]constants.AutoSubscribeTrigger{constants.AutoSubscribeCreated, constants.AutoSubscribeAssigned, constants.AutoSubscribeMentioned, constants.AutoSubscribeEdited}),
}

// SettingsPatch is the settings part of the request.
func (r *UpdateNotificationPreferencesRequest) SettingsPatch() apiresource.NotificationSettingsPatch {
	return apiresource.NotificationSettingsPatch{Default: r.Default, Types: r.Types, AutoSubscribe: r.AutoSubscribe}
}

func (*UpdateNotificationPreferencesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateNotificationPreferencesRequest)
}

// Updates your notification preferences.
//
// Only the fields you send change. `types` merges by type: a type you send replaces your setting for it, a type sent as `null` goes back to your default, and types you leave out are unchanged. Required types (`security.*`) always email, so their `channels` must include `email`.
type UpdateNotificationPreferencesEndpoint struct{}

func (e *UpdateNotificationPreferencesEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateNotificationPreferencesRequest, *apiresource.NotificationPreferences] {
	return (&apiendpoint.APIEndpoint[*UpdateNotificationPreferencesRequest, *apiresource.NotificationPreferences]{
		Title:             "Update Notification Preferences",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             preferencesRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeNotificationPreferences,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateNotificationPreferencesRequest) (*apiresource.NotificationPreferences, *apierror.APIError) {
			return svc.(NotificationPreferenceSvc).UpdateNotificationPreferences
		},
	})
}
