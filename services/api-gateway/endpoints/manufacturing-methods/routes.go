package manufacturingmethodep

import (
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

const methodsRoute = "/v1/operations/manufacturing-methods"

var methodIncludeFields = []string{"item", "sales_order_line", "operations", "materials"}

func methodIncludes() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{ObjectType: constants.ObjectTypeManufacturingMethod, Fields: methodIncludeFields})
}

const operationsRoute = methodsRoute + "/{manufacturing_method_id}/operations"

var operationIncludeFields = []string{"work_center", "machines", "scanning_station"}

func operationIncludes() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{ObjectType: constants.ObjectTypeMethodOperation, Fields: operationIncludeFields})
}

const materialsRoute = methodsRoute + "/{manufacturing_method_id}/materials"

var materialIncludeFields = []string{"item", "operation"}

func materialIncludes() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{ObjectType: constants.ObjectTypeMethodMaterial, Fields: materialIncludeFields})
}
