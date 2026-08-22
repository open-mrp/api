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

// Request to remove one item's planning overrides.
type DeleteItemSettingRequest struct {
	// Item whose overrides are being removed.
	ItemID string `path:"item_id" validate:"required"`
}

// Removes one item's planning overrides, returning it to the account defaults and its product line's conventions.
//
// Fails with a not-found error when the item has no overrides, rather than reporting success: a mistyped item ID would otherwise read as a change that never happened.
type DeleteItemSettingEndpoint struct{}

func (e *DeleteItemSettingEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteItemSettingRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteItemSettingRequest, *apiresource.EmptyResource]{
		Title:             "Delete Production Schedule Item Setting",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedule-settings/items/{item_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionScheduleItemSetting,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteItemSettingRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ProductionScheduleSettingsSvc).DeleteItemSetting
		},
	})
}
