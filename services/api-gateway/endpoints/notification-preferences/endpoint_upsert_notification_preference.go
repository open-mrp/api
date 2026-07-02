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

// Request to create or replace a notification preference for the caller.
//
// The preference is keyed by (caller, category), so repeating the request with the same category replaces the existing preference.
type UpsertNotificationPreferenceRequest struct {
	// The notification category this preference applies to.
	//
	// Omit (or `null`) to set the caller's global default.
	Category field.Clearable[string] `json:"category,omitzero"`
	// Whether in-app (bell) notifications are delivered for this category.
	InAppEnabled bool `json:"in_app_enabled"`
	// Whether email notifications are delivered for this category.
	EmailEnabled bool `json:"email_enabled"`
	// Whether push notifications are delivered for this category.
	PushEnabled bool `json:"push_enabled"`
	// How email delivery for this category is batched.
	//
	// - `instant`: send an email as soon as an eligible notification occurs.
	// - `hourly`: batch eligible notifications into a single hourly email.
	// - `daily`: batch eligible notifications into a single daily email.
	// - `off`: never send email for this category, even when email delivery is otherwise enabled.
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

// Creates or replaces a notification channel preference for the caller.
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
