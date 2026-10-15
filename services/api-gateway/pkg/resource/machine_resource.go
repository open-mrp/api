package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

// A piece of production equipment, such as a CNC router or press, at a work center.
type Machine struct {
	// Machine ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=machine"`
	// Display name of the machine.
	//
	// Unique within your account (case-insensitive).
	Name string `json:"name" validate:"required"`
	// Serial number of the machine.
	//
	// Unique within your account (case-insensitive).
	SerialNumber string `json:"serial_number" validate:"required"`
	// Whether the machine can be newly assigned.
	//
	// `active`: in use. `archived`: retired; it stays on the batches, downtime events and operations that already name it, but can't be newly assigned to them or moved into a work center.
	Status constants.ArchiveStatus `json:"status" validate:"required"`
	// The work center this machine belongs to.
	//
	// Null unless included.
	WorkCenter *WorkCenter `json:"work_center" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

const SampleMachineName = "CNC Router"
const SampleMachineSerialNumber = "SN-2024-0001"

// WorkCenter is left nil here because SampleWorkCenter embeds SampleMachine in its machines list, and a back-reference would be an initialization cycle.
var SampleMachine = &Machine{
	ID:           SampleMachineID,
	Object:       constants.ObjectTypeMachine,
	Name:         SampleMachineName,
	SerialNumber: SampleMachineSerialNumber,
	Status:       constants.ArchiveStatusActive,
	WorkCenter:   nil,
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Machine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMachine)
}
