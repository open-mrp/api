package domain

import "time"

// Production represents the output of a production step.
type Production struct {
	ID               string
	ItemID           string   `audit:"item_id"`
	ItemSKU          string   `audit:"item_sku"`
	ItemDescription  *string  `audit:"item_description"`
	ItemTypeCode     string   `audit:"item_type_code"`
	Quantity         Quantity `audit:"quantity"`
	ProductionStepID string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// UpdateProductionParams holds the parameters for updating a production output.
type UpdateProductionParams struct {
	AccountID        string
	ProductionStepID string
	ProductionID     string
	ItemID           *string
	QuantityValue    *string
	QuantityUnitID   *string
}
