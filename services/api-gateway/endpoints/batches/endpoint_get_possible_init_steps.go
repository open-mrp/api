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

// Request to retrieve the production steps a batch can be initialized at.
type GetPossibleInitStepsRequest struct {
	// Batch ID.
	BatchID string `path:"id" validate:"required"`
	// Scanning station ID to evaluate initialization steps from.
	ScanningStationID string `json:"scanning_station_id" validate:"required"`
}

var sampleGetPossibleInitStepsRequest = &GetPossibleInitStepsRequest{
	ScanningStationID: apiresource.SampleScanningStationID,
}

func (*GetPossibleInitStepsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleGetPossibleInitStepsRequest)
}

// Returns the production steps a batch can be initialized at from a given scanning station.
//
// Use this to drive the step picker on a scanning terminal when an operator scans a batch that has not been scanned before. Initializing is the batch's first scan, so there is no prior step to advance from: the steps offered are the ones assigned to the given scanning station that produce the batch's own item. A batch whose item nothing at that station makes comes back with an empty list. For a batch that has already been scanned, use Get Possible Next Steps instead.
type GetPossibleInitStepsEndpoint struct{}

func (e *GetPossibleInitStepsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPossibleInitStepsRequest, *apiresource.List[apiresource.ScanningProductionStepInfo]] {
	return (&apiendpoint.APIEndpoint[*GetPossibleInitStepsRequest, *apiresource.List[apiresource.ScanningProductionStepInfo]]{
		Title:               "Get Possible Init Steps",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/batches/{id}/init-steps",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ReadOnly:            true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionRead}},
		ObjectType:          constants.ObjectTypeScanningProductionStepInfo,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPossibleInitStepsRequest) (*apiresource.List[apiresource.ScanningProductionStepInfo], *apierror.APIError) {
			return svc.(BatchSvc).GetPossibleInitSteps
		},
	})
}
