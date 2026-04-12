package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetProductionRunRequest is the request to retrieve a single production run by ID.
type GetProductionRunRequest struct {
	// The ID of the production run to retrieve.
	ProductionRunID string `path:"id" validate:"required"`
	// The fields to include in the response.
	Includes []string `query:"include"`
}

type GetProductionRunEndpoint struct{}

func (e *GetProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetProductionRunRequest, *apiresource.ProductionRunDetail] {
	return &apiendpoint.APIEndpoint[*GetProductionRunRequest, *apiresource.ProductionRunDetail]{
		Title:             "Get Production Run",
		Description:       "Returns a single production run by its ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs/{id}",
		Request:           &GetProductionRunRequest{},
		Response:          &apiresource.ProductionRunDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetProductionRunRequest) (*apiresource.ProductionRunDetail, *apierror.APIError) {
			return svc.(ProductionRunSvc).GetProductionRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductionRun,
			Fields:     []string{"responsible_user"},
		}),
	}
}
