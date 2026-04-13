package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a production step.
type GetProductionStepRequest struct {
	// Production step ID.
	ProductionStepID string `path:"id" validate:"required"`
}

type GetProductionStepEndpoint struct{}

func (e *GetProductionStepEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetProductionStepRequest, *apiresource.ProductionStep] {
	return &apiendpoint.APIEndpoint[*GetProductionStepRequest, *apiresource.ProductionStep]{
		Title:             "Get Production Step",
		Description:       "Returns a production step by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-steps/{id}",
		Request:           &GetProductionStepRequest{},
		Response:          &apiresource.ProductionStep{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError) {
			return svc.(ProductionStepSvc).GetProductionStep
		},
	}
}
