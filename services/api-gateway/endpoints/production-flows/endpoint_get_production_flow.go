package productionflowep

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

// Request to retrieve the production flow graph for an item.
type GetProductionFlowRequest struct {
	// ID of the item whose production flow to retrieve.
	ItemID string `path:"item_id" validate:"required"`
}

// Returns the production flow graph for the given item.
//
// The graph contains the step(s) that produce the item, every upstream step that feeds them, and any connected downstream steps, with each step's production output, consumptions, and connections. The list of steps is empty if no production step produces the item.
//
// Kept for 1.0.forge-preview.9 callers only; 1.0.forge has manufacturing methods instead.
type GetProductionFlowEndpoint struct{}

func (e *GetProductionFlowEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetProductionFlowRequest, *apiresource.ProductionFlow] {
	return (&apiendpoint.APIEndpoint[*GetProductionFlowRequest, *apiresource.ProductionFlow]{
		Title:             "Get Production Flow",
		Method:            http.MethodGet,
		Route:             "/v1/operations/production-flows/by-item/{item_id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionFlow,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSteps, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetProductionFlowRequest) (*apiresource.ProductionFlow, *apierror.APIError) {
			return svc.(ProductionFlowSvc).GetProductionFlow
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductionFlow,
			Fields: []string{
				"steps", "steps.production", "steps.production.produced_item", "steps.consumptions", "steps.consumptions.consumed_item", "steps.machines", "steps.scanning_station", "steps.department", "steps.in_steps", "steps.out_steps",
			},
		}),
	})
}
