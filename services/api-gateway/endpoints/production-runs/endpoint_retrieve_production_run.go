package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a production run by ID.
type RetrieveProductionRunRequest struct {
	// Production run ID.
	ProductionRunID string `path:"id" validate:"required"`
	// Fields to include in the response.
	Includes []string `query:"include"`
}

// Returns a production run by ID.
type RetrieveProductionRunEndpoint struct{}

func (e *RetrieveProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveProductionRunRequest, *apiresource.ProductionRunDetail] {
	return (&apiendpoint.APIEndpoint[*RetrieveProductionRunRequest, *apiresource.ProductionRunDetail]{
		Title:             "Retrieve Production Run",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveProductionRunRequest) (*apiresource.ProductionRunDetail, *apierror.APIError) {
			return svc.(ProductionRunSvc).GetProductionRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductionRun,
			Fields:     []string{"responsible_user"},
		}),
	})
}
