package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// A single stage of work in an item's production flow, with its output, material inputs, cost rates, and graph connections.
type ProductionStep struct {
	// Production step ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_step"`
	// Display name of the step.
	Name string `json:"name" validate:"required"`
	// Free-form notes about the step.
	Notes *string `json:"notes"`
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
	// The item and quantity this step produces.
	Production *ProductionOutput `json:"production" expandable:"true"`
	// Materials this step consumes as inputs, with their quantities and expected waste.
	Consumptions *List[Consumption] `json:"consumptions" expandable:"true"`
	// Machines assigned to this step.
	Machines *List[Machine] `json:"machines" expandable:"true"`
	// Scanning station where this step's batches are scanned.
	ScanningStation *ScanningStation `json:"scanning_station" expandable:"true"`
	// Steps that feed into this step.
	InSteps *List[ProductionStep] `json:"in_steps" expandable:"true"`
	// Steps that this step feeds into.
	OutSteps *List[ProductionStep] `json:"out_steps" expandable:"true"`
	// Department responsible for this step.
	Department *Department `json:"department" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleProductionStepNotes = "Torque all fasteners to spec before releasing to the next step."

var SampleProductionStep = &ProductionStep{
	ID:              SampleProductionStepID,
	Object:          constants.ObjectTypeProductionStep,
	Name:            "Final Assembly",
	Notes:           &sampleProductionStepNotes,
	LevelingFactor:  "1.000000000000000000000000000000",
	Allowances:      "0.000000000000000000000000000000",
	LaborRate:       SampleRate,
	LaborTime:       SampleRate,
	OverheadRate:    SampleRate,
	Production:      SampleProductionOutput,
	Consumptions:    NewList([]Consumption{*SampleConsumption}, PageInfo{}),
	Machines:        NewList([]Machine{*SampleMachine}, PageInfo{}),
	ScanningStation: SampleScanningStation,
	Department:      SampleDepartment,
	InSteps:         NewList([]ProductionStep{}, PageInfo{}),
	OutSteps:        NewList([]ProductionStep{}, PageInfo{}),
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductionStep) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionStep)
}
