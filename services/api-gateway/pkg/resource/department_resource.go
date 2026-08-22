package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleDepartmentID = "dp_m0jayebxnkos"
const SampleDepartmentName = "Fabrication"

// ---------------------------------------------------------------------------
// Department — full department resource
// ---------------------------------------------------------------------------

// A functional area of a production operation, such as fabrication or packaging, that groups scanning stations and machines.
type Department struct {
	// Department ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=department"`
	// Display name of the department.
	//
	// Unique within the account.
	Name string `json:"name" validate:"required"`
	// Free-form notes about the department.
	Notes *string `json:"notes"`
	// The storage location where this department operates.
	Location *Location `json:"location" expandable:"true"`
	// Scanning stations in this department.
	ScanningStations *List[ScanningStation] `json:"scanning_stations" expandable:"true"`
	// Machines in this department.
	Machines *List[Machine] `json:"machines" expandable:"true"`
	// Hourly labor rate for work done in this department, such as a changeover technician.
	//
	// Production scheduling costs changeovers with the constraint department's rate when one is set, falling back to the account-wide changeover labor rate setting.
	LaborRate *Rate `json:"labor_rate"`
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
	LaborRate:        nil,
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Department) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleDepartment)
}
