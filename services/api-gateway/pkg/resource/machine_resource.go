package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// A piece of production equipment, such as a CNC router or press, assigned to a department.
type Machine struct {
	// Machine ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=machine"`
	// Display name of the machine.
	//
	// Unique within the account.
	Name string `json:"name" validate:"required"`
	// Serial number of the machine.
	SerialNumber string `json:"serial_number" validate:"required"`
	// Free-form notes about the machine.
	Notes *string `json:"notes"`
	// The department this machine belongs to.
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
