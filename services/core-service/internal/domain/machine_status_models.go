package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
)

// MachineCampaign is one campaign on a machine, with how far through it the floor is.
type MachineCampaign struct {
	ProductionScheduleLineID string
	ItemID                   string
	SKU                      string
	WeekStartDate            time.Time
	WeekIndex                int32
	PlannedQuantity          float64
	ScannedQuantity          float64
	// RemainingQuantity never goes below zero: an over-run is reported through the scanned figure rather than by making what is left negative.
	RemainingQuantity  float64
	Unit               string
	ReleasedBatchCount int64
	ScannedBatchCount  int64
	PlannedRunHours    float64
	StatusCode         string
	ProductionRunID    *string
}

// MachineDowntimeSummary is an open stoppage, as a floor display needs it.
type MachineDowntimeSummary struct {
	EventID    string
	Reason     string
	ReasonName string
	OEEBucket  string
	StartedAt  time.Time
	Note       *string
}

// MachineStatus is one machine's whole picture: what it is on, what is left, what is next.
type MachineStatus struct {
	MachineID      string
	MachineName    string
	DepartmentID   *string
	DepartmentName *string
	Status         constants.MachineWorkStatus
	Downtime       *MachineDowntimeSummary
	// Current is the campaign being worked: the earliest released one with batches still unscanned. A machine whose released work is all scanned has finished it, so the next campaign becomes current instead.
	Current *MachineCampaign
	// Next is what follows Current in the plan, so an operator can set up for it.
	Next *MachineCampaign
	// This week's totals for the machine, which is the number management tracks.
	WeekPlannedQuantity float64
	WeekScannedQuantity float64
	WeekPlannedRunHours float64
	Unit                string
}

// MachineStatusResult is the whole floor at one moment.
type MachineStatusResult struct {
	// ProductionScheduleID is the published version the picture is read from, empty when nothing is published — in which case every machine is idle rather than the endpoint failing, since the floor still exists when planning has not caught up.
	ProductionScheduleID string
	WeekStartDate        time.Time
	Machines             []MachineStatus
}

// ListMachineStatusParams asks for the floor at a moment.
type ListMachineStatusParams struct {
	AccountID string
	// AsOf defaults to now. Present so a caller can ask what a given week looked like.
	AsOf time.Time
	// DepartmentIDs narrows to one part of the plant; empty means every machine.
	DepartmentIDs []string
}

// MachineForStatusRow is one machine as stored, before any plan or downtime is attached to it.
type MachineForStatusRow struct {
	ID   string
	Name string
	// DepartmentID is NOT NULL on machine, so an empty string is the "unassigned" case.
	DepartmentID   string
	DepartmentName *string
}

// OpenDowntimeForStatusRow is an open stoppage as stored.
type OpenDowntimeForStatusRow struct {
	ID         string
	MachineID  string
	ReasonCode string
	ReasonName *string
	OEEBucket  *string
	StartedAt  time.Time
	Note       *string
}

// ListScheduleLinesForStatusParams scopes the plan read to one published version from a week forward.
type ListScheduleLinesForStatusParams struct {
	AccountID            string
	ProductionScheduleID string
	FromWeek             time.Time
}

// ScheduleLineForStatusRow is one schedule line with its scan progress, as stored.
type ScheduleLineForStatusRow struct {
	ID              string
	MachineID       string
	ItemID          string
	WeekIndex       int32
	WeekStartDate   time.Time
	PlannedQuantity float64
	PlannedRunHours float64
	StatusCode      string
	ProductionRunID *string
	// PlannedUnitAbbreviation is nil when the line's unit row is missing.
	PlannedUnitAbbreviation *string
	// SKU is nil when the item has no policy row on the schedule.
	SKU                *string
	ReleasedBatchCount int64
	ScannedBatchCount  int64
	ScannedQuantity    float64
}
