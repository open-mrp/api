package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreateProductionStepRequest is the request to create a new production step.
type CreateProductionStepRequest struct {
	// The name of the production step.
	Name string `json:"name" validate:"required,max=255"`
	// Optional notes about the production step.
	Notes *string `json:"notes,omitempty"`
	// The leveling factor as a decimal string.
	LevelingFactor string `json:"leveling_factor" validate:"required"`
	// The allowances as a decimal string.
	Allowances string `json:"allowances" validate:"required"`
	// The scanning station ID.
	ScanningStationID *string `json:"scanning_station_id,omitempty" validate:"omitempty,max=191"`
	// The department ID.
	DepartmentID *string `json:"department_id,omitempty" validate:"omitempty,max=191"`
	// The labor rate configuration.
	LaborRate CreateRateInput `json:"labor_rate" validate:"required"`
	// The labor time configuration.
	LaborTime CreateRateInput `json:"labor_time" validate:"required"`
	// The overhead rate configuration.
	OverheadRate CreateRateInput `json:"overhead_rate" validate:"required"`
	// The production output configuration.
	Production CreateProductionInput `json:"production" validate:"required"`
	// The consumptions for this step.
	Consumptions []CreateConsumptionInput `json:"consumptions"`
}

// CreateRateInput holds the input for creating a rate.
type CreateRateInput struct {
	// The rate value as a decimal string.
	Value string `json:"value" validate:"required"`
	// The numerator unit ID.
	NumeratorUnitID string `json:"numerator_unit_id" validate:"required,max=191"`
	// The denominator unit ID.
	DenominatorUnitID string `json:"denominator_unit_id" validate:"required,max=191"`
}

// CreateProductionInput holds the input for creating a production output.
type CreateProductionInput struct {
	// The item ID to produce.
	ItemID string `json:"item_id" validate:"required,max=191"`
	// The quantity value as a decimal string.
	QuantityValue string `json:"quantity_value" validate:"required"`
	// The quantity unit ID.
	QuantityUnitID string `json:"quantity_unit_id" validate:"required,max=191"`
}

// CreateConsumptionInput holds the input for creating a consumption within a step.
type CreateConsumptionInput struct {
	// The item ID being consumed.
	ItemID string `json:"item_id" validate:"required,max=191"`
	// The quantity value as a decimal string.
	QuantityValue string `json:"quantity_value" validate:"required"`
	// The quantity unit ID.
	QuantityUnitID string `json:"quantity_unit_id" validate:"required,max=191"`
	// The waste quantity value as a decimal string.
	WasteQuantityValue string `json:"waste_quantity_value" validate:"required"`
	// The waste quantity unit ID.
	WasteQuantityUnitID string `json:"waste_quantity_unit_id" validate:"required,max=191"`
	// Optional instructions for how this material is consumed.
	Instructions *string `json:"instructions,omitempty"`
}

var sampleCreateProductionStepScanningStationID = apiresource.SampleScanningStationID
var sampleCreateProductionStepRequest = &CreateProductionStepRequest{
	Name:              "Mixing",
	LevelingFactor:    "1.10",
	Allowances:        "0.05",
	ScanningStationID: &sampleCreateProductionStepScanningStationID,
	LaborRate: CreateRateInput{
		Value:             "25.00",
		NumeratorUnitID:   apiresource.SampleUnitID,
		DenominatorUnitID: apiresource.SampleUnitID,
	},
	LaborTime: CreateRateInput{
		Value:             "1.5",
		NumeratorUnitID:   apiresource.SampleUnitID,
		DenominatorUnitID: apiresource.SampleUnitID,
	},
	OverheadRate: CreateRateInput{
		Value:             "15.00",
		NumeratorUnitID:   apiresource.SampleUnitID,
		DenominatorUnitID: apiresource.SampleUnitID,
	},
	Production: CreateProductionInput{
		ItemID:         apiresource.SampleItemID,
		QuantityValue:  "100",
		QuantityUnitID: apiresource.SampleUnitID,
	},
	Consumptions: []CreateConsumptionInput{
		{
			ItemID:              apiresource.SampleItemID,
			QuantityValue:       "50",
			QuantityUnitID:      apiresource.SampleUnitID,
			WasteQuantityValue:  "2",
			WasteQuantityUnitID: apiresource.SampleUnitID,
		},
	},
}

func (*CreateProductionStepRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateProductionStepRequest)
}

type CreateProductionStepEndpoint struct{}

func (e *CreateProductionStepEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductionStepRequest, *apiresource.ProductionStep] {
	return &apiendpoint.APIEndpoint[*CreateProductionStepRequest, *apiresource.ProductionStep]{
		Title:             "Create Production Step",
		Description:       "Creates a new production step with production output, rates, and consumptions.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/production-steps",
		Request:           &CreateProductionStepRequest{},
		Response:          &apiresource.ProductionStep{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError) {
			return svc.(ProductionStepSvc).CreateProductionStep
		},
	}
}
