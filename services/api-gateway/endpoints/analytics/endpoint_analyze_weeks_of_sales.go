package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// AnalyzeWeeksOfSalesRequest is the request to analyze weeks of sales.
type AnalyzeWeeksOfSalesRequest struct {
	// The number of weeks to use for the sales period. Defaults to 4.
	//
	// A period is a divisor of demand, so zero and negative values are rejected rather than quietly substituted with the default — a caller who asked for an impossible period should be told, not handed the answer to a different question.
	PeriodInWeeks *int32 `query:"period_in_weeks" validate:"omitempty,min=1,max=520"`
}

// Returns weeks-of-sales metrics per product line, including on-hand quantity, average weekly sales, and weeks of inventory remaining.
//
// On-hand stock is the available receipts of the line's sale products, deleted items included, net of what has been drawn against them. Sales are the quantities ordered on the account's orders issued in the trailing period. Both are stated in the product line's base unit, converting each line and receipt from its own unit.
type AnalyzeWeeksOfSalesEndpoint struct{}

func (e *AnalyzeWeeksOfSalesEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeWeeksOfSalesRequest, *apiresource.AnalyzeWeeksOfSalesResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeWeeksOfSalesRequest, *apiresource.AnalyzeWeeksOfSalesResponse]{
		Title:               "Analyze Weeks of Sales",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/core/analytics/weeks-of-sales",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainInventory, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeWeeksOfSalesRequest) (*apiresource.AnalyzeWeeksOfSalesResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeWeeksOfSales
		},
	})
}
