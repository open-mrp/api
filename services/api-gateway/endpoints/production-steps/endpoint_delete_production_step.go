package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a production step.
type DeleteProductionStepRequest struct {
	// Production step ID.
	ProductionStepID string `path:"id" validate:"required"`
}

// Deletes a production step and its associated data.
//
// The step's connections in the production flow graph are removed as part of the deletion.
type DeleteProductionStepEndpoint struct{}

func (e *DeleteProductionStepEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductionStepRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteProductionStepRequest, *apiresource.EmptyResource]{
		Title:             "Delete Production Step",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/production-steps/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductionStepRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ProductionStepSvc).DeleteProductionStep
		},
	})
}
