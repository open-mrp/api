package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// Production step with all nested data.
type ProductionStep struct {
	// Production step ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_step"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Notes.
	Notes *string `json:"notes"`
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
	// Production output.
	//
	// Expandable via include[]=production.
	Production *ProductionOutput `json:"production" expandable:"true"`
	// Materials consumed by this step.
	//
	// Expandable via include[]=consumptions.
	Consumptions *List[Consumption] `json:"consumptions" expandable:"true"`
	// Machines assigned to this step.
	//
	// Expandable via include[]=machines.
	Machines *List[Machine] `json:"machines" expandable:"true"`
	// Scanning station where this step is scanned, if assigned.
	//
	// Expandable via include[]=scanning_station.
	ScanningStation *ScanningStation `json:"scanning_station" expandable:"true"`
	// Input steps feeding into this step.
	//
	// Expandable via include[]=in_steps.
	InSteps *List[ProductionStep] `json:"in_steps" expandable:"true"`
	// Output steps this step feeds into.
	//
	// Expandable via include[]=out_steps.
	OutSteps *List[ProductionStep] `json:"out_steps" expandable:"true"`
	// Department.
	//
	// Expandable via include[]=department.
	Department *Department `json:"department" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProductionStep = &ProductionStep{
	ID:              SampleProductionStepID,
	Object:          constants.ObjectTypeProductionStep,
	Name:            "Final Assembly",
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
