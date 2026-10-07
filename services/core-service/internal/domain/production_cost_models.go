package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// AnalyzeProductionCostsParams scopes a production cost report to the batches scanned at a production step within [StartDate, EndDate].
type AnalyzeProductionCostsParams struct {
	AccountID string
	StartDate time.Time
	EndDate   time.Time
	// ItemIDs and ProductLineIDs select the parts the selected items' production consumes, with the selected items themselves.
	ItemIDs        []string
	ProductLineIDs []string
	// DepartmentIDs match the department of the station a batch was scanned at.
	DepartmentIDs []string
	// CategoryIDs match the category of the batch's item.
	CategoryIDs []string
}

// ProductionCostRef names a department or item category a production cost report groups by.
type ProductionCostRef struct {
	ID   string
	Name string
}

// ProductionCostBatches is one kind of output — productive, seconds or waste — summed across a group of batches.
type ProductionCostBatches struct {
	// BaseQuantity sums each batch's quantity in its dimension's base unit, offset included.
	BaseQuantity decimal.Decimal
	// Count is how many of the batches recorded this kind of output.
	Count int64
}

// ProductionCostRow is the batches one production step recorded at one department's stations, of items in one category.
type ProductionCostRow struct {
	ProductionStepID string
	// Department is nil for batches scanned at no station, or at a station whose department no longer exists.
	Department *ProductionCostRef
	Category   ProductionCostRef
	Productive ProductionCostBatches
	Seconds    ProductionCostBatches
	Waste      ProductionCostBatches
}

// ProductionCostStep is what costing one run of a production step reads. Step.Production is the step's earliest-created production.
type ProductionCostStep struct {
	Step         ProductionFlowStep
	Consumptions []CostFlowConsumption
}

// ProducedQuantity is an amount produced, in the base unit of its dimension.
type ProducedQuantity struct {
	UnitID string
	Value  decimal.Decimal
}

// ProductionCost is the cost of some output: money in the currency base unit, labor time in hours, and what was produced, one entry per dimension.
type ProductionCost struct {
	Materials  decimal.Decimal
	Labor      decimal.Decimal
	Overhead   decimal.Decimal
	Total      decimal.Decimal
	LaborHours decimal.Decimal
	Produced   []ProducedQuantity
}

// ProductionCostSet splits a group's cost by the kind of output it went into.
type ProductionCostSet struct {
	Productive ProductionCost
	Seconds    ProductionCost
	Waste      ProductionCost
	Total      ProductionCost
}

// ProductionCostGroup is the cost of the batches in one department, one category, or one department and category.
type ProductionCostGroup struct {
	// Department is nil in a category group, and for batches with no department.
	Department *ProductionCostRef
	// Category is nil in a department group.
	Category *ProductionCostRef
	Costs    ProductionCostSet
}

// ProductionCostReport is what production cost over a window, overall and by department and item category.
type ProductionCostReport struct {
	CurrencyUnitID string
	TimeUnitID     string
	// Units are the currency, time and produced units, keyed by id. They travel with the report, so a caller who may read it need not also be allowed to browse units.
	Units                map[string]*Unit
	Totals               ProductionCostSet
	Departments          []ProductionCostGroup
	Categories           []ProductionCostGroup
	DepartmentCategories []ProductionCostGroup
}
