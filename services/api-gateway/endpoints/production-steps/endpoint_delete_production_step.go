package productionstepep

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

// Request to delete a production step.
type DeleteProductionStepRequest struct {
	// Production step ID.
	ProductionStepID string `path:"id" validate:"required"`
}

// Deletes a production step.
//
// The step's connections to its upstream and downstream steps are removed as part of the deletion, so the neighboring steps are left unconnected to each other. Deleting a step that was already deleted returns an already-deleted error rather than a not-found error.
type DeleteProductionStepEndpoint struct{}

func (e *DeleteProductionStepEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductionStepRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteProductionStepRequest, *object.Deleted]{
		Title:             "Delete Production Step",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/production-steps/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProductionSteps, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductionStepRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ProductionStepSvc).DeleteProductionStep, constants.ObjectTypeProductionStep, "id")
		},
	})
}
