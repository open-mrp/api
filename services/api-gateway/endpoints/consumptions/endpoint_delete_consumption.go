package consumptionep

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

// Request to delete a consumption.
type DeleteConsumptionRequest struct {
	// Production step ID.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// Consumption ID.
	ConsumptionID string `path:"id" validate:"required"`
}

// Removes a material input from a production step.
//
// Any production-flow connections established through this consumption are disconnected, and the remaining consumptions are re-linked. A deleted stub is returned; deleting it again reports that it has already been deleted.
//
// Kept for 1.0.forge-preview.9 callers only; 1.0.forge has manufacturing methods instead.
type DeleteConsumptionEndpoint struct{}

func (e *DeleteConsumptionEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteConsumptionRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteConsumptionRequest, *object.Deleted]{
		Title:             "Delete Consumption",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/production-steps/{production_step_id}/consumptions/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeConsumption,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSteps, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteConsumptionRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ConsumptionSvc).DeleteConsumption, constants.ObjectTypeConsumption, "id")
		},
	})
}
