package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
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

// Returns the production steps a batch can be advanced to from a given scanning station.
//
// Use this to drive the step picker on a scanning terminal after an operator scans a batch. Traversal starts at the batch: an open batch that has already been scanned offers the steps that come after its current step, while a closed batch is followed downstream to the batches it produced and the search continues from there. Only steps assigned to the given scanning station are returned, so a batch with nothing left to do at that station comes back with an empty list.
type GetPossibleNextStepsEndpoint struct{}

func (e *GetPossibleNextStepsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPossibleNextStepsRequest, *apiresource.List[apiresource.ScanningProductionStepInfo]] {
	return (&apiendpoint.APIEndpoint[*GetPossibleNextStepsRequest, *apiresource.List[apiresource.ScanningProductionStepInfo]]{
		Title:               "Get Possible Next Steps",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/batches/{id}/next-steps",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ReadOnly:            true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionRead}},
		ObjectType:          constants.ObjectTypeScanningProductionStepInfo,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPossibleNextStepsRequest) (*apiresource.List[apiresource.ScanningProductionStepInfo], *apierror.APIError) {
			return svc.(BatchSvc).GetPossibleNextSteps
		},
	})
}
