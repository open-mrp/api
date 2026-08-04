package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// One campaign on a machine, with how far through it the floor is.
//
// A campaign is one item scheduled to run on one machine for one week. Progress is taken from the batches the floor has scanned against it rather than reported by hand, so it advances on its own as a shift runs.
type MachineCampaign struct {
	// The schedule line this campaign came from.
	ScheduleLine *Entity `json:"schedule_line" validate:"required"`
	// The item being made.
	Item *Entity `json:"item" validate:"required"`
	// SKU of the item.
	SKU string `json:"sku" validate:"required"`
	// First day of the week the campaign belongs to.
	WeekStartDate time.Time `json:"week_starts_at" validate:"required"`
	// Zero-based week offset from the start of the horizon.
	WeekIndex int32 `json:"week_index"`
	// Quantity the plan asked for.
	PlannedQuantity float64 `json:"planned_quantity"`
	// Quantity the floor has scanned so far.
	ScannedQuantity float64 `json:"scanned_quantity"`
	// Quantity still to make.
	//
	// Never negative: an over-run shows up in `scanned_quantity` rather than as negative remaining work.
	RemainingQuantity float64 `json:"remaining_quantity"`
	// Unit the quantities are counted in.
	Unit *string `json:"unit"`
	// Batches issued to the floor for this campaign.
	ReleasedBatchCount int64 `json:"released_batch_count"`
	// Batches of this campaign the floor has scanned.
	ScannedBatchCount int64 `json:"scanned_batch_count"`
	// Machine hours the plan allocates to the campaign.
	PlannedRunHours float64 `json:"planned_run_hours"`
	// Where the campaign is in its lifecycle.
	//
	// - `planned`: scheduled, but not yet released to the floor.
	// - `released`: issued to the floor as a production run, so batches can be scanned against it.
	// - `in_progress`: being run.
	// - `complete`: finished.
	// - `cancelled`: will not be run.
	Status constants.ProductionScheduleLineStatus `json:"status" validate:"required"`
	// The run carrying this campaign's work, once its week has been released.
	ProductionRun *Entity `json:"production_run"`
}

// An open stoppage on a machine.
type MachineDowntimeSummary struct {
	// The downtime event.
	Event *Entity `json:"event" validate:"required"`
	// Why the machine stopped.
	Reason *MachineDowntimeReasonSummary `json:"reason" validate:"required"`
	// When the machine went down.
	StartedAt time.Time `json:"started_at" validate:"required"`
	// Free-text note left by whoever logged it.
	Note *string `json:"note"`
}

// What one machine is doing right now.
//
// Assembled from the published schedule, the batches the floor has scanned against it, and any open downtime. A machine with an open stoppage reads `down` even when it has a released campaign, because a broken machine is not producing whatever the plan says.
type MachineStatus struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=machine_status"`
	// The machine.
	Machine *Entity `json:"machine" validate:"required"`
	// The department the machine belongs to.
	Department *Entity `json:"department"`
	// What the machine is doing.
	//
	// - `running`: a released campaign with work still to scan.
	// - `idle`: nothing released to it.
	// - `down`: an open downtime event, which outranks running.
	Status constants.MachineWorkStatus `json:"status" validate:"required"`
	// The open stoppage, when the machine is down.
	Downtime *MachineDowntimeSummary `json:"downtime"`
	// The campaign the machine is working on now.
	//
	// The earliest released campaign that still has batches left to scan. Once its last batch is scanned it stops being current and the queue moves on, which is what makes a floor display advance by itself.
	Current *MachineCampaign `json:"current"`
	// What the machine takes on next, so an operator can set up for it.
	//
	// When the machine has no current campaign it is between jobs, and this is the earliest campaign still ahead of it.
	Next *MachineCampaign `json:"next"`
	// Quantity planned on this machine for the current week.
	//
	// Summed across every campaign scheduled on the machine that week, not just the current one.
	WeekPlannedQuantity float64 `json:"week_planned_quantity"`
	// Quantity scanned on this machine so far in the current week.
	WeekScannedQuantity float64 `json:"week_scanned_quantity"`
	// Machine hours the plan allocates on this machine for the current week.
	WeekPlannedRunHours float64 `json:"week_planned_run_hours"`
	// Unit the week's quantities are counted in.
	Unit *string `json:"unit"`
}

var (
	sampleMachineStatusUnit        = "pr"
	sampleMachineStatusMachineName = "Knitter 3"
	sampleMachineStatusSKU         = "MZ-GREIGE-CREW"
	sampleMachineStatusNextSKU     = "MZ-GREIGE-QTR"
)

var SampleMachineStatus = &MachineStatus{
	Object:              constants.ObjectTypeMachineStatus,
	Machine:             NewEntity(SampleMachineID, constants.ObjectTypeMachine, &sampleMachineStatusMachineName, nil),
	Department:          NewEntity(SampleDepartmentID, constants.ObjectTypeDepartment, new(SampleDepartmentName), nil),
	Status:              constants.MachineWorkStatusRunning,
	WeekPlannedQuantity: 600,
	WeekScannedQuantity: 120,
	WeekPlannedRunHours: 5,
	Unit:                &sampleMachineStatusUnit,
	Current: &MachineCampaign{
		ScheduleLine:       NewEntity(SampleProductionScheduleLineID, constants.ObjectTypeProductionScheduleLine, nil, nil),
		Item:               NewEntity(SampleItemID, constants.ObjectTypeItem, nil, &sampleMachineStatusSKU),
		SKU:                sampleMachineStatusSKU,
		WeekStartDate:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
		PlannedQuantity:    360,
		ScannedQuantity:    120,
		RemainingQuantity:  240,
		Unit:               &sampleMachineStatusUnit,
		ReleasedBatchCount: 6,
		ScannedBatchCount:  2,
		PlannedRunHours:    3,
		Status:             constants.ProductionScheduleLineStatusReleased,
		ProductionRun:      NewEntity(SampleProductionRunID, constants.ObjectTypeProductionRun, nil, nil),
	},
	Next: &MachineCampaign{
		ScheduleLine:      NewEntity(SampleProductionScheduleLineID, constants.ObjectTypeProductionScheduleLine, nil, nil),
		Item:              NewEntity(SampleItemID, constants.ObjectTypeItem, nil, &sampleMachineStatusNextSKU),
		SKU:               sampleMachineStatusNextSKU,
		WeekStartDate:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
		PlannedQuantity:   240,
		ScannedQuantity:   0,
		RemainingQuantity: 240,
		Unit:              &sampleMachineStatusUnit,
		PlannedRunHours:   2,
		Status:            constants.ProductionScheduleLineStatusPlanned,
	},
}

func (*MachineStatus) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMachineStatus)
}
