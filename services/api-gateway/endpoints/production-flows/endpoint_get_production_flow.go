package productionflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetProductionFlowRequest is the request to retrieve the production flow graph for an item.
type GetProductionFlowRequest struct {
	// Item ID.
	ItemID string `path:"item_id" validate:"required"`
}

// Returns the production flow graph for the given item, including all production steps, their consumptions, and connections.
type GetProductionFlowEndpoint struct{}

func (e *GetProductionFlowEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetProductionFlowRequest, *apiresource.ProductionFlow] {
	return (&apiendpoint.APIEndpoint[*GetProductionFlowRequest, *apiresource.ProductionFlow]{
		Title:             "Get Production Flow",
		Method:            http.MethodGet,
		Route:             "/v1/operations/production-flows/by-item/{item_id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetProductionFlowRequest) (*apiresource.ProductionFlow, *apierror.APIError) {
			return svc.(ProductionFlowSvc).GetProductionFlow
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductionFlow,
			Fields: []string{
				"steps",
				"steps.production",
				"steps.production.produced_item",
				"steps.consumptions",
				"steps.consumptions.consumed_item",
				"steps.consumptions.quantity",
				"steps.consumptions.waste_quantity",
				"steps.machines",
				"steps.scanning_station",
				"steps.department",
				"steps.in_steps",
				"steps.out_steps",
			},
		}),
	})
}
