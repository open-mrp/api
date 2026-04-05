package consumptionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetConsumptionRequest is the request to retrieve a single consumption.
type GetConsumptionRequest struct {
	// The ID of the production step.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// The ID of the consumption to retrieve.
	ConsumptionID string `path:"id" validate:"required"`
}

type GetConsumptionEndpoint struct{}

func (e *GetConsumptionEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetConsumptionRequest, *apiresource.Consumption] {
	return &apiendpoint.APIEndpoint[*GetConsumptionRequest, *apiresource.Consumption]{
		Title:             "Get Consumption",
		Description:       "Returns a single consumption by its ID within a production step.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/production-steps/{production_step_id}/consumptions/{id}",
		ContentType:       "application/json",
		Request:           &GetConsumptionRequest{},
		Response:          &apiresource.Consumption{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
			return svc.(ConsumptionSvc).GetConsumption
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeConsumption,
			Fields:     []string{"consumed_item"},
		}),
	}
}
