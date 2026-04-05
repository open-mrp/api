package consumptionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteConsumptionRequest is the request to delete a consumption.
type DeleteConsumptionRequest struct {
	// The ID of the production step.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// The ID of the consumption to delete.
	ConsumptionID string `path:"id" validate:"required"`
}

type DeleteConsumptionEndpoint struct{}

func (e *DeleteConsumptionEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteConsumptionRequest, *apiresource.Consumption] {
	return &apiendpoint.APIEndpoint[*DeleteConsumptionRequest, *apiresource.Consumption]{
		Title:             "Delete Consumption",
		Description:       "Deletes a consumption from a production step.",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/production-steps/{production_step_id}/consumptions/{id}",
		ContentType:       "application/json",
		Request:           &DeleteConsumptionRequest{},
		Response:          &apiresource.Consumption{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
			return svc.(ConsumptionSvc).DeleteConsumption
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeConsumption,
			Fields:     []string{"consumed_item"},
		}),
	}
}
