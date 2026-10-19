package notificationdefaultsep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to update your account's notification defaults.
type UpdateNotificationDefaultsRequest struct {
	// Where notifications go when `types` has no entry for their type. Send `null` to go back to OpenMRP's: in-app only.
	Default field.Clearable[apiresource.NotificationSettingInput] `json:"default,omitzero"`
	// Settings for single notification types, keyed by type such as `order.updated`. A type sent as `null` loses its setting; types you leave out are unchanged.
	Types map[string]*apiresource.NotificationSettingInput `json:"types,omitzero"`
	// When people are subscribed to records automatically: `created`, `assigned`, `mentioned`, `edited`. Send `null` to go back to OpenMRP's: `created`, `assigned` and `mentioned`.
	AutoSubscribe field.Clearable[[]constants.AutoSubscribeTrigger] `json:"auto_subscribe,omitzero"`
}

var sampleUpdateNotificationDefaultsRequest = &UpdateNotificationDefaultsRequest{
	Types: map[string]*apiresource.NotificationSettingInput{
		string(constants.NotificationCategoryOrderUpdated): {
			Channels: []constants.NotificationChannel{constants.NotificationChannelInApp, constants.NotificationChannelEmail},
		},
	},
}

func (*UpdateNotificationDefaultsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateNotificationDefaultsRequest)
}

// Updates your account's notification defaults.
//
// Only the fields you send change. `types` merges by type: a type you send replaces its setting, a type sent as `null` loses it, and types you leave out are unchanged. Required types (`security.*`) always email, so their `channels` must include `email`.
type UpdateNotificationDefaultsEndpoint struct{}

func (e *UpdateNotificationDefaultsEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateNotificationDefaultsRequest, *apiresource.NotificationDefaults] {
	return (&apiendpoint.APIEndpoint[*UpdateNotificationDefaultsRequest, *apiresource.NotificationDefaults]{
		Title:             "Update Notification Defaults",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             route,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeNotificationDefaults,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateNotificationDefaultsRequest) (*apiresource.NotificationDefaults, *apierror.APIError) {
			return svc.(Svc).Update
		},
	})
}
