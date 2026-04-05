package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateProductionStepRequest is the request to update a production step.
type UpdateProductionStepRequest struct {
	// The ID of the production step to update.
	ProductionStepID string `path:"id" validate:"required"`
	// The new name.
	Name *string `json:"name,omitempty"`
	// The new leveling factor as a decimal string.
	LevelingFactor *string `json:"leveling_factor,omitempty"`
	// The new allowances as a decimal string.
	Allowances *string `json:"allowances,omitempty"`
	// The new scanning station ID.
	ScanningStationID *string `json:"scanning_station_id,omitempty" nullable:"true"`
}

var sampleUpdateProductionStepName = "Assembly Step A"
var sampleUpdateProductionStepLevelingFactor = "1.15"
var sampleUpdateProductionStepScanningStationID = apiresource.SampleScanningStationID
var sampleUpdateProductionStepRequest = &UpdateProductionStepRequest{
	Name:              &sampleUpdateProductionStepName,
	LevelingFactor:    &sampleUpdateProductionStepLevelingFactor,
	ScanningStationID: &sampleUpdateProductionStepScanningStationID,
}

func (*UpdateProductionStepRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductionStepRequest)
}

type UpdateProductionStepEndpoint struct{}

func (e *UpdateProductionStepEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductionStepRequest, *apiresource.ProductionStep] {
	return &apiendpoint.APIEndpoint[*UpdateProductionStepRequest, *apiresource.ProductionStep]{
		Title:             "Update Production Step",
		Description:       "Partially updates a production step.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/production-steps/{id}",
		Request:           &UpdateProductionStepRequest{},
		Response:          &apiresource.ProductionStep{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError) {
			return svc.(ProductionStepSvc).UpdateProductionStep
		},
	}
}
