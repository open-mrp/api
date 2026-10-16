package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleItemID = "it_pej07ckhvu62"
const SampleItemSKU = "ALM-2024-1001"

// Something you buy, make, stock or sell: a material, part, product or service.
type Item struct {
	// Item ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item"`
	// What kind of item this is.
	//
	// `material`: bought and consumed in production. `part`: produced and used to make other items. `product`: a physical item you sell. `service`: something you sell that is never stocked, picked or shipped, such as installation. Can't change after creation.
	Type constants.ItemTypeCode `json:"type" validate:"required"`
	// Whether the item can be added to new orders, production and purchasing.
	//
	// `archived` items stay on the records already using them.
	Status constants.ArchiveStatus `json:"status" validate:"required"`
	// Your stock keeping unit, unique (case-insensitive) within your account.
	SKU string `json:"sku" validate:"required"`
	// Description of the item.
	//
	// Shown to customers in the portal for visible products.
	Description *string `json:"description"`
	// The category the item is classified under.
	Category *ItemCategory `json:"category" expandable:"true"`
	// The item's attributes, at most one per property, such as `Size: Large`.
	Attributes *List[Attribute] `json:"attributes" expandable:"true"`
	// What one stocking unit of the item costs you.
	//
	// For items made by a production flow, it's restated from the flow when anything the item is built from changes. `null` without `costs:read`, and always `null` for portal users.
	UnitCost *UnitPrice `json:"unit_cost" sensitive:"cost"`
	// How fast the item is consumed, such as `120 pr / day`.
	//
	// Calculated from demand; read-only. Always `null` for portal users.
	BurnRate *Rate `json:"burn_rate" sensitive:"internal"`
	// Where the item comes from when an order line or plan doesn't say.
	//
	// `stock`: taken from inventory. `make`: produced for the demand. `buy`: purchased for the demand. `null` uses the product line's, then the account default. Always `null` for portal users.
	Supply *constants.Supply `json:"supply" sensitive:"internal"`
	// Selling details. Products and services always have them; a material or part has them once you send `sales`, and `null` otherwise.
	Sales *ItemSales `json:"sales"`
	// Purchasing details, for materials. `null` for other types, and always `null` for portal users.
	Material *ItemMaterial `json:"material" sensitive:"internal"`
	// The manufacturing method used for new production of this item when nothing more specific applies.
	//
	// Always `null` for portal users.
	DefaultManufacturingMethod *ManufacturingMethod `json:"default_manufacturing_method" expandable:"true" sensitive:"internal"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last-updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// An item's selling details.
type ItemSales struct {
	// Base price on sales orders, before price lists apply.
	UnitPrice *UnitPrice `json:"unit_price" validate:"required"`
	// The product line the item is sold under.
	//
	// Customers are granted access to whole lines, so an item with no line is never shown in the customer portal. The line also sets the default commission and freight policies.
	ProductLine *ProductLine `json:"product_line" expandable:"true"`
	// Whether customers with access to the item's product line can see and order it in the portal.
	//
	// Defaults to `hidden`.
	PortalVisibility constants.CustomerPortalVisibility `json:"portal_visibility" validate:"required"`
	// How long production or purchasing takes before the item can ship, in business days.
	//
	// An order line is promised by the later of this and the customer's lead time. `null` when the item adds none.
	LeadTime *Quantity `json:"lead_time"`
}

// A material's purchasing details.
type ItemMaterial struct {
	// Reorder when on-hand stock falls to this quantity.
	//
	// `null` when there is no order point.
	OrderPoint *Quantity `json:"order_point"`
	// How long the material usually takes to arrive after it's ordered, in a time unit.
	//
	// `null` when unknown.
	LeadTime *Quantity `json:"lead_time"`
}

var sampleDescription = "6061-T6 aluminum sheet, 4ft x 8ft, 0.125in thick"

var SampleItem = &Item{
	ID:          SampleItemID,
	Object:      constants.ObjectTypeItem,
	Type:        constants.ItemTypeCodeProduct,
	Status:      constants.ArchiveStatusActive,
	SKU:         SampleItemSKU,
	Description: &sampleDescription,
	Category:    SampleItemCategory,
	Attributes:  NewList([]Attribute{*SampleAttribute}, PageInfo{}),
	UnitCost:    SampleUnitPrice,
	BurnRate:    SampleRate,
	Supply:      new(constants.SupplyStock),
	Sales: &ItemSales{
		UnitPrice:        SampleUnitPrice,
		ProductLine:      SampleProductLine,
		PortalVisibility: constants.CustomerPortalVisibilityVisible,
		LeadTime:         &Quantity{Value: "10", Unit: sampleDayUnit},
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Item) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleItem)
}

// The stock position for an item: what is in stock, what is already committed, and what is still free to sell.
//
// All four quantities are reported in the same unit — the base unit of the item's category.
// Derived figures, not stored rows: each is netted out of the ledger at read time, so none of them
// carries a quantity id.
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
	Object constants.ObjectType `json:"object" validate:"required,enum=item_costs"`
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
	// Unit of the costs' numerator — the currency they are counted in.
	//
	// Always resolved in full. A cost is unreadable without the currency it is priced in, so neither unit sits behind an include.
	NumeratorUnit *Unit `json:"numerator_unit"`
	// Unit of the costs' denominator — the item unit they are counted per, so each cost reads as currency per this unit.
	//
	// Always resolved in full, for the same reason as `numerator_unit`.
	DenominatorUnit *Unit `json:"denominator_unit"`
}

var SampleItemCosts = &ItemCosts{
	Object:             constants.ObjectTypeItemCosts,
	DirectMaterialCost: "5.000000000000000000000000000000",
	DirectLaborCost:    "3.000000000000000000000000000000",
	OverheadCost:       "2.000000000000000000000000000000",
	TotalCost:          "10.000000000000000000000000000000",
	NumeratorUnit:      SampleCurrencyUnit,
	DenominatorUnit:    SampleUnit,
}

func (*ItemCosts) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleItemCosts)
}

// One day of an item's trend series.
type ItemTrendPoint struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item_trend_point"`
	// Start of the UTC calendar day this point closes.
	OccurredAt time.Time `json:"occurred_at" validate:"required"`
	// Value of the trend metric at the end of that day, in the series' `unit`.
	Value string `json:"value" validate:"required" format:"decimal"`
}

// An item's trend metric over the last 30 UTC calendar days, one point per day.
type ItemTrends struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item_trends"`
	// The trend type that was requested.
	TrendType constants.ItemTrendType `json:"trend_type" validate:"required"`
	// Unit every point's value is counted in: the base unit of the item's category.
	//
	// Always resolved in full. A level is unreadable without its unit, so it is not behind an include.
	Unit *Unit `json:"unit"`
	// Exactly 30 points, oldest first, one for each UTC day ending today.
	Points *List[ItemTrendPoint] `json:"points" validate:"required"`
}

var SampleItemTrends = &ItemTrends{
	Object:    constants.ObjectTypeItemTrends,
	TrendType: constants.ItemTrendTypeInventory,
	Unit:      SampleUnit,
	Points: NewList([]ItemTrendPoint{
		{
			Object:     constants.ObjectTypeItemTrendPoint,
			OccurredAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
			Value:      "100",
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
	//
	// Null to customer and supplier portal users: they are your own team's notes.
	Notes *string `json:"notes" sensitive:"internal"`
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
// Both quantities are expressed in the item's own base unit, not in the unit submitted with the request, and both arrive with that unit resolved.
type ReconciledItemResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=reconciled_item_result"`
	// The item that was reconciled, named by id and SKU.
	Item *Entity `json:"item" validate:"required"`
	// Quantity before the reconciliation.
	PreviousQuantity *Quantity `json:"previous_quantity" validate:"required"`
	// Quantity after the reconciliation.
	NewQuantity *Quantity `json:"new_quantity" validate:"required"`
}

// A submitted row that was skipped rather than reconciled.
//
// A skipped row is reported by the SKU it was submitted under rather than as an item reference, because the usual reason to skip one is that no item carries that SKU — there is nothing to point at.
type SkippedItemResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=skipped_item_result"`
	// Item SKU, as submitted.
	SKU string `json:"sku" validate:"required"`
	// Human-readable reason the item was skipped.
	Reason string `json:"reason" validate:"required"`
}

// A submitted row that could not be reconciled.
type ReconcileErrorResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=reconcile_error_result"`
	// The item the row named, named by id and SKU.
	//
	// Always set: a row whose SKU matched no item is reported under `skipped_items` rather than here.
	Item *Entity `json:"item" validate:"required"`
	// Error message.
	Error string `json:"error" validate:"required"`
}

var SampleBulkReconcileItemsResponse = &BulkReconcileItemsResponse{
	Object: constants.ObjectTypeBulkReconcileItemsResponse,
	ReconciledItems: NewList([]ReconciledItemResult{
		{
			Object:           constants.ObjectTypeReconciledItemResult,
			Item:             SampleItemEntity,
			PreviousQuantity: SampleQuantity,
			NewQuantity:      SampleQuantity,
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
	// A lot that came from the item's category is counted in the category's unit; otherwise the item's own base unit is used. The unit is returned even when no rule supplies a lot size, so a form can show what is being counted with the quantity left blank.
	Unit *Unit `json:"unit" expandable:"true"`
	// Which rule in the chain produced this lot.
	//
	// - `item_override`: a lot size set on the item itself.
	// - `item_category`: the lot of the item's category.
	// - `account_default`: the account-wide fallback.
	//
	// Empty when no rule in the chain supplies a lot, which is the same case `quantity` reports as `0`.
	Source constants.ItemLotSource `json:"source" validate:"required"`
	// The item category the lot came from.
	//
	// Present only when `source` is `item_category`.
	ItemCategory *Entity `json:"item_category"`
}

var SampleItemLotDefault = &ItemLotDefault{
	Object:       constants.ObjectTypeItemLotDefault,
	Item:         NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil),
	Quantity:     60,
	Source:       constants.ItemLotSourceItemCategory,
	ItemCategory: NewEntity(SampleItemCategoryID, constants.ObjectTypeItemCategory, nil, nil),
}

func (*ItemLotDefault) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleItemLotDefault)
}
