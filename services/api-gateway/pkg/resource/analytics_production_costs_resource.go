package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

// AnalyzeProductionCostsResponse is what production cost over a window: overall, by department, by item category, and by both.
type AnalyzeProductionCostsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=analyze_production_costs_response"`
	// Start of the window, inclusive.
	StartsAt time.Time `json:"starts_at" validate:"required"`
	// End of the window, inclusive.
	EndsAt time.Time `json:"ends_at" validate:"required"`
	// The currency every money figure in the report is in.
	CurrencyUnit *Unit `json:"currency_unit" validate:"required"`
	// The unit every `labor_time` in the report is in.
	TimeUnit *Unit `json:"time_unit" validate:"required"`
	// The cost of every batch the report covers.
	Totals *ProductionCostTotals `json:"totals" validate:"required"`
	// The cost per department of the station each batch was scanned at, by department name; batches with no department come last.
	Departments *List[ProductionCostDepartment] `json:"departments" validate:"required"`
	// The cost per category of each batch's item, by category name.
	Categories *List[ProductionCostCategory] `json:"categories" validate:"required"`
	// The cost per department and item category, by department name then category name; batches with no department come last.
	DepartmentCategories *List[ProductionCostDepartmentCategory] `json:"department_categories" validate:"required"`
}

// ProductionCost is what one kind of output cost: the material it consumed, the labor and overhead its labor time was charged, and what it was.
type ProductionCost struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_cost"`
	// Raw material consumed, waste allowance included, in `currency_unit`.
	Materials string `json:"materials" validate:"required" format:"decimal"`
	// Labor time priced at each step's labor rate, in `currency_unit`.
	Labor string `json:"labor" validate:"required" format:"decimal"`
	// Labor time priced at each step's overhead rate, in `currency_unit`.
	Overhead string `json:"overhead" validate:"required" format:"decimal"`
	// Materials, labor and overhead together, in `currency_unit`.
	Total string `json:"total" validate:"required" format:"decimal"`
	// Labor time, after each step's leveling factor and allowances, in `time_unit`.
	LaborTime string `json:"labor_time" validate:"required" format:"decimal"`
	// What was produced, in the base unit of its dimension: one entry per dimension, so a report mixing pieces and weights states each rather than adding them.
	Produced *List[ComputedQuantity] `json:"produced" validate:"required"`
}

// ProductionCostTotals is what a production cost report's batches cost, by the kind of output they went into.
type ProductionCostTotals struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_cost_totals"`
	// First-quality output: each batch's quantity.
	Productive *ProductionCost `json:"productive" validate:"required"`
	// Second-quality output the batches recorded.
	Seconds *ProductionCost `json:"seconds" validate:"required"`
	// Waste the batches recorded.
	Waste *ProductionCost `json:"waste" validate:"required"`
	// Productive, seconds and waste together.
	Total *ProductionCost `json:"total" validate:"required"`
}

// ProductionCostDepartment is what the batches scanned at one department's stations cost.
type ProductionCostDepartment struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_cost_department"`
	// The department. Null for batches scanned at no station.
	Department *Entity `json:"department"`
	// First-quality output: each batch's quantity.
	Productive *ProductionCost `json:"productive" validate:"required"`
	// Second-quality output the batches recorded.
	Seconds *ProductionCost `json:"seconds" validate:"required"`
	// Waste the batches recorded.
	Waste *ProductionCost `json:"waste" validate:"required"`
	// Productive, seconds and waste together.
	Total *ProductionCost `json:"total" validate:"required"`
}

// ProductionCostCategory is what the batches of one item category cost.
type ProductionCostCategory struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_cost_category"`
	// The category of the batches' items.
	Category *Entity `json:"category" validate:"required"`
	// First-quality output: each batch's quantity.
	Productive *ProductionCost `json:"productive" validate:"required"`
	// Second-quality output the batches recorded.
	Seconds *ProductionCost `json:"seconds" validate:"required"`
	// Waste the batches recorded.
	Waste *ProductionCost `json:"waste" validate:"required"`
	// Productive, seconds and waste together.
	Total *ProductionCost `json:"total" validate:"required"`
}

// ProductionCostDepartmentCategory is what the batches of one item category scanned at one department's stations cost.
type ProductionCostDepartmentCategory struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_cost_department_category"`
	// The department. Null for batches scanned at no station.
	Department *Entity `json:"department"`
	// The category of the batches' items.
	Category *Entity `json:"category" validate:"required"`
	// First-quality output: each batch's quantity.
	Productive *ProductionCost `json:"productive" validate:"required"`
	// Second-quality output the batches recorded.
	Seconds *ProductionCost `json:"seconds" validate:"required"`
	// Waste the batches recorded.
	Waste *ProductionCost `json:"waste" validate:"required"`
	// Productive, seconds and waste together.
	Total *ProductionCost `json:"total" validate:"required"`
}

func sampleProductionCost(materials, labor, overhead, total, laborTime, produced string) *ProductionCost {
	return &ProductionCost{
		Object:    constants.ObjectTypeProductionCost,
		Materials: materials,
		Labor:     labor,
		Overhead:  overhead,
		Total:     total,
		LaborTime: laborTime,
		Produced: NewList([]ComputedQuantity{{
			Object:       constants.ObjectTypeComputedQuantity,
			Value:        produced,
			DisplayValue: produced + " ea",
			Unit:         SampleEachUnit,
		}}, PageInfo{}),
	}
}

var (
	sampleProductionCostProductive = sampleProductionCost("1200", "640", "512", "2352", "25.6", "4800")
	sampleProductionCostSeconds    = sampleProductionCost("60", "32", "25.6", "117.6", "1.28", "240")
	sampleProductionCostWaste      = sampleProductionCost("30", "16", "12.8", "58.8", "0.64", "120")
	sampleProductionCostTotal      = sampleProductionCost("1290", "688", "550.4", "2528.4", "27.52", "5160")
)

var SampleProductionCostTotals = &ProductionCostTotals{
	Object:     constants.ObjectTypeProductionCostTotals,
	Productive: sampleProductionCostProductive,
	Seconds:    sampleProductionCostSeconds,
	Waste:      sampleProductionCostWaste,
	Total:      sampleProductionCostTotal,
}

var SampleProductionCostDepartment = &ProductionCostDepartment{
	Object:     constants.ObjectTypeProductionCostDepartment,
	Department: NewEntity(SampleDepartmentID, constants.ObjectTypeDepartment, new(SampleDepartmentName), nil),
	Productive: sampleProductionCostProductive,
	Seconds:    sampleProductionCostSeconds,
	Waste:      sampleProductionCostWaste,
	Total:      sampleProductionCostTotal,
}

var SampleProductionCostCategory = &ProductionCostCategory{
	Object:     constants.ObjectTypeProductionCostCategory,
	Category:   NewEntity(SampleItemCategoryID, constants.ObjectTypeItemCategory, new(SampleItemCategoryName), nil),
	Productive: sampleProductionCostProductive,
	Seconds:    sampleProductionCostSeconds,
	Waste:      sampleProductionCostWaste,
	Total:      sampleProductionCostTotal,
}

var SampleProductionCostDepartmentCategory = &ProductionCostDepartmentCategory{
	Object:     constants.ObjectTypeProductionCostDepartmentCategory,
	Department: NewEntity(SampleDepartmentID, constants.ObjectTypeDepartment, new(SampleDepartmentName), nil),
	Category:   NewEntity(SampleItemCategoryID, constants.ObjectTypeItemCategory, new(SampleItemCategoryName), nil),
	Productive: sampleProductionCostProductive,
	Seconds:    sampleProductionCostSeconds,
	Waste:      sampleProductionCostWaste,
	Total:      sampleProductionCostTotal,
}

var SampleAnalyzeProductionCostsResponse = &AnalyzeProductionCostsResponse{
	Object:               constants.ObjectTypeAnalyzeProductionCostsResponse,
	StartsAt:             SampleAnalyticsPeriodStart,
	EndsAt:               SampleAnalyticsPeriodEnd,
	CurrencyUnit:         SampleCurrencyUnit,
	TimeUnit:             newSampleUnit("Hour", "hr", constants.UnitTypeTime),
	Totals:               SampleProductionCostTotals,
	Departments:          NewList([]ProductionCostDepartment{*SampleProductionCostDepartment}, PageInfo{}),
	Categories:           NewList([]ProductionCostCategory{*SampleProductionCostCategory}, PageInfo{}),
	DepartmentCategories: NewList([]ProductionCostDepartmentCategory{*SampleProductionCostDepartmentCategory}, PageInfo{}),
}

func (*ProductionCost) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleProductionCostProductive)
}

func (*ProductionCostTotals) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionCostTotals)
}

func (*ProductionCostDepartment) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionCostDepartment)
}

func (*ProductionCostCategory) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionCostCategory)
}

func (*ProductionCostDepartmentCategory) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductionCostDepartmentCategory)
}

func (*AnalyzeProductionCostsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAnalyzeProductionCostsResponse)
}
