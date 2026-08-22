package consumptionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a consumption.
type DeleteConsumptionRequest struct {
	// Production step ID.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// Consumption ID.
	ConsumptionID string `path:"id" validate:"required"`
}

// Removes a material input from a production step.
//
// Any production-flow connections established through this consumption are disconnected, and the remaining consumptions are re-linked. The deleted consumption is returned; deleting it again reports that it has already been deleted.
type DeleteConsumptionEndpoint struct{}

func (e *DeleteConsumptionEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteConsumptionRequest, *apiresource.Consumption] {
	return (&apiendpoint.APIEndpoint[*DeleteConsumptionRequest, *apiresource.Consumption]{
		Title:             "Delete Consumption",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/production-steps/{production_step_id}/consumptions/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeConsumption,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSteps, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
			return svc.(ConsumptionSvc).DeleteConsumption
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeConsumption,
			Fields:     []string{"consumed_item"},
		}),
	})
}
