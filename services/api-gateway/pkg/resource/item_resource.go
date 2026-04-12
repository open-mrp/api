package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleItemID = "it_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleItemSKU = "ALM-2024-1001"

// Item represents an inventory item (product, material, or part).
type Item struct {
	// The unique identifier for the item.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// The stock keeping unit code.
	SKU string `json:"sku" validate:"required"`
	// A description of the item.
	Description *string `json:"description"`
	// Additional notes about the item.
	Notes *string `json:"notes"`
	// The item type code.
	ItemTypeCode constants.ItemTypeCode `json:"type" validate:"required"`
	// The item category.
	Category *ItemCategory `json:"category" expandable:"true"`
	// The unit value rate for this item.
	UnitValue *Rate `json:"unit_value" expandable:"true"`
	// The unit cost rate for this item.
	UnitCost *Rate `json:"unit_cost" expandable:"true"`
	// The burn rate for this item.
	BurnRate *Rate `json:"burn_rate" expandable:"true"`
	// The attributes assigned to this item.
	Attributes *List[Attribute] `json:"attributes"`
	// Whether the item has unsaved changes.
	IsDirty bool `json:"is_dirty"`
	// The timestamp when the item was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the item was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleDescription = "6061-T6 aluminum sheet, 4ft x 8ft, 0.125in thick"

var SampleItem = &Item{
	ID:           SampleItemID,
	Object:       constants.ObjectTypeItem,
	SKU:          SampleItemSKU,
	Description:  &sampleDescription,
	Notes:        nil,
	ItemTypeCode: constants.ItemTypeCodeProduct,
	Category:     SampleItemCategory,
	UnitValue:    SampleRate,
	UnitCost:     SampleRate,
	BurnRate:     SampleRate,
	Attributes:   NewList([]Attribute{*SampleAttribute}, PageInfo{}),
	IsDirty:      false,
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Item) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleItem)
}

// ItemInventory represents inventory quantities for an item.
type ItemInventory struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// The on-hand quantity and its unit.
	OnHand *QuantityInfo `json:"on_hand" validate:"required"`
	// The reserved quantity and its unit.
	Reserved *QuantityInfo `json:"reserved" validate:"required"`
	// The available-to-promise quantity and its unit.
	AvailableToPromise *QuantityInfo `json:"available_to_promise" validate:"required"`
	// The short quantity and its unit.
	Short *QuantityInfo `json:"short" validate:"required"`
}

// QuantityInfo represents a quantity with its associated unit.
type QuantityInfo struct {
	// The decimal quantity value.
	Value string `json:"value" validate:"required" format:"decimal"`
	// The unit for this quantity.
	Unit *Unit `json:"unit"`
}

var SampleQuantityInfo = &QuantityInfo{
	Value: "100.000000000000000000000000000000",
	Unit:  SampleUnit,
}

var SampleItemInventory = &ItemInventory{
	Object:             constants.ObjectTypeItem,
	OnHand:             SampleQuantityInfo,
	Reserved:           SampleQuantityInfo,
	AvailableToPromise: SampleQuantityInfo,
	Short:              SampleQuantityInfo,
}

func (*ItemInventory) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleItemInventory)
}

// ItemCosts represents cost breakdown for an item.
type ItemCosts struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// The direct material cost.
	DirectMaterialCost string `json:"direct_material_cost" validate:"required" format:"decimal"`
	// The direct labor cost.
	DirectLaborCost string `json:"direct_labor_cost" validate:"required" format:"decimal"`
	// The overhead cost.
	OverheadCost string `json:"overhead_cost" validate:"required" format:"decimal"`
	// The total cost.
	TotalCost string `json:"total_cost" validate:"required" format:"decimal"`
	// The unit for cost values.
	Unit *Unit `json:"unit"`
}

var SampleItemCosts = &ItemCosts{
	Object:             constants.ObjectTypeItem,
	DirectMaterialCost: "5.000000000000000000000000000000",
	DirectLaborCost:    "3.000000000000000000000000000000",
	OverheadCost:       "2.000000000000000000000000000000",
	TotalCost:          "10.000000000000000000000000000000",
	Unit:               SampleUnit,
}

func (*ItemCosts) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleItemCosts)
}

// ItemTrendPoint represents a single trend data point.
type ItemTrendPoint struct {
	// The date of the trend data point.
	OccurredAt time.Time `json:"occurred_at" validate:"required"`
	// The value at this date.
	Value string `json:"value" validate:"required" format:"decimal"`
}

// ItemTrends represents historical trend data for an item.
type ItemTrends struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// The trend type that was requested.
	TrendType string `json:"trend_type" validate:"required"`
	// The trend data points.
	Points *List[ItemTrendPoint] `json:"points" validate:"required"`
}

var SampleItemTrends = &ItemTrends{
	Object:    constants.ObjectTypeItem,
	TrendType: "on_hand",
	Points: NewList([]ItemTrendPoint{
		{
			OccurredAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
			Value:      "100.000000000000000000000000000000",
		},
	}, PageInfo{}),
}

func (*ItemTrends) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleItemTrends)
}

// ExportItem represents an item with inventory for export.
type ExportItem struct {
	// The unique identifier for the item.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// The stock keeping unit code.
	SKU string `json:"sku" validate:"required"`
	// A description of the item.
	Description *string `json:"description"`
	// Additional notes about the item.
	Notes *string `json:"notes"`
	// The item type code.
	ItemTypeCode constants.ItemTypeCode `json:"type" validate:"required"`
	// The category name.
	CategoryName string `json:"category_name"`
	// The on-hand quantity.
	OnHandQuantity string `json:"on_hand_quantity" format:"decimal"`
	// The on-hand unit.
	OnHandUnit *Unit `json:"on_hand_unit"`
	// The timestamp when the item was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the item was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleExportItem = &ExportItem{
	ID:             SampleItemID,
	Object:         constants.ObjectTypeItem,
	SKU:            SampleItemSKU,
	Description:    &sampleDescription,
	Notes:          nil,
	ItemTypeCode:   constants.ItemTypeCodeProduct,
	CategoryName:   SampleItemCategoryName,
	OnHandQuantity: "100.000000000000000000000000000000",
	OnHandUnit:     SampleUnit,
	CreatedAt:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:      timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ExportItem) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleExportItem)
}

// ExportItemsResponse represents the export items response when returning JSON.
// Export endpoints may use arrays; they typically return an Excel file instead.
type ExportItemsResponse struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// The exported items.
	Items []*ExportItem `json:"items" validate:"required"`
	// The total count of exported items.
	Count int64 `json:"count"`
}

var SampleExportItemsResponse = &ExportItemsResponse{
	Object: constants.ObjectTypeList,
	Items:  []*ExportItem{SampleExportItem},
	Count:  1,
}

func (*ExportItemsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleExportItemsResponse)
}

// BulkReconcileItemsResponse represents the response from bulk reconciling items.
type BulkReconcileItemsResponse struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=bulk_reconcile_items_response"`
	// The reconciled items.
	ReconciledItems []ReconciledItemResult `json:"reconciled_items" validate:"required"`
	// The skipped items.
	SkippedItems []SkippedItemResult `json:"skipped_items" validate:"required"`
	// The errors.
	Errors []ReconcileErrorResult `json:"errors" validate:"required"`
}

// ReconciledItemResult represents a successfully reconciled item.
type ReconciledItemResult struct {
	// The item ID.
	ItemID string `json:"item_id" validate:"required"`
	// The SKU.
	SKU string `json:"sku" validate:"required"`
	// The previous quantity.
	PreviousQuantity float64 `json:"previous_quantity" validate:"required"`
	// The new quantity.
	NewQuantity float64 `json:"new_quantity" validate:"required"`
}

// SkippedItemResult represents a skipped item during reconciliation.
type SkippedItemResult struct {
	// The SKU.
	SKU string `json:"sku" validate:"required"`
	// The reason for skipping.
	Reason string `json:"reason" validate:"required"`
}

// ReconcileErrorResult represents an error during reconciliation.
type ReconcileErrorResult struct {
	// The SKU.
	SKU string `json:"sku" validate:"required"`
	// The error message.
	Error string `json:"error" validate:"required"`
}
