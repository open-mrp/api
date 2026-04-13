package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Consumption input resolved by SKU.
type BulkCreateConsumptionInput struct {
	// SKU of the consumed material.
	SKU string `json:"sku" validate:"required"`
	// Consumption quantity measure.
	Measure float64 `json:"measure" validate:"required,gt=0"`
	// Instructions for this consumption.
	Instructions *string `json:"instructions,omitempty"`
}

// Production output input resolved by SKU.
type BulkCreateProductionOutputInput struct {
	// SKU of the produced item.
	SKU string `json:"sku" validate:"required"`
	// Production quantity measure.
	Measure float64 `json:"measure" validate:"required,gt=0"`
}

// Production step input for bulk creation.
type BulkCreateProductionStepInput struct {
	// Display name.
	Name string `json:"name" validate:"required"`
	// Consumptions.
	Consumptions []BulkCreateConsumptionInput `json:"consumptions"`
	// Production outputs. At least one is required.
	Productions []BulkCreateProductionOutputInput `json:"productions" validate:"required,min=1"`
	// Labor rate in dollars per hour.
	LaborRate float64 `json:"labor_rate" validate:"required,gt=0"`
	// Labor time value.
	LaborTime float64 `json:"labor_time" validate:"required,gt=0"`
	// Labor time unit abbreviation (default: "hr"). One of: hr, minute, second, day.
	LaborTimeUnit *string `json:"labor_time_unit,omitempty"`
	// Overhead rate in dollars per hour.
	OverheadRate float64 `json:"overhead_rate" validate:"required,gt=0"`
	// Allowances factor (default: 0).
	Allowances *float64 `json:"allowances,omitempty"`
	// Leveling factor (default: 0).
	LevelingFactor *float64 `json:"leveling_factor,omitempty"`
	// Scanning station name, resolved by name.
	Station *string `json:"station,omitempty"`
}

// Request to bulk create production steps.
type BulkCreateProductionStepsRequest struct {
	// Production steps to create.
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
		Description:       "Creates multiple production steps in a single request.",
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
