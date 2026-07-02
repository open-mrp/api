package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a production run.
type DeleteProductionRunRequest struct {
	// Production run ID.
	ProductionRunID string `path:"id" validate:"required"`
}

// Deletes a production run.
//
// All batches recorded against the run are deleted, linked orders are detached from the run, and reserved inventory for those orders is released.
type DeleteProductionRunEndpoint struct{}

func (e *DeleteProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductionRunRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteProductionRunRequest, *apiresource.EmptyResource]{
		Title:             "Delete Production Run",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionRuns, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductionRunRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ProductionRunSvc).DeleteProductionRun
		},
	})
}
