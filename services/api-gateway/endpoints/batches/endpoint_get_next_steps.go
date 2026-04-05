package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetPossibleNextStepsRequest is the request to retrieve possible next production steps for a batch.
type GetPossibleNextStepsRequest struct {
	// The ID of the batch.
	BatchID string `path:"id" validate:"required"`
	// The ID of the scanning station to evaluate next steps from.
	ScanningStationID string `json:"scanning_station_id" validate:"required"`
}

var sampleGetPossibleNextStepsRequest = &GetPossibleNextStepsRequest{
	ScanningStationID: apiresource.SampleScanningStationID,
}

func (*GetPossibleNextStepsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleGetPossibleNextStepsRequest)
}

type GetPossibleNextStepsEndpoint struct{}

func (e *GetPossibleNextStepsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPossibleNextStepsRequest, *apiresource.List[apiresource.ScanningProductionStepInfo]] {
	return &apiendpoint.APIEndpoint[*GetPossibleNextStepsRequest, *apiresource.List[apiresource.ScanningProductionStepInfo]]{
		Title:             "Get Possible Next Steps",
		Description:       "Returns the possible next production steps for a batch at a given scanning station.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/batches/{id}/next-steps",
		Request:           &GetPossibleNextStepsRequest{},
		Response:          &apiresource.List[apiresource.ScanningProductionStepInfo]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPossibleNextStepsRequest) (*apiresource.List[apiresource.ScanningProductionStepInfo], *apierror.APIError) {
			return svc.(BatchSvc).GetPossibleNextSteps
		},
	}
}
