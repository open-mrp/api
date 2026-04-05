package domain

import "time"

// Consumption represents a material consumed by a production step.
type Consumption struct {
	ID               string
	ItemID           string   `audit:"item_id"`
	ItemSKU          string   `audit:"item_sku"`
	ItemDescription  *string  `audit:"item_description"`
	ItemTypeCode     string   `audit:"item_type_code"`
	Quantity         Quantity `audit:"quantity"`
	WasteQuantity    Quantity `audit:"waste_quantity"`
	Instructions     *string  `audit:"instructions"`
	ProductionStepID string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CreateConsumptionParams holds the parameters for creating a consumption.
type CreateConsumptionParams struct {
	AccountID           string
	ProductionStepID    string
	ItemID              string
	QuantityValue       string
	QuantityUnitID      string
	WasteQuantityValue  string
	WasteQuantityUnitID string
	Instructions        *string
}

// UpdateConsumptionParams holds the parameters for updating a consumption.
type UpdateConsumptionParams struct {
	AccountID           string
	ProductionStepID    string
	ConsumptionID       string
	ItemID              *string
	QuantityValue       *string
	QuantityUnitID      *string
	WasteQuantityValue  *string
	WasteQuantityUnitID *string
	Instructions        *string
}

// DeleteConsumptionParams holds the parameters for deleting a consumption.
type DeleteConsumptionParams struct {
	AccountID        string
	ProductionStepID string
	ConsumptionID    string
}
