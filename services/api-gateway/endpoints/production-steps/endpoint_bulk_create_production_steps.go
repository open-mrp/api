package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// BulkCreateConsumptionInput represents a consumption input resolved by SKU.
type BulkCreateConsumptionInput struct {
	// The item SKU for the consumed material.
	SKU string `json:"sku" validate:"required"`
	// The consumption quantity measure.
	Measure float64 `json:"measure" validate:"required,gt=0"`
	// Optional instructions for this consumption.
	Instructions *string `json:"instructions,omitempty"`
}

// BulkCreateProductionOutputInput represents a production output input resolved by SKU.
type BulkCreateProductionOutputInput struct {
	// The item SKU for the produced item.
	SKU string `json:"sku" validate:"required"`
	// The production quantity measure.
	Measure float64 `json:"measure" validate:"required,gt=0"`
}

// BulkCreateProductionStepInput represents a single production step to create.
type BulkCreateProductionStepInput struct {
	// The name of the production step.
	Name string `json:"name" validate:"required"`
	// The consumptions for this step.
	Consumptions []BulkCreateConsumptionInput `json:"consumptions"`
	// The production outputs for this step. At least one is required.
	Productions []BulkCreateProductionOutputInput `json:"productions" validate:"required,min=1"`
	// The labor rate (dollars per hour).
	LaborRate float64 `json:"labor_rate" validate:"required,gt=0"`
	// The labor time value.
	LaborTime float64 `json:"labor_time" validate:"required,gt=0"`
	// The unit abbreviation for labor time (default: "hr"). Must be one of: hr, minute, second, day.
	LaborTimeUnit *string `json:"labor_time_unit,omitempty"`
	// The overhead rate (dollars per hour).
	OverheadRate float64 `json:"overhead_rate" validate:"required,gt=0"`
	// The allowances factor (default: 0).
	Allowances *float64 `json:"allowances,omitempty"`
	// The leveling factor (default: 0).
	LevelingFactor *float64 `json:"leveling_factor,omitempty"`
	// The scanning station name to associate (resolved by name).
	Station *string `json:"station,omitempty"`
}

// BulkCreateProductionStepsRequest is the request to create multiple production steps.
type BulkCreateProductionStepsRequest struct {
	// The production steps to create.
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

type BulkCreateProductionStepsEndpoint struct{}

func (e *BulkCreateProductionStepsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkCreateProductionStepsRequest, *apiresource.BulkCreateProductionStepsResponse] {
	return &apiendpoint.APIEndpoint[*BulkCreateProductionStepsRequest, *apiresource.BulkCreateProductionStepsResponse]{
		Title:             "Bulk Create Production Steps",
		Description:       "Creates multiple production steps in a single bulk operation.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/production-steps/actions/bulk-create",
		ContentType:       "application/json",
		Request:           &BulkCreateProductionStepsRequest{},
		Response:          &apiresource.BulkCreateProductionStepsResponse{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkCreateProductionStepsRequest) (*apiresource.BulkCreateProductionStepsResponse, *apierror.APIError) {
			return svc.(ProductionStepSvc).BulkCreateProductionSteps
		},
	}
}
