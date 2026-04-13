package productionflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetProductionFlowRequest is the request to retrieve the production flow graph for an item.
type GetProductionFlowRequest struct {
	// Item ID.
	ItemID string `path:"item_id" validate:"required"`
}

type GetProductionFlowEndpoint struct{}

func (e *GetProductionFlowEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetProductionFlowRequest, *apiresource.ProductionFlow] {
	return &apiendpoint.APIEndpoint[*GetProductionFlowRequest, *apiresource.ProductionFlow]{
		Title:             "Get Production Flow",
		Description:       "Returns the production flow graph for the given item, including all production steps, their consumptions, and connections.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/production-flows/by-item/{item_id}",
		ContentType:       "application/json",
		Request:           &GetProductionFlowRequest{},
		Response:          &apiresource.ProductionFlow{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetProductionFlowRequest) (*apiresource.ProductionFlow, *apierror.APIError) {
			return svc.(ProductionFlowSvc).GetProductionFlow
		},
	}
}
