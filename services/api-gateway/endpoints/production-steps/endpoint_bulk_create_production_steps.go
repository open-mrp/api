package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Consumption input resolved by SKU.
type BulkCreateConsumptionInput struct {
	// SKU of the consumed material.
	SKU string `json:"sku" validate:"required"`
	// Quantity consumed, in the item's base unit.
	Measure float64 `json:"measure" validate:"required,gt=0"`
	// Instructions for this consumption.
	Instructions field.Optional[string] `json:"instructions,omitzero"`
}

// Production output input resolved by SKU.
type BulkCreateProductionOutputInput struct {
	// SKU of the produced item.
	SKU string `json:"sku" validate:"required"`
	// Quantity produced, in the item's base unit.
	Measure float64 `json:"measure" validate:"required,gt=0"`
}

// Production step input for bulk creation.
type BulkCreateProductionStepInput struct {
	// Display name of the production step.
	//
	// Used to match existing steps: if a step with this name already exists in the account, that step is updated in place instead of creating a new one.
	Name string `json:"name" validate:"required"`
	// Materials consumed by the step, resolved by SKU.
	Consumptions []BulkCreateConsumptionInput `json:"consumptions"`
	// Items produced by the step, resolved by SKU.
	//
	// At least one is required.
	Productions []BulkCreateProductionOutputInput `json:"productions" validate:"required,min=1"`
	// Labor rate in dollars per hour.
	LaborRate float64 `json:"labor_rate" validate:"required,gt=0"`
	// Labor time required per unit of output.
	//
	// Stored as `labor_time_unit` per one base unit of the first production output.
	LaborTime float64 `json:"labor_time" validate:"required,gt=0"`
	// Labor time unit abbreviation.
	//
	// Accepted values (case-insensitive): `hr`, `minute`, `min`, `second`, `sec`, `day`. Defaults to `hr`.
	LaborTimeUnit field.Optional[string] `json:"labor_time_unit,omitzero"`
	// Overhead rate in dollars per hour.
	OverheadRate float64 `json:"overhead_rate" validate:"required,gt=0"`
	// Allowance correction factor applied to labor time in cost calculations.
	//
	// Defaults to `0`.
	Allowances field.Optional[float64] `json:"allowances,omitzero"`
	// Leveling correction factor applied to labor time in cost calculations.
	//
	// Defaults to `0`.
	LevelingFactor field.Optional[float64] `json:"leveling_factor,omitzero"`
	// Name of an existing scanning station to assign to the step.
	//
	// Resolved by exact name; rows referencing an unknown station are skipped.
	Station field.Optional[string] `json:"station,omitzero"`
}

// Request to bulk create production steps.
type BulkCreateProductionStepsRequest struct {
	// Production steps to create or update.
	Steps []BulkCreateProductionStepInput `json:"steps" validate:"required"`
}

var sampleBulkCreateProductionStepsRequest = &BulkCreateProductionStepsRequest{
	Steps: []BulkCreateProductionStepInput{
		{
			Name:         "Mixing",
			LaborRate:    25.00,
			LaborTime:    1.5,
			OverheadRate: 15.00,
			Productions: []BulkCreateProductionOutputInput{
				{
					SKU:     apiresource.SampleItemSKU,
					Measure: 100,
				},
			},
			Consumptions: []BulkCreateConsumptionInput{
				{
					SKU:     "RAW-FLOUR-001",
					Measure: 50,
				},
			},
		},
	},
}

func (*BulkCreateProductionStepsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkCreateProductionStepsRequest)
}

// Creates or updates multiple production steps in a single request.
//
// Rows are matched to existing steps by name: matches are updated in place (replacing their production outputs and consumptions) and the rest are created. Each row succeeds or fails independently; failures are reported per row in the response instead of failing the whole request.
type BulkCreateProductionStepsEndpoint struct{}

func (e *BulkCreateProductionStepsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkCreateProductionStepsRequest, *apiresource.BulkCreateProductionStepsResponse] {
	return (&apiendpoint.APIEndpoint[*BulkCreateProductionStepsRequest, *apiresource.BulkCreateProductionStepsResponse]{
		Title:             "Bulk Create Production Steps",
		Method:            http.MethodPost,
		Route:             "/v1/operations/production-steps/actions/bulk-create",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkCreateProductionStepsRequest) (*apiresource.BulkCreateProductionStepsResponse, *apierror.APIError) {
			return svc.(ProductionStepSvc).BulkCreateProductionSteps
		},
	})
}
