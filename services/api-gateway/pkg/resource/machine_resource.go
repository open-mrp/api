package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// Machine represents a machine within an account.
type Machine struct {
	// The unique identifier for the machine.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=machine"`
	// The display name of the machine.
	Name string `json:"name" validate:"required"`
	// The serial number of the machine.
	SerialNumber string `json:"serial_number" validate:"required"`
	// Optional notes about the machine.
	Notes *string `json:"notes"`
	// The department this machine belongs to.
	Department *Department `json:"department" expandable:"true"`
	// The timestamp when the machine was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the machine was last updated.
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
