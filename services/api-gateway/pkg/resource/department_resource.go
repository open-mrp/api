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

// Department resource.
type Department struct {
	// Department ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=department"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Notes about the department.
	Notes *string `json:"notes"`
	// Associated location.
	Location *Location `json:"location" expandable:"true"`
	// Scanning stations in this department.
	ScanningStations *List[ScanningStation] `json:"scanning_stations" expandable:"true"`
	// Machines in this department.
	Machines *List[Machine] `json:"machines" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
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
