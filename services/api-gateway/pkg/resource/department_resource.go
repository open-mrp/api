package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleDepartmentID = "dp_01gf7a8200er3ar3pkfrb6kk30"
const SampleDepartmentName = "Fabrication"

// ---------------------------------------------------------------------------
// Department — full department resource
// ---------------------------------------------------------------------------

// Department represents a department within an account.
type Department struct {
	// The unique identifier for the department.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=department"`
	// The display name of the department.
	Name string `json:"name" validate:"required"`
	// Optional notes about the department.
	Notes *string `json:"notes"`
	// The location associated with this department.
	Location *Location `json:"location" expandable:"true"`
	// The scanning stations belonging to this department.
	ScanningStations *List[ScanningStation] `json:"scanning_stations" expandable:"true"`
	// The machines belonging to this department.
	Machines *List[Machine] `json:"machines" expandable:"true"`
	// The timestamp when the department was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the department was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleDepartment = &Department{
	ID:               SampleDepartmentID,
	Object:           constants.ObjectTypeDepartment,
	Name:             SampleDepartmentName,
	Notes:            nil,
	Location:         SampleLocation,
	ScanningStations: NewList([]ScanningStation{*SampleScanningStation}, PageInfo{}),
	Machines:         NewList([]Machine{*SampleMachine}, PageInfo{}),
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Department) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleDepartment)
}
