package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// ProductionFlow represents the production flow graph for an item.
type ProductionFlow struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_flow"`
	// The steps in the production flow graph.
	Steps []ProductionFlowStep `json:"steps" validate:"required"`
}

// ProductionFlowStep represents a single step in the production flow.
type ProductionFlowStep struct {
	// The unique identifier for the production step.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_step"`
	// The production step name.
	Name string `json:"name" validate:"required"`
	// The production output for this step.
	Production *ProductionFlowProduction `json:"production" validate:"required"`
	// The consumptions (inputs) for this step.
	Consumptions []ProductionFlowConsumption `json:"consumptions" validate:"required"`
	// The steps that feed into this step.
	InSteps []ProductionFlowStepRef `json:"in_steps" validate:"required"`
	// The steps that this step feeds into.
	OutSteps []ProductionFlowStepRef `json:"out_steps" validate:"required"`
	// The scanning station, if assigned.
	ScanningStation *ProductionFlowStepRef `json:"scanning_station"`
	// The leveling factor as a decimal string.
	LevelingFactor string `json:"leveling_factor" validate:"required" format:"decimal"`
	// The allowances as a decimal string.
	Allowances string `json:"allowances" validate:"required" format:"decimal"`
	// The labor rate for this step.
	LaborRate *Rate `json:"labor_rate"`
	// The labor time for this step.
	LaborTime *Rate `json:"labor_time"`
	// The overhead rate for this step.
	OverheadRate *Rate `json:"overhead_rate"`
}

// ProductionFlowProduction represents the production output of a step.
type ProductionFlowProduction struct {
	// The unique identifier for the production record.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production"`
	// The produced item.
	Item *ProductionFlowItemRef `json:"item" validate:"required"`
	// The produced quantity.
	Quantity *Quantity `json:"quantity" validate:"required"`
}

// ProductionFlowConsumption represents a consumption input of a step.
type ProductionFlowConsumption struct {
	// The unique identifier for the consumption record.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=consumption"`
	// The consumed item.
	Item *ProductionFlowItemRef `json:"item" validate:"required"`
	// The consumed quantity.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// The waste quantity.
	WasteQuantity *Quantity `json:"waste_quantity" validate:"required"`
	// Optional instructions for this consumption.
	Instructions *string `json:"instructions"`
}

// ProductionFlowStepRef is a lightweight reference to a production step.
type ProductionFlowStepRef struct {
	// The unique identifier.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required"`
}

// ProductionFlowItemRef is a lightweight reference to an item.
type ProductionFlowItemRef struct {
	// The unique identifier.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// The item SKU.
	SKU string `json:"sku" validate:"required"`
}

// --- Sample data ---

var sampleFlowInstructions = "Mix with water before adding"

var SampleProductionFlowStepRef = &ProductionFlowStepRef{
	ID:     SampleProductionStepID,
	Object: constants.ObjectTypeProductionStep,
}

var SampleProductionFlowItemRef = &ProductionFlowItemRef{
	ID:     SampleItemID,
	Object: constants.ObjectTypeItem,
	SKU:    SampleItemSKU,
}

var SampleProductionFlowProduction = &ProductionFlowProduction{
	ID:       "pn_01jm4r6700f8nwq3v5hx2d9ktp",
	Object:   constants.ObjectTypeProduction,
	Item:     SampleProductionFlowItemRef,
	Quantity: SampleQuantity,
}

var SampleProductionFlowConsumption = ProductionFlowConsumption{
	ID:            SampleConsumptionID,
	Object:        constants.ObjectTypeConsumption,
	Item:          SampleProductionFlowItemRef,
	Quantity:      SampleQuantity,
	WasteQuantity: SampleQuantity,
	Instructions:  &sampleFlowInstructions,
}

var SampleProductionFlowStep = ProductionFlowStep{
	ID:             SampleProductionStepID,
	Object:         constants.ObjectTypeProductionStep,
	Name:           "Final Assembly",
	Production:     SampleProductionFlowProduction,
	Consumptions:   []ProductionFlowConsumption{SampleProductionFlowConsumption},
	InSteps:        []ProductionFlowStepRef{*SampleProductionFlowStepRef},
	OutSteps:       []ProductionFlowStepRef{},
	LevelingFactor: "1.000000000000000000000000000000",
	Allowances:     "0.000000000000000000000000000000",
	LaborRate:      SampleRate,
	LaborTime:      SampleRate,
	OverheadRate:   SampleRate,
}

var SampleProductionFlow = &ProductionFlow{
	Object: constants.ObjectTypeProductionFlow,
	Steps:  []ProductionFlowStep{SampleProductionFlowStep},
}

func (*ProductionFlow) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionFlow)
}

func (*ProductionFlowStep) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&SampleProductionFlowStep)
}

func (*ProductionFlowProduction) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionFlowProduction)
}

func (*ProductionFlowConsumption) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&SampleProductionFlowConsumption)
}

func (*ProductionFlowStepRef) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionFlowStepRef)
}

func (*ProductionFlowItemRef) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionFlowItemRef)
}
