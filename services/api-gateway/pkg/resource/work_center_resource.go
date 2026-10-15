package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleWorkCenterID = "wkcc_m0jayebxnkos"
const SampleWorkCenterName = "Fabrication"

// A place where production work is done, such as fabrication or packaging, with the machines and scanning stations that do it.
//
// Operations in a manufacturing method run at a work center. Its `labor_rate` costs the labor done there.
type WorkCenter struct {
	// Work center ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=work_center"`
	// Display name of the work center.
	//
	// Unique within your account (case-insensitive).
	Name string `json:"name" validate:"required"`
	// The location where the work center operates.
	//
	// Null unless included.
	Location *Location `json:"location" expandable:"true"`
	// Scanning stations at this work center, newest first.
	//
	// Null unless included. Holds the first 10; when there are more, `page_info.next_page_url` lists the rest.
	ScanningStations *List[ScanningStation] `json:"scanning_stations" expandable:"true"`
	// Machines at this work center, newest first.
	//
	// Null unless included. Holds the first 10; when there are more, `page_info.next_page_url` lists the rest.
	Machines *List[Machine] `json:"machines" expandable:"true"`
	// Hourly labor rate for work done at this work center, such as `45.00 usd / hr`.
	//
	// Operations at the work center default to it, and production scheduling costs changeovers with the constraint work center's rate. Null when none is set, or unless the caller holds `costs:read`; customer and supplier portal users never see it.
	LaborRate *UnitPrice `json:"labor_rate" sensitive:"cost"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleWorkCenter = &WorkCenter{
	ID:               SampleWorkCenterID,
	Object:           constants.ObjectTypeWorkCenter,
	Name:             SampleWorkCenterName,
	Location:         SampleLocation,
	ScanningStations: NewList([]ScanningStation{*SampleScanningStation}, PageInfo{}),
	Machines:         NewList([]Machine{*SampleMachine}, PageInfo{}),
	LaborRate:        nil,
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*WorkCenter) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleWorkCenter)
}
