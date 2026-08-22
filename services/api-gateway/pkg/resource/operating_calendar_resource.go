package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
)

const (
	SampleOperatingCalendarID        = "occd_7f2m9qk4wzxb"
	SampleOperatingCalendarClosureID = "occdcn_3vh8yt5nqp1r"
)

// OperatingCalendar is the set of days one party to a shipment operates.
//
// A `ship` calendar is the plant tendering freight to a carrier; a `receive` calendar is a customer's dock accepting it. Ship-by dates are resolved against both, so an order is never committed to a day nobody can act on.
type OperatingCalendar struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=operating_calendar"`
	// Unique identifier.
	ID string `json:"id" validate:"required"`
	// Short stable identifier, unique per account.
	Code string `json:"code" validate:"required"`
	// Human-readable name.
	Name string `json:"name" validate:"required"`
	// Which side of a shipment this calendar describes.
	Kind constants.OperatingCalendarKind `json:"kind" validate:"required"`
	// Open weekdays as seven characters of '0' or '1', Monday first. "1111100" is Monday to Friday; "1111000" is a Monday-to-Thursday plant.
	DaysOfWeek string `json:"days_of_week" validate:"required"`
	// Local time freight has to be tendered by, as "15:00". Only a shipping calendar carries one.
	CutoffAt *string `json:"cutoff_at"`
	// IANA zone the cutoff is read in. Null on a receiving calendar means it is taken from the ship-to address.
	Timezone *string `json:"timezone"`
	// Whether this is the calendar used when nothing more specific is linked. Exactly one per kind.
	IsDefault bool `json:"is_default" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// OperatingCalendarClosure is one date a calendar is shut — a holiday, or a day of a shutdown week.
type OperatingCalendarClosure struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=operating_calendar_closure"`
	// Unique identifier.
	ID string `json:"id" validate:"required"`
	// The calendar this closure belongs to.
	OperatingCalendarID string `json:"operating_calendar_id" validate:"required"`
	// The date nothing operates.
	ClosedOn time.Time `json:"closed_on" validate:"required"`
	// What the closure is, such as "Thanksgiving Day" or "Summer shutdown".
	Name string `json:"name" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var (
	sampleCalendarCutoffAt = "15:00"
	sampleCalendarTimezone = "America/Chicago"
	sampleCalendarCreated  = time.Date(2026, time.January, 6, 9, 14, 0, 0, time.UTC)
)

// The sample is the configuration the feature exists for: a plant that ships Monday to Thursday and hands freight over by 3pm.
var SampleOperatingCalendar = &OperatingCalendar{
	Object:     constants.ObjectTypeOperatingCalendar,
	ID:         SampleOperatingCalendarID,
	Code:       "default_ship",
	Name:       "Shipping days",
	Kind:       constants.OperatingCalendarKindShip,
	DaysOfWeek: "1111000",
	CutoffAt:   &sampleCalendarCutoffAt,
	Timezone:   &sampleCalendarTimezone,
	IsDefault:  true,
	CreatedAt:  sampleCalendarCreated,
	UpdatedAt:  sampleCalendarCreated,
}

var SampleOperatingCalendarClosure = &OperatingCalendarClosure{
	Object:              constants.ObjectTypeOperatingCalendarClosure,
	ID:                  SampleOperatingCalendarClosureID,
	OperatingCalendarID: SampleOperatingCalendarID,
	ClosedOn:            time.Date(2026, time.November, 26, 0, 0, 0, 0, time.UTC),
	Name:                "Thanksgiving Day",
	CreatedAt:           sampleCalendarCreated,
	UpdatedAt:           sampleCalendarCreated,
}

func (*OperatingCalendar) SchemaExample() any        { return SampleOperatingCalendar }
func (*OperatingCalendarClosure) SchemaExample() any { return SampleOperatingCalendarClosure }
