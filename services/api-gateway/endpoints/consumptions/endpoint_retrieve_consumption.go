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
)

// Request to retrieve a consumption.
type RetrieveConsumptionRequest struct {
	// Production step ID.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// Consumption ID.
	ConsumptionID string `path:"id" validate:"required"`
}

// Returns a consumption by ID within a production step.
//
// Kept for 1.0.forge-preview.9 callers only; 1.0.forge has manufacturing methods instead.
type RetrieveConsumptionEndpoint struct{}

func (e *RetrieveConsumptionEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveConsumptionRequest, *apiresource.Consumption] {
	return (&apiendpoint.APIEndpoint[*RetrieveConsumptionRequest, *apiresource.Consumption]{
		Title:             "Retrieve Consumption",
		Method:            http.MethodGet,
		Route:             "/v1/operations/production-steps/{production_step_id}/consumptions/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeConsumption,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSteps, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
			return svc.(ConsumptionSvc).GetConsumption
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeConsumption,
			Fields:     consumptionEndpointIncludes,
		}),
	})
}
