package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// AnalyzeQuarterlyOrdersRequest is the request to analyze quarterly order data.
type AnalyzeQuarterlyOrdersRequest struct {
	// Optional sales rep IDs to filter by.
	SalesRepIDs []string `json:"sales_rep_ids,omitempty"`
	// Optional item IDs to filter by.
	ItemIDs []string `json:"item_ids,omitempty"`
	// Optional product line IDs to filter by.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
	// Optional customer IDs to filter by.
	CustomerIDs []string `json:"customer_ids,omitempty"`
	// Optional customer group IDs to filter by.
	CustomerGroupIDs []string `json:"customer_group_ids,omitempty"`
	// Calendar years to cover, the current one included. Defaults to 5.
	YearsBack field.Optional[int32] `json:"years_back,omitzero" validate:"omitempty,min=1,max=100"`
}

// Returns the ordered value of sales orders by the year and quarter they were issued, for the last few calendar years.
//
// Each year's quarters and total are the ordered value of sale lines — quantity times unit price, converted between units — on sales orders issued in that quarter (UTC), whatever their status now. Estimates, which have not been issued, and purchase orders are left out. Customers include their child accounts. Sales reps see only their own orders.
type AnalyzeQuarterlyOrdersEndpoint struct{}

func (e *AnalyzeQuarterlyOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeQuarterlyOrdersRequest, *apiresource.AnalyzeQuarterlyOrdersResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeQuarterlyOrdersRequest, *apiresource.AnalyzeQuarterlyOrdersResponse]{
		Title:               "Analyze Quarterly Orders",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/quarterly-orders",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainInvoices, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeQuarterlyOrdersRequest) (*apiresource.AnalyzeQuarterlyOrdersResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeQuarterlyOrders
		},
	})
}
