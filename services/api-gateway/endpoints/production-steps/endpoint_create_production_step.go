package productionstepep

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

// Request to create a production step.
type CreateProductionStepRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Notes.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// Leveling factor as a decimal string.
	LevelingFactor string `json:"leveling_factor" validate:"required"`
	// Allowances as a decimal string.
	Allowances string `json:"allowances" validate:"required"`
	// Scanning station ID.
	ScanningStationID field.Optional[string] `json:"scanning_station_id,omitzero" validate:"omitempty"`
	// Department ID.
	DepartmentID field.Optional[string] `json:"department_id,omitzero" validate:"omitempty"`
	// Labor rate configuration.
	LaborRate CreateRateInput `json:"labor_rate" validate:"required"`
	// Labor time configuration.
	LaborTime CreateRateInput `json:"labor_time" validate:"required"`
	// Overhead rate configuration.
	OverheadRate CreateRateInput `json:"overhead_rate" validate:"required"`
	// Production output configuration.
	Production CreateProductionInput `json:"production" validate:"required"`
	// Consumptions.
	Consumptions []CreateConsumptionInput `json:"consumptions"`
}

// Rate configuration input.
type CreateRateInput struct {
	// Value as a decimal string.
	Value string `json:"value" validate:"required"`
	// Numerator unit ID.
	NumeratorUnitID string `json:"numerator_unit_id" validate:"required"`
	// Denominator unit ID.
	DenominatorUnitID string `json:"denominator_unit_id" validate:"required"`
}

// Production output input.
type CreateProductionInput struct {
	// Item ID.
	ItemID string `json:"item_id" validate:"required"`
	// Quantity value as a decimal string.
	QuantityValue string `json:"quantity_value" validate:"required"`
	// Quantity unit ID.
	QuantityUnitID string `json:"quantity_unit_id" validate:"required"`
}

// Consumption input for a production step.
type CreateConsumptionInput struct {
	// Item ID.
	ItemID string `json:"item_id" validate:"required"`
	// Quantity value as a decimal string.
	QuantityValue string `json:"quantity_value" validate:"required"`
	// Quantity unit ID.
	QuantityUnitID string `json:"quantity_unit_id" validate:"required"`
	// Waste quantity value as a decimal string.
	WasteQuantityValue string `json:"waste_quantity_value" validate:"required"`
	// Waste quantity unit ID.
	WasteQuantityUnitID string `json:"waste_quantity_unit_id" validate:"required"`
	// Instructions for how this material is consumed.
	Instructions field.Optional[string] `json:"instructions,omitzero"`
}

var sampleCreateProductionStepScanningStationID = apiresource.SampleScanningStationID
var sampleCreateProductionStepRequest = &CreateProductionStepRequest{
	Name:              "Mixing",
	LevelingFactor:    "1.10",
	Allowances:        "0.05",
	ScanningStationID: field.Some(sampleCreateProductionStepScanningStationID),
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

// Creates a production step with production output, rates, and consumptions.
type CreateProductionStepEndpoint struct{}

func (e *CreateProductionStepEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductionStepRequest, *apiresource.ProductionStep] {
	return (&apiendpoint.APIEndpoint[*CreateProductionStepRequest, *apiresource.ProductionStep]{
		Title:             "Create Production Step",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-steps",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionStep,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError) {
			return svc.(ProductionStepSvc).CreateProductionStep
		},
		LocationFunc: func(resp *apiresource.ProductionStep) string {
			return "/v1/operations/production-steps/" + resp.ID
		},
	})
}
