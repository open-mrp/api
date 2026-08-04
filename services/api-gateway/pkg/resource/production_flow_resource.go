package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// The production flow graph for an item.
//
// Contains the step(s) that produce the item, every upstream step that feeds them, and any connected downstream steps.
type ProductionFlow struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_flow"`
	// Steps in the production flow graph, with the item's producing step(s) listed first.
	Steps *List[ProductionFlowStep] `json:"steps" expandable:"true"`
}

// A stage of work within an item's production flow, with its output, material inputs, cost rates, and links to the steps around it.
type ProductionFlowStep struct {
	// Production step ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_step"`
	// Display name of the step.
	Name string `json:"name" validate:"required"`
	// Free-form notes about this step.
	Notes *string `json:"notes"`
	// The item and quantity this step produces.
	Production *ProductionFlowProduction `json:"production" expandable:"true"`
	// Materials this step consumes as inputs, with their quantities and expected waste.
	Consumptions *List[ProductionFlowConsumption] `json:"consumptions" expandable:"true"`
	// Steps that feed into this step.
	//
	// Restricted to steps that are themselves part of this flow.
	InSteps *List[ProductionStep] `json:"in_steps" expandable:"true"`
	// Steps that this step feeds into.
	//
	// Restricted to steps that are themselves part of this flow.
	OutSteps *List[ProductionStep] `json:"out_steps" expandable:"true"`
	// Machines assigned to this step.
	Machines *List[Machine] `json:"machines" expandable:"true"`
	// Department responsible for this step.
	Department *Department `json:"department" expandable:"true"`
	// Scanning station where this step's batches are scanned.
	ScanningStation *ScanningStation `json:"scanning_station" expandable:"true"`
	// Leveling correction factor applied to labor time in cost calculations, as a decimal string.
	//
	// Effective labor time per unit is `labor_time × (1 + leveling_factor) × (1 + allowances)`.
	LevelingFactor string `json:"leveling_factor" validate:"required" format:"decimal"`
	// Allowance correction factor applied to labor time in cost calculations, as a decimal string.
	//
	// Effective labor time per unit is `labor_time × (1 + leveling_factor) × (1 + allowances)`.
	Allowances string `json:"allowances" validate:"required" format:"decimal"`
	// Cost of labor for this step, expressed as a rate of currency per unit of time (e.g. `$` per `hr`).
	LaborRate *Rate `json:"labor_rate"`
	// Labor duration for this step, expressed as a rate (e.g. time per unit of output).
	LaborTime *Rate `json:"labor_time"`
	// Overhead cost for this step, expressed as a rate of currency per unit of time (e.g. `$` per `hr`).
	OverheadRate *Rate `json:"overhead_rate"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// The item and quantity produced by a step in the production flow.
type ProductionFlowProduction struct {
	// Production record ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production"`
	// Item produced by the step.
	ProducedItem *Item `json:"produced_item" expandable:"true"`
	// Quantity of the item this step produces.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// A material consumed by a step in the production flow, with its quantity and expected waste.
type ProductionFlowConsumption struct {
	// Consumption record ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=consumption"`
	// Item consumed by the step.
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

var sampleProductionFlowStepNotes = "Torque all fasteners to spec before releasing to the next step."

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
	ID:              SampleProductionStepID,
	Object:          constants.ObjectTypeProductionStep,
	Name:            "Final Assembly",
	Notes:           &sampleProductionFlowStepNotes,
	Production:      SampleProductionFlowProduction,
	Consumptions:    NewList([]ProductionFlowConsumption{SampleProductionFlowConsumption}, PageInfo{}),
	InSteps:         NewList([]ProductionStep{}, PageInfo{}),
	OutSteps:        NewList([]ProductionStep{}, PageInfo{}),
	Machines:        NewList([]Machine{*SampleMachine}, PageInfo{}),
	Department:      SampleDepartment,
	ScanningStation: SampleScanningStation,
	LevelingFactor:  "1.000000000000000000000000000000",
	Allowances:      "0.000000000000000000000000000000",
	LaborRate:       SampleRate,
	LaborTime:       SampleRate,
	OverheadRate:    SampleRate,
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
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
