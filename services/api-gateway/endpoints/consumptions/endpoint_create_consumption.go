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

// Request to create a consumption.
type CreateConsumptionRequest struct {
	// Production step ID.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// Item ID.
	ItemID string `json:"item_id" validate:"required"`
	// Consumed quantity value.
	QuantityValue string `json:"quantity_value" validate:"required"`
	// Consumed quantity unit ID.
	QuantityUnitID string `json:"quantity_unit_id" validate:"required"`
	// Waste quantity value.
	WasteQuantityValue string `json:"waste_quantity_value" validate:"required"`
	// Waste quantity unit ID.
	WasteQuantityUnitID string `json:"waste_quantity_unit_id" validate:"required"`
	// Instructions for how this material is consumed.
	Instructions *string `json:"instructions,omitempty"`
}

var sampleCreateConsumptionRequest = &CreateConsumptionRequest{
	ItemID:              apiresource.SampleItemID,
	QuantityValue:       "10.000000000000000000000000000000",
	QuantityUnitID:      apiresource.SampleUnitID,
	WasteQuantityValue:  "0.500000000000000000000000000000",
	WasteQuantityUnitID: apiresource.SampleUnitID,
	Instructions:        new("Mix with water before adding"),
}

func (*CreateConsumptionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateConsumptionRequest)
}

// Creates a consumption within a production step.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
			return svc.(ConsumptionSvc).CreateConsumption
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeConsumption,
			Fields:     []string{"consumed_item"},
		}),
	})
}
