package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteProductionRunRequest is the request to delete a production run.
type DeleteProductionRunRequest struct {
	// The ID of the production run to delete.
	ProductionRunID string `path:"id" validate:"required"`
}

type DeleteProductionRunEndpoint struct{}

func (e *DeleteProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductionRunRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteProductionRunRequest, *apiresource.EmptyResource]{
		Title:             "Delete Production Run",
		Description:       "Deletes a production run and its associated batches and order links.",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs/{id}",
		Request:           &DeleteProductionRunRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductionRunRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ProductionRunSvc).DeleteProductionRun
		},
	}
}
