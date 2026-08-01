package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleMachineDowntimeEventID = "mcdt_0192a4c17b3e4f8a91c2d05e77"
const SampleMachineDowntimeReasonID = "mcdttp_01seedbreakdown"

// A reason a machine stopped running.
//
// The `oee_bucket` decides which OEE term the stoppage charges: `availability` losses reduce run time, `performance` losses are minor stops and speed loss, `quality` losses cover rework and holds, and `not_scheduled` time is removed from the OEE calculation entirely rather than counted against it.
type MachineDowntimeReason struct {
	// Downtime reason ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=machine_downtime_reason"`
	// Stable code used when logging downtime.
	Code constants.MachineDowntimeReasonCode `json:"code" validate:"required"`
	// Display name of the reason.
	Name string `json:"name" validate:"required"`
	// Which OEE term this reason charges.
	OeeBucket constants.OeeBucket `json:"oee_bucket" validate:"required"`
	// Whether the stoppage was scheduled in advance, such as preventive maintenance.
	PlanningStatus constants.DowntimePlanningStatus `json:"planning_status" validate:"required"`
	// Display order, ascending.
	SortOrder int32 `json:"sort_order" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleMachineDowntimeReason = &MachineDowntimeReason{
	ID:             SampleMachineDowntimeReasonID,
	Object:         constants.ObjectTypeMachineDowntimeReason,
	Code:           constants.MachineDowntimeReasonCodeBreakdown,
	Name:           "Breakdown",
	OeeBucket:      constants.OeeBucketAvailability,
	PlanningStatus: constants.DowntimePlanningStatusUnplanned,
	SortOrder:      10,
	CreatedAt:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:      timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*MachineDowntimeReason) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMachineDowntimeReason)
}

// The reason for a stoppage, as carried on a downtime event.
//
// A denormalized view of the reason taxonomy: the stable code plus the display name and OEE bucket resolved from it at read time.
type MachineDowntimeReasonSummary struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=machine_downtime_reason"`
	// Stable code identifying the reason.
	Code constants.MachineDowntimeReasonCode `json:"code" validate:"required"`
	// Display name of the reason.
	Name *string `json:"name"`
	// Which OEE term this reason charges.
	OeeBucket *constants.OeeBucket `json:"oee_bucket"`
}

// A period during which a machine was not running.
//
// Downtime is what makes OEE Availability a measurement rather than an estimate. An event with no `ended_at` is still open, meaning the machine is down right now; a machine can only have one open event at a time.
type MachineDowntimeEvent struct {
	// Downtime event ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=machine_downtime_event"`
	// The machine that stopped.
	Machine *Machine `json:"machine" expandable:"true"`
	// The department the machine belongs to, captured when the event was logged.
	Department *Department `json:"department" expandable:"true"`
	// Why the machine stopped.
	Reason *MachineDowntimeReasonSummary `json:"reason" validate:"required"`
	// When the machine stopped.
	StartedAt time.Time `json:"started_at" validate:"required"`
	// When the machine started running again.
	EndedAt *time.Time `json:"ended_at"`
	// How long the machine was down, in seconds.
	DurationSeconds *int32 `json:"duration_seconds"`
	// The business day the stoppage is counted against.
	ShiftDate time.Time `json:"shift_at" validate:"required"`
	// The shift the stoppage is counted against.
	ShiftCode *string `json:"shift_code"`
	// What the machine was running when it stopped.
	Item *Item `json:"item" expandable:"true"`
	// The production run in progress when the machine stopped.
	ProductionRun *Entity `json:"production_run"`
	// The batch in progress when the machine stopped.
	Batch *Entity `json:"batch"`
	// The scheduled campaign the stoppage interrupted.
	ScheduleLine *Entity `json:"schedule_line"`
	// Free-form notes about the stoppage.
	Note *string `json:"note"`
	// The actor that logged the event — a user, API key, or agent. Expandable.
	ReportedBy *Actor `json:"reported_by" expandable:"true"`
	// How the event was recorded.
	Source constants.MachineDowntimeSource `json:"source" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var (
	sampleDowntimeReasonName = "Breakdown"
	sampleDowntimeOeeBucket  = constants.OeeBucketAvailability
	sampleDowntimeDuration   = int32(1380)
	sampleDowntimeShiftCode  = "A"
	sampleDowntimeNote       = "Needle bar jam; replaced needle 42."
)

var SampleMachineDowntimeEvent = &MachineDowntimeEvent{
	ID:         SampleMachineDowntimeEventID,
	Object:     constants.ObjectTypeMachineDowntimeEvent,
	Machine:    SampleMachine,
	Department: SampleDepartment,
	Reason: &MachineDowntimeReasonSummary{
		Object:    constants.ObjectTypeMachineDowntimeReason,
		Code:      constants.MachineDowntimeReasonCodeBreakdown,
		Name:      &sampleDowntimeReasonName,
		OeeBucket: &sampleDowntimeOeeBucket,
	},
	StartedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	EndedAt:         timeutil.TimestampToTimePtr(sampleUpdatedAtTimestamp),
	DurationSeconds: &sampleDowntimeDuration,
	ShiftDate:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	ShiftCode:       &sampleDowntimeShiftCode,
	Item:            SampleItem,
	ProductionRun:   NewEntity(SampleProductionRunID, constants.ObjectTypeProductionRun, nil, nil),
	Batch:           NewEntity(SampleBatchID, constants.ObjectTypeBatch, nil, nil),
	ScheduleLine:    NewEntity(SampleProductionScheduleLineID, constants.ObjectTypeProductionScheduleLine, nil, nil),
	Note:            &sampleDowntimeNote,
	ReportedBy:      SampleActor,
	Source:          constants.MachineDowntimeSourceManual,
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*MachineDowntimeEvent) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMachineDowntimeEvent)
}
