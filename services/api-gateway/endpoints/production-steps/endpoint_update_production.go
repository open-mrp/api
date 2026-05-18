package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to update a production output.
type UpdateProductionRequest struct {
	// Production step ID.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// Production ID.
	ProductionID string `path:"id" validate:"required"`
	// Item ID.
	ItemID *string `json:"item_id,omitempty" nullable:"false" validate:"omitempty"`
	// Quantity value as a decimal string.
	QuantityValue *string `json:"quantity_value,omitempty" nullable:"false"`
	// Quantity unit ID.
	QuantityUnitID *string `json:"quantity_unit_id,omitempty" nullable:"false" validate:"omitempty"`
}

var sampleUpdateProductionItemID = apiresource.SampleItemID
var sampleUpdateProductionQuantityValue = "500"
var sampleUpdateProductionQuantityUnitID = apiresource.SampleUnitID
var sampleUpdateProductionRequest = &UpdateProductionRequest{
	ItemID:         &sampleUpdateProductionItemID,
	QuantityValue:  &sampleUpdateProductionQuantityValue,
	QuantityUnitID: &sampleUpdateProductionQuantityUnitID,
}

func (*UpdateProductionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductionRequest)
}

// Partially updates a production output within a production step.
type UpdateProductionEndpoint struct{}

func (e *UpdateProductionEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductionRequest, *apiresource.ProductionOutput] {
	return (&apiendpoint.APIEndpoint[*UpdateProductionRequest, *apiresource.ProductionOutput]{
		Title:             "Update Production",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-steps/{production_step_id}/productions/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductionRequest) (*apiresource.ProductionOutput, *apierror.APIError) {
			return svc.(ProductionStepSvc).UpdateProduction
		},
	})
}
