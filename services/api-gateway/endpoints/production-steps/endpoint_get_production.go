package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a production output.
type GetProductionRequest struct {
	// Production step ID.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// Production ID.
	ProductionID string `path:"id" validate:"required"`
}

type GetProductionEndpoint struct{}

func (e *GetProductionEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetProductionRequest, *apiresource.ProductionOutput] {
	return &apiendpoint.APIEndpoint[*GetProductionRequest, *apiresource.ProductionOutput]{
		Title:             "Get Production",
		Description:       "Returns a production output by ID within a production step.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
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
