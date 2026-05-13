package consumptionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a consumption.
type RetrieveConsumptionRequest struct {
	// Production step ID.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// Consumption ID.
	ConsumptionID string `path:"id" validate:"required"`
}

type RetrieveConsumptionEndpoint struct{}

func (e *RetrieveConsumptionEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveConsumptionRequest, *apiresource.Consumption] {
	return &apiendpoint.APIEndpoint[*RetrieveConsumptionRequest, *apiresource.Consumption]{
		Title:             "Retrieve Consumption",
		Description:       "Returns a consumption by ID within a production step.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/production-steps/{production_step_id}/consumptions/{id}",
		ContentType:       "application/json",
		Request:           &RetrieveConsumptionRequest{},
		Response:          &apiresource.Consumption{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
			return svc.(ConsumptionSvc).GetConsumption
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeConsumption,
			Fields:     []string{"consumed_item"},
		}),
	}
}
