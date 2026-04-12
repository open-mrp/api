package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListBatchesByProductionRunRequest is the request to list batches for a production run.
type ListBatchesByProductionRunRequest struct {
	// The ID of the production run.
	ProductionRunID string `path:"id" validate:"required"`
	apiresource.PaginationRequest
}

type ListBatchesByProductionRunEndpoint struct{}

func (e *ListBatchesByProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListBatchesByProductionRunRequest, *apiresource.List[apiresource.Batch]] {
	return &apiendpoint.APIEndpoint[*ListBatchesByProductionRunRequest, *apiresource.List[apiresource.Batch]]{
		Title:             "List Batches by Production Run",
		Description:       "Returns a paginated list of batches associated with a production run.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs/{id}/batches",
		Request:           &ListBatchesByProductionRunRequest{},
		Response:          &apiresource.List[apiresource.Batch]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListBatchesByProductionRunRequest) (*apiresource.List[apiresource.Batch], *apierror.APIError) {
			return svc.(ProductionRunSvc).ListBatchesByProductionRun
		},
	}
}
