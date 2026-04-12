package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateProductionRequest is the request to update a production output.
type UpdateProductionRequest struct {
	// The ID of the production step.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// The ID of the production to update.
	ProductionID string `path:"id" validate:"required"`
	// The new item ID.
	ItemID *string `json:"item_id,omitempty" nullable:"false" validate:"omitempty,max=191"`
	// The new quantity value as a decimal string.
	QuantityValue *string `json:"quantity_value,omitempty" nullable:"false"`
	// The new quantity unit ID.
	QuantityUnitID *string `json:"quantity_unit_id,omitempty" nullable:"false" validate:"omitempty,max=191"`
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

type UpdateProductionEndpoint struct{}

func (e *UpdateProductionEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductionRequest, *apiresource.ProductionOutput] {
	return &apiendpoint.APIEndpoint[*UpdateProductionRequest, *apiresource.ProductionOutput]{
		Title:             "Update Production",
		Description:       "Partially updates a production output within a production step.",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-steps/{production_step_id}/productions/{id}",
		Request:           &UpdateProductionRequest{},
		Response:          &apiresource.ProductionOutput{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductionRequest) (*apiresource.ProductionOutput, *apierror.APIError) {
			return svc.(ProductionStepSvc).UpdateProduction
		},
	}
}
