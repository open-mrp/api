package domain

import (
	"strings"
	"time"

	"github.com/open-mrp/apikit/pagination"
	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/shared/actor"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
)

// MovementFields is what an inventory movement records beyond its item and quantity.
type MovementFields struct {
	Type        constants.InventoryMovementType
	Actor       actor.Actor
	LocationID  *string
	LotID       *string
	Reason      *constants.InventoryMovementReason
	RelatedType constants.RecordType
	RelatedID   string
	// UnitCost is the item's cost per unit when the movement was recorded; nil when the item had none.
	UnitCost *UnitPrice
}

// LegacyMovementType is the business event of a row written before movements stored their type, read from the
// channel code it was written under. A scan that added stock produced it; one that removed stock consumed it. A user
// correction can't tell an adjustment from a reconciliation and reads as an adjustment.
func LegacyMovementType(actionTypeCode string, measure decimal.Decimal) constants.InventoryMovementType {
	switch constants.InventoryActionType(actionTypeCode) {
	case constants.InventoryActionTypeScan:
		if measure.IsNegative() {
			return constants.InventoryMovementTypeConsumption
		}
		return constants.InventoryMovementTypeOutput
	case constants.InventoryActionTypeSystemAction:
		return constants.InventoryMovementTypeReceipt
	default:
		return constants.InventoryMovementTypeAdjustment
	}
}

// LegacyMovementActor is who caused a row that stored no actor: the scanning station it was scanned at, else the
// caller recorded as its responsible user (a user, API key or agent, told apart by its ID), else OpenMRP itself.
func LegacyMovementActor(scanningStationID, responsibleUserID *string) actor.Actor {
	if scanningStationID != nil && *scanningStationID != "" {
		return actor.New(constants.ActorTypeDevice, *scanningStationID)
	}
	if responsibleUserID != nil && *responsibleUserID != "" {
		if a, ok := actorFromBareID(*responsibleUserID); ok {
			return a
		}
	}
	return actor.System()
}

// legacyAPIKeyIDPrefix is the API key ID prefix written before IDs were composed.
const legacyAPIKeyIDPrefix = "apky_"

func actorFromBareID(actorID string) (actor.Actor, bool) {
	switch {
	case strings.HasPrefix(actorID, string(id.UserIDPrefix)+"_"), strings.HasPrefix(actorID, string(id.AccountUserIDPrefix)+"_"):
		return actor.New(constants.ActorTypeUser, actorID), true
	case strings.HasPrefix(actorID, string(id.APIKeyIDPrefix)+"_"), strings.HasPrefix(actorID, legacyAPIKeyIDPrefix):
		return actor.New(constants.ActorTypeAPIKey, actorID), true
	case strings.HasPrefix(actorID, string(id.AgentDefinitionIDPrefix)+"_"):
		return actor.New(constants.ActorTypeAgent, actorID), true
	}
	return actor.Actor{}, false
}

// CreateInventoryMovementParams is a movement a caller records by hand: an adjustment by an amount or a
// reconciliation to a count.
type CreateInventoryMovementParams struct {
	ItemID string
	// Reconcile sets stock to exactly Measure; otherwise Measure is added (negative to remove).
	Reconcile  bool
	Measure    decimal.Decimal
	UnitID     string
	LocationID *string
	// LotID names an existing lot of the item; LotNumber names one by number, creating it when new. At most one is set.
	LotID      *string
	LotNumber  *string
	CustomerID *string
	Reason     *constants.InventoryMovementReason
}

// BulkCreateInventoryMovementRow is one row of a bulk movement request: the item by ID or SKU.
type BulkCreateInventoryMovementRow struct {
	ItemID     *string
	SKU        *string
	Measure    decimal.Decimal
	UnitID     string
	LocationID *string
	LotNumber  *string
	Reason     *constants.InventoryMovementReason
}

// BulkCreateInventoryMovementsParams is a bulk movement request; every row applies the same operation.
type BulkCreateInventoryMovementsParams struct {
	Reconcile bool
	Rows      []BulkCreateInventoryMovementRow
}

// ResolvedInventoryMovementRow is a bulk movement row with its item resolved; ItemID is empty when no item matched,
// which the write reports as that row's error. ID is the movement the row will create.
type ResolvedInventoryMovementRow struct {
	ID         string
	ItemID     string
	SKU        string
	Reconcile  bool
	Measure    decimal.Decimal
	UnitID     string
	LocationID *string
	LotNumber  *string
	Reason     *constants.InventoryMovementReason
}

// ListInventoryLevelsParams selects inventory levels: a page of items, optionally only those named, with their stock
// at the given locations (all when empty), now or as of an instant.
type ListInventoryLevelsParams struct {
	Cursor      *string
	Limit       int32
	ItemIDs     []string
	LocationIDs []string
	SiteIDs     []string
	AsOf        *time.Time
}

// StockLevel is an item's stock position at a set of locations (every location when LocationID is nil), in its
// stocking unit. With AsOf only OnHand is known.
type StockLevel struct {
	ItemID             string
	LocationID         *string
	UnitID             string
	OnHand             decimal.Decimal
	Reserved           *decimal.Decimal
	AvailableToPromise *decimal.Decimal
	Short              *decimal.Decimal
}

// ListInventoryLevelsResult is a page of inventory levels.
type ListInventoryLevelsResult struct {
	Levels   []*StockLevel
	PageInfo pagination.PageInfo
}
