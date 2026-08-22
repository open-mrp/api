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

// Request to list per-item planning overrides.
type ListItemSettingsRequest struct{}

// Returns every per-item planning override in the account.
//
// Only items that have been given an override appear here. An item with none is planned on the account defaults and its product line's conventions, which is the normal case — this is the list of exceptions, not a list of every item.
type ListItemSettingsEndpoint struct{}

func (e *ListItemSettingsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListItemSettingsRequest, *apiresource.List[apiresource.ProductionScheduleItemSetting]] {
	return (&apiendpoint.APIEndpoint[*ListItemSettingsRequest, *apiresource.List[apiresource.ProductionScheduleItemSetting]]{
		Title:             "List Production Schedule Item Settings",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedule-settings/items",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionScheduleItemSetting,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListItemSettingsRequest) (*apiresource.List[apiresource.ProductionScheduleItemSetting], *apierror.APIError) {
			return svc.(ProductionScheduleSettingsSvc).ListItemSettings
		},
	})
}
