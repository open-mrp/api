package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/pagination"
)

// OEE buckets. A reason's bucket decides which OEE term its downtime charges. OeeBucketNotScheduled is the odd one out: it is removed from the Availability denominator entirely rather than counted as a loss against it, because a machine nobody planned to run has no OEE rather than 0% OEE.
//
// These are string-typed views of the shared enums in shared/constants — the single source of truth for values that cross the gRPC contract — kept here so repository code that works in plain storage strings does not re-declare the vocabulary.
const (
	OeeBucketAvailability               = string(constants.OeeBucketAvailability)
	OeeBucketPerformance                = string(constants.OeeBucketPerformance)
	OeeBucketQuality                    = string(constants.OeeBucketQuality)
	OeeBucketNotScheduled               = string(constants.OeeBucketNotScheduled)
	MachineDowntimeReasonCodeChangeover = string(constants.MachineDowntimeReasonCodeChangeover)
)

// Downtime event sources. Manual is a person in the UI; scanner is the shop-floor station; inferred is a system-derived gap; api is an integration. String-typed views of constants.MachineDowntimeSource, the single source of truth.
const (
	MachineDowntimeSourceManual   = string(constants.MachineDowntimeSourceManual)
	MachineDowntimeSourceScanner  = string(constants.MachineDowntimeSourceScanner)
	MachineDowntimeSourceInferred = string(constants.MachineDowntimeSourceInferred)
	MachineDowntimeSourceAPI      = string(constants.MachineDowntimeSourceAPI)
)

type MachineDowntimeReason struct {
	ID        string
	Code      string
	Name      string
	OeeBucket string
	IsPlanned bool
	SortOrder int32
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MachineDowntimeEvent struct {
	ID        string
	AccountID string
	MachineID string `audit:"machine_id"`
	// DepartmentID and ProductionStepID are resolved from the machine at write time so downtime rolls up by department without joining through machine on read.
	DepartmentID     *string
	ProductionStepID *string

	ReasonCode      string `audit:"reason_code"`
	ReasonName      *string
	ReasonOeeBucket *string
	ReasonIsPlanned *bool

	StartedAt time.Time  `audit:"started_at"`
	EndedAt   *time.Time `audit:"ended_at"`
	// DurationSeconds is materialized when the event closes; nil while still down.
	DurationSeconds *int32

	ShiftDate time.Time
	ShiftCode *string

	ItemID          *string `audit:"item_id"`
	ProductionRunID *string `audit:"production_run_id"`
	BatchID         *string `audit:"batch_id"`
	ScheduleLineID  *string

	Note         *string `audit:"note"`
	ReportedByID string
	SourceCode   string `audit:"source_code"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsOpen reports whether the machine is still down.
func (e *MachineDowntimeEvent) IsOpen() bool { return e.EndedAt == nil }

type ListMachineDowntimeEventsParams struct {
	AccountID     string
	Cursor        *string
	Limit         int32
	Query         *string
	MachineIDs    []string
	DepartmentIDs []string
	ReasonCodes   []string
	OpenOnly      bool
	StartDate     *time.Time
	EndDate       *time.Time
}

type ListMachineDowntimeEventsResult struct {
	Events   []*MachineDowntimeEvent
	PageInfo pagination.PageInfo
}

type GetMachineDowntimeEventParams struct {
	AccountID string
	EventID   string
}

type CreateMachineDowntimeEventParams struct {
	AccountID  string
	MachineID  string
	ReasonCode string
	StartedAt  time.Time
	EndedAt    *time.Time
	// Duration is an alternative way of saying when the stoppage ended: the end is derived from the start plus this. Supplying both is rejected rather than reconciled, because the two disagreeing is a caller bug and picking a winner would hide it.
	Duration        *DowntimeDurationInput
	ItemID          *string
	ProductionRunID *string
	BatchID         *string
	Note            *string
	SourceCode      *string
	ReportedByID    string
}

// DowntimeDurationInput is how long a machine was down on its way in: a decimal string and the unit of time it counts.
//
// Carried as a quantity rather than a minute count so an operator can say "two days" without doing the arithmetic, and so the unit the number was entered in is the one that gets validated.
type DowntimeDurationInput struct {
	Value  string
	UnitID string
}

type UpdateMachineDowntimeEventParams struct {
	AccountID  string
	EventID    string
	ReasonCode *string
	StartedAt  *time.Time
	// MachineID moves the stoppage to another machine, re-resolving the department and step it is charged to. Correcting the machine is the one thing a mis-logged event most often needs, and deleting and re-logging would lose the record of who reported it and when.
	MachineID *string
	// The nullable columns are Clearable: unset leaves the column unchanged, clear nulls it. Clearing EndedAt reopens an event closed by mistake.
	EndedAt field.Clearable[time.Time]
	// Duration restates the end as a length of time from the start. Clearing it reopens the event, the same as clearing EndedAt.
	Duration        field.Clearable[DowntimeDurationInput]
	ItemID          field.Clearable[string]
	ProductionRunID field.Clearable[string]
	BatchID         field.Clearable[string]
	Note            field.Clearable[string]
}

type DeleteMachineDowntimeEventParams struct {
	AccountID string
	EventID   string
}
