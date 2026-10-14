package productionrunep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete a production run.
type DeleteProductionRunRequest struct {
	// Production run ID.
	ProductionRunID string `path:"id" validate:"required"`
}

// Deletes a production run.
//
// All batches recorded against the run are deleted, linked orders are detached from the run, and the inventory those orders had reserved is released. Any production schedule lines that were released as this run revert to planned so the same work can be released again.
type DeleteProductionRunEndpoint struct{}

func (e *DeleteProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductionRunRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteProductionRunRequest, *object.Deleted]{
		Title:             "Delete Production Run",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductionRuns, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductionRunRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ProductionRunSvc).DeleteProductionRun, constants.ObjectTypeProductionRun, "id")
		},
	})
}
