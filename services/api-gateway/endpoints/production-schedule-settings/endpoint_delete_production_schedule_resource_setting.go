package productionschedulesettingsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to remove a per-resource planning override.
type DeleteResourceSettingRequest struct {
	// ID of the resource setting.
	SettingID string `path:"id" validate:"required"`
}

// Removes a per-resource planning override, returning that resource to the account defaults.
type DeleteResourceSettingEndpoint struct{}

func (e *DeleteResourceSettingEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteResourceSettingRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteResourceSettingRequest, *apiresource.EmptyResource]{
		Title:             "Delete Production Schedule Resource Setting",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedule-settings/resources/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionScheduleResourceSetting,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteResourceSettingRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ProductionScheduleSettingsSvc).DeleteResourceSetting
		},
	})
}
