package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a production step.
type DeleteProductionStepRequest struct {
	// Production step ID.
	ProductionStepID string `path:"id" validate:"required"`
}

// Deletes a production step.
//
// The step's connections to its upstream and downstream steps are removed as part of the deletion, so the neighboring steps are left unconnected to each other. Deleting a step that was already deleted returns an already-deleted error rather than a not-found error.
type DeleteProductionStepEndpoint struct{}

func (e *DeleteProductionStepEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductionStepRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteProductionStepRequest, *apiresource.EmptyResource]{
		Title:               "Delete Production Step",
		Method:              http.MethodDelete,
		Route:               "/v1/operations/production-steps/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductionSteps, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductionStepRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ProductionStepSvc).DeleteProductionStep
		},
	})
}
