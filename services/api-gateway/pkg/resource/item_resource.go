package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleItemID = "it_pej07ckhvu62"
const SampleItemSKU = "ALM-2024-1001"

// An entry in your catalog: something you sell, consume, or build with.
type Item struct {
	// Item ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// Stock keeping unit code, unique within the account.
	SKU string `json:"sku" validate:"required"`
	// Item description.
	Description *string `json:"description"`
	// Free-form notes about the item.
	Notes *string `json:"notes"`
	// What kind of item this is.
	//
	// - `product`: a finished product.
	// - `material`: a raw material or component consumed in production.
	// - `part`: a part used in production.
	ItemTypeCode constants.ItemTypeCode `json:"type" validate:"required"`
	// The category this item belongs to.
	//
	// The category's unit group determines the base unit the item's rates (`unit_value`, `unit_cost`, `burn_rate`) are expressed in.
	Category *ItemCategory `json:"category" expandable:"true"`
	// Selling value per unit, expressed as a rate (e.g. `$25.50 / kg`).
	UnitValue *Rate `json:"unit_value" expandable:"true"`
	// Cost per unit, expressed as a rate (e.g. `$10.00 / kg`).
	//
	// For items a production flow produces, retrieving the item's costs recomputes this from the flow and stores the result here, so it can change without the item having been edited.
	UnitCost *Rate `json:"unit_cost" expandable:"true"`
	// Rate at which this item is consumed in production, expressed as a quantity over time (e.g. `100 kg / hr`).
	BurnRate *Rate `json:"burn_rate" expandable:"true"`
	// Attributes assigned to this item.
	Attributes *List[Attribute] `json:"attributes" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Item) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleItem)
}

// The stock position for an item: what is in stock, what is already committed, and what is still free to sell.
//
// All four quantities are reported in the same unit — the base unit of the item's category.
type ItemInventory struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item_inventory"`
	// Physical quantity currently in stock.
	OnHand *Quantity `json:"on_hand" expandable:"true"`
	// Quantity committed to existing orders and therefore not free to allocate.
	Reserved *Quantity `json:"reserved" expandable:"true"`
	// Quantity free to commit to new orders, i.e. on-hand minus reserved minus short.
	AvailableToPromise *Quantity `json:"available_to_promise" expandable:"true"`
	// Quantity by which demand exceeds available supply (the unfulfillable shortfall).
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

// The per-unit production cost breakdown for an item, computed from the production flow that produces it.
type ItemCosts struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// Cost of materials consumed to produce one unit of the item, including the portion consumed as waste.
	//
	// Counts raw materials only. Parts and sub-products consumed along the way are not priced here; their labor, overhead, and material costs are already included through the steps that produce them.
	DirectMaterialCost string `json:"direct_material_cost" validate:"required" format:"decimal"`
	// Labor cost to produce one unit of the item.
	//
	// Based on each step's labor time after its leveling factor and allowances are applied, priced at that step's labor rate.
	DirectLaborCost string `json:"direct_labor_cost" validate:"required" format:"decimal"`
	// Overhead cost allocated to one unit of the item.
	//
	// Applied over the same corrected labor time as `direct_labor_cost`, priced at each step's overhead rate.
	OverheadCost string `json:"overhead_cost" validate:"required" format:"decimal"`
	// Total cost to produce one unit of the item (material + labor + overhead).
	TotalCost string `json:"total_cost" validate:"required" format:"decimal"`
	// The unit of the item the per-unit costs are expressed against.
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

// A single measurement in an item's trend series.
type ItemTrendPoint struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item_trend_point"`
	// Timestamp of the data point.
	OccurredAt time.Time `json:"occurred_at" validate:"required"`
	// Recorded value of the trend metric at `occurred_at`.
	Value string `json:"value" validate:"required" format:"decimal"`
}

// Historical trend data for an item, as a time-ordered series of measurements.
type ItemTrends struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// The trend type that was requested.
	//
	// Currently the only supported value is `inventory`.
	TrendType string `json:"trend_type" validate:"required"`
	// Trend data points, oldest first.
	//
	// At most one point is returned per calendar day: when several measurements were recorded on the same day, the earliest one is kept.
	Points *List[ItemTrendPoint] `json:"points" validate:"required"`
}

var SampleItemTrends = &ItemTrends{
	Object:    constants.ObjectTypeItem,
	TrendType: "on_hand",
	Points: NewList([]ItemTrendPoint{
		{
			Object:     constants.ObjectTypeItemTrendPoint,
			OccurredAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
			Value:      "100.000000000000000000000000000000",
		},
	}, PageInfo{}),
}

func (*ItemTrends) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleItemTrends)
}

// A single row of the items export: an item together with its on-hand inventory.
type ExportItem struct {
	// Item ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// Stock keeping unit code, unique within the account.
	SKU string `json:"sku" validate:"required"`
	// Item description.
	Description *string `json:"description"`
	// Free-form notes about the item.
	Notes *string `json:"notes"`
	// What kind of item this is.
	//
	// - `product`: a finished product.
	// - `material`: a raw material or component consumed in production.
	// - `part`: a part used in production.
	ItemTypeCode constants.ItemTypeCode `json:"type" validate:"required"`
	// Name of the item's category.
	CategoryName string `json:"category_name"`
	// Physical quantity currently in stock, expressed in `on_hand_unit`.
	OnHandQuantity string `json:"on_hand_quantity" format:"decimal"`
	// Unit of measure for `on_hand_quantity`.
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

// The JSON shape of the items export.
//
// The export endpoint itself responds with an Excel file; this documents the equivalent structured payload.
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

// The outcome of a bulk inventory reconciliation, reported as three separate lists.
type BulkReconcileItemsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=bulk_reconcile_items_response"`
	// Items whose inventory was successfully reconciled.
	ReconciledItems *List[ReconciledItemResult] `json:"reconciled_items" validate:"required"`
	// Items that were skipped, e.g. because no item with the given SKU exists.
	SkippedItems *List[SkippedItemResult] `json:"skipped_items" validate:"required"`
	// Items that failed to reconcile, e.g. because the given unit does not exist or the inventory write failed.
	Errors *List[ReconcileErrorResult] `json:"errors" validate:"required"`
}

// An item whose on-hand quantity was successfully reconciled.
//
// Both quantities are expressed in the item's own base unit, not in the unit submitted with the request.
type ReconciledItemResult struct {
	// Item ID.
	ItemID string `json:"item_id" validate:"required"`
	// Item SKU.
	SKU string `json:"sku" validate:"required"`
	// On-hand quantity before the reconciliation.
	PreviousQuantity float64 `json:"previous_quantity" validate:"required"`
	// On-hand quantity after the reconciliation.
	NewQuantity float64 `json:"new_quantity" validate:"required"`
}

// A submitted row that was skipped rather than reconciled.
type SkippedItemResult struct {
	// Item SKU.
	SKU string `json:"sku" validate:"required"`
	// Human-readable reason the item was skipped.
	Reason string `json:"reason" validate:"required"`
}

// A submitted row that could not be reconciled.
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

// The lot an item is made in — how many, counted in what.
//
// A lot is the quantity production is issued in: a doff, a pallet, a batch. The unit is what makes it meaningful, since 60 pairs and 60 eaches are different lots.
type ItemLotDefault struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item_lot_default"`
	// The item the lot was resolved for.
	Item *Entity `json:"item" validate:"required"`
	// Units in one lot.
	//
	// `0` means the item has no lot convention, not that its lot is zero.
	Quantity float64 `json:"quantity"`
	// The unit the lot is counted in.
	//
	// A lot that came from a product line is counted in that line's unit; otherwise the item's own base unit is used. The unit is returned even when no rule supplies a lot size, so a form can show what is being counted with the quantity left blank.
	Unit *Unit `json:"unit" expandable:"true"`
	// Which rule in the chain produced this lot.
	//
	// - `item_override`: a lot size set on the item itself.
	// - `product_line`: the convention of the line the item sells under.
	// - `downstream_product_line`: inherited from the finished goods this item becomes, for intermediates that are not themselves sold.
	// - `account_default`: the account-wide fallback.
	//
	// Empty when no rule in the chain supplies a lot, which is the same case `quantity` reports as `0`.
	Source constants.ItemLotSource `json:"source" validate:"required"`
	// The product line the convention came from.
	//
	// Present only when `source` is `product_line` or `downstream_product_line`; an item override and the account default do not come from a line.
	ProductLine *Entity `json:"product_line"`
}

var SampleItemLotDefault = &ItemLotDefault{
	Object:      constants.ObjectTypeItemLotDefault,
	Item:        NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil),
	Quantity:    60,
	Source:      constants.ItemLotSourceDownstreamProductLine,
	ProductLine: NewEntity(SampleProductLineID, constants.ObjectTypeProductLine, nil, nil),
}

func (*ItemLotDefault) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleItemLotDefault)
}
