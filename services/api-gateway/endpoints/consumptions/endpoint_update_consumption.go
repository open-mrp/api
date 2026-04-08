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

// UpdateConsumptionRequest is the request to partially update a consumption.
type UpdateConsumptionRequest struct {
	// The ID of the production step.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// The ID of the consumption to update.
	ConsumptionID string `path:"id" validate:"required"`
	// The ID of the item being consumed.
	ItemID *string `json:"item_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The decimal value of the quantity consumed.
	QuantityValue *string `json:"quantity_value,omitempty"`
	// The unit ID for the quantity consumed.
	QuantityUnitID *string `json:"quantity_unit_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The decimal value of the waste quantity.
	WasteQuantityValue *string `json:"waste_quantity_value,omitempty"`
	// The unit ID for the waste quantity.
	WasteQuantityUnitID *string `json:"waste_quantity_unit_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// Optional instructions for how this material is consumed.
	Instructions *string `json:"instructions,omitempty"`
}

var sampleUpdateConsumptionRequest = &UpdateConsumptionRequest{
	QuantityValue:  new("20.000000000000000000000000000000"),
	QuantityUnitID: new(apiresource.SampleUnitID),
}

func (*UpdateConsumptionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateConsumptionRequest)
}

type UpdateConsumptionEndpoint struct{}

func (e *UpdateConsumptionEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateConsumptionRequest, *apiresource.Consumption] {
	return &apiendpoint.APIEndpoint[*UpdateConsumptionRequest, *apiresource.Consumption]{
		Title:             "Update Consumption",
		Description:       "Partially updates a consumption within a production step.",
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
	}
}
