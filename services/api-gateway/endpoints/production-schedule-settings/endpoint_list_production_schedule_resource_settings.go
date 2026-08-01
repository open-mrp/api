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

// Request to list per-resource planning overrides.
type ListResourceSettingsRequest struct{}

// Returns the per-machine, per-department and per-step planning overrides.
//
// This is where machines are marked as the planning constraint, and where a department or step declares how many weeks after the constraint its work starts.
type ListResourceSettingsEndpoint struct{}

func (e *ListResourceSettingsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListResourceSettingsRequest, *apiresource.List[apiresource.ProductionScheduleResourceSetting]] {
	return (&apiendpoint.APIEndpoint[*ListResourceSettingsRequest, *apiresource.List[apiresource.ProductionScheduleResourceSetting]]{
		Title:             "List Production Schedule Resource Settings",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedule-settings/resources",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionScheduleResourceSetting,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListResourceSettingsRequest) (*apiresource.List[apiresource.ProductionScheduleResourceSetting], *apierror.APIError) {
			return svc.(ProductionScheduleSettingsSvc).ListResourceSettings
		},
	})
}
