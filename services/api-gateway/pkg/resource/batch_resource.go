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

const SampleBatchID = "bt_017313a7df2d7ac8d895809747"
const SampleScanningStationID = "scst_0129335dd6286056a97024fcc1"
const SampleProductionStepID = "prst_0159474175bb59f4b1990404ee"
const SampleProductionRunID = "prru_0141c28081df4faac0fe726c41"
const SampleMachineID = "mc_0177d18f55a1615f783d3bf8d0"

// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// ProductionRun — production run reference
// ---------------------------------------------------------------------------

// Production run sub-resource.
type ProductionRun struct {
	// Production run ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_run"`
	// Production run number.
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

// Lot associated with a batch.
type BatchLot struct {
	// Lot number.
	LotNumber string `json:"lot_number" validate:"required"`
	// Lot type (material or productionRun).
	Type string `json:"type" validate:"required"`
}

// Production batch.
type Batch struct {
	// Batch ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=batch"`
	// Item.
	Item *Item `json:"item" expandable:"true"`
	// Quantity produced.
	Quantity *Quantity `json:"quantity" expandable:"true"`
	// Time measurement in seconds.
	Seconds *Quantity `json:"seconds" expandable:"true"`
	// Waste measurement.
	Waste *Quantity `json:"waste" expandable:"true"`
	// Scanning station.
	ScanningStation *ScanningStation `json:"scanning_station" expandable:"true"`
	// Department of the scanning station.
	Department *Department `json:"department" expandable:"true"`
	// Production step.
	ProductionStep *ProductionStep `json:"production_step" expandable:"true"`
	// Production run.
	ProductionRun *ProductionRun `json:"production_run" expandable:"true"`
	// Machines used.
	Machines *List[Machine] `json:"machines" expandable:"true"`
	// Associated lots.
	Lots *List[BatchLot] `json:"lots"`
	// Input batch IDs.
	InputBatchIDs []string `json:"input_batch_ids"`
	// Output batch IDs.
	OutputBatchIDs []string `json:"output_batch_ids"`
	// Closed timestamp.
	ClosedAt *time.Time `json:"closed_at"`
	// Scanned timestamp.
	ScannedAt *time.Time `json:"scanned_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last-updated timestamp.
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

// Batch within a production flow graph, including input and output edges.
type BatchFlowNode struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=batch_flow_node"`
	// Batch at this node.
	Batch Batch `json:"batch"`
	// IDs of batches that feed into this batch.
	InputBatchIDs []string `json:"input_batch_ids"`
	// IDs of batches this batch feeds into.
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

// Material consumption data for a scanning operation.
type ScanningConsumption struct {
	// SKU.
	SKU string `json:"sku" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=scanning_consumption"`
	// Demand measure value.
	DemandMeasure string `json:"demand_measure" validate:"required" format:"decimal"`
	// Demand unit abbreviation.
	DemandUnit string `json:"demand_unit" validate:"required"`
	// Inventory measure value.
	InventoryMeasure string `json:"inventory_measure" validate:"required" format:"decimal"`
	// Inventory unit abbreviation.
	InventoryUnit string `json:"inventory_unit" validate:"required"`
	// Consumption instructions.
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

// Aggregated summary of open batches.
type OpenBatchSummary struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=open_batch_summary"`
	// Department name.
	DepartmentName string `json:"department_name" validate:"required"`
	// Item associated with this summary.
	Item *Item `json:"item" validate:"required"`
	// Scanning station associated with this summary.
	ScanningStation *ScanningStation `json:"scanning_station" validate:"required"`
	// Count of open batches.
	Count string `json:"count" validate:"required" format:"decimal"`
	// Unit abbreviation.
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

// Production step information for the scanning next-steps response.
type ScanningProductionStepInfo struct {
	// Production step ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=scanning_production_step_info"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Whether this step supports multi-part batches.
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
