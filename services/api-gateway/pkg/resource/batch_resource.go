package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// ---------------------------------------------------------------------------
// Sample IDs
// ---------------------------------------------------------------------------

const SampleBatchID = "bt_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleScanningStationID = "scst_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleProductionStepID = "prst_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleProductionRunID = "prru_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleMachineID = "mc_01jm4r6700f8nwq3v5hx2d9ktp"

// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// ProductionRun — production run reference
// ---------------------------------------------------------------------------

// ProductionRun represents a production run sub-resource.
type ProductionRun struct {
	// The unique identifier for the production run.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_run"`
	// The production run number.
	Number string `json:"number" validate:"required"`
}

var SampleProductionRun = &ProductionRun{
	ID:     SampleProductionRunID,
	Object: constants.ObjectTypeProductionRun,
	Number: "PR-001",
}

func (*ProductionRun) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionRun)
}

// ---------------------------------------------------------------------------
// Batch — full batch resource
// ---------------------------------------------------------------------------

// BatchLot represents a lot associated with a batch.
type BatchLot struct {
	// The lot number.
	LotNumber string `json:"lot_number" validate:"required"`
	// The lot type (material or productionRun).
	Type string `json:"type" validate:"required"`
}

// Batch represents a production batch.
type Batch struct {
	// The unique identifier for the batch.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=batch"`
	// The item associated with this batch.
	Item *Item `json:"item" expandable:"true"`
	// The quantity produced in this batch.
	Quantity *Quantity `json:"quantity" expandable:"true"`
	// The time measurement for this batch in seconds.
	Seconds *Quantity `json:"seconds" expandable:"true"`
	// The waste measurement for this batch.
	Waste *Quantity `json:"waste" expandable:"true"`
	// The scanning station where this batch was scanned.
	ScanningStation *ScanningStation `json:"scanning_station" expandable:"true"`
	// The department associated with this batch's scanning station.
	Department *Department `json:"department" expandable:"true"`
	// The production step this batch belongs to.
	ProductionStep *ProductionStep `json:"production_step" expandable:"true"`
	// The production run this batch belongs to.
	ProductionRun *ProductionRun `json:"production_run" expandable:"true"`
	// The machines used for this batch.
	Machines *List[Machine] `json:"machines" expandable:"true"`
	// The lots associated with this batch.
	Lots *List[BatchLot] `json:"lots"`
	// The IDs of batches that feed into this batch.
	InputBatchIDs []string `json:"input_batch_ids"`
	// The IDs of batches that this batch feeds into.
	OutputBatchIDs []string `json:"output_batch_ids"`
	// The timestamp when the batch was closed.
	ClosedAt *time.Time `json:"closed_at"`
	// The timestamp when the batch was scanned.
	ScannedAt *time.Time `json:"scanned_at"`
	// The timestamp when the batch was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the batch was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleScannedAt = timeutil.TimestampToTime(sampleCreatedAtTimestamp)

var SampleBatch = &Batch{
	ID:              SampleBatchID,
	Object:          constants.ObjectTypeBatch,
	Item:            SampleItem,
	Quantity:        SampleQuantity,
	Seconds:         SampleQuantity,
	Waste:           SampleQuantity,
	ScanningStation: SampleScanningStation,
	Department:      SampleDepartment,
	ProductionStep:  nil,
	ProductionRun:   SampleProductionRun,
	Machines:        NewList([]Machine{*SampleMachine}, PageInfo{}),
	Lots:            NewList([]BatchLot{}, PageInfo{}),
	InputBatchIDs:   []string{},
	OutputBatchIDs:  []string{},
	ClosedAt:        nil,
	ScannedAt:       &sampleScannedAt,
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Batch) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleBatch)
}

// ---------------------------------------------------------------------------
// BatchFlowNode — batch with flow graph edges
// ---------------------------------------------------------------------------

// BatchFlowNode represents a batch within a production flow graph, including its input and output edges.
type BatchFlowNode struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=batch_flow_node"`
	// The batch at this node.
	Batch Batch `json:"batch"`
	// The IDs of batches that feed into this batch.
	InputBatchIDs []string `json:"input_batch_ids"`
	// The IDs of batches that this batch feeds into.
	OutputBatchIDs []string `json:"output_batch_ids"`
}

var SampleBatchFlowNode = &BatchFlowNode{
	Object:         constants.ObjectTypeBatchFlowNode,
	Batch:          *SampleBatch,
	InputBatchIDs:  []string{},
	OutputBatchIDs: []string{},
}

func (*BatchFlowNode) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleBatchFlowNode)
}

// ---------------------------------------------------------------------------
// ScanningConsumption — consumption data for a scanning operation
// ---------------------------------------------------------------------------

// ScanningConsumption represents the material consumption data for a scanning operation.
type ScanningConsumption struct {
	// The stock keeping unit code.
	SKU string `json:"sku" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=scanning_consumption"`
	// The demand measure value.
	DemandMeasure string `json:"demand_measure" validate:"required" format:"decimal"`
	// The demand unit abbreviation.
	DemandUnit string `json:"demand_unit" validate:"required"`
	// The inventory measure value.
	InventoryMeasure string `json:"inventory_measure" validate:"required" format:"decimal"`
	// The inventory unit abbreviation.
	InventoryUnit string `json:"inventory_unit" validate:"required"`
	// Optional instructions for this consumption.
	Instructions *string `json:"instructions"`
}

var SampleScanningConsumption = &ScanningConsumption{
	SKU:              SampleItemSKU,
	Object:           constants.ObjectTypeScanningConsumption,
	DemandMeasure:    "10.000000000000000000000000000000",
	DemandUnit:       "kg",
	InventoryMeasure: "100.000000000000000000000000000000",
	InventoryUnit:    "kg",
	Instructions:     nil,
}

func (*ScanningConsumption) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleScanningConsumption)
}

// ---------------------------------------------------------------------------
// OpenBatchSummary — summary of open batches
// ---------------------------------------------------------------------------

// OpenBatchSummary represents an aggregated summary of open batches.
type OpenBatchSummary struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=open_batch_summary"`
	// The department name.
	DepartmentName string `json:"department_name" validate:"required"`
	// The item associated with this summary.
	Item *Item `json:"item" validate:"required"`
	// The scanning station associated with this summary.
	ScanningStation *ScanningStation `json:"scanning_station" validate:"required"`
	// The count of open batches.
	Count string `json:"count" validate:"required" format:"decimal"`
	// The unit abbreviation.
	Unit string `json:"unit" validate:"required"`
}

var SampleOpenBatchSummary = &OpenBatchSummary{
	Object:         constants.ObjectTypeOpenBatchSummary,
	DepartmentName: "Production",
	Item: &Item{
		ID:     SampleItemID,
		Object: constants.ObjectTypeItem,
		SKU:    SampleItemSKU,
	},
	ScanningStation: SampleScanningStation,
	Count:           "5.000000000000000000000000000000",
	Unit:            "kg",
}

func (*OpenBatchSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOpenBatchSummary)
}

// ---------------------------------------------------------------------------
// ScanningProductionStepInfo — production step info for scanning
// ---------------------------------------------------------------------------

// ScanningProductionStepInfo provides production step information for the scanning next-steps response.
type ScanningProductionStepInfo struct {
	// The unique identifier for the production step.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=scanning_production_step_info"`
	// The display name of the production step.
	Name string `json:"name" validate:"required"`
	// Whether this production step supports multi-part batches.
	IsMultiPart bool `json:"is_multi_part"`
}

var SampleScanningProductionStepInfo = &ScanningProductionStepInfo{
	ID:          SampleProductionStepID,
	Object:      constants.ObjectTypeScanningProductionStepInfo,
	Name:        "Mixing",
	IsMultiPart: false,
}

func (*ScanningProductionStepInfo) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleScanningProductionStepInfo)
}
