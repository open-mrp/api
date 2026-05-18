package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve possible next production steps for a batch.
type GetPossibleNextStepsRequest struct {
	// Batch ID.
	BatchID string `path:"id" validate:"required"`
	// Scanning station ID to evaluate next steps from.
	ScanningStationID string `json:"scanning_station_id" validate:"required"`
}

var sampleGetPossibleNextStepsRequest = &GetPossibleNextStepsRequest{
	ScanningStationID: apiresource.SampleScanningStationID,
}

func (*GetPossibleNextStepsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleGetPossibleNextStepsRequest)
}

// Returns possible next production steps for a batch at a given scanning station.
type GetPossibleNextStepsEndpoint struct{}

func (e *GetPossibleNextStepsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPossibleNextStepsRequest, *apiresource.List[apiresource.ScanningProductionStepInfo]] {
	return (&apiendpoint.APIEndpoint[*GetPossibleNextStepsRequest, *apiresource.List[apiresource.ScanningProductionStepInfo]]{
		Title:             "Get Possible Next Steps",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/batches/{id}/next-steps",
		Request:           &GetPossibleNextStepsRequest{},
		Response:          &apiresource.List[apiresource.ScanningProductionStepInfo]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPossibleNextStepsRequest) (*apiresource.List[apiresource.ScanningProductionStepInfo], *apierror.APIError) {
			return svc.(BatchSvc).GetPossibleNextSteps
		},
	}).WithDocSource(e)
}
