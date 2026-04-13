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
	// Labor rate.
	LaborRate *Rate `json:"labor_rate"`
	// Labor time.
	LaborTime *Rate `json:"labor_time"`
	// Overhead rate.
	OverheadRate *Rate `json:"overhead_rate"`
	// Production output.
	Production *ProductionOutput `json:"production"`
	// Consumptions.
	Consumptions []Consumption `json:"consumptions"`
	// Machines assigned to this step.
	Machines []ProductionStepMachine `json:"machines"`
	// Scanning station.
	ScanningStation *ProductionStepScanStation `json:"scanning_station"`
	// Input steps feeding into this step.
	InSteps []ProductionStepRef `json:"in_steps"`
	// Output steps this step feeds into.
	OutSteps []ProductionStepRef `json:"out_steps"`
	// Department.
	Department *ProductionStepDepartment `json:"department"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Minimal production step reference.
type ProductionStepRef struct {
	// Production step ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_step"`
	// Display name.
	Name string `json:"name" validate:"required"`
}

// Scanning station reference on a production step.
type ProductionStepScanStation struct {
	// Scanning station ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=scanning_station"`
	// Display name.
	Name string `json:"name" validate:"required"`
}

// Department reference on a production step.
type ProductionStepDepartment struct {
	// Department ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=department"`
}

// Machine reference on a production step.
type ProductionStepMachine struct {
	// Machine ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=machine"`
	// Display name.
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
