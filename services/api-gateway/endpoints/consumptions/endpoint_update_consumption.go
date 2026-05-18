package consumptionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to partially update a consumption.
type UpdateConsumptionRequest struct {
	// Production step ID.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// Consumption ID.
	ConsumptionID string `path:"id" validate:"required"`
	// Item ID.
	ItemID *string `json:"item_id,omitempty" nullable:"false" validate:"omitempty"`
	// Consumed quantity value.
	QuantityValue *string `json:"quantity_value,omitempty" nullable:"false"`
	// Consumed quantity unit ID.
	QuantityUnitID *string `json:"quantity_unit_id,omitempty" nullable:"false" validate:"omitempty"`
	// Waste quantity value.
	WasteQuantityValue *string `json:"waste_quantity_value,omitempty" nullable:"false"`
	// Waste quantity unit ID.
	WasteQuantityUnitID *string `json:"waste_quantity_unit_id,omitempty" nullable:"false" validate:"omitempty"`
	// Instructions for how this material is consumed.
	Instructions *string `json:"instructions,omitempty" nullable:"true"`
}

var sampleUpdateConsumptionRequest = &UpdateConsumptionRequest{
	QuantityValue:  new("20.000000000000000000000000000000"),
	QuantityUnitID: new(apiresource.SampleUnitID),
}

func (*UpdateConsumptionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateConsumptionRequest)
}

// Partially updates a consumption within a production step.
type UpdateConsumptionEndpoint struct{}

func (e *UpdateConsumptionEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateConsumptionRequest, *apiresource.Consumption] {
	return (&apiendpoint.APIEndpoint[*UpdateConsumptionRequest, *apiresource.Consumption]{
		Title:             "Update Consumption",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/production-steps/{production_step_id}/consumptions/{id}",
		ContentType:       "application/json",
		Request:           &UpdateConsumptionRequest{},
		Response:          &apiresource.Consumption{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
			return svc.(ConsumptionSvc).UpdateConsumption
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeConsumption,
			Fields:     []string{"consumed_item"},
		}),
	}).WithDocSource(e)
}
