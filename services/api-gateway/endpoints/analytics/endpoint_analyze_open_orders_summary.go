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

// AnalyzeOpenOrdersSummaryRequest selects the open sales orders to total.
type AnalyzeOpenOrdersSummaryRequest struct {
	OpenOrderFilters
}

// Returns the money on open sales orders: what is ordered, what of it is back-ordered, and what has been invoiced against it so far.
//
// An open order is a sales order that has been issued and not yet completed; estimates and purchase orders are never open. Only sale lines count, each valued at its quantity times its unit price, converted between the two units. Sales reps see only their own orders.
type AnalyzeOpenOrdersSummaryEndpoint struct{}

func (e *AnalyzeOpenOrdersSummaryEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeOpenOrdersSummaryRequest, *apiresource.AnalyzeOpenOrdersSummaryResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeOpenOrdersSummaryRequest, *apiresource.AnalyzeOpenOrdersSummaryResponse]{
		Title:               "Analyze Open Orders Summary",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/open-orders/summary",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		ReadOnly:            true,
		ObjectType:          constants.ObjectTypeOpenOrdersSummary,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainInvoices, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeOpenOrdersSummaryRequest) (*apiresource.AnalyzeOpenOrdersSummaryResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeOpenOrdersSummary
		},
	})
}
