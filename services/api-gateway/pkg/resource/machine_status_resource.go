package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// One campaign on a machine, with how far through it the floor is.
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
	// Constraint hours the campaign consumes.
	PlannedRunHours float64 `json:"planned_run_hours"`
	// Where the campaign is in its lifecycle.
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
	// The campaign being worked: the earliest released one with batches still to scan.
	Current *MachineCampaign `json:"current"`
	// What follows, so an operator can set up for it.
	Next *MachineCampaign `json:"next"`
	// Planned for this machine this week.
	WeekPlannedQuantity float64 `json:"week_planned_quantity"`
	// Scanned on this machine this week.
	WeekScannedQuantity float64 `json:"week_scanned_quantity"`
	// Constraint hours planned on this machine this week.
	WeekPlannedRunHours float64 `json:"week_planned_run_hours"`
	// Unit the week's quantities are counted in.
	Unit *string `json:"unit"`
}

var (
	sampleMachineStatusUnit        = "pr"
	sampleMachineStatusMachineName = "Merz 1"
	sampleMachineStatusSKU         = "MZ-GREIGE-CREW"
)

var SampleMachineStatus = &MachineStatus{
	Object:              constants.ObjectTypeMachineStatus,
	Machine:             NewEntity(SampleMachineID, constants.ObjectTypeMachine, &sampleMachineStatusMachineName, nil),
	Status:              constants.MachineWorkStatusRunning,
	WeekPlannedQuantity: 360,
	WeekScannedQuantity: 120,
	WeekPlannedRunHours: 60,
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
		PlannedRunHours:    60,
		Status:             constants.ProductionScheduleLineStatusReleased,
	},
}

func (*MachineStatus) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMachineStatus)
}
