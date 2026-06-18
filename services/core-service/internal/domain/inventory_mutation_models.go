package domain

import "github.com/shopspring/decimal"

// InventoryUpdateParams describes a single inventory change (receipt or issue) to apply after a production step execution. Positive measure = receipt; negative = issue.
type InventoryUpdateParams struct {
	AccountID         string
	ItemID            string
	Measure           decimal.Decimal
	UnitID            string
	ActionType        string
	ScanningStationID string
	ResponsibleUserID *string
	BatchID           *string
}

// OrderReservationReductionParams describes a shortfall reduction on an order item's reservation.
type OrderReservationReductionParams struct {
	OrderID   string
	AccountID string
	ItemID    string
	Measure   decimal.Decimal
	UnitID    string
}

// ConsumptionAllocationParams describes a reservation allocation for material consumption.
type ConsumptionAllocationParams struct {
	OrderID   string
	AccountID string
	ItemID    string
	Measure   decimal.Decimal
	UnitID    string
}

// ConsumptionAllocationResult is the result of allocating reservations for consumption.
type ConsumptionAllocationResult struct {
	// RemainingMeasure is the quantity that could not be allocated from existing reservations and must be deducted from general inventory instead.
	RemainingMeasure decimal.Decimal
	RemainingUnitID  string
}

// MaterialDemandItem represents a single material demand entry from a BOM explosion.
type MaterialDemandItem struct {
	ItemID  string
	Measure decimal.Decimal
	UnitID  string
}

// CreateMaterialReservationParams holds parameters for creating a reserved inventory issue for a material demand linked to a sales order.
type CreateMaterialReservationParams struct {
	AccountID string
	ItemID    string
	Measure   decimal.Decimal
	UnitID    string
	OrderID   string
}

// CreateInventoryReceiptParams holds parameters for creating an inventory receipt.
type CreateInventoryReceiptParams struct {
	AccountID       string
	OwnerAccountID  string
	HolderAccountID string
	ItemID          string
	Measure         decimal.Decimal
	UnitID          string
	LocationID      *string
	LotID           *string
}

// CreateInventoryIssueParams holds parameters for creating an inventory issue.
type CreateInventoryIssueParams struct {
	AccountID  string
	ItemID     string
	Measure    decimal.Decimal
	UnitID     string
	LocationID *string
	LotID      *string
}

// CreateInventoryLogParams holds parameters for creating an inventory snapshot log.
type CreateInventoryLogParams struct {
	AccountID string
	ItemID    string
	Measure   decimal.Decimal
	UnitID    string
}

// CreateInventoryChangeLogParams holds parameters for creating an inventory change audit entry.
type CreateInventoryChangeLogParams struct {
	AccountID         string
	ItemID            string
	Measure           decimal.Decimal
	UnitID            string
	ActionType        string
	ScanningStationID *string
	ResponsibleUserID *string
	InventoryLogID    *string
}
