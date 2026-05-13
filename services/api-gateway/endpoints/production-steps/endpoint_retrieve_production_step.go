package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a production step.
type RetrieveProductionStepRequest struct {
	// Production step ID.
	ProductionStepID string `path:"id" validate:"required"`
}

type RetrieveProductionStepEndpoint struct{}

func (e *RetrieveProductionStepEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveProductionStepRequest, *apiresource.ProductionStep] {
	return &apiendpoint.APIEndpoint[*RetrieveProductionStepRequest, *apiresource.ProductionStep]{
		Title:             "Retrieve Production Step",
		Description:       "Returns a production step by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-steps/{id}",
		Request:           &RetrieveProductionStepRequest{},
		Response:          &apiresource.ProductionStep{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError) {
			return svc.(ProductionStepSvc).GetProductionStep
		},
	}
}
