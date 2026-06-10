package domain

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/augno/api/shared/pagination"
)

// Item represents an inventory item (product, material, or part).
type Item struct {
	ID             string
	SKU            string  `audit:"sku"`
	Description    *string `audit:"description"`
	Notes          *string `audit:"notes"`
	ItemTypeCode   string  `audit:"item_type_code"`
	ItemCategoryID string  `audit:"item_category_id"`
	CategoryName   string  `audit:"category_name"`
	UnitValueID    string
	UnitCostID     string
	BurnRateID     string
	AccountID      string
	IsDirty        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time

	// Joined data (populated when listing/getting items)
	UnitValue *Rate
	UnitCost  *Rate
	BurnRate  *Rate
	Category  *ItemCategory

	// Many-to-many joined data
	Attributes []*ItemAttribute `audit:"attributes"`
}

// ItemCategory represents an item category (joined data).
type ItemCategory struct {
	ID                   string
	Name                 string
	ItemCategoryTypeCode string
	UnitGroupID          string
	CreatedAt            time.Time
	UpdatedAt            time.Time

	// Joined from unit_group when loading items/materials/products/parts.
	UnitGroupName       string
	UnitGroupTypeCode   string
	UnitGroupBaseUnitID string
	UnitGroupCreatedAt  time.Time
	UnitGroupUpdatedAt  time.Time
	Properties          []ItemCategoryProperty

	// Populated when category.unit_group.base_unit is included.
	UnitGroupBaseUnit *LightUnit
	// Populated when category.unit_group.associated_units is included.
	UnitGroupAssociatedUnits []*UnitGroupUnit
}

// ItemAttribute represents an attribute on an item (joined via _item_attributes).
type ItemAttribute struct {
	ID         string
	Value      string
	ColorCode  *string
	Order      int32
	PropertyID string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Rate represents a rate value (unit_value, unit_cost, or burn_rate).
type Rate struct {
	ID                               string
	Value                            string `audit:"value"`
	NumeratorUnitID                  string `audit:"numerator_unit_id"`
	NumeratorUnitName                string `audit:"numerator_unit_name"`
	NumeratorUnitAbbreviation        string `audit:"numerator_unit_abbreviation"`
	NumeratorUnitType                string `audit:"numerator_unit_type"`
	NumeratorUnitRatioNumerator      string
	NumeratorUnitRatioDenominator    string
	NumeratorUnitOffsetNumerator     string
	NumeratorUnitOffsetDenominator   string
	NumeratorUnitCreatedAt           time.Time
	NumeratorUnitUpdatedAt           time.Time
	DenominatorUnitID                string `audit:"denominator_unit_id"`
	DenominatorUnitName              string `audit:"denominator_unit_name"`
	DenominatorUnitAbbreviation      string `audit:"denominator_unit_abbreviation"`
	DenominatorUnitType              string `audit:"denominator_unit_type"`
	DenominatorUnitRatioNumerator    string
	DenominatorUnitRatioDenominator  string
	DenominatorUnitOffsetNumerator   string
	DenominatorUnitOffsetDenominator string
	DenominatorUnitCreatedAt         time.Time
	DenominatorUnitUpdatedAt         time.Time
	CreatedAt                        time.Time
	UpdatedAt                        time.Time
}

type ListItemsParams struct {
	AccountID                string
	Cursor                   *string
	Limit                    int32
	Query                    *string
	Types                    []string
	CategoryIDs              []string
	AttributeIDs             []string
	SupplierID               *string
	StartDate                *time.Time
	EndDate                  *time.Time
	IsExactMatch             bool
	OnlyInitialSubassemblies bool
	Includes                 []string
	ProductLineIDs           []string
	CustomerIDs              []string
}

type ListItemsResult struct {
	Items    []*Item
	PageInfo pagination.PageInfo
}

type GetItemParams struct {
	AccountID string
	ItemID    string
	Includes  []string
}

// ItemInventory represents inventory quantities for an item.
type ItemInventory struct {
	OnHand             string
	OnHandUnitID       string
	Reserved           string
	ReservedUnitID     string
	AvailableToPromise string
	ATPUnitID          string
	Short              string
	ShortUnitID        string
	UnitAbbreviation   string
	UnitType           string
}

// ItemCosts represents cost breakdown for an item.
type ItemCosts struct {
	DirectMaterialCost string
	DirectLaborCost    string
	OverheadCost       string
	TotalCost          string
	UnitID             string
}

// CostFlowConsumption represents a single consumption with cost-relevant data.
type CostFlowConsumption struct {
	ConsumedItemType    string          // e.g. "material", "part", "product"
	ConsumptionQuantity decimal.Decimal // quantity consumed
	WasteQuantity       decimal.Decimal // waste quantity
	UnitCost            decimal.Decimal // consumed item's unit cost
}

// ItemTrend represents a single trend data point.
type ItemTrend struct {
	Date  time.Time
	Value string
}

// ItemTrends represents historical trend data for an item.
type ItemTrends struct {
	TrendType string
	Points    []*ItemTrend
}

// ExportItemsResult represents the export response.
type ExportItemsResult struct {
	Items []*ExportItem
	Count int64
}

// ExportItem represents an item with inventory for export.
type ExportItem struct {
	Item
	OnHandQuantity string
	OnHandUnitID   string
}

// UpdateItemParams holds parameters for partially updating an item.
type UpdateItemParams struct {
	AccountID         string
	ItemID            string
	SKU               *string
	Description       *string
	UpdateDescription bool // true if the caller explicitly set the description field (even to null)
	Notes             *string
	UpdateNotes       bool // true if the caller explicitly set the notes field (even to null)
}

// AddItemAttributeParams holds parameters for adding an attribute to an item.
type AddItemAttributeParams struct {
	AccountID   string
	ItemID      string
	AttributeID string
}

// RemoveItemAttributeParams holds parameters for removing an attribute from an item.
type RemoveItemAttributeParams struct {
	AccountID   string
	ItemID      string
	AttributeID string
}

// ChangeItemCategoryParams holds parameters for changing an item's category.
type ChangeItemCategoryParams struct {
	AccountID  string
	ItemID     string
	CategoryID string
}

// UpdateItemInventoryParams holds parameters for updating item inventory.
type UpdateItemInventoryParams struct {
	AccountID      string
	ItemID         string
	QuantityChange *float64
	Reconcile      *bool
	CustomerID     *string
	LocationID     *string
	LotNumber      *string
	UnitID         *string
}

// BulkCreateItemsParams holds parameters for bulk creating items.
type BulkCreateItemsParams struct {
	AccountID string
	Items     []BulkCreateItemInput
	Type      string
}

// BulkCreateItemInput represents a single item to create in a bulk operation.
type BulkCreateItemInput struct {
	SKU            string
	Description    *string
	ItemCategoryID string
	ProductLineID  *string
	// AttributeIDs are connected to the new (or existing upserted) item in the same tx.
	AttributeIDs []string
}

// BulkCreateItemResult represents the result of creating a single item in a bulk operation.
type BulkCreateItemResult struct {
	SKU     string
	Success bool
	Error   *string
	ItemID  *string
}

// BulkReconcileItemsParams holds parameters for bulk reconciling item inventory.
type BulkReconcileItemsParams struct {
	AccountID         string
	Data              []BulkReconcileItemInput
	ReconcileType     string // "addition" or "force"
	ResponsibleUserID *string
}

// BulkReconcileItemInput represents a single item to reconcile.
type BulkReconcileItemInput struct {
	SKU      string
	Unit     string
	Quantity float64
}

// BulkReconcileItemsResult holds the results of a bulk reconciliation.
type BulkReconcileItemsResult struct {
	ReconciledItems []ReconciledItem
	SkippedItems    []SkippedItem
	Errors          []ReconcileError
}

// ReconciledItem represents a successfully reconciled item.
type ReconciledItem struct {
	ItemID           string
	SKU              string
	PreviousQuantity float64
	NewQuantity      float64
}

// SkippedItem represents an item that was skipped during reconciliation.
type SkippedItem struct {
	SKU    string
	Reason string
}

// ReconcileError represents an error that occurred during reconciliation.
type ReconcileError struct {
	SKU   string
	Error string
}

// InventoryItemResult represents an item with its on-hand inventory quantity.
type InventoryItemResult struct {
	Item             *Item
	OnHandQuantity   float64
	OnHandUnitID     string
	OnHandUnitAbbrev string
	OnHandUnitType   string
}

// ListInventoriesParams contains the parameters for listing inventories.
type ListInventoriesParams struct {
	Cursor    *string
	Limit     int32
	Query     *string
	AccountID string
}

// ListInventoriesResult represents the result of listing all items with inventory.
type ListInventoriesResult struct {
	Items    []*InventoryItemResult
	Count    int64
	PageInfo pagination.PageInfo
}

// ItemSKUInfo represents minimal item info fetched by SKU for bulk operations.
type ItemSKUInfo struct {
	SKU        string
	ItemID     string
	BaseUnitID string
}

// BulkOnHandInventory represents bulk on-hand inventory for multiple items.
type BulkOnHandInventory struct {
	ItemID           string
	OnHandQuantity   float64
	UnitID           string
	UnitAbbreviation string
	UnitType         string
}

// BulkCreateProductionStepsParams holds parameters for bulk creating production steps.
type BulkCreateProductionStepsParams struct {
	AccountID string
	Steps     []BulkCreateProductionStepInput
}

// BulkCreateConsumptionInput represents a consumption input in a bulk create operation (resolved by SKU).
type BulkCreateConsumptionInput struct {
	SKU          string
	Measure      string
	Instructions *string
}

// BulkCreateProductionInput represents a production output input in a bulk create operation (resolved by SKU).
type BulkCreateProductionInput struct {
	SKU     string
	Measure string
}

// BulkCreateProductionStepInput represents a single production step to create in a bulk operation.
type BulkCreateProductionStepInput struct {
	Name           string
	Consumptions   []BulkCreateConsumptionInput
	Productions    []BulkCreateProductionInput
	LaborRate      string
	LaborTime      string
	LaborTimeUnit  *string
	OverheadRate   string
	Allowances     *string
	LevelingFactor *string
	Station        *string
}

// BulkCreateProductionStepResult represents the result of creating a single production step.
type BulkCreateProductionStepResult struct {
	Name             string
	Success          bool
	Error            *string
	ProductionStepID *string
	Action           string // "created", "updated", or "skipped"
}

// BurnRateConsumptionLog is a single negative consumption entry used to compute burn rate.
type BurnRateConsumptionLog struct {
	Value     string
	UnitID    string
	CreatedAt time.Time
}
