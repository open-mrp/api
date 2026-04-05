package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteProductionStepRequest is the request to delete a production step.
type DeleteProductionStepRequest struct {
	// The ID of the production step to delete.
	ProductionStepID string `path:"id" validate:"required"`
}

type DeleteProductionStepEndpoint struct{}

func (e *DeleteProductionStepEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductionStepRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteProductionStepRequest, *apiresource.EmptyResource]{
		Title:             "Delete Production Step",
		Description:       "Deletes a production step and its associated data.",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/production-steps/{id}",
		ContentType:       "application/json",
		Request:           &DeleteProductionStepRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductionStepRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ProductionStepSvc).DeleteProductionStep
		},
	}
}
