package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// AnalyzeProductionCostsRequest is the request to cost a window's production.
type AnalyzeProductionCostsRequest struct {
	// Start of the window, inclusive: batches scanned at or after it are costed.
	StartDate time.Time `json:"starts_at" validate:"required"`
	// End of the window, inclusive: batches scanned at or before it are costed.
	EndDate time.Time `json:"ends_at" validate:"required"`
	// Restrict the report to these items' production: the items themselves and every part their steps consume, recursively upstream. Combines with `product_line_ids`.
	ItemIDs []string `json:"item_ids,omitzero"`
	// Restrict the report to the production of these product lines' items: the parts their steps consume, recursively upstream, and any of the items no step produces. An item a step produces is selected only through `item_ids`.
	//
	// Product lines that lead to no item, with no `item_ids`, do not restrict the report.
	ProductLineIDs []string `json:"product_line_ids,omitzero"`
	// Restrict the report to batches scanned at these departments' stations.
	DepartmentIDs []string `json:"department_ids,omitzero"`
	// Restrict the report to batches of items in these categories.
	CategoryIDs []string `json:"category_ids,omitzero"`
}

// Costs the production scanned over a window: overall, per department, per item category, and per department and category.
//
// Every batch scanned at a production step within the window counts, open or closed. Each kind of output is charged for the runs of its step it amounts to: a batch's productive quantity, and the seconds and waste it recorded, are each carried into the unit the step's production is entered in and divided by what one run produces. One run costs its raw material consumption (quantity plus waste allowance, at each material's unit cost) and its labor time — the step's labor time per unit, stretched by its leveling factor and allowances — priced at the step's labor rate and again at its overhead rate.
//
// Labor time is read in its own units on both sides: seconds per pair on a step producing eaches is half those seconds per each, and minutes or seconds are carried into hours before they meet a rate. The dashboard's report, for a step whose labor time is per a different unit from the one it produces in, read a figure in hours as if it were in the labor time's own unit, scaling the labor time, and the labor and overhead priced from it, by that unit's size in hours: seconds by 1/3600. A step producing 1 pr with a labor time of 2 min/ea at $10/hr is 4 minutes and $0.67 of labor here, where the dashboard reported $0.01. A labor time per a unit of another dimension than the step's output, which the dashboard could not cost at all, is read as per unit produced.
//
// Money is in `currency_unit` and labor time in `time_unit`, rounded to 10 decimal places. What was produced is stated per dimension, in its base unit.
type AnalyzeProductionCostsEndpoint struct{}

func (e *AnalyzeProductionCostsEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeProductionCostsRequest, *apiresource.AnalyzeProductionCostsResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeProductionCostsRequest, *apiresource.AnalyzeProductionCostsResponse]{
		Title:               "Analyze Production Costs",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/production-costs",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		ReadOnly:            true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAnalyzeProductionCostsResponse,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeProductionCostsRequest) (*apiresource.AnalyzeProductionCostsResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeProductionCosts
		},
	})
}
