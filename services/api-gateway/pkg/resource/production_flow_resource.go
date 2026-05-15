package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// ProductionFlow is the production flow graph for an item.
type ProductionFlow struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_flow"`
	// Steps in the production flow graph.
	Steps *List[ProductionFlowStep] `json:"steps" validate:"required"`
}

// ProductionFlowStep is a step in the production flow.
type ProductionFlowStep struct {
	// Production step ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_step"`
	// Production step name.
	Name string `json:"name" validate:"required"`
	// Production output for this step. Expandable via include[]=steps.production.
	Production *ProductionFlowProduction `json:"production" expandable:"true"`
	// Consumptions (inputs) for this step. Expandable via include[]=steps.consumptions.
	Consumptions *List[ProductionFlowConsumption] `json:"consumptions" expandable:"true"`
	// Steps that feed into this step. Expandable via include[]=steps.in_steps.
	InSteps *List[ProductionStep] `json:"in_steps" expandable:"true"`
	// Steps that this step feeds into. Expandable via include[]=steps.out_steps.
	OutSteps *List[ProductionStep] `json:"out_steps" expandable:"true"`
	// Machines assigned to this step. Expandable via include[]=steps.machines.
	Machines *List[Machine] `json:"machines" expandable:"true"`
	// Department for this step. Expandable via include[]=steps.department.
	Department *Department `json:"department" expandable:"true"`
	// Scanning station, if assigned. Expandable via include[]=steps.scanning_station.
	ScanningStation *ScanningStation `json:"scanning_station" expandable:"true"`
	// Leveling factor as a decimal string.
	LevelingFactor string `json:"leveling_factor" validate:"required" format:"decimal"`
	// Allowances as a decimal string.
	Allowances string `json:"allowances" validate:"required" format:"decimal"`
	// Labor rate.
	LaborRate *Rate `json:"labor_rate"`
	// Labor time.
	LaborTime *Rate `json:"labor_time"`
	// Overhead rate.
	OverheadRate *Rate `json:"overhead_rate"`
}

// ProductionFlowProduction is the production output of a flow step.
type ProductionFlowProduction struct {
	// Production record ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production"`
	// Produced item. Expandable via include[]=produced_item.
	ProducedItem *Item `json:"produced_item" expandable:"true"`
	// Produced quantity.
	Quantity *Quantity `json:"quantity" validate:"required"`
}

// ProductionFlowConsumption is a consumption input of a flow step.
type ProductionFlowConsumption struct {
	// Consumption record ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=consumption"`
	// Consumed item. Expandable via include[]=consumed_item.
	ConsumedItem *Item `json:"consumed_item" expandable:"true"`
	// Consumed quantity.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// Waste quantity.
	WasteQuantity *Quantity `json:"waste_quantity" validate:"required"`
	// Consumption instructions.
	Instructions *string `json:"instructions"`
}

// --- Sample data ---

var sampleFlowInstructions = "Mix with water before adding"

var SampleProductionFlowProduction = &ProductionFlowProduction{
	ID:           "pn_01jm4r6700f8nwq3v5hx2d9ktp",
	Object:       constants.ObjectTypeProduction,
	ProducedItem: SampleItem,
	Quantity:     SampleQuantity,
}

var SampleProductionFlowConsumption = ProductionFlowConsumption{
	ID:            SampleConsumptionID,
	Object:        constants.ObjectTypeConsumption,
	ConsumedItem:  SampleItem,
	Quantity:      SampleQuantity,
	WasteQuantity: SampleQuantity,
	Instructions:  &sampleFlowInstructions,
}

var SampleProductionFlowStep = ProductionFlowStep{
	ID:             SampleProductionStepID,
	Object:         constants.ObjectTypeProductionStep,
	Name:           "Final Assembly",
	Production:     SampleProductionFlowProduction,
	Consumptions:   NewList([]ProductionFlowConsumption{SampleProductionFlowConsumption}, PageInfo{}),
	InSteps:        NewList([]ProductionStep{}, PageInfo{}),
	OutSteps:       NewList([]ProductionStep{}, PageInfo{}),
	Machines:       NewList([]Machine{*SampleMachine}, PageInfo{}),
	Department:     SampleDepartment,
	LevelingFactor: "1.000000000000000000000000000000",
	Allowances:     "0.000000000000000000000000000000",
	LaborRate:      SampleRate,
	LaborTime:      SampleRate,
	OverheadRate:   SampleRate,
}

var SampleProductionFlow = &ProductionFlow{
	Object: constants.ObjectTypeProductionFlow,
	Steps:  NewList([]ProductionFlowStep{SampleProductionFlowStep}, PageInfo{}),
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
