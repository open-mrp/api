package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListBatchesByScanningStationRequest is the request to list batches for a scanning station.
type ListBatchesByScanningStationRequest struct {
	// The ID of the scanning station to list batches for.
	ScanningStationID string `path:"id" validate:"required"`
	apiresource.PaginationRequest
}

type ListBatchesByScanningStationEndpoint struct{}

func (e *ListBatchesByScanningStationEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListBatchesByScanningStationRequest, *apiresource.List[apiresource.Batch]] {
	return &apiendpoint.APIEndpoint[*ListBatchesByScanningStationRequest, *apiresource.List[apiresource.Batch]]{
		Title:             "List Batches by Scanning Station",
		Description:       "Returns a paginated list of batches for a given scanning station.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/scanning-stations/{id}/batches",
		Request:           &ListBatchesByScanningStationRequest{},
		Response:          &apiresource.List[apiresource.Batch]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListBatchesByScanningStationRequest) (*apiresource.List[apiresource.Batch], *apierror.APIError) {
			return svc.(BatchSvc).ListBatchesByScanningStation
		},
	}
}
