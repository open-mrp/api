package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to update a production step.
type UpdateProductionStepRequest struct {
	// Production step ID.
	ProductionStepID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" validate:"omitempty,max=255"`
	// Leveling factor as a decimal string.
	LevelingFactor *string `json:"leveling_factor,omitempty"`
	// Allowances as a decimal string.
	Allowances *string `json:"allowances,omitempty"`
	// Scanning station ID.
	ScanningStationID *string `json:"scanning_station_id,omitempty" validate:"omitempty"`
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

// Partially updates a production step.
type UpdateProductionStepEndpoint struct{}

func (e *UpdateProductionStepEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductionStepRequest, *apiresource.ProductionStep] {
	return (&apiendpoint.APIEndpoint[*UpdateProductionStepRequest, *apiresource.ProductionStep]{
		Title:             "Update Production Step",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-steps/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionStep,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError) {
			return svc.(ProductionStepSvc).UpdateProductionStep
		},
	})
}
