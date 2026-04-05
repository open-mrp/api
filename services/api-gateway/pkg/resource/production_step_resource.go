package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// ProductionStep represents a production step with all nested data.
type ProductionStep struct {
	// The unique identifier for the production step.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_step"`
	// The display name of the production step.
	Name string `json:"name" validate:"required"`
	// Optional notes about the production step.
	Notes *string `json:"notes"`
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
	// The production output for this step.
	Production *ProductionOutput `json:"production"`
	// The consumptions for this step.
	Consumptions []Consumption `json:"consumptions"`
	// The machines assigned to this step.
	Machines []ProductionStepMachine `json:"machines"`
	// The scanning station for this step.
	ScanningStation *ProductionStepScanStation `json:"scanning_station"`
	// The input steps that feed into this step.
	InSteps []ProductionStepRef `json:"in_steps"`
	// The output steps that this step feeds into.
	OutSteps []ProductionStepRef `json:"out_steps"`
	// The department this step belongs to.
	Department *ProductionStepDepartment `json:"department"`
	// The timestamp when the production step was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the production step was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// ProductionStepRef is a minimal production step reference.
type ProductionStepRef struct {
	// The unique identifier for the production step.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_step"`
	// The display name of the production step.
	Name string `json:"name" validate:"required"`
}

// ProductionStepScanStation is a scanning station reference on a production step.
type ProductionStepScanStation struct {
	// The unique identifier for the scanning station.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=scanning_station"`
	// The display name of the scanning station.
	Name string `json:"name" validate:"required"`
}

// ProductionStepDepartment is a department reference on a production step.
type ProductionStepDepartment struct {
	// The unique identifier for the department.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=department"`
}

// ProductionStepMachine is a machine reference on a production step.
type ProductionStepMachine struct {
	// The unique identifier for the machine.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=machine"`
	// The display name of the machine.
	Name string `json:"name" validate:"required"`
}

var SampleProductionStep = &ProductionStep{
	ID:             SampleProductionStepID,
	Object:         constants.ObjectTypeProductionStep,
	Name:           "Final Assembly",
	LevelingFactor: "1.000000000000000000000000000000",
	Allowances:     "0.000000000000000000000000000000",
	LaborRate:      SampleRate,
	LaborTime:      SampleRate,
	OverheadRate:   SampleRate,
	Production:     SampleProductionOutput,
	Consumptions:   []Consumption{*SampleConsumption},
	Machines:       []ProductionStepMachine{},
	InSteps:        []ProductionStepRef{},
	OutSteps:       []ProductionStepRef{},
	CreatedAt:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:      timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductionStep) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionStep)
}

var SampleProductionStepRef = &ProductionStepRef{
	ID:     SampleProductionStepID,
	Object: constants.ObjectTypeProductionStep,
	Name:   "Final Assembly",
}

func (*ProductionStepRef) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionStepRef)
}

func (*ProductionStepScanStation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&ProductionStepScanStation{
		ID:     SampleScanningStationID,
		Object: constants.ObjectTypeScanningStation,
		Name:   SampleScanningStationName,
	})
}

func (*ProductionStepDepartment) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&ProductionStepDepartment{
		ID:     SampleDepartmentID,
		Object: constants.ObjectTypeDepartment,
	})
}

func (*ProductionStepMachine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&ProductionStepMachine{
		ID:     SampleMachineID,
		Object: constants.ObjectTypeMachine,
		Name:   "CNC Router",
	})
}
