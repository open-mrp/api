package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update a production output.
type UpdateProductionRequest struct {
	// Production step ID.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// Production ID.
	ProductionID string `path:"id" validate:"required"`
	// Item ID.
	ItemID field.Optional[string] `json:"item_id,omitzero" validate:"omitempty"`
	// Quantity value as a decimal string.
	QuantityValue field.Optional[string] `json:"quantity_value,omitzero"`
	// Quantity unit ID.
	QuantityUnitID field.Optional[string] `json:"quantity_unit_id,omitzero" validate:"omitempty"`
}

var sampleUpdateProductionItemID = apiresource.SampleItemID
var sampleUpdateProductionQuantityValue = "500"
var sampleUpdateProductionQuantityUnitID = apiresource.SampleUnitID
var sampleUpdateProductionRequest = &UpdateProductionRequest{
	ItemID:         field.Some(sampleUpdateProductionItemID),
	QuantityValue:  field.Some(sampleUpdateProductionQuantityValue),
	QuantityUnitID: field.Some(sampleUpdateProductionQuantityUnitID),
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
		ObjectType:        constants.ObjectTypeProduction,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductionRequest) (*apiresource.ProductionOutput, *apierror.APIError) {
			return svc.(ProductionStepSvc).UpdateProduction
		},
	})
}
