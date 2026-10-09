package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// ExportOpenOrderLinesRequest filters which open sale lines land in the exported file.
type ExportOpenOrderLinesRequest struct {
	OpenOrderFilters
}

// Starts an export of the open sales orders' sale lines, oldest order first, and returns the job that tracks it.
//
// The file has one row per line with its customer, sales rep, product, quantities in the item's base unit, unit price and totals. The unit price is what has been invoiced per unit, or zero before anything is. Sales reps export only their own orders, and without the unit cost column. An export matching more than 50,000 lines fails with a request to narrow the filters.
type ExportOpenOrderLinesEndpoint struct{}

func (e *ExportOpenOrderLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportOpenOrderLinesRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportOpenOrderLinesRequest, *apiresource.Job]{
		Title:               "Export Open Order Lines",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/core/analytics/open-order-lines/actions/export",
		SuccessStatusCode:   http.StatusAccepted,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeJob,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead}},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportOpenOrderLinesRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(AnalyticsSvc).ExportOpenOrderLines
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
