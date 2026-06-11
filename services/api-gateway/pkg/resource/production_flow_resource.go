package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// ProductionFlow is the production flow graph for an item.
type ProductionFlow struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_flow"`
	// Steps in the production flow graph.
	//
	// Expandable via include[]=steps.
	Steps *List[ProductionFlowStep] `json:"steps" expandable:"true"`
}

// ProductionFlowStep is a step in the production flow.
type ProductionFlowStep struct {
	// Production step ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_step"`
	// Production step name.
	Name string `json:"name" validate:"required"`
	// Notes.
	Notes *string `json:"notes"`
	// Production output for this step.
	//
	// Expandable via include[]=steps.production.
	Production *ProductionFlowProduction `json:"production" expandable:"true"`
	// Consumptions (inputs) for this step.
	//
	// Expandable via include[]=steps.consumptions.
	Consumptions *List[ProductionFlowConsumption] `json:"consumptions" expandable:"true"`
	// Steps that feed into this step.
	//
	// Expandable via include[]=steps.in_steps.
	InSteps *List[ProductionStep] `json:"in_steps" expandable:"true"`
	// Steps that this step feeds into.
	//
	// Expandable via include[]=steps.out_steps.
	OutSteps *List[ProductionStep] `json:"out_steps" expandable:"true"`
	// Machines assigned to this step.
	//
	// Expandable via include[]=steps.machines.
	Machines *List[Machine] `json:"machines" expandable:"true"`
	// Department for this step.
	//
	// Expandable via include[]=steps.department.
	Department *Department `json:"department" expandable:"true"`
	// Scanning station, if assigned.
	//
	// Expandable via include[]=steps.scanning_station.
	ScanningStation *ScanningStation `json:"scanning_station" expandable:"true"`
	// Leveling factor as a decimal string.
	LevelingFactor string `json:"leveling_factor" validate:"required" format:"decimal"`
	// Allowances as a decimal string.
	Allowances string `json:"allowances" validate:"required" format:"decimal"`
	// Cost of labor for this step, expressed as a rate (e.g. currency per unit of output).
	LaborRate *Rate `json:"labor_rate"`
	// Labor duration for this step, expressed as a rate (e.g. time per unit of output).
	LaborTime *Rate `json:"labor_time"`
	// Overhead cost for this step, expressed as a rate (e.g. currency per unit of output).
	OverheadRate *Rate `json:"overhead_rate"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// ProductionFlowProduction is the production output of a flow step.
type ProductionFlowProduction struct {
	// Production record ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production"`
	// Produced item.
	//
	// Expandable via include[]=produced_item.
	ProducedItem *Item `json:"produced_item" expandable:"true"`
	// Produced quantity.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// ProductionFlowConsumption is a consumption input of a flow step.
type ProductionFlowConsumption struct {
	// Consumption record ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=consumption"`
	// Consumed item.
	//
	// Expandable via include[]=consumed_item.
	ConsumedItem *Item `json:"consumed_item" expandable:"true"`
	// Quantity of the item consumed by this step.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// Quantity of the consumed item expected to be lost as scrap or waste.
	WasteQuantity *Quantity `json:"waste_quantity" validate:"required"`
	// Instructions for how this material is consumed.
	Instructions *string `json:"instructions"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// --- Sample data ---

var sampleFlowInstructions = "Mix with water before adding"

var SampleProductionFlowProduction = &ProductionFlowProduction{
	ID:           SampleProductionID,
	Object:       constants.ObjectTypeProduction,
	ProducedItem: SampleItem,
	Quantity:     SampleQuantity,
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

var SampleProductionFlowConsumption = ProductionFlowConsumption{
	ID:            SampleConsumptionID,
	Object:        constants.ObjectTypeConsumption,
	ConsumedItem:  SampleItem,
	Quantity:      SampleQuantity,
	WasteQuantity: SampleQuantity,
	Instructions:  &sampleFlowInstructions,
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
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
	CreatedAt:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:      timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
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
