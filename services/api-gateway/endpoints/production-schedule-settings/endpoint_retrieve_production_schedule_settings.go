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
// The whole set is always returned. An account that has never saved settings reads back the values the solver would apply anyway, so a caller never has to know which assumptions are in play; `settings_status` says whether the values were saved on the account or are those defaults.
//
// Per-machine, per-department and per-step overrides of these assumptions are read separately.
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
