package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// Machine within an account.
type Machine struct {
	// Machine ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=machine"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Serial number.
	SerialNumber string `json:"serial_number" validate:"required"`
	// Notes.
	Notes *string `json:"notes"`
	// Associated department.
	Department *Department `json:"department" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

const SampleMachineName = "CNC Router"
const SampleMachineSerialNumber = "SN-2024-0001"

var SampleMachine = &Machine{
	ID:           SampleMachineID,
	Object:       constants.ObjectTypeMachine,
	Name:         SampleMachineName,
	SerialNumber: SampleMachineSerialNumber,
	Notes:        nil,
	Department:   nil,
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Machine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMachine)
}
