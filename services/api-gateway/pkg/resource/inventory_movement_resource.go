package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

func init() {
	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeInventoryMovement,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "item", ObjectType: constants.ObjectTypeItem},
			{Key: "location", ObjectType: constants.ObjectTypeLocation},
			{Key: "lot", ObjectType: constants.ObjectTypeLot},
			{Key: "related", ObjectType: constants.ObjectTypeRecord},
		},
	})
	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeInventoryLevel,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "item", ObjectType: constants.ObjectTypeItem},
			{Key: "location", ObjectType: constants.ObjectTypeLocation},
		},
	})
	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeLotSizeDefault,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "item", ObjectType: constants.ObjectTypeItem},
			{Key: "item_category", ObjectType: constants.ObjectTypeItemCategory},
		},
	})
}

const SampleInventoryMovementID = "inmv_wgxiepfsyfmz"

// A change to an item's stock at one location.
//
// Every receipt, shipment, production consumption and output, transfer, adjustment and count is a movement. Movements can't be edited or deleted; a mistake is corrected with another movement.
type InventoryMovement struct {
	// Inventory movement ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=inventory_movement"`
	// What kind of movement this is.
	//
	// - `receipt`: stock received from a supplier.
	// - `shipment`: stock shipped to a customer, or put back when a shipment is undone.
	// - `consumption`: stock used by production.
	// - `output`: stock produced.
	// - `transfer`: stock moved between locations.
	// - `adjustment`: a manual change by an amount.
	// - `reconciliation`: a count that set stock to an exact quantity.
	// - `return`: stock returned by a customer.
	//
	// More types may be added. How the movement was recorded, such as by a scan, is `actor`, not `type`.
	Type constants.InventoryMovementType `json:"type" validate:"required"`
	// The item whose stock moved.
	Item *Item `json:"item" expandable:"true"`
	// How much stock moved, in the unit it was recorded in.
	//
	// Positive values added stock, negative values removed it. A reconciliation that matched the stock already on hand records `0`.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// Where the stock moved. `null` when no location was recorded.
	Location *Location `json:"location" expandable:"true"`
	// The traceability lot the stock moved in. `null` when no lot was recorded.
	Lot *Lot `json:"lot" expandable:"true"`
	// Who caused the movement: a user, API key, agent, device, or OpenMRP itself.
	Actor *ActorRef `json:"actor" validate:"required"`
	// Why stock was corrected.
	//
	// `damaged`, `count_correction`, `found`, `scrap` or `other`. Set only on adjustments and reconciliations that gave one.
	Reason *constants.InventoryMovementReason `json:"reason"`
	// The document behind the movement, such as the shipment it shipped on or the batch a scan moved. `null` when there is none, and unless included.
	Related *Record `json:"related" expandable:"true"`
	// The item's cost per unit when the movement was recorded.
	//
	// A receipt carries the price the goods were bought at. `null` when the item had no cost.
	UnitCost *UnitPrice `json:"unit_cost"`
	// When the stock moved.
	OccurredAt time.Time `json:"occurred_at" validate:"required"`
	// When the movement was recorded.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

var sampleMovementReason = constants.InventoryMovementReasonCountCorrection

var SampleInventoryMovement = &InventoryMovement{
	ID:         SampleInventoryMovementID,
	Object:     constants.ObjectTypeInventoryMovement,
	Type:       constants.InventoryMovementTypeAdjustment,
	Item:       SampleItem,
	Quantity:   SampleQuantity,
	Location:   SampleLocation,
	Lot:        SampleLot,
	Actor:      SampleActorRef,
	Reason:     &sampleMovementReason,
	Related:    SampleRecord,
	UnitCost:   SampleUnitPrice,
	OccurredAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*InventoryMovement) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleInventoryMovement)
}

// An item's stock position.
//
// `on_hand`: in your facility, including customer-owned stock you hold. `reserved`: held for existing orders. `available_to_promise`: free for new orders. `short`: demand not yet covered.
type InventoryLevel struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=inventory_level"`
	// The item this level is for.
	Item *Item `json:"item" expandable:"true"`
	// The location this level covers.
	//
	// `null` when it covers every location, or several. Set only when the list was filtered to exactly one location.
	Location *Location `json:"location" expandable:"true"`
	// Stock in your facility, in the item's stocking unit: what was received or produced and not yet drawn on. Includes customer-owned stock you hold.
	OnHand *Quantity `json:"on_hand" validate:"required"`
	// Stock held for existing orders, in the item's stocking unit. `null` for a level as of a past instant.
	Reserved *Quantity `json:"reserved"`
	// Stock free for new orders: `on_hand` less `reserved` and `short`. `null` for a level as of a past instant.
	AvailableToPromise *Quantity `json:"available_to_promise"`
	// Demand no stock covers yet, in the item's stocking unit. `null` for a level as of a past instant.
	Short *Quantity `json:"short"`

	itemID, locationID string
}

// SetRefs records the item and location the level is for, which its includes load.
func (l *InventoryLevel) SetRefs(itemID string, locationID *string) {
	l.itemID = itemID
	if locationID != nil {
		l.locationID = *locationID
	}
}

// Refs is the item and location ID the level is for; the location is empty when it covers several.
func (l *InventoryLevel) Refs() (itemID, locationID string) { return l.itemID, l.locationID }

var SampleInventoryLevel = &InventoryLevel{
	Object:             constants.ObjectTypeInventoryLevel,
	Item:               SampleItem,
	Location:           nil,
	OnHand:             SampleQuantity,
	Reserved:           SampleQuantity,
	AvailableToPromise: SampleQuantity,
	Short:              SampleQuantity,
}

func (*InventoryLevel) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleInventoryLevel)
}

// The quantity an item is made in, and which rule supplied it.
type LotSizeDefault struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=lot_size_default"`
	// The item the lot size is for.
	Item *Item `json:"item" expandable:"true"`
	// The quantity this item is made in, such as one doff.
	//
	// `null` when no lot convention applies. `source` says which rule supplied it.
	LotSize *Quantity `json:"lot_size"`
	// Which rule supplied `lot_size`, most specific first.
	//
	// - `item_override`: a lot size set for this item in planning.
	// - `item_category`: the item category's `default_lot_size`.
	// - `account_default`: the account-wide default.
	//
	// `null` when no rule applies.
	Source *constants.ItemLotSource `json:"source"`
	// The item category the lot size came from. `null` unless `source` is `item_category`, and unless included.
	ItemCategory *ItemCategory `json:"item_category" expandable:"true"`

	itemID, itemCategoryID string
}

// SetRefs records the item and the item category the lot size came from, which its includes load.
func (d *LotSizeDefault) SetRefs(itemID, itemCategoryID string) {
	d.itemID, d.itemCategoryID = itemID, itemCategoryID
}

// Refs is the item and item category ID; the category is empty unless the lot size came from it.
func (d *LotSizeDefault) Refs() (itemID, itemCategoryID string) { return d.itemID, d.itemCategoryID }

var sampleLotSizeSource = constants.ItemLotSourceItemCategory

var SampleLotSizeDefault = &LotSizeDefault{
	Object:       constants.ObjectTypeLotSizeDefault,
	Item:         SampleItem,
	LotSize:      SampleQuantity,
	Source:       &sampleLotSizeSource,
	ItemCategory: SampleItemCategory,
}

func (*LotSizeDefault) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleLotSizeDefault)
}
