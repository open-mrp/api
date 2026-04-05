package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetProductionRequest is the request to retrieve a single production output.
type GetProductionRequest struct {
	// The ID of the production step.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// The ID of the production to retrieve.
	ProductionID string `path:"id" validate:"required"`
}

type GetProductionEndpoint struct{}

func (e *GetProductionEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetProductionRequest, *apiresource.ProductionOutput] {
	return &apiendpoint.APIEndpoint[*GetProductionRequest, *apiresource.ProductionOutput]{
		Title:             "Get Production",
		Description:       "Returns a single production output by its ID within a production step.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/production-steps/{production_step_id}/productions/{id}",
		Request:           &GetProductionRequest{},
		Response:          &apiresource.ProductionOutput{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetProductionRequest) (*apiresource.ProductionOutput, *apierror.APIError) {
			return svc.(ProductionStepSvc).GetProduction
		},
	}
}
