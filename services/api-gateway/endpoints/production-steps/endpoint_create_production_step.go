package productionstepep

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

// Request to create a production step.
type CreateProductionStepRequest struct {
	// Display name of the step.
	//
	// Must be unique within the account.
	Name string `json:"name" validate:"required,max=255"`
	// Free-form notes about the step.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// Leveling correction factor applied to labor time in cost calculations, as a decimal string.
	//
	// Effective labor time per unit is `labor_time × (1 + leveling_factor) × (1 + allowances)`, so `0` applies no leveling correction.
	LevelingFactor string `json:"leveling_factor" validate:"required"`
	// Allowance correction factor applied to labor time in cost calculations, as a decimal string.
	//
	// Effective labor time per unit is `labor_time × (1 + leveling_factor) × (1 + allowances)`, so `0` applies no allowance.
	Allowances string `json:"allowances" validate:"required"`
	// Scanning station where batches at this step are scanned.
	ScanningStationID field.Optional[string] `json:"scanning_station_id,omitzero" validate:"omitempty"`
	// Department responsible for this step.
	DepartmentID field.Optional[string] `json:"department_id,omitzero" validate:"omitempty"`
	// Cost of labor for this step, expressed as a rate of currency per unit of time (e.g. `$` per `hr`).
	//
	// The numerator unit must be a currency and the denominator must not be.
	LaborRate CreateRateInput `json:"labor_rate" validate:"required"`
	// Labor duration for this step, expressed as a rate (e.g. time per unit of output).
	LaborTime CreateRateInput `json:"labor_time" validate:"required"`
	// Overhead cost for this step, expressed as a rate of currency per unit of time (e.g. `$` per `hr`).
	//
	// The numerator unit must be a currency and the denominator must not be.
	OverheadRate CreateRateInput `json:"overhead_rate" validate:"required"`
	// The item and quantity this step produces.
	Production CreateProductionInput `json:"production" validate:"required"`
	// Materials consumed by the step.
	Consumptions []CreateConsumptionInput `json:"consumptions,omitzero"`
}

// A rate, expressed as a value together with the units of its numerator and denominator (for example, `25.00` `$` per `hr`).
type CreateRateInput struct {
	// Value as a decimal string.
	Value string `json:"value" validate:"required"`
	// Unit of the rate's numerator.
	NumeratorUnitID string `json:"numerator_unit_id" validate:"required"`
	// Unit of the rate's denominator.
	DenominatorUnitID string `json:"denominator_unit_id" validate:"required"`
}

// The item and quantity a production step produces.
type CreateProductionInput struct {
	// Item the step produces.
	ItemID string `json:"item_id" validate:"required"`
	// Quantity value as a decimal string.
	QuantityValue string `json:"quantity_value" validate:"required"`
	// Unit for `quantity_value`.
	QuantityUnitID string `json:"quantity_unit_id" validate:"required"`
}

// A material a production step consumes, with its quantity and expected waste.
type CreateConsumptionInput struct {
	// Material the step consumes.
	ItemID string `json:"item_id" validate:"required"`
	// Quantity value as a decimal string.
	QuantityValue string `json:"quantity_value" validate:"required"`
	// Unit for `quantity_value`.
	QuantityUnitID string `json:"quantity_unit_id" validate:"required"`
	// Quantity expected to be lost as scrap or waste, as a decimal string.
	WasteQuantityValue string `json:"waste_quantity_value" validate:"required"`
	// Unit for `waste_quantity_value`.
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

// Creates a production step with its production output, cost rates, and consumptions.
//
// The step is automatically connected into the production flow graph based on the items it produces and consumes.
//
// Returns a conflict error if a production step with the same name already exists in the account.
type CreateProductionStepEndpoint struct{}

func (e *CreateProductionStepEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductionStepRequest, *apiresource.ProductionStep] {
	return (&apiendpoint.APIEndpoint[*CreateProductionStepRequest, *apiresource.ProductionStep]{
		Title:               "Create Production Step",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/production-steps",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductionSteps, Action: types.ActionCreate}},
		ObjectType:          constants.ObjectTypeProductionStep,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError) {
			return svc.(ProductionStepSvc).CreateProductionStep
		},
		LocationFunc: func(resp *apiresource.ProductionStep) string {
			return "/v1/operations/production-steps/" + resp.ID
		},
	})
}
