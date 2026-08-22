package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
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

// UpsertRateParams is a rate in a bulk upsert, with units referenced fuzzily (by id,
// name, or abbreviation) and resolved server-side. Rate rows are never mutated:
// updates insert fresh rate rows and re-point the step at them.
type UpsertRateParams struct {
	Value           string
	NumeratorUnit   UnitIdentifier
	DenominatorUnit UnitIdentifier
}

// UpsertProductionParams is a production output in a bulk upsert, with the item and
// unit referenced fuzzily.
type UpsertProductionParams struct {
	Item          ItemIdentifier
	QuantityValue string
	QuantityUnit  UnitIdentifier
}

// UpsertStepConsumptionParams is a consumption in a bulk upsert, with the item and
// units referenced fuzzily. Waste defaults to zero in the quantity unit when omitted.
type UpsertStepConsumptionParams struct {
	Item               ItemIdentifier
	QuantityValue      string
	QuantityUnit       UnitIdentifier
	WasteQuantityValue *string
	WasteQuantityUnit  *UnitIdentifier
	Instructions       *string
}

// UpsertProductionStepParams is a single production step in a bulk upsert, matched by
// name (case-insensitive) within the account. Items, units, the department, and the
// scanning station are referenced fuzzily and resolved server-side. The department is
// create-only: a matched row stating a different department is rejected. The
// production and the consumptions are replaced wholesale on update. Flow DAG edges are
// not part of the input — they are auto-derived from item flows after the batch
// commits, mirroring single create.
type UpsertProductionStepParams struct {
	Name            string
	Notes           *string
	LevelingFactor  *string
	Allowances      *string
	ScanningStation *ObjectIdentifier
	Department      *ObjectIdentifier
	LaborRate       UpsertRateParams
	LaborTime       UpsertRateParams
	OverheadRate    UpsertRateParams
	Production      UpsertProductionParams
	Consumptions    []UpsertStepConsumptionParams
}

// BulkUpsertProductionStepsParams holds the parameters for bulk upserting production
// steps.
type BulkUpsertProductionStepsParams struct {
	ProductionSteps []UpsertProductionStepParams
}

// BulkUpsertProductionStepsResult is the aggregate result of a bulk production step
// upsert.
type BulkUpsertProductionStepsResult struct {
	CreatedIDs []string
	UpdatedIDs []string
}

// The Resolved* types below are UpsertProductionStep* after fuzzy resolution: every
// item, unit, department, and scanning-station reference replaced by its resolved ID.
// They are what the accept phase stores on the job row and what the executing worker
// writes from, so they must carry only resolved IDs — no fuzzy identifiers.
//
// They carry no JSON tags: the job_items payload is marshaled and unmarshaled by the
// engine against these same types, and it is an internal column no client reads.

// ResolvedUpsertRate is a rate with its units resolved.
type ResolvedUpsertRate struct {
	Value             string
	NumeratorUnitID   string
	DenominatorUnitID string
}

// ResolvedUpsertProduction is a production output with its item and unit resolved.
type ResolvedUpsertProduction struct {
	ItemID         string
	QuantityValue  string
	QuantityUnitID string
}

// ResolvedUpsertConsumption is a consumption with its item and units resolved. A nil
// WasteQuantityUnitID means waste defaults to the consumption's own quantity unit.
type ResolvedUpsertConsumption struct {
	ItemID              string
	QuantityValue       string
	QuantityUnitID      string
	WasteQuantityValue  *string
	WasteQuantityUnitID *string
	Instructions        *string
}

// ResolvedUpsertStepRow is a production step upsert row after fuzzy resolution. A nil
// DepartmentID or ScanningStationID means the row did not name one.
type ResolvedUpsertStepRow struct {
	Name              string
	Notes             *string
	LevelingFactor    *string
	Allowances        *string
	ScanningStationID *string
	DepartmentID      *string
	LaborRate         ResolvedUpsertRate
	LaborTime         ResolvedUpsertRate
	OverheadRate      ResolvedUpsertRate
	Production        ResolvedUpsertProduction
	Consumptions      []ResolvedUpsertConsumption
}

// ProductionStepBulkRow is the slim row shape used to match bulk upsert rows against
// existing production steps.
type ProductionStepBulkRow struct {
	ID                string
	Name              string
	Notes             *string
	LevelingFactor    string
	Allowances        string
	ScanningStationID *string
	DepartmentID      *string
	LaborRateID       string
	LaborTimeID       string
	OverheadRateID    string
}

// UpdateProductionStepForBulkUpsertParams holds the full-row write used by the bulk
// upsert update path. Rate IDs point at freshly inserted rate rows.
type UpdateProductionStepForBulkUpsertParams struct {
	AccountID         string
	ProductionStepID  string
	Name              string
	Notes             *string
	LevelingFactor    string
	Allowances        string
	ScanningStationID *string
	LaborRateID       string
	LaborTimeID       string
	OverheadRateID    string
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
