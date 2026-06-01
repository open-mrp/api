package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleItemID = "it_0131e386ac683e8c29a71f6f1f"
const SampleItemSKU = "ALM-2024-1001"

// Item is an inventory item (product, material, or part).
type Item struct {
	// Item ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// Stock keeping unit code.
	SKU string `json:"sku" validate:"required"`
	// Item description.
	Description *string `json:"description"`
	// Notes.
	Notes *string `json:"notes"`
	// Item type code.
	ItemTypeCode constants.ItemTypeCode `json:"type" validate:"required"`
	// Item category.
	Category *ItemCategory `json:"category" expandable:"true"`
	// Unit value rate.
	UnitValue *Rate `json:"unit_value" expandable:"true"`
	// Unit cost rate.
	UnitCost *Rate `json:"unit_cost" expandable:"true"`
	// Burn rate.
	BurnRate *Rate `json:"burn_rate" expandable:"true"`
	// Attributes assigned to this item.
	Attributes *List[Attribute] `json:"attributes" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

func ExpandableItemStub(id, sku string, ts time.Time) *Item {
	if id == "" {
		id = "it_unknown"
	}
	if sku == "" {
		sku = "ITEM"
	}
	if ts.IsZero() {
		ts = time.Unix(0, 0).UTC()
	}
	return &Item{
		ID:           id,
		Object:       constants.ObjectTypeItem,
		SKU:          sku,
		ItemTypeCode: constants.ItemTypeCodeProduct,
		CreatedAt:    ts,
		UpdatedAt:    ts,
	}
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
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Item) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleItem)
}

// ItemInventory contains inventory quantities for an item.
type ItemInventory struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item_inventory"`
	// On-hand quantity. Expandable via include[]=on_hand.
	OnHand *Quantity `json:"on_hand" expandable:"true"`
	// Reserved quantity. Expandable via include[]=reserved.
	Reserved *Quantity `json:"reserved" expandable:"true"`
	// Available-to-promise quantity. Expandable via include[]=available_to_promise.
	AvailableToPromise *Quantity `json:"available_to_promise" expandable:"true"`
	// Short quantity. Expandable via include[]=short.
	Short *Quantity `json:"short" expandable:"true"`
}

var SampleItemInventory = &ItemInventory{
	Object:             constants.ObjectTypeItemInventory,
	OnHand:             SampleQuantity,
	Reserved:           SampleQuantity,
	AvailableToPromise: SampleQuantity,
	Short:              SampleQuantity,
}

func (*ItemInventory) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleItemInventory)
}

// ItemCosts is the cost breakdown for an item.
type ItemCosts struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// Direct material cost.
	DirectMaterialCost string `json:"direct_material_cost" validate:"required" format:"decimal"`
	// Direct labor cost.
	DirectLaborCost string `json:"direct_labor_cost" validate:"required" format:"decimal"`
	// Overhead cost.
	OverheadCost string `json:"overhead_cost" validate:"required" format:"decimal"`
	// Total cost.
	TotalCost string `json:"total_cost" validate:"required" format:"decimal"`
	// Unit for cost values.
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

// ItemTrendPoint is a single trend data point.
type ItemTrendPoint struct {
	// Timestamp of the data point.
	OccurredAt time.Time `json:"occurred_at" validate:"required"`
	// Value at this date.
	Value string `json:"value" validate:"required" format:"decimal"`
}

// ItemTrends is the historical trend data for an item.
type ItemTrends struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// Requested trend type.
	TrendType string `json:"trend_type" validate:"required"`
	// Trend data points.
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

// ExportItem is an item with inventory for export.
type ExportItem struct {
	// Item ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// Stock keeping unit code.
	SKU string `json:"sku" validate:"required"`
	// Item description.
	Description *string `json:"description"`
	// Notes.
	Notes *string `json:"notes"`
	// Item type code.
	ItemTypeCode constants.ItemTypeCode `json:"type" validate:"required"`
	// Category name.
	CategoryName string `json:"category_name"`
	// On-hand quantity.
	OnHandQuantity string `json:"on_hand_quantity" format:"decimal"`
	// On-hand unit.
	OnHandUnit *Unit `json:"on_hand_unit"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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

// ExportItemsResponse is the export items response in JSON format.
// Export endpoints typically return an Excel file instead.
type ExportItemsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// Exported items.
	Items []*ExportItem `json:"items" validate:"required"`
	// Total count of exported items.
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

// BulkReconcileItemsResponse is the response from bulk reconciling items.
type BulkReconcileItemsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=bulk_reconcile_items_response"`
	// Successfully reconciled items.
	ReconciledItems *List[ReconciledItemResult] `json:"reconciled_items" validate:"required"`
	// Skipped items.
	SkippedItems *List[SkippedItemResult] `json:"skipped_items" validate:"required"`
	// Reconciliation errors.
	Errors *List[ReconcileErrorResult] `json:"errors" validate:"required"`
}

// ReconciledItemResult is a successfully reconciled item.
type ReconciledItemResult struct {
	// Item ID.
	ItemID string `json:"item_id" validate:"required"`
	// Item SKU.
	SKU string `json:"sku" validate:"required"`
	// Previous quantity.
	PreviousQuantity float64 `json:"previous_quantity" validate:"required"`
	// New quantity.
	NewQuantity float64 `json:"new_quantity" validate:"required"`
}

// SkippedItemResult is a skipped item during reconciliation.
type SkippedItemResult struct {
	// Item SKU.
	SKU string `json:"sku" validate:"required"`
	// Reason for skipping.
	Reason string `json:"reason" validate:"required"`
}

// ReconcileErrorResult is an error during reconciliation.
type ReconcileErrorResult struct {
	// Item SKU.
	SKU string `json:"sku" validate:"required"`
	// Error message.
	Error string `json:"error" validate:"required"`
}

var SampleBulkReconcileItemsResponse = &BulkReconcileItemsResponse{
	Object: constants.ObjectTypeBulkReconcileItemsResponse,
	ReconciledItems: NewList([]ReconciledItemResult{
		{
			ItemID:           SampleItemID,
			SKU:              SampleItemSKU,
			PreviousQuantity: 10,
			NewQuantity:      12,
		},
	}, PageInfo{}),
	SkippedItems: NewList([]SkippedItemResult{}, PageInfo{}),
	Errors:       NewList([]ReconcileErrorResult{}, PageInfo{}),
}

func (*BulkReconcileItemsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleBulkReconcileItemsResponse)
}
