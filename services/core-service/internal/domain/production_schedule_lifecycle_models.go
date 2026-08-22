package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/pagination"
)

// Deviation types name what changed about a line. The reason code carries why, and is required only for edits inside a frozen week. These alias the shared enum so the vocabulary has a single source of truth.
const (
	DeviationTypeLineAdded       = string(constants.ScheduleDeviationTypeLineAdded)
	DeviationTypeLineRemoved     = string(constants.ScheduleDeviationTypeLineRemoved)
	DeviationTypeQuantityChanged = string(constants.ScheduleDeviationTypeQuantityChanged)
	DeviationTypeMachineChanged  = string(constants.ScheduleDeviationTypeMachineChanged)
	DeviationTypeResequenced     = string(constants.ScheduleDeviationTypeResequenced)
	DeviationTypeWeekMoved       = string(constants.ScheduleDeviationTypeWeekMoved)
)

type ScheduleDeviationType struct {
	ID        string
	Code      string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ProductionScheduleDeviation struct {
	ID                   string
	AccountID            string
	ProductionScheduleID string
	// Nil for a removed line, whose row no longer exists, and for an added one at the moment the deviation is written.
	ProductionScheduleLineID *string

	DeviationTypeCode string

	// IsFrozenWeek is materialized at write time from the schedule's frozen_through_date as it stood at that moment. Deriving it on read would let a later publish retroactively reclassify past edits and the adherence KPI would drift.
	IsFrozenWeek bool

	WeekIndex *int32
	MachineID *string
	ItemID    *string

	BeforeJSON []byte
	AfterJSON  []byte

	DeltaQuantity float64
	DeltaRunHours float64

	ReasonCode *string
	ReasonNote *string

	ActorID   string
	CreatedAt time.Time
}

type ListProductionScheduleDeviationsParams struct {
	AccountID  string
	ScheduleID string
	Cursor     *string
	Limit      int32
	Query      *string
	// FrozenOnly filters to (or away from) frozen-week deviations. Nil returns both.
	FrozenOnly *bool
}

type ListProductionScheduleDeviationsResult struct {
	Deviations []*ProductionScheduleDeviation
	PageInfo   pagination.PageInfo
}

type CreateProductionScheduleLineParams struct {
	AccountID  string
	ScheduleID string
	WeekIndex  int32
	MachineID  string
	ItemID     string
	Quantity   float64
	Lots       *int32
	RunHours   *float64
	ReasonCode *string
	ReasonNote *string
}

type UpdateProductionScheduleLineParams struct {
	AccountID     string
	ScheduleID    string
	LineID        string
	WeekIndex     *int32
	MachineID     *string
	Quantity      *float64
	Lots          *int32
	RunHours      *float64
	SequenceIndex *int32
	StatusCode    *string
	// ReasonCode is Clearable: unset leaves the column unchanged, clear nulls it.
	ReasonCode field.Clearable[string]
	ReasonNote *string
}

type DeleteProductionScheduleLineParams struct {
	AccountID  string
	ScheduleID string
	LineID     string
	ReasonCode *string
	ReasonNote *string
}

type PublishProductionScheduleParams struct {
	AccountID  string
	ScheduleID string
}

type UpdateLineRepoParams struct {
	AccountID       string
	LineID          string
	MachineID       *string
	WeekIndex       *int32
	WeekStartDate   *time.Time
	PlannedQuantity *float64
	PlannedLots     *int32
	PlannedRunHours *float64
	SequenceIndex   *int32
	StatusCode      *string
	ReasonCode      *string
	ClearReasonCode bool
}

type FrozenLineTotals struct {
	LineCount       int64
	PlannedQuantity float64
}

// WeekReleaseState is how much of one planned week has already gone to the floor.
type WeekReleaseState struct {
	TotalLines    int64
	ReleasedLines int64
	// ExistingProductionRunID names a run the week is already tied to, so a repeat release can point at it instead of failing with nothing to look at.
	ExistingProductionRunID *string
}

// ReleaseScheduleWeekParams asks for one planned week to become a production run.
type ReleaseScheduleWeekParams struct {
	AccountID            string
	ProductionScheduleID string
	WeekIndex            int32
	ResponsibleUserID    string
	ScanningStationID    *string
	// SkipCarryForward issues the whole week as new tickets, leaving an earlier week's unworked doffs where they are. Off by default: reprinting a ticket the floor is already holding is the failure this exists to prevent, and a planner has to ask for it deliberately.
	SkipCarryForward bool
}

// ReleaseBatch is one batch a release puts into the run: a single lot off one campaign.
type ReleaseBatch struct {
	ItemID   string
	SKU      string
	Quantity float64
	// BatchID is empty on a preview of a batch that would be created; a carried-forward batch already exists and names itself even in a preview.
	BatchID string
	// CarriedForwardFrom is the number of the run this ticket came off when the batch already existed. Empty means the release creates it new.
	//
	// This is what the confirmation is really telling a planner: these doffs are already printed and on the floor, so nobody needs to print them again.
	CarriedForwardFrom string
}

// CarryForwardBatch is an unworked ticket an earlier week issued, and a candidate to be moved into the run being released.
type CarryForwardBatch struct {
	BatchID             string
	ProductionRunID     string
	ProductionRunNumber string
	ProductionStepID    *string
	Quantity            float64
	UnitID              string
}

// ListCarryForwardBatchesParams asks for one item's unworked tickets from weeks that have already begun.
type ListCarryForwardBatchesParams struct {
	AccountID string
	ItemID    string
	// WeekStartDate is the week being released. Only runs from weeks strictly before it are eligible: a week still ahead has not been missed, it simply has not been worked.
	WeekStartDate time.Time
	Limit         int32
}

// ReleasedScheduleLine is one campaign and the lots it broke into.
type ReleasedScheduleLine struct {
	ProductionScheduleLineID string
	ItemID                   string
	SKU                      string
	MachineID                string
	MachineName              *string
	PlannedQuantity          float64
	LotUnits                 float64
	// Unit is what the quantity and the lot are counted in. "6 × 60" on a release confirmation is not an instruction until it says 6 × 60 of what.
	Unit string
	// CarriedForwardQuantity is how much of PlannedQuantity is covered by tickets an earlier week already issued.
	CarriedForwardQuantity float64
	Batches                []ReleaseBatch
}

// ReleaseScheduleWeekResult is the run a release produced, and what went into it.
type ReleaseScheduleWeekResult struct {
	ProductionRun     *ProductionRun
	WeekIndex         int32
	WeekStartDate     time.Time
	ReleasedLineCount int32
	BatchCount        int32
	// CarriedForwardBatchCount is how many of BatchCount were moved off an earlier run rather than created. Tickets for these are already printed.
	CarriedForwardBatchCount int32
	TotalQuantity            float64
	Lines                    []ReleasedScheduleLine
}

// ReleaseScheduleWeekPreview is what a release would do, with nothing written.
//
// A preview is not just a courtesy here: a release creates a numbered run and dozens of batch rows, and undoing that by hand is real work. IsReleasable is false when the week is empty or already released, and BlockedReason says which.
type ReleaseScheduleWeekPreview struct {
	WeekIndex     int32
	WeekStartDate time.Time
	LineCount     int32
	BatchCount    int32
	// CarriedForwardBatchCount is how many of BatchCount would be moved off an earlier run rather than created new.
	CarriedForwardBatchCount int32
	TotalQuantity            float64
	Lines                    []ReleasedScheduleLine
	IsReleasable             bool
	BlockedReason            *string
	ExistingProductionRunID  *string
}
