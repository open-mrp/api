package domain

import "github.com/shopspring/decimal"

// ProductionStepDetail is the full production step with its production and consumptions.
type ProductionStepDetail struct {
	ID           string
	Name         string
	Production   StepProduction
	Consumptions []StepConsumption
}

// StepProduction represents the output of a production step.
type StepProduction struct {
	ID           string
	ProducedItem LightItem
	Quantity     BatchQuantity
}

// StepConsumption represents a single consumption of a production step.
type StepConsumption struct {
	ID            string
	ConsumedItem  LightItem
	Quantity      BatchQuantity
	WasteQuantity BatchQuantity
	Instructions  *string
}

// NextStepQuantitiesResult holds the result of calculating quantities for the next production step.
type NextStepQuantitiesResult struct {
	Quantity       decimal.Decimal
	ItemID         string
	ProducedUnitID string
}

// ScanningProductionStepInfo holds information about a production step available at a scanning station.
type ScanningProductionStepInfo struct {
	ID          string
	Name        string
	IsMultiPart bool
}
