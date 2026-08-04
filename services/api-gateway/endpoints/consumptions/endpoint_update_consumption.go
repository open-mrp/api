package consumptionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update a consumption.
type UpdateConsumptionRequest struct {
	// Production step ID.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// Consumption ID.
	ConsumptionID string `path:"id" validate:"required"`
	// ID of the item to consume.
	//
	// Changing the item disconnects any production-flow link based on the previous item and re-links the flow using the new item.
	ItemID field.Optional[string] `json:"item_id,omitzero" validate:"omitempty"`
	// Amount of the item consumed, as a decimal string.
	//
	// The consumed quantity only changes when this and `quantity_unit_id` are sent together; sending either one alone leaves it untouched.
	QuantityValue field.Optional[string] `json:"quantity_value,omitzero"`
	// ID of the unit of measure for `quantity_value`.
	//
	// Send it together with `quantity_value`, even when the unit is not changing.
	QuantityUnitID field.Optional[string] `json:"quantity_unit_id,omitzero" validate:"omitempty"`
	// Amount of the item expected to be lost as waste, as a decimal string.
	//
	// The waste quantity only changes when this and `waste_quantity_unit_id` are sent together; sending either one alone leaves it untouched.
	WasteQuantityValue field.Optional[string] `json:"waste_quantity_value,omitzero"`
	// ID of the unit of measure for `waste_quantity_value`.
	//
	// Send it together with `waste_quantity_value`, even when the unit is not changing.
	WasteQuantityUnitID field.Optional[string] `json:"waste_quantity_unit_id,omitzero" validate:"omitempty"`
	// Instructions for how this material is consumed.
	Instructions field.Optional[string] `json:"instructions,omitzero"`
}

var sampleUpdateConsumptionRequest = &UpdateConsumptionRequest{
	QuantityValue:  field.Some("20.000000000000000000000000000000"),
	QuantityUnitID: field.Some(apiresource.SampleUnitID),
}

func (*UpdateConsumptionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateConsumptionRequest)
}

// Updates a production step's material input.
//
// Omitted fields are left unchanged. Each quantity is only rewritten when its value and unit are sent together, and changing the consumed item recomputes the production flow around the step.
type UpdateConsumptionEndpoint struct{}

func (e *UpdateConsumptionEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateConsumptionRequest, *apiresource.Consumption] {
	return (&apiendpoint.APIEndpoint[*UpdateConsumptionRequest, *apiresource.Consumption]{
		Title:             "Update Consumption",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/production-steps/{production_step_id}/consumptions/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeConsumption,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSteps, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateConsumptionRequest) (*apiresource.Consumption, *apierror.APIError) {
			return svc.(ConsumptionSvc).UpdateConsumption
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeConsumption,
			Fields:     []string{"consumed_item"},
		}),
	})
}
