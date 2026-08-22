package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// A material the step consumes, matched to an existing item by SKU.
//
// Materials added this way record no expected waste.
type BulkCreateConsumptionInput struct {
	// SKU of the consumed material.
	SKU string `json:"sku" validate:"required"`
	// Quantity consumed, in the item's base unit.
	Measure float64 `json:"measure" validate:"required,gt=0"`
	// Instructions for this consumption.
	Instructions field.Optional[string] `json:"instructions,omitzero"`
}

// An item the step produces, matched to an existing item by SKU.
type BulkCreateProductionOutputInput struct {
	// SKU of the produced item.
	SKU string `json:"sku" validate:"required"`
	// Quantity produced, in the item's base unit.
	Measure float64 `json:"measure" validate:"required,gt=0"`
}

// A production step to create or update.
type BulkCreateProductionStepInput struct {
	// Display name of the production step.
	//
	// Used to match existing steps: if a step with this name already exists in the account, that step is updated in place instead of creating a new one.
	Name string `json:"name" validate:"required"`
	// Materials consumed by the step, matched by SKU.
	Consumptions []BulkCreateConsumptionInput `json:"consumptions"`
	// Items produced by the step, matched by SKU.
	Productions []BulkCreateProductionOutputInput `json:"productions" validate:"required,min=1"`
	// Labor rate in dollars per hour.
	LaborRate float64 `json:"labor_rate" validate:"required,gt=0"`
	// Labor time required per unit of output.
	//
	// Recorded as `labor_time_unit` per one base unit of the first item in `productions`.
	LaborTime float64 `json:"labor_time" validate:"required,gt=0"`
	// Unit that `labor_time` is expressed in.
	//
	// One of `hr`, `min`, `minute`, `sec`, `second`, or `day`; a row naming anything else is skipped. Labor time is read as hours when this is omitted.
	LaborTimeUnit field.Optional[string] `json:"labor_time_unit,omitzero"`
	// Overhead rate in dollars per hour.
	OverheadRate float64 `json:"overhead_rate" validate:"required,gt=0"`
	// Allowance correction factor applied to labor time in cost calculations.
	//
	// When omitted, no allowance adjustment is applied.
	Allowances field.Optional[float64] `json:"allowances,omitzero"`
	// Leveling correction factor applied to labor time in cost calculations.
	//
	// When omitted, no leveling adjustment is applied.
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
// Rows are matched to existing steps by name. A matched step has its production outputs and consumptions replaced, and its allowances, leveling factor, and scanning station overwritten from the row — omitting any of those three resets it — while its labor rate, labor time, and overhead rate keep the values they already had. Unmatched rows are created in full, and every created or updated step is reconnected into the production flow graph from the items it produces and consumes.
//
// Each row succeeds or fails independently: a row referencing an unknown SKU, scanning station, or labor time unit is skipped and reported in the response instead of failing the whole request.
type BulkCreateProductionStepsEndpoint struct{}

func (e *BulkCreateProductionStepsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkCreateProductionStepsRequest, *apiresource.BulkCreateProductionStepsResponse] {
	return (&apiendpoint.APIEndpoint[*BulkCreateProductionStepsRequest, *apiresource.BulkCreateProductionStepsResponse]{
		Title:               "Bulk Create Production Steps",
		Method:              http.MethodPost,
		Route:               "/v1/operations/production-steps/actions/bulk-create",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductionSteps, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkCreateProductionStepsRequest) (*apiresource.BulkCreateProductionStepsResponse, *apierror.APIError) {
			return svc.(ProductionStepSvc).BulkCreateProductionSteps
		},
	})
}
