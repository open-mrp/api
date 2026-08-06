package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// ProductionStepDetail is the full production step with its production and consumptions.
// carries an export's filters — the list params without pagination, plus the cap
// that keeps one request from building an unbounded workbook
type ExportProductionStepsParams struct {
	AccountID string
	Query     *string
	Limit     int32
}

// carries one step and its consumptions as the export sheet lays them out. The
// read model nests rates and quantities; the sheet wants them flat.
type ProductionStepExport struct {
	ID                       string
	Name                     string
	DepartmentName           *string
	ScanningStationName      *string
	LaborRate                *string
	LaborRateCurrencyUnit    *string
	LaborRateTimeUnit        *string
	LaborTime                *string
	LaborTimeUnit            *string
	LaborTimePerUnit         *string
	OverheadRate             *string
	OverheadRateCurrencyUnit *string
	OverheadRateTimeUnit     *string
	Allowances               string
	LevelingFactor           string
	Notes                    *string
	ProducedItemSKU          string
	ProducedQuantity         string
	ProducedUnit             string
	Consumptions             []ProductionStepExportConsumption
}

// carries one consumption of an exported step, one sheet row each
type ProductionStepExportConsumption struct {
	ItemSKU       string
	Quantity      string
	Unit          string
	WasteQuantity *string
	WasteUnit     *string
	Instructions  *string
}

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
	CreatedAt     time.Time
	UpdatedAt     time.Time
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
