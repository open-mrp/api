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

// Request to read one item's planning overrides.
type RetrieveItemSettingRequest struct {
	// Item whose overrides are being read.
	ItemID string `path:"item_id" validate:"required"`
}

// Returns the planning overrides for one item.
//
// Fails with a not-found error when the item has none, rather than returning an empty set of overrides: an item with no overrides is planned on the account defaults and its product line's conventions, and reporting that as a resource would suggest there is something here to edit.
type RetrieveItemSettingEndpoint struct{}

func (e *RetrieveItemSettingEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveItemSettingRequest, *apiresource.ProductionScheduleItemSetting] {
	return (&apiendpoint.APIEndpoint[*RetrieveItemSettingRequest, *apiresource.ProductionScheduleItemSetting]{
		Title:             "Retrieve Production Schedule Item Setting",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedule-settings/items/{item_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionScheduleItemSetting,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveItemSettingRequest) (*apiresource.ProductionScheduleItemSetting, *apierror.APIError) {
			return svc.(ProductionScheduleSettingsSvc).GetItemSetting
		},
	})
}
