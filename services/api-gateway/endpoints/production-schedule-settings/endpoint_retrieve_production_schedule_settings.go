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

// Request to retrieve the account's planning assumptions.
type RetrieveProductionScheduleSettingsRequest struct{}

// Returns the planning assumptions production schedules are solved against.
//
// Always fully populated: an account that has never saved settings gets the solver's own defaults rather than nulls, so a caller never has to know which values would otherwise be assumed. `settings_status` distinguishes the two.
type RetrieveProductionScheduleSettingsEndpoint struct{}

func (e *RetrieveProductionScheduleSettingsEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveProductionScheduleSettingsRequest, *apiresource.ProductionScheduleSettings] {
	return (&apiendpoint.APIEndpoint[*RetrieveProductionScheduleSettingsRequest, *apiresource.ProductionScheduleSettings]{
		Title:             "Retrieve Production Schedule Settings",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedule-settings",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionScheduleSettings,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveProductionScheduleSettingsRequest) (*apiresource.ProductionScheduleSettings, *apierror.APIError) {
			return svc.(ProductionScheduleSettingsSvc).GetSettings
		},
	})
}
