package consumptionep

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

// Request to create a consumption.
type CreateConsumptionRequest struct {
	// ID of the production step that consumes the material.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// ID of the item to consume.
	ItemID string `json:"item_id" validate:"required"`
	// Amount of the item consumed, as a decimal string.
	QuantityValue string `json:"quantity_value" validate:"required"`
	// ID of the unit of measure for `quantity_value`.
	QuantityUnitID string `json:"quantity_unit_id" validate:"required"`
	// Amount of the item lost as waste, as a decimal string, tracked separately from the consumed quantity.
	WasteQuantityValue string `json:"waste_quantity_value" validate:"required"`
	// ID of the unit of measure for `waste_quantity_value`.
	WasteQuantityUnitID string `json:"waste_quantity_unit_id" validate:"required"`
	// Instructions for how this material is consumed.
	Instructions field.Optional[string] `json:"instructions,omitzero"`
}

var sampleCreateConsumptionRequest = &CreateConsumptionRequest{
	ItemID:              apiresource.SampleItemID,
	QuantityValue:       "10.000000000000000000000000000000",
	QuantityUnitID:      apiresource.SampleUnitID,
	WasteQuantityValue:  "0.500000000000000000000000000000",
	WasteQuantityUnitID: apiresource.SampleUnitID,
	Instructions:        field.Some("Mix with water before adding"),
}

func (*CreateConsumptionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateConsumptionRequest)
}

// Creates a consumption within a production step.
//
// Adding a consumption recomputes the production flow: if another production step produces the consumed item, the two steps are linked upstream/downstream automatically.
type CreateConsumptionEndpoint struct{}

func (e *CreateConsumptionEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateConsumptionRequest, *apiresource.Consumption] {
	return (&apiendpoint.APIEndpoint[*CreateConsumptionRequest, *apiresource.Consumption]{
		Title:             "Create Consumption",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-steps/{production_step_id}/consumptions",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeConsumption,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
			return svc.(ConsumptionSvc).CreateConsumption
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeConsumption,
			Fields:     []string{"consumed_item"},
		}),
	})
}
