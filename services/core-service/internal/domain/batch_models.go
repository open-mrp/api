package domain

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/augno/api/shared/pagination"
)

// LightUnit is a lightweight unit reference returned as a sub-resource.
type LightUnit struct {
	ID                string
	Name              string
	Abbreviation      string
	Type              string
	RatioNumerator    string
	RatioDenominator  string
	OffsetNumerator   string
	OffsetDenominator string
	IsBaseUnit        bool
	AccountID         *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// LightItem is a lightweight item reference returned as a sub-resource.
type LightItem struct {
	ID  string
	SKU string
	// Type is the item_type_code (e.g. "material", "part", "finished_good"). Populated where the query provides it; empty otherwise.
	Type string
}

// LightScanningStation is a lightweight scanning station reference returned as a sub-resource.
type LightScanningStation struct {
	ID   string
	Name string
}

// LightProductionStep is a lightweight production step reference returned as a sub-resource.
type LightProductionStep struct {
	ID   string
	Name string
}

// LightProductionRun is a lightweight production run reference returned as a sub-resource.
type LightProductionRun struct {
	ID     string
	Number string
}

// LightMachine is a lightweight machine reference returned as a sub-resource.
type LightMachine struct {
	ID   string
	Name string
}

// BatchQuantity represents a quantity with a unit.
type BatchQuantity struct {
	ID      string
	Measure decimal.Decimal
	Unit    LightUnit
}

// BatchLot represents a lot associated with a batch.
type BatchLot struct {
	LotNumber string
	Type      string // "material" or "productionRun"
}

// Batch is the full batch domain model with all associated data.
type Batch struct {
	ID              string
	Item            LightItem             `audit:"item"`
	Quantity        BatchQuantity         `audit:"quantity"`
	Seconds         *BatchQuantity        `audit:"seconds"`
	Waste           *BatchQuantity        `audit:"waste"`
	ScanningStation *LightScanningStation `audit:"scanning_station"`
	DepartmentID    *string               `audit:"department_id"`
	DepartmentName  *string               `audit:"department_name"`
	ProductionStep  *LightProductionStep  `audit:"production_step"`
	ProductionRun   *LightProductionRun   `audit:"production_run"`
	Machines        []LightMachine
	Lots            []BatchLot
	InputBatchIDs   []string
	OutputBatchIDs  []string
	ClosedAt        *time.Time `audit:"closed_at"`
	ScannedAt       *time.Time `audit:"scanned_at"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// BaseBatch is a lighter version of Batch used for mutation responses.
type BaseBatch struct {
	ID              string
	Item            LightItem             `audit:"item"`
	Quantity        BatchQuantity         `audit:"quantity"`
	Seconds         *BatchQuantity        `audit:"seconds"`
	Waste           *BatchQuantity        `audit:"waste"`
	ScanningStation *LightScanningStation `audit:"scanning_station"`
	DepartmentID    *string               `audit:"department_id"`
	DepartmentName  *string               `audit:"department_name"`
	ProductionStep  *LightProductionStep  `audit:"production_step"`
	ProductionRun   *LightProductionRun   `audit:"production_run"`
	ProductionRunID *string
	ClosedAt        *time.Time `audit:"closed_at"`
	ScannedAt       *time.Time `audit:"scanned_at"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// BatchFlowNode is a batch with its input and output batch IDs for flow graph rendering.
type BatchFlowNode struct {
	Batch          Batch
	InputBatchIDs  []string
	OutputBatchIDs []string
}

// ListBatchesByScanningStationParams holds the parameters for listing batches by scanning station.
type ListBatchesByScanningStationParams struct {
	AccountID         string
	ScanningStationID string
	Cursor            *string
	Limit             int32
	Query             *string
}

// ListBatchesByScanningStationResult holds the result of listing batches by scanning station.
type ListBatchesByScanningStationResult struct {
	Batches  []*Batch
	PageInfo pagination.PageInfo
}

// CreateBatchParams holds the parameters for creating a new batch.
type CreateBatchParams struct {
	AccountID         string
	ItemID            string
	Quantity          CreateQuantityParams
	Seconds           *CreateQuantityParams
	Waste             *CreateQuantityParams
	ProductionStepID  string
	ScanningStationID string
	ProductionRunID   string
	// MachineIDs are the machines the batch runs on. Attainment attributes production through this link, so a batch created without it is work no machine gets credit for.
	MachineIDs []string
}

// CreateQuantityParams holds the parameters for creating a quantity record.
type CreateQuantityParams struct {
	Measure decimal.Decimal
	UnitID  string
}

// MoveBatchesParams holds the parameters for moving batches.
type MoveBatchesParams struct {
	BatchIDs          []string
	ProductionStepID  string
	ScanningStationID string
}

// MergeBatchesParams holds the parameters for merging batches.
type MergeBatchesParams struct {
	BatchIDs          []string
	ScanningStationID string
	ProductionStepID  string
}

// SplitBatchParams holds the parameters for splitting a batch.
type SplitBatchParams struct {
	BatchIDs          []string
	ScanningStationID string
	ProductionStepID  string
	Firsts            BatchQuantity
	Seconds           *BatchQuantity
	Waste             *BatchQuantity
	CloseBatch        bool
}

// GetConsumptionParams holds the parameters for getting scanning station consumption.
type GetConsumptionParams struct {
	ScanningStationID string
	BatchIDs          []string
	ProductionStepID  *string
	SplitQuantity     *BatchQuantity
}

// InventorySnapshot represents a point-in-time inventory measure for an item.
type InventorySnapshot struct {
	AvailableToPromiseMeasure          decimal.Decimal
	AvailableToPromiseUnitAbbreviation string
}

// UndoBatchScanEvent is the outbox event payload for reversing the inventory a scan recorded against a batch that has just been deleted.
//
// The reversal itself is keyed off the batch: every row a scan writes carries it on `batch_id`, and those columns have no foreign key, so the tags outlive the batch row. What does not outlive it is the lineage — the flow edges go with the batch — so the seconds and waste the scan released reservations for are snapshotted here at delete time.
type UndoBatchScanEvent struct {
	BatchID           string `json:"batch_id"`
	ScanningStationID string `json:"scanning_station_id,omitempty"`
	ResponsibleUserID string `json:"responsible_user_id,omitempty"`
	OrderID           string `json:"order_id,omitempty"`
	ProducedItemID    string `json:"produced_item_id,omitempty"`
	ShortfallMeasure  string `json:"shortfall_measure,omitempty"`
	ShortfallUnitID   string `json:"shortfall_unit_id,omitempty"`
}

// LineageShortfall is the production run a batch belongs to and the scrap accumulated across its upstream lineage. A scan releases reservations for that scrap; undoing the scan puts the same amount back.
type LineageShortfall struct {
	ProductionRunID string
	Seconds         decimal.Decimal
	Waste           decimal.Decimal
}

// Total is the quantity that will never be produced, and so the quantity whose reservation moves.
func (l LineageShortfall) Total() decimal.Decimal {
	return l.Seconds.Add(l.Waste)
}

// ExecuteProductionStepEvent is the outbox event payload for executing a production step side-effect.
type ExecuteProductionStepEvent struct {
	ProductionStepID  string  `json:"production_step_id"`
	ScanningStationID string  `json:"scanning_station_id"`
	ItemID            string  `json:"item_id"`
	BatchQuantityID   string  `json:"batch_quantity_id"`
	BatchMeasure      string  `json:"batch_measure"`
	BatchUnitID       string  `json:"batch_unit_id"`
	ResponsibleUserID *string `json:"responsible_user_id,omitempty"`
	ProducedBatchID   *string `json:"produced_batch_id,omitempty"`
	ProduceInventory  bool    `json:"produce_inventory"`
}
