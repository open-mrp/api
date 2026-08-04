package notificationpreferenceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create or replace one of the caller's notification preferences.
//
// A user has at most one preference per category, so sending the same category again replaces the previous settings outright — every channel is written from this request, not merged with what was there before.
//
// Chat notifications are the only ones these settings currently govern: notifications in every other category reach the in-app feed and are never emailed, whatever is stored here.
type UpsertNotificationPreferenceRequest struct {
	// The notification category these settings apply to, such as `chat.message`.
	//
	// Leave it out to set the global default used for every category without its own preference.
	Category field.Clearable[string] `json:"category,omitzero"`
	// Whether notifications in this category appear in the user's in-app feed.
	//
	// A direct @mention is always delivered in-app, even when this is off.
	InAppEnabled bool `json:"in_app_enabled"`
	// Whether notifications in this category are also emailed to the user.
	//
	// Email is additionally suppressed for a conversation the user has muted, and only sent on the cadence set by `digest`.
	EmailEnabled bool `json:"email_enabled"`
	// Whether notifications in this category are also sent as push notifications.
	//
	// Push delivery is not available yet; the choice is stored for when it is.
	PushEnabled bool `json:"push_enabled"`
	// How often email for this category is sent.
	//
	// - `instant`: send an email as soon as an eligible notification occurs.
	// - `hourly`: collect eligible notifications into a single hourly email.
	// - `daily`: collect eligible notifications into a single daily email.
	// - `off`: never send email for this category, even when email is otherwise enabled.
	//
	// This governs email only; in-app delivery is unaffected. Batched sending is not running yet, so `hourly` and `daily` currently hold email back in the same way as `off`.
	Digest field.Optional[constants.NotificationDigest] `json:"digest,omitzero" default:"instant"`
}

var sampleUpsertNotificationPreferenceRequest = &UpsertNotificationPreferenceRequest{
	Category:     field.Set("chat.message"),
	InAppEnabled: true,
	EmailEnabled: false,
	PushEnabled:  false,
	Digest:       field.Some(constants.NotificationDigestInstant),
}

func (*UpsertNotificationPreferenceRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpsertNotificationPreferenceRequest)
}

// Creates or replaces one of the current user's notification preferences, either their global default or the override for a single category.
//
// The preference applies only to the account being acted in, and the category must be one the platform recognizes. Callers without a user membership in that account cannot hold preferences and are refused.
type UpsertNotificationPreferenceEndpoint struct{}

func (e *UpsertNotificationPreferenceEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpsertNotificationPreferenceRequest, *apiresource.NotificationPreference] {
	return (&apiendpoint.APIEndpoint[*UpsertNotificationPreferenceRequest, *apiresource.NotificationPreference]{
		Title:               "Upsert Notification Preference",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/messaging/preferences",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeNotificationPreference,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpsertNotificationPreferenceRequest) (*apiresource.NotificationPreference, *apierror.APIError) {
			return svc.(NotificationPreferenceSvc).UpsertNotificationPreference
		},
	})
}
