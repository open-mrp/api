package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const (
	SampleManufacturingMethodID = "pnmd_vlblgqvnciwk"
	SampleMethodOperationID     = "mdoe_xxg8pkhbylow"
	SampleMethodMaterialID      = "mdml_fmzfzf0e8dn3"
)

// How an item is made: the operations it goes through and the materials it consumes.
//
// Methods are versioned. Edit a draft, then activate it; each item has one active method, and production runs keep the version they were released with.
type ManufacturingMethod struct {
	// Manufacturing method ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=manufacturing_method"`
	// The item this method makes.
	//
	// Null unless included.
	Item *Item `json:"item" expandable:"true"`
	// The method's version number for its item, starting at 1. Each new version of an item's method takes the next number.
	Version int32 `json:"version" validate:"required"`
	// `draft`: editable, not in use. `active`: the method used for new production of this item. `archived`: a previous version, kept for history.
	Status constants.ManufacturingMethodStatus `json:"status" validate:"required"`
	// How much of the item one run of the method makes. Material quantities and labor and machine times are per this much output. Defaults to one of the item's stocking unit.
	OutputQuantity *Quantity `json:"output_quantity" validate:"required"`
	// The sales order line this method was made for, when it is specific to one order.
	//
	// Always null for now: every method is a standard method of its item.
	SalesOrderLine *SalesOrderLine `json:"sales_order_line" expandable:"true"`
	// The method's operations, in order.
	//
	// Null unless included. Shows the first 10; `page_info.next_page_url` continues the list.
	Operations *List[MethodOperation] `json:"operations" expandable:"true"`
	// The materials the method's operations consume.
	//
	// Null unless included. Shows the first 10; `page_info.next_page_url` continues the list.
	Materials *List[MethodMaterial] `json:"materials" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// One step of work in a method, done at a work center with its machines and scanning station.
//
// `sequence` says whether it starts after the previous operation or alongside it.
type MethodOperation struct {
	// Operation ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=method_operation"`
	// Where the operation comes in its method, starting at 1.
	Position int32 `json:"position" validate:"required"`
	// When the operation starts.
	//
	// - `after_previous`: once the previous operation is done.
	// - `with_previous`: alongside the previous operation.
	Sequence constants.OperationSequence `json:"sequence" validate:"required"`
	// Who does the work. `inside`: done in-house at the operation's work center.
	Type constants.OperationType `json:"type" validate:"required"`
	// The work center the operation is done at. Always set for an `inside` operation.
	//
	// Null unless included.
	WorkCenter *WorkCenter `json:"work_center" expandable:"true"`
	// The machines the operation runs on.
	//
	// Null unless included.
	Machines *List[Machine] `json:"machines" expandable:"true"`
	// The scanning station the operation's batches are scanned at.
	//
	// Null unless included.
	ScanningStation *ScanningStation `json:"scanning_station" expandable:"true"`
	// Time to set up for one run of the operation, such as `0.5 hr`, whatever the run's size. Null when there is none.
	SetupTime *Quantity `json:"setup_time"`
	// Labor time per unit of output, such as `0.02 hr / pr`.
	//
	// Effective time is `labor_time × (1 + leveling_percent ÷ 100) × (1 + allowance_percent ÷ 100)`.
	LaborTime *Rate `json:"labor_time"`
	// Machine time per unit of output, such as `0.01 hr / pr`. Null when not tracked.
	MachineTime *Rate `json:"machine_time"`
	// Cost of an hour of labor, such as `$20 / hr`. Defaults to the work center's labor rate.
	//
	// Null unless the caller holds `costs:read`; customer and supplier portal users never see it.
	LaborRate *UnitPrice `json:"labor_rate" sensitive:"cost"`
	// Overhead cost per unit of time, such as `$5 / hr`.
	//
	// Null unless the caller holds `costs:read`; customer and supplier portal users never see it.
	OverheadRate *UnitPrice `json:"overhead_rate" sensitive:"cost"`
	// Correction to labor time for the pace of the work, as a whole-number percent such as `"15"` for 15%.
	LevelingPercent string `json:"leveling_percent" validate:"required" format:"decimal"`
	// Allowance added to labor time for breaks and fatigue, as a whole-number percent such as `"10"` for 10%.
	AllowancePercent string `json:"allowance_percent" validate:"required" format:"decimal"`
	// Instructions for the people doing the operation.
	WorkInstructions *string `json:"work_instructions"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// An item an operation consumes, and how much per the method's output quantity.
//
// `scrap_quantity` is the expected loss on top of `quantity`. `supply` says where it comes from: `make` (its own method), `stock` (inventory) or `buy` (purchased for the run).
type MethodMaterial struct {
	// Material ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=method_material"`
	// The item consumed.
	//
	// Null unless included.
	Item *Item `json:"item" expandable:"true"`
	// How much of the item is consumed per the method's output quantity.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// The expected loss on top of `quantity`.
	ScrapQuantity *Quantity `json:"scrap_quantity" validate:"required"`
	// The operation that consumes the material.
	//
	// Null unless included.
	Operation *MethodOperation `json:"operation" expandable:"true"`
	// Where the material comes from.
	//
	// - `make`: production pegged to this work, by the material's own method.
	// - `stock`: taken from inventory, however it was replenished.
	// - `buy`: purchased for this work.
	Supply constants.Supply `json:"supply" validate:"required"`
	// Instructions for handling the material in the operation.
	WorkInstructions *string `json:"work_instructions"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleWorkInstructions = "Trim loose threads before passing the batch on."

var SampleMethodOperation = &MethodOperation{
	ID:               SampleMethodOperationID,
	Object:           constants.ObjectTypeMethodOperation,
	Position:         1,
	Sequence:         constants.OperationSequenceAfterPrevious,
	Type:             constants.OperationTypeInside,
	WorkCenter:       SampleWorkCenter,
	Machines:         NewList([]Machine{*SampleMachine}, PageInfo{}),
	ScanningStation:  SampleScanningStation,
	SetupTime:        SampleQuantity,
	LaborTime:        SampleRate,
	MachineTime:      SampleRate,
	LaborRate:        SampleUnitPrice,
	OverheadRate:     SampleUnitPrice,
	LevelingPercent:  "15",
	AllowancePercent: "10",
	WorkInstructions: &sampleWorkInstructions,
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

var SampleMethodMaterial = &MethodMaterial{
	ID:               SampleMethodMaterialID,
	Object:           constants.ObjectTypeMethodMaterial,
	Item:             SampleItem,
	Quantity:         SampleQuantity,
	ScrapQuantity:    SampleQuantity,
	Operation:        SampleMethodOperation,
	Supply:           constants.SupplyStock,
	WorkInstructions: &sampleWorkInstructions,
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

var SampleManufacturingMethod = &ManufacturingMethod{
	ID:             SampleManufacturingMethodID,
	Object:         constants.ObjectTypeManufacturingMethod,
	Item:           SampleItem,
	Version:        2,
	Status:         constants.ManufacturingMethodStatusActive,
	OutputQuantity: SampleQuantity,
	Operations:     NewList([]MethodOperation{*SampleMethodOperation}, PageInfo{}),
	Materials:      NewList([]MethodMaterial{*SampleMethodMaterial}, PageInfo{}),
	CreatedAt:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:      timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ManufacturingMethod) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleManufacturingMethod)
}

func (*MethodOperation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMethodOperation)
}

func (*MethodMaterial) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMethodMaterial)
}
