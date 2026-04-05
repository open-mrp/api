package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// ProductionStep is the full production step domain model with all associated data.
type ProductionStep struct {
	ID              string
	Name            string                `audit:"name"`
	Notes           *string               `audit:"notes"`
	LevelingFactor  string                `audit:"leveling_factor"`
	Allowances      string                `audit:"allowances"`
	LaborRate       *ProductionStepRate   `audit:"labor_rate"`
	LaborTime       *ProductionStepRate   `audit:"labor_time"`
	OverheadRate    *ProductionStepRate   `audit:"overhead_rate"`
	Production      *Production           `audit:"production"`
	Consumptions    []Consumption         `audit:"consumptions"`
	Machines        []LightMachine        `audit:"machines"`
	ScanningStation *LightScanningStation `audit:"scanning_station"`
	InSteps         []LightProductionStep `audit:"in_steps"`
	OutSteps        []LightProductionStep `audit:"out_steps"`
	DepartmentID    *string               `audit:"department_id"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ProductionStepRate represents a rate associated with a production step (labor, overhead, etc).
type ProductionStepRate struct {
	ID              string
	Value           string
	NumeratorUnit   LightUnit
	DenominatorUnit LightUnit
}

// ListProductionStepsParams holds the parameters for listing production steps.
type ListProductionStepsParams struct {
	AccountID          string
	Cursor             *string
	Limit              int32
	Query              *string
	ItemIDs            []string
	MachineIDs         []string
	ScanningStationIDs []string
	InputStepIDs       []string
	OutputStepIDs      []string
	StartDate          *time.Time
	EndDate            *time.Time
}

// ListProductionStepsResult holds the result of listing production steps.
type ListProductionStepsResult struct {
	Steps    []*ProductionStep
	PageInfo pagination.PageInfo
}

// CreateProductionStepParams holds the parameters for creating a production step.
type CreateProductionStepParams struct {
	AccountID         string
	Name              string
	Notes             *string
	LevelingFactor    string
	Allowances        string
	ScanningStationID *string
	DepartmentID      *string
	LaborRate         CreateRateParams
	LaborTime         CreateRateParams
	OverheadRate      CreateRateParams
	Production        CreateProductionParams
	Consumptions      []CreateStepConsumptionParams
}

// CreateRateParams holds the parameters for creating a rate record.
type CreateRateParams struct {
	Value             string
	NumeratorUnitID   string
	DenominatorUnitID string
}

// CreateProductionParams holds the parameters for creating a production output.
type CreateProductionParams struct {
	ItemID         string
	QuantityValue  string
	QuantityUnitID string
}

// CreateStepConsumptionParams holds the parameters for creating a consumption within a production step create.
type CreateStepConsumptionParams struct {
	ItemID              string
	QuantityValue       string
	QuantityUnitID      string
	WasteQuantityValue  string
	WasteQuantityUnitID string
	Instructions        *string
}

// UpdateProductionStepParams holds the parameters for updating a production step.
type UpdateProductionStepParams struct {
	AccountID         string
	ProductionStepID  string
	Name              *string
	LevelingFactor    *string
	Allowances        *string
	ScanningStationID *string
}
