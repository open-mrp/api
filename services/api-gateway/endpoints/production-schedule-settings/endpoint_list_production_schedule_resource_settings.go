package productionschedulesettingsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list per-resource planning overrides.
type ListResourceSettingsRequest struct{}

// Returns every per-machine, per-department and per-step override of the account's planning assumptions.
//
// An override exists only for a resource that has been given one: this is where a machine is taken out of the plan, and where a production step declares how many weeks its work starts after the step that feeds it. Anything absent from this list is planned on the account settings alone.
//
// The account's full set of overrides is returned at once — there are no filters and nothing to page through.
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
