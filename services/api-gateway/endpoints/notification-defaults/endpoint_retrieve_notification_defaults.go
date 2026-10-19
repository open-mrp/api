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
)

// Request to retrieve your account's notification defaults.
type RetrieveNotificationDefaultsRequest struct{}

// Retrieves your account's notification defaults.
//
// They apply to everyone in the account after their own preferences and their role's defaults.
type RetrieveNotificationDefaultsEndpoint struct{}

func (e *RetrieveNotificationDefaultsEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveNotificationDefaultsRequest, *apiresource.NotificationDefaults] {
	return (&apiendpoint.APIEndpoint[*RetrieveNotificationDefaultsRequest, *apiresource.NotificationDefaults]{
		Title:             "Retrieve Notification Defaults",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             route,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeNotificationDefaults,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveNotificationDefaultsRequest) (*apiresource.NotificationDefaults, *apierror.APIError) {
			return svc.(Svc).Get
		},
	})
}
