package repository

import (
	gosql "database/sql"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/actor"
	"github.com/open-mrp/api/shared/constants"
)

// movementColumns are the movement columns of an inventory_change_log row as read.
type movementColumns struct {
	TypeCode, ActorType, ActorID, LocationID, LotID, ReasonCode, RelatedType, RelatedID gosql.NullString
	UnitCostAmount, UnitCostCurrency, UnitCostPerUnitID, LotNumber                      gosql.NullString
}

// fields is the row's movement. A row written before movements stored their type and actor (00115) reads them derived
// from the channel code and the station or user it recorded, as the inventory_movement_types backfill will store them.
func (c movementColumns) fields(actionTypeCode, quantityValue string, stationID, userID *string) domain.MovementFields {
	m := domain.MovementFields{
		Type:        constants.InventoryMovementType(c.TypeCode.String),
		LocationID:  nullStringPtr(c.LocationID),
		LotID:       nullStringPtr(c.LotID),
		LotNumber:   nullStringPtr(c.LotNumber),
		RelatedType: constants.RecordType(c.RelatedType.String),
		RelatedID:   c.RelatedID.String,
	}
	if !c.TypeCode.Valid {
		measure, _ := decimal.NewFromString(quantityValue)
		m.Type = domain.LegacyMovementType(actionTypeCode, measure)
	}
	if a, err := actor.FromColumns(c.ActorType.String, c.ActorID); c.ActorType.Valid && err == nil {
		m.Actor = a
	} else {
		m.Actor = domain.LegacyMovementActor(stationID, userID)
	}
	if c.ReasonCode.Valid {
		reason := constants.InventoryMovementReason(c.ReasonCode.String)
		m.Reason = &reason
	}
	if c.UnitCostAmount.Valid && c.UnitCostCurrency.Valid {
		m.UnitCost = &domain.UnitPrice{
			Amount:    c.UnitCostAmount.String,
			Currency:  constants.Currency(c.UnitCostCurrency.String),
			PerUnitID: c.UnitCostPerUnitID.String,
		}
	}
	return m
}
